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
