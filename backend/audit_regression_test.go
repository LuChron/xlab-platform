package main

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
