package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type unavailableActorStore struct{ Store }

func (unavailableActorStore) For(string) (Store, error) {
	return nil, errors.New("missing Org2 certificate")
}

func TestMemoryActivateRejectsInsufficientCapacity(t *testing.T) {
	store := newMemoryStoreWithLedger(&Ledger{
		Applications: []ProjectApplication{{
			ID: "APP-1", ProjectName: "Large team", RequestedSeats: 99,
			StartDate: "2026-10-01", EndDate: "2026-11-01", Status: "APPROVED",
		}},
		Spaces: []Space{{ID: "A1", Status: "OPEN", Capacity: 2}},
	}, "admin", AdminMSP)

	_, err := store.ActivateProject("APP-1", []string{"A1"})
	if err == nil || !strings.Contains(err.Error(), "容量") {
		t.Fatalf("expected insufficient capacity error, got %v", err)
	}
}

func TestMemoryCloseRejectsUnresolvedException(t *testing.T) {
	store := newMemoryStoreWithLedger(&Ledger{
		Projects: []Project{{ID: "PRJ-1", Status: "ACTIVE"}},
		Borrows:  []BorrowRecord{{ID: "BOR-1", ProjectID: "PRJ-1", Status: "EXCEPTION"}},
	}, "admin", AdminMSP)

	err := store.CloseProject("PRJ-1", "done")
	if err == nil || !strings.Contains(err.Error(), "未归还") {
		t.Fatalf("expected unresolved exception to block close, got %v", err)
	}
	store.data.Borrows[0].Status = "RESOLVED"
	if err := store.CloseProject("PRJ-1", "done"); err != nil {
		t.Fatalf("resolved exception still blocked close: %v", err)
	}
}

func TestMemberHandlersEnforceTeamOwnership(t *testing.T) {
	gin.SetMode(gin.TestMode)
	member := User{
		Username: "team", Role: RoleMember, Org: UserMSP,
		TeamID: "team-a", SubjectName: "林同学",
	}

	t.Run("milestone", func(t *testing.T) {
		store := newMemoryStoreWithLedger(&Ledger{
			Projects: []Project{{ID: "PRJ-1", LeadID: "team-b", Status: "ACTIVE"}},
		}, "member", UserMSP)
		status := invokeMemberHandler(t, milestoneHandler(store), member, "/projects/PRJ-1/milestone", map[string]string{"id": "PRJ-1"}, `{"note":"changed"}`)
		if status != http.StatusForbidden {
			t.Fatalf("cross-team milestone status = %d, want 403", status)
		}
	})

	t.Run("borrow", func(t *testing.T) {
		store := newMemoryStoreWithLedger(&Ledger{
			Projects:  []Project{{ID: "PRJ-1", LeadID: "team-b", Status: "ACTIVE"}},
			Equipment: []Equipment{{ID: "EQ-1", Name: "Camera", Status: "AVAILABLE"}},
		}, "member", UserMSP)
		status := invokeMemberHandler(t, createBorrowHandler(store), member, "/borrows", nil, `{"equipmentId":"EQ-1","projectId":"PRJ-1","borrower":"forged","purpose":"shoot","dueDate":"2026-11-01"}`)
		if status != http.StatusForbidden {
			t.Fatalf("cross-team borrow status = %d, want 403", status)
		}
	})

	t.Run("return", func(t *testing.T) {
		store := newMemoryStoreWithLedger(&Ledger{
			Borrows: []BorrowRecord{{ID: "BOR-1", BorrowerID: "team-b", Status: "BORROWED"}},
		}, "member", UserMSP)
		status := invokeMemberHandler(t, returnRequestHandler(store), member, "/borrows/BOR-1/return-request", map[string]string{"id": "BOR-1"}, `{}`)
		if status != http.StatusForbidden {
			t.Fatalf("non-borrower return status = %d, want 403", status)
		}
	})
}

func TestMemberHandlersDeriveIdentityFromSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	member := User{
		Username: "team", Role: RoleMember, Org: UserMSP,
		TeamID: "team-a", SubjectName: "林同学",
	}
	store := newMemoryStoreWithLedger(&Ledger{}, "member", UserMSP)

	status := invokeMemberHandler(t, createApplicationHandler(store), member, "/applications", nil,
		`{"projectName":"Project","leadId":"forged","leadName":"forged","members":[],"requestedSeats":1,"startDate":"2026-10-01","endDate":"2026-11-01","ideaSummary":"summary"}`)
	if status != http.StatusCreated {
		t.Fatalf("application status = %d, want 201", status)
	}
	if len(store.data.Applications) != 1 {
		t.Fatalf("applications = %d, want 1", len(store.data.Applications))
	}
	app := store.data.Applications[0]
	if app.LeadID != member.TeamID || app.LeadName != member.SubjectName {
		t.Fatalf("application identity was not derived from session: %+v", app)
	}

	store.data.Projects = append(store.data.Projects, Project{ID: "PRJ-1", LeadID: member.TeamID, Status: "ACTIVE"})
	store.data.Equipment = append(store.data.Equipment, Equipment{ID: "EQ-1", Name: "Camera", Status: "AVAILABLE"})
	status = invokeMemberHandler(t, createBorrowHandler(store), member, "/borrows", nil,
		`{"equipmentId":"EQ-1","projectId":"PRJ-1","borrower":"forged","purpose":"shoot","dueDate":"2026-11-01"}`)
	if status != http.StatusCreated {
		t.Fatalf("borrow status = %d, want 201", status)
	}
	borrow := store.data.Borrows[len(store.data.Borrows)-1]
	if borrow.BorrowerID != member.TeamID || borrow.Borrower != member.SubjectName {
		t.Fatalf("borrow identity was not derived from session: %+v", borrow)
	}
}

func TestWriteHandlerFailsClosedWhenSignerIsUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	member := User{
		Username: "team", Role: RoleMember, Org: UserMSP,
		TeamID: "team-a", SubjectName: "林同学",
	}
	status := invokeMemberHandler(t, createApplicationHandler(unavailableActorStore{}), member, "/applications", nil,
		`{"projectName":"Project","requestedSeats":1,"startDate":"2026-10-01","endDate":"2026-11-01","ideaSummary":"summary"}`)
	if status != http.StatusServiceUnavailable {
		t.Fatalf("unavailable signer status = %d, want 503", status)
	}
}

func TestMemoryStoreUsesRequestOrganizationInActivityLog(t *testing.T) {
	base := newMemoryStore()
	memberStore, err := base.For(UserMSP)
	if err != nil {
		t.Fatal(err)
	}
	_, err = memberStore.CreateApplication(ApplicationInput{
		ProjectName: "Project", LeadID: "team-a", LeadName: "Team A",
		RequestedSeats: 1, StartDate: "2026-12-01", EndDate: "2026-12-20", IdeaSummary: "summary",
	})
	if err != nil {
		t.Fatal(err)
	}
	ledger, err := base.Ledger()
	if err != nil {
		t.Fatal(err)
	}
	if len(ledger.Activities) == 0 || ledger.Activities[0].ActorMSP != UserMSP {
		t.Fatalf("member activity was not attributed to Org2: %#v", ledger.Activities)
	}
}

func invokeMemberHandler(t *testing.T, handler gin.HandlerFunc, user User, path string, params map[string]string, body string) int {
	t.Helper()
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("session", Session{User: user})
	for key, value := range params {
		ctx.Params = append(ctx.Params, gin.Param{Key: key, Value: value})
	}
	handler(ctx)
	return recorder.Code
}

// 回归：非法验收结论必须在校验阶段就失败，不能先把 condition/note/returnAt 写进台账。
func TestMemoryConfirmReturnValidatesBeforeMutating(t *testing.T) {
	store := newMemoryStoreWithLedger(&Ledger{
		Equipment: []Equipment{{ID: "EQ-1", Name: "Camera", Status: "RETURN_REQUESTED"}},
		Borrows: []BorrowRecord{{
			ID: "BOR-1", EquipmentID: "EQ-1", Status: "RETURN_REQUESTED",
		}},
	}, "admin", AdminMSP)

	err := store.ConfirmReturn("BOR-1", "INVALID", "should not save", nil)
	if err == nil || !strings.Contains(err.Error(), "验收结论") {
		t.Fatalf("expected invalid condition to be rejected, got %v", err)
	}

	ledger, err := store.Ledger()
	if err != nil {
		t.Fatal(err)
	}
	rec := ledger.Borrows[0]
	if rec.Status != "RETURN_REQUESTED" || rec.Condition != "" || rec.ExceptionNote != "" || rec.ReturnAt != "" {
		t.Fatalf("failed request mutated borrow record: %#v", rec)
	}
	if len(ledger.Activities) != 0 {
		t.Fatalf("failed request left activity records: %#v", ledger.Activities)
	}
}

