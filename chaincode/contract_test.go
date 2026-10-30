package main

import (
	"crypto/x509"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/hyperledger/fabric-chaincode-go/shimtest"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type testIdentity struct {
	id    string
	msp   string
	idErr bool
}

func (i *testIdentity) GetID() (string, error) {
	if i.idErr {
		return "", errors.New("identity unavailable")
	}
	return i.id, nil
}
func (i *testIdentity) GetMSPID() (string, error) { return i.msp, nil }
func (i *testIdentity) GetAttributeValue(string) (string, bool, error) {
	return "", false, nil
}
func (i *testIdentity) AssertAttributeValue(string, string) error { return nil }
func (i *testIdentity) GetX509Certificate() (*x509.Certificate, error) {
	return nil, nil
}

type chaincodeFixture struct {
	contract *XLabContract
	stub     *shimtest.MockStub
	ctx      *contractapi.TransactionContext
	nextTx   int
}

func newChaincodeFixture(id, msp string) *chaincodeFixture {
	stub := shimtest.NewMockStub("xlab", nil)
	ctx := new(contractapi.TransactionContext)
	ctx.SetStub(stub)
	ctx.SetClientIdentity(&testIdentity{id: id, msp: msp})
	return &chaincodeFixture{contract: new(XLabContract), stub: stub, ctx: ctx}
}

func (f *chaincodeFixture) as(id, msp string) {
	f.ctx.SetClientIdentity(&testIdentity{id: id, msp: msp})
}

func (f *chaincodeFixture) tx(t *testing.T, fn func() error) error {
	t.Helper()
	f.nextTx++
	txID := "tx-" + string(rune('a'+f.nextTx))
	f.stub.MockTransactionStart(txID)
	defer f.stub.MockTransactionEnd(txID)
	return fn()
}

func (f *chaincodeFixture) put(t *testing.T, key string, value interface{}) {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.tx(t, func() error { return f.stub.PutState(key, payload) }); err != nil {
		t.Fatal(err)
	}
}

func TestInitLedgerRequiresAdminAndRunsOnlyOnce(t *testing.T) {
	f := newChaincodeFixture("team-a", UserMSP)
	err := f.tx(t, func() error { return f.contract.InitLedger(f.ctx) })
	if err == nil || !strings.Contains(err.Error(), "权限不足") {
		t.Fatalf("expected member initialization to be rejected, got %v", err)
	}

	f.as("admin", AdminMSP)
	if err := f.tx(t, func() error { return f.contract.InitLedger(f.ctx) }); err != nil {
		t.Fatalf("first admin initialization failed: %v", err)
	}

	var equipment Equipment
	if err := json.Unmarshal(f.stub.State[equipmentPrefix+"EQ-001"], &equipment); err != nil {
		t.Fatal(err)
	}
	equipment.Status = "BORROWED"
	f.put(t, equipmentPrefix+equipment.ID, equipment)

	err = f.tx(t, func() error { return f.contract.InitLedger(f.ctx) })
	if err == nil {
		t.Fatal("expected repeated initialization to be rejected")
	}
	if err := json.Unmarshal(f.stub.State[equipmentPrefix+"EQ-001"], &equipment); err != nil {
		t.Fatal(err)
	}
	if equipment.Status != "BORROWED" {
		t.Fatalf("repeated initialization overwrote equipment state: %s", equipment.Status)
	}
}

func TestInitLedgerRejectsPartiallyPopulatedLedger(t *testing.T) {
	f := newChaincodeFixture("admin", AdminMSP)
	f.put(t, spacePrefix+"LEGACY", Space{ID: "LEGACY", Status: "OPEN"})
	err := f.tx(t, func() error { return f.contract.InitLedger(f.ctx) })
	if err == nil || !strings.Contains(err.Error(), "已存在") {
		t.Fatalf("expected partial ledger state to block initialization, got %v", err)
	}
	if f.stub.State[equipmentPrefix+"EQ-001"] != nil {
		t.Fatal("initialization wrote seed equipment into a partially populated ledger")
	}
}

func TestActivateProjectRejectsInsufficientCapacity(t *testing.T) {
	f := newChaincodeFixture("admin", AdminMSP)
	f.put(t, applicationPrefix+"APP-1", ProjectApplication{
		ID: "APP-1", ProjectName: "Large team", RequestedSeats: 99,
		StartDate: "2026-10-01", EndDate: "2026-11-01", Status: "APPROVED",
	})
	f.put(t, spacePrefix+"A1", Space{ID: "A1", Status: "OPEN", Capacity: 2})

	err := f.tx(t, func() error {
		return f.contract.ActivateProject(f.ctx, "PRJ-1", "APP-1", `["A1"]`)
	})
	if err == nil || !strings.Contains(err.Error(), "容量") {
		t.Fatalf("expected insufficient capacity error, got %v", err)
	}
}