// 回归：续期理由不得覆盖阶段成果 milestone（milestone 由 SubmitMilestone 单独管理）。
func TestMemoryExtendDoesNotOverwriteMilestone(t *testing.T) {
	store := newMemoryStoreWithLedger(&Ledger{
		Projects: []Project{{
			ID: "PRJ-1", Name: "AR", Status: "ACTIVE",
			StartDate: "2026-10-01", EndDate: "2026-11-01",
			Milestone: "完成可操作原型", MilestoneHash: "c4d8f2e6",
			SpaceIDs: []string{"A1"},
		}},
		Spaces: []Space{{ID: "A1", Status: "OPEN", Capacity: 4}},
	}, "admin", AdminMSP)

	if err := store.ExtendProject("PRJ-1", "2026-12-01", "这是续期理由"); err != nil {
		t.Fatalf("extend failed: %v", err)
	}

	ledger, _ := store.Ledger()
	p := ledger.Projects[0]
	if p.Milestone != "完成可操作原型" || p.MilestoneHash != "c4d8f2e6" {
		t.Fatalf("extend overwrote milestone: %q / %q", p.Milestone, p.MilestoneHash)
	}
	if p.EndDate != "2026-12-01" {
		t.Fatalf("extend did not update endDate: %q", p.EndDate)
	}
}

// 回归：续期必须晚于当前期限，不能通过"缩期"绕过空间冲突校验。
func TestMemoryExtendRejectsShrinkingPeriod(t *testing.T) {
	store := newMemoryStoreWithLedger(&Ledger{
		Projects: []Project{{
			ID: "PRJ-1", Name: "AR", Status: "ACTIVE",
			StartDate: "2026-10-01", EndDate: "2026-11-01",
			SpaceIDs: []string{"A1"},
		}},
		Spaces: []Space{{ID: "A1", Status: "OPEN", Capacity: 4}},
	}, "admin", AdminMSP)

	err := store.ExtendProject("PRJ-1", "2026-10-15", "缩短")
	if err == nil || !strings.Contains(err.Error(), "晚于当前期限") {
		t.Fatalf("expected shrinking period to be rejected, got %v", err)
	}
}

// 回归：领取交接必须校验成色枚举（GOOD / FAIR），任意字符串不得入库。
func TestMemoryConfirmPickupValidatesCondition(t *testing.T) {
	store := newMemoryStoreWithLedger(&Ledger{
		Equipment: []Equipment{{ID: "EQ-1", Name: "Camera", Status: "APPROVED"}},
		Borrows: []BorrowRecord{{
			ID: "BOR-1", EquipmentID: "EQ-1", Status: "APPROVED",
		}},
	}, "admin", AdminMSP)

	err := store.ConfirmPickup("BOR-1", "PRETTY_GOOD", nil)
	if err == nil || !strings.Contains(err.Error(), "领取成色") {
		t.Fatalf("expected invalid pickup condition to be rejected, got %v", err)
	}

	ledger, _ := store.Ledger()
	rec := ledger.Borrows[0]
	if rec.Status != "APPROVED" || rec.Condition != "" || rec.PickupAt != "" {
		t.Fatalf("failed pickup mutated borrow record: %#v", rec)
	}
	if ledger.Equipment[0].Status != "APPROVED" {
		t.Fatalf("failed pickup mutated equipment status: %s", ledger.Equipment[0].Status)
	}
}

// 回归：公开账本不能泄露申请单和借用人信息；项目团队只能看到本团队记录。
func TestLedgerHandlerScopesPrivateRecords(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := newMemoryStoreWithLedger(&Ledger{
		Applications: []ProjectApplication{
			{ID: "APP-A", LeadID: "team-a"},
			{ID: "APP-B", LeadID: "team-b"},
		},
		Borrows: []BorrowRecord{
			{ID: "BOR-A", BorrowerID: "team-a"},
			{ID: "BOR-B", BorrowerID: "team-b"},
		},
	}, "admin", AdminMSP)

	invoke := func(user *User) Ledger {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/api/ledger", nil)
		if user != nil {
			ctx.Set("session", Session{User: *user})
		}
		ledgerHandler(store)(ctx)
		if recorder.Code != http.StatusOK {
			t.Fatalf("ledger status = %d, want 200", recorder.Code)
		}
		var got Ledger
		if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		return got
	}

	visitor := invoke(nil)
	if len(visitor.Applications) != 0 || len(visitor.Borrows) != 0 {
		t.Fatalf("visitor received private records: applications=%d borrows=%d", len(visitor.Applications), len(visitor.Borrows))
	}

	member := invoke(&User{Role: RoleMember, TeamID: "team-a"})
	if len(member.Applications) != 1 || member.Applications[0].ID != "APP-A" {
		t.Fatalf("member applications were not scoped: %#v", member.Applications)
	}
	if len(member.Borrows) != 1 || member.Borrows[0].ID != "BOR-A" {
		t.Fatalf("member borrows were not scoped: %#v", member.Borrows)
	}

	admin := invoke(&User{Role: RoleAdmin})
	if len(admin.Applications) != 2 || len(admin.Borrows) != 2 {
		t.Fatalf("admin ledger was unexpectedly filtered: applications=%d borrows=%d", len(admin.Applications), len(admin.Borrows))
	}
}

// 回归：链上活动流水必须按身份裁剪。团队只见本团队申请/项目/借用相关记录
// （外加全局台账事件）；访客不得读取业务正文 detail；管理方保留全量。
func TestLedgerHandlerScopesActivities(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := newMemoryStoreWithLedger(&Ledger{
		Applications: []ProjectApplication{
			{ID: "APP-A", LeadID: "team-a"},
			{ID: "APP-B", LeadID: "team-b"},
		},
		Borrows: []BorrowRecord{
			{ID: "BOR-A", BorrowerID: "team-a"},
			{ID: "BOR-B", BorrowerID: "team-b"},
		},
		Projects: []Project{
			{ID: "PRJ-A", LeadID: "team-a"},
			{ID: "PRJ-B", LeadID: "team-b"},
		},
		Activities: []Activity{
			{TxID: "tx-app-a", Action: "SUBMIT_APPLICATION", ObjectType: "application", ObjectID: "APP-A", Detail: "团队A项目名"},
			{TxID: "tx-app-b", Action: "SUBMIT_APPLICATION", ObjectType: "application", ObjectID: "APP-B", Detail: "团队B项目名"},
			{TxID: "tx-bor-a", Action: "REQUEST_BORROW", ObjectType: "borrow", ObjectID: "BOR-A", Detail: "EQ-1 → 团队A成员"},
			{TxID: "tx-bor-b", Action: "REQUEST_BORROW", ObjectType: "borrow", ObjectID: "BOR-B", Detail: "EQ-2 → 团队B成员"},
			{TxID: "tx-prj-a", Action: "SUBMIT_MILESTONE", ObjectType: "project", ObjectID: "PRJ-A", Detail: "团队A里程碑"},
			{TxID: "tx-prj-b", Action: "SUBMIT_MILESTONE", ObjectType: "project", ObjectID: "PRJ-B", Detail: "团队B里程碑"},
			{TxID: "tx-init", Action: "INIT_LEDGER", ObjectType: "ledger", ObjectID: "xlab", Detail: "初始化台账"},
		},
	}, "admin", AdminMSP)

	invoke := func(user *User) []Activity {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/api/ledger", nil)
		if user != nil {
			ctx.Set("session", Session{User: *user})
		}
		ledgerHandler(store)(ctx)
		if recorder.Code != http.StatusOK {
			t.Fatalf("ledger status = %d, want 200", recorder.Code)
		}
		var got Ledger
		if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		return got.Activities
	}

	visitor := invoke(nil)
	if len(visitor) != 7 {
		t.Fatalf("visitor activity count = %d, want 7 (流水公开可核验)", len(visitor))
	}
	for _, a := range visitor {
		if a.Detail != "" {
			t.Fatalf("visitor activity leaked detail: %#v", a)
		}
	}

	member := invoke(&User{Role: RoleMember, TeamID: "team-a"})
	if len(member) != 4 {
		t.Fatalf("member activity count = %d, want 4 (本团队3条+全局1条): %#v", len(member), member)
	}
	other := map[string]bool{"APP-B": true, "BOR-B": true, "PRJ-B": true}
	for _, a := range member {
		if other[a.ObjectID] {
			t.Fatalf("member saw other-team activity: %#v", a)
		}
	}

	admin := invoke(&User{Role: RoleAdmin})
	if len(admin) != 7 {
		t.Fatalf("admin activity count = %d, want 7", len(admin))
	}
	for _, a := range admin {
		if a.Detail == "" {
			t.Fatalf("admin activity detail unexpectedly redacted: %#v", a)
		}
	}
}