func TestCloseProjectRejectsUnresolvedException(t *testing.T) {
	f := newChaincodeFixture("admin", AdminMSP)
	f.put(t, projectPrefix+"PRJ-1", Project{ID: "PRJ-1", Status: "ACTIVE"})
	f.put(t, borrowPrefix+"BOR-1", BorrowRecord{ID: "BOR-1", ProjectID: "PRJ-1", Status: "EXCEPTION"})

	err := f.tx(t, func() error { return f.contract.CloseProject(f.ctx, "PRJ-1", "done") })
	if err == nil || !strings.Contains(err.Error(), "未归还") {
		t.Fatalf("expected unresolved exception to block close, got %v", err)
	}

	f.put(t, borrowPrefix+"BOR-1", BorrowRecord{ID: "BOR-1", ProjectID: "PRJ-1", Status: "RESOLVED"})
	if err := f.tx(t, func() error { return f.contract.CloseProject(f.ctx, "PRJ-1", "done") }); err != nil {
		t.Fatalf("resolved exception still blocked close: %v", err)
	}
}

func TestMemberActionsRequireOrgAndObjectOwner(t *testing.T) {
	f := newChaincodeFixture("admin", AdminMSP)
	err := f.tx(t, func() error {
		return f.contract.SubmitProjectApplication(f.ctx, "APP-1", "Project", "team-a", "Team A", `[]`, "2", `[]`, "2026-10-01", "2026-11-01", "summary", "hash", "m1", "mh")
	})
	if err == nil || !strings.Contains(err.Error(), "权限不足") {
		t.Fatalf("expected Org1 application submission to be rejected, got %v", err)
	}
	f.as("team-a", UserMSP)
	if err := f.tx(t, func() error {
		return f.contract.SubmitProjectApplication(f.ctx, "APP-2", "Project", "team-a", "Team A", `[]`, "2", `[]`, "2026-10-01", "2026-11-01", "summary", "hash", "m1", "mh")
	}); err != nil {
		t.Fatalf("member application submission failed: %v", err)
	}
	var app ProjectApplication
	if err := json.Unmarshal(f.stub.State[applicationPrefix+"APP-2"], &app); err != nil {
		t.Fatal(err)
	}
	if app.OwnerIdentity != "team-a" {
		t.Fatalf("application owner identity = %q, want team-a", app.OwnerIdentity)
	}

	f.put(t, projectPrefix+"PRJ-1", Project{ID: "PRJ-1", Status: "ACTIVE", OwnerIdentity: "team-a"})
	f.as("team-b", UserMSP)
	err = f.tx(t, func() error { return f.contract.SubmitMilestone(f.ctx, "PRJ-1", "changed", "hash") })
	if err == nil || !strings.Contains(err.Error(), "归属") {
		t.Fatalf("expected cross-team milestone to be rejected, got %v", err)
	}

	f.put(t, equipmentPrefix+"EQ-1", Equipment{ID: "EQ-1", Name: "Camera", Status: "AVAILABLE"})
	err = f.tx(t, func() error {
		return f.contract.RequestEquipmentBorrow(f.ctx, "BOR-1", "EQ-1", "PRJ-1", "team-b", "Team B", "shoot", "hash", "2026-11-01")
	})
	if err == nil || !strings.Contains(err.Error(), "归属") {
		t.Fatalf("expected cross-team borrow to be rejected, got %v", err)
	}

	f.put(t, borrowPrefix+"BOR-2", BorrowRecord{ID: "BOR-2", Status: "BORROWED", BorrowerIdentity: "team-a"})
	err = f.tx(t, func() error { return f.contract.RequestEquipmentReturn(f.ctx, "BOR-2", "return") })
	if err == nil || !strings.Contains(err.Error(), "归属") {
		t.Fatalf("expected non-borrower return to be rejected, got %v", err)
	}
}

func TestMemberWriteRejectsUnverifiableClientIdentity(t *testing.T) {
	f := newChaincodeFixture("team-a", UserMSP)
	f.ctx.SetClientIdentity(&testIdentity{msp: UserMSP, idErr: true})
	err := f.tx(t, func() error {
		return f.contract.SubmitProjectApplication(f.ctx, "APP-1", "Project", "team-a", "Team A", `[]`, "2", `[]`, "2026-10-01", "2026-11-01", "summary", "hash", "m1", "mh")
	})
	if err == nil || !strings.Contains(err.Error(), "客户身份") {
		t.Fatalf("expected unverifiable identity to be rejected, got %v", err)
	}
}

func TestGetActivityLogSortsByTimestamp(t *testing.T) {
	f := newChaincodeFixture("admin", AdminMSP)
	f.put(t, activityPrefix+"tx-a", Activity{TxID: "tx-a", Timestamp: "2026-10-25T12:00:00Z"})
	f.put(t, activityPrefix+"tx-b", Activity{TxID: "tx-b", Timestamp: "2026-10-25T12:00:00Z"})
	f.put(t, activityPrefix+"tx-z", Activity{TxID: "tx-z", Timestamp: "2026-10-25T08:00:00Z"})

	items, err := f.contract.GetActivityLog(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 || items[0].TxID != "tx-b" || items[1].TxID != "tx-a" {
		t.Fatalf("expected newest activity first, got %#v", items)
	}
}

// 回归：维护入口不得把仍挂在有效借用链（REQUESTED/PENDING）上的设备改回 AVAILABLE，
// 否则可对同一设备再开第二条借用单，破坏设备独占约束。
func TestMaintenanceRejectsReleasingEquipmentWithActiveBorrow(t *testing.T) {
	f := newChaincodeFixture("admin", AdminMSP)
	f.put(t, equipmentPrefix+"EQ-1", Equipment{ID: "EQ-1", Name: "Camera", Status: "REQUESTED", ProjectID: "PRJ-1", CurrentBorrow: "BOR-1"})
	f.put(t, borrowPrefix+"BOR-1", BorrowRecord{ID: "BOR-1", EquipmentID: "EQ-1", ProjectID: "PRJ-1", Status: "PENDING"})

	err := f.tx(t, func() error {
		return f.contract.UpdateEquipmentStatus(f.ctx, "EQ-1", "AVAILABLE", "GOOD", "release")
	})
	if err == nil || !strings.Contains(err.Error(), "未结清") {
		t.Fatalf("expected maintenance to reject releasing equipment with active borrow, got %v", err)
	}

	var eq Equipment
	if err := json.Unmarshal(f.stub.State[equipmentPrefix+"EQ-1"], &eq); err != nil {
		t.Fatal(err)
	}
	if eq.Status != "REQUESTED" {
		t.Fatalf("maintenance mutated equipment status to %s despite active borrow", eq.Status)
	}
}

// 回归：续期理由不得覆盖阶段成果 milestone。
func TestExtendDoesNotOverwriteMilestone(t *testing.T) {
	f := newChaincodeFixture("admin", AdminMSP)
	f.put(t, projectPrefix+"PRJ-1", Project{
		ID: "PRJ-1", Status: "ACTIVE", StartDate: "2026-10-01", EndDate: "2026-11-01",
		Milestone: "完成可操作原型", MilestoneHash: "c4d8f2e6", SpaceIDs: []string{"A1"},
	})
	f.put(t, spacePrefix+"A1", Space{ID: "A1", Status: "OPEN", Capacity: 4})

	if err := f.tx(t, func() error { return f.contract.ExtendProject(f.ctx, "PRJ-1", "2026-12-01", "这是续期理由") }); err != nil {
		t.Fatalf("extend failed: %v", err)
	}

	var p Project
	if err := json.Unmarshal(f.stub.State[projectPrefix+"PRJ-1"], &p); err != nil {
		t.Fatal(err)
	}
	if p.Milestone != "完成可操作原型" || p.MilestoneHash != "c4d8f2e6" {
		t.Fatalf("extend overwrote milestone: %q / %q", p.Milestone, p.MilestoneHash)
	}
	if p.EndDate != "2026-12-01" {
		t.Fatalf("extend did not update endDate: %q", p.EndDate)
	}
}

// 回归：续期必须晚于当前期限，禁止缩期。
func TestExtendRejectsShrinkingPeriod(t *testing.T) {
	f := newChaincodeFixture("admin", AdminMSP)
	f.put(t, projectPrefix+"PRJ-1", Project{
		ID: "PRJ-1", Status: "ACTIVE", StartDate: "2026-10-01", EndDate: "2026-11-01", SpaceIDs: []string{"A1"},
	})
	f.put(t, spacePrefix+"A1", Space{ID: "A1", Status: "OPEN", Capacity: 4})

	err := f.tx(t, func() error { return f.contract.ExtendProject(f.ctx, "PRJ-1", "2026-10-15", "缩短") })
	if err == nil || !strings.Contains(err.Error(), "晚于当前期限") {
		t.Fatalf("expected shrinking period to be rejected, got %v", err)
	}
}

// 回归：领取交接必须校验成色枚举（GOOD / FAIR）。
func TestPickupValidatesCondition(t *testing.T) {
	f := newChaincodeFixture("admin", AdminMSP)
	f.put(t, equipmentPrefix+"EQ-1", Equipment{ID: "EQ-1", Name: "Camera", Status: "APPROVED"})
	f.put(t, borrowPrefix+"BOR-1", BorrowRecord{ID: "BOR-1", EquipmentID: "EQ-1", Status: "APPROVED"})

	err := f.tx(t, func() error { return f.contract.ConfirmEquipmentPickup(f.ctx, "BOR-1", "PRETTY_GOOD", "[]") })
	if err == nil || !strings.Contains(err.Error(), "领取成色") {
		t.Fatalf("expected invalid pickup condition to be rejected, got %v", err)
	}

	var rec BorrowRecord
	if err := json.Unmarshal(f.stub.State[borrowPrefix+"BOR-1"], &rec); err != nil {
		t.Fatal(err)
	}
	if rec.Status != "APPROVED" || rec.Condition != "" {
		t.Fatalf("failed pickup mutated borrow record: %#v", rec)
	}
}

// 回归：借用期限必须落在有效的项目周期内，不能提交已经逾期或晚于项目结束日的借用单。
func TestRequestBorrowValidatesDueDateAgainstProject(t *testing.T) {
	f := newChaincodeFixture("team-a", UserMSP)
	f.put(t, projectPrefix+"PRJ-1", Project{
		ID: "PRJ-1", Status: "ACTIVE", EndDate: "2099-12-31", OwnerIdentity: "team-a",
	})
	f.put(t, equipmentPrefix+"EQ-1", Equipment{ID: "EQ-1", Name: "Camera", Status: "AVAILABLE"})

	err := f.tx(t, func() error {
		return f.contract.RequestEquipmentBorrow(f.ctx, "BOR-PAST", "EQ-1", "PRJ-1", "team-a", "Team A", "shoot", "hash", "2000-01-01")
	})
	if err == nil || !strings.Contains(err.Error(), "不能早于今天") {
		t.Fatalf("expected past due date to be rejected, got %v", err)
	}

	f.put(t, projectPrefix+"PRJ-1", Project{
		ID: "PRJ-1", Status: "ACTIVE", EndDate: "2026-11-01", OwnerIdentity: "team-a",
	})
	err = f.tx(t, func() error {
		return f.contract.RequestEquipmentBorrow(f.ctx, "BOR-LATE", "EQ-1", "PRJ-1", "team-a", "Team A", "shoot", "hash", "2026-11-02")
	})
	if err == nil || !strings.Contains(err.Error(), "项目期限") {
		t.Fatalf("expected due date after project end to be rejected, got %v", err)
	}
}

// 回归：归还验收必须先校验结论再写状态——MockStub 没有交易回滚，
// 非法 condition 若先写字段就会污染账本；同时结论必须归一化为大写。
func TestConfirmReturnValidatesVerdictBeforeMutation(t *testing.T) {
	f := newChaincodeFixture("admin", AdminMSP)
	f.put(t, equipmentPrefix+"EQ-1", Equipment{ID: "EQ-1", Name: "Camera", Status: "BORROWED"})
	f.put(t, borrowPrefix+"BOR-1", BorrowRecord{ID: "BOR-1", EquipmentID: "EQ-1", Status: "RETURN_REQUESTED"})

	err := f.tx(t, func() error {
		return f.contract.ConfirmEquipmentReturn(f.ctx, "BOR-1", "pretty_good", "hash", "note", "[]")
	})
	if err == nil || !strings.Contains(err.Error(), "验收结论") {
		t.Fatalf("expected invalid return verdict to be rejected, got %v", err)
	}
	var rec BorrowRecord
	if err := json.Unmarshal(f.stub.State[borrowPrefix+"BOR-1"], &rec); err != nil {
		t.Fatal(err)
	}
	if rec.Status != "RETURN_REQUESTED" || rec.Condition != "" || rec.ReturnAt != "" {
		t.Fatalf("invalid verdict mutated borrow record: %#v", rec)
	}
	var eq Equipment
	if err := json.Unmarshal(f.stub.State[equipmentPrefix+"EQ-1"], &eq); err != nil {
		t.Fatal(err)
	}
	if eq.Status != "BORROWED" {
		t.Fatalf("invalid verdict mutated equipment: %#v", eq)
	}

	// 小写合法值必须归一化为大写入库，与演示模式口径一致。
	if err := f.tx(t, func() error {
		return f.contract.ConfirmEquipmentReturn(f.ctx, "BOR-1", "normal", "hash", "note", "[]")
	}); err != nil {
		t.Fatalf("lowercase 'normal' should be accepted: %v", err)
	}
	if err := json.Unmarshal(f.stub.State[borrowPrefix+"BOR-1"], &rec); err != nil {
		t.Fatal(err)
	}
	if rec.Status != "RETURNED" || rec.Condition != "NORMAL" {
		t.Fatalf("verdict not normalized/recorded: %#v", rec)
	}
}