// 回归：预计归还日不能早于当天，也不能晚于关联项目期限。
func TestMemoryRequestBorrowValidatesDueDateAgainstProject(t *testing.T) {
	store := newMemoryStoreWithLedger(&Ledger{
		Projects:  []Project{{ID: "PRJ-1", Status: "ACTIVE", EndDate: "2099-12-31"}},
		Equipment: []Equipment{{ID: "EQ-1", Name: "Camera", Status: "AVAILABLE"}},
	}, "member", UserMSP)

	base := BorrowInput{EquipmentID: "EQ-1", ProjectID: "PRJ-1", Borrower: "Team A", Purpose: "shoot"}
	base.DueDate = "2000-01-01"
	if _, err := store.RequestBorrow(base); err == nil || !strings.Contains(err.Error(), "不能早于今天") {
		t.Fatalf("expected past due date to be rejected, got %v", err)
	}

	store.data.Projects[0].EndDate = "2026-11-01"
	base.DueDate = "2026-11-02"
	if _, err := store.RequestBorrow(base); err == nil || !strings.Contains(err.Error(), "项目期限") {
		t.Fatalf("expected due date after project end to be rejected, got %v", err)
	}
}

// 回归：团队不能通过历史接口绕过账本列表过滤，读取其他团队的申请或借用单。
func TestHistoryHandlerRejectsCrossTeamPrivateRecords(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := newMemoryStoreWithLedger(&Ledger{
		Applications: []ProjectApplication{{ID: "APP-B", LeadID: "team-b"}},
		Borrows:      []BorrowRecord{{ID: "BOR-B", BorrowerID: "team-b"}},
	}, "member", UserMSP)
	member := User{Role: RoleMember, TeamID: "team-a"}

	for _, tc := range []struct {
		objectType string
		id         string
	}{
		{objectType: "application", id: "APP-B"},
		{objectType: "borrow", id: "BOR-B"},
	} {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/api/history/"+tc.objectType+"/"+tc.id, nil)
		ctx.Params = []gin.Param{{Key: "type", Value: tc.objectType}, {Key: "id", Value: tc.id}}
		ctx.Set("session", Session{User: member})
		historyHandler(store)(ctx)
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("cross-team %s history status = %d, want 403", tc.objectType, recorder.Code)
		}
	}
}

// 回归：项目历史不得绕过归属校验。团队只能读自己项目的历史，
// 其他团队的项目按 403 处理（与申请/借用历史同一规则）。
func TestHistoryHandlerRejectsCrossTeamProjectHistory(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := newMemoryStoreWithLedger(&Ledger{
		Projects: []Project{{ID: "PRJ-B", LeadID: "team-b"}},
	}, "member", UserMSP)
	member := User{Role: RoleMember, TeamID: "team-a"}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/history/project/PRJ-B", nil)
	ctx.Params = []gin.Param{{Key: "type", Value: "project"}, {Key: "id", Value: "PRJ-B"}}
	ctx.Set("session", Session{User: member})
	historyHandler(store)(ctx)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("cross-team project history status = %d, want 403, body: %s", recorder.Code, recorder.Body.String())
	}
}

// 回归：过期的会话必须从内存中移除，避免长期运行时 sessions map 无界增长。
func TestAuthServiceResolveRemovesExpiredSessions(t *testing.T) {
	auth := NewAuthService()
	sess, err := auth.Login("team", "xlab2026")
	if err != nil {
		t.Fatal(err)
	}

	auth.mu.Lock()
	auth.sessions[sess.Token] = Session{Token: sess.Token, ExpiresAt: time.Now().Add(-time.Minute)}
	auth.sessions["expired-other"] = Session{Token: "expired-other", ExpiresAt: time.Now().Add(-time.Hour)}
	auth.sessions["alive-other"] = Session{Token: "alive-other", ExpiresAt: time.Now().Add(time.Hour)}
	auth.mu.Unlock()

	if _, ok := auth.Resolve(sess.Token); ok {
		t.Fatal("expired session should not resolve")
	}
	auth.mu.RLock()
	_, expiredGone := auth.sessions[sess.Token]
	_, otherGone := auth.sessions["expired-other"]
	_, aliveKept := auth.sessions["alive-other"]
	auth.mu.RUnlock()
	if expiredGone {
		t.Fatal("expired session was not removed from memory")
	}
	if otherGone {
		t.Fatal("other expired session was not removed")
	}
	if !aliveKept {
		t.Fatal("live session must be kept")
	}
}
