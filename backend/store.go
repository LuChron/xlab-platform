package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// Store 是账本访问的统一抽象。
// 两个实现：
//   - memoryStore：演示兜底，进程内保存，Fabric 未启动时也能完整走通业务
//   - fabricStore：真实调用 Fabric Gateway
//
// 两者行为与错误语义保持一致，前端无需区分。
type Store interface {
	Ledger() (*Ledger, error)

	CreateApplication(input ApplicationInput) (*ProjectApplication, error)
	ReviewApplication(id string, approved bool, note string) error
	ActivateProject(appID string, spaceIDs []string) (*Project, error)
	ExtendProject(projectID, newEndDate, note string) error
	CloseProject(projectID, note string) error
	SubmitMilestone(projectID, milestone string) error

	RequestBorrow(input BorrowInput) (*BorrowRecord, error)
	ReviewBorrow(id string, approved bool, note string) error
	ConfirmPickup(id, condition string, photos []string) error
	RequestReturn(id, note string) error
	ConfirmReturn(id, condition, note string, photos []string) error
	ResolveException(id, note string) error

	History(objectType, id string) ([]HistoryEntry, error)

	// Chain 返回真实链状态（高度 + 最近区块哈希链），供"链上记录"页展示真实出块。
	// 演示模式没有区块，返回空信息，前端据 Mode 判断是否展示。
	Chain() (*ChainInfo, error)

	Mode() string
	Close()
}

// ActorAware 可选能力：按调用方所属组织切换签名身份。
// 演示模式没有真实证书，不实现该接口，请求原样使用同一个内存账本。
type ActorAware interface {
	For(mspID string) (Store, error)
}

// HistoryEntry 对象变更历史
type HistoryEntry struct {
	TxID      string `json:"txId"`
	Timestamp string `json:"timestamp"`
	IsDelete  bool   `json:"isDelete"`
	Value     string `json:"value"`
}

// ApplicationInput 入驻申请入参
type ApplicationInput struct {
	ProjectName    string   `json:"projectName"`
	LeadID         string   `json:"leadId"`
	LeadName       string   `json:"leadName"`
	Members        []string `json:"members"`
	RequestedSeats int      `json:"requestedSeats"`
	PreferredSpace []string `json:"preferredSpace"`
	StartDate      string   `json:"startDate"`
	EndDate        string   `json:"endDate"`
	IdeaSummary    string   `json:"ideaSummary"`
	Milestone      string   `json:"milestone"`
}

// BorrowInput 借用申请入参
type BorrowInput struct {
	EquipmentID string `json:"equipmentId"`
	ProjectID   string `json:"projectId"`
	BorrowerID  string `json:"-"`
	Borrower    string `json:"borrower"`
	Purpose     string `json:"purpose"`
	DueDate     string `json:"dueDate"`
}

// ---------------- 工具 ----------------

func nowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// hashOf 计算内容摘要，链上只存这个值
func hashOf(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte("\x00"))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func pseudoTxID(seed string) string {
	return hashOf(seed, nowISO(), fmt.Sprintf("%d", time.Now().UnixNano()))
}

// nextID 生成不与既有记录冲突的业务编号。
// 种子数据里已有 APP-2026-001/002，直接用计数器会撞号，因此这里持续递增到空位。
func (m *memoryStore) nextID(prefix string, taken func(string) bool) string {
	year := time.Now().Format("2006")
	for {
		(*m.seq)++
		id := fmt.Sprintf("%s-%s-%03d", prefix, year, *m.seq)
		if !taken(id) {
			return id
		}
	}
}

func (m *memoryStore) applicationExists(id string) bool {
	for _, a := range m.data.Applications {
		if a.ID == id {
			return true
		}
	}
	return false
}

func (m *memoryStore) projectExists(id string) bool {
	for _, p := range m.data.Projects {
		if p.ID == id {
			return true
		}
	}
	return false
}

func (m *memoryStore) borrowExists(id string) bool {
	for _, b := range m.data.Borrows {
		if b.ID == id {
			return true
		}
	}
	return false
}

// ---------------- memoryStore ----------------

type memoryStore struct {
	mu       *sync.RWMutex
	data     *Ledger
	seq      *int
	actor    string
	actorMSP string
}

func newMemoryStore() *memoryStore {
	return newMemoryStoreWithLedger(seedDemo(), "Org1MSP/Admin", AdminMSP)
}

func newMemoryStoreWithLedger(data *Ledger, actor, actorMSP string) *memoryStore {
	seq := 0
	return &memoryStore{
		mu: new(sync.RWMutex), data: data, seq: &seq,
		actor: actor, actorMSP: actorMSP,
	}
}

func (m *memoryStore) Mode() string { return "demo" }

// Chain 演示模式没有区块，返回空信息，前端据 Mode 判断不展示区块视图
func (m *memoryStore) Chain() (*ChainInfo, error) {
	return &ChainInfo{Blocks: []Block{}}, nil
}

// For 演示模式没有真实证书，但仍按请求组织记录活动归属。
func (m *memoryStore) For(mspID string) (Store, error) {
	actors := map[string]string{
		AdminMSP: "Org1MSP/Admin",
		UserMSP:  "Org2MSP/TeamUser",
	}
	actor, ok := actors[mspID]
	if !ok {
		return nil, fmt.Errorf("未配置的组织 %s", mspID)
	}
	return &memoryStore{
		mu: m.mu, data: m.data, seq: m.seq,
		actor: actor, actorMSP: mspID,
	}, nil
}
func (m *memoryStore) Close() {}

func (m *memoryStore) Ledger() (*Ledger, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	snapshot := *m.data
	snapshot.Equipment = append([]Equipment(nil), m.data.Equipment...)
	snapshot.Spaces = append([]Space(nil), m.data.Spaces...)
	snapshot.Projects = append([]Project(nil), m.data.Projects...)
	snapshot.Applications = append([]ProjectApplication(nil), m.data.Applications...)
	snapshot.Borrows = append([]BorrowRecord(nil), m.data.Borrows...)
	snapshot.Allocations = append([]SpaceAllocation(nil), m.data.Allocations...)
	snapshot.Activities = append([]Activity(nil), m.data.Activities...)
	for i := range snapshot.Borrows {
		snapshot.Borrows[i].Overdue = isOverdue(snapshot.Borrows[i])
	}
	sort.Slice(snapshot.Equipment, func(i, j int) bool { return snapshot.Equipment[i].ID < snapshot.Equipment[j].ID })
	sort.Slice(snapshot.Spaces, func(i, j int) bool { return snapshot.Spaces[i].ID < snapshot.Spaces[j].ID })
	return &snapshot, nil
}

func (m *memoryStore) log(action, objType, objID, detail string) {
	m.data.Activities = append([]Activity{{
		TxID: pseudoTxID(objID + action), Action: action, ObjectType: objType,
		ObjectID: objID, Actor: m.actor, ActorMSP: m.actorMSP, Detail: detail, Timestamp: nowISO(),
	}}, m.data.Activities...)
	if len(m.data.Activities) > 200 {
		m.data.Activities = m.data.Activities[:200]
	}
}

func (m *memoryStore) CreateApplication(in ApplicationInput) (*ProjectApplication, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if strings.TrimSpace(in.ProjectName) == "" {
		return nil, fmt.Errorf("项目名称不能为空")
	}
	if strings.TrimSpace(in.LeadName) == "" {
		return nil, fmt.Errorf("负责人不能为空")
	}
	if in.RequestedSeats < 1 {
		return nil, fmt.Errorf("申请工位数必须大于 0")
	}
	if len(in.Members) == 0 {
		in.Members = []string{in.LeadName}
	}
	if _, _, err := parsePeriod(in.StartDate, in.EndDate); err != nil {
		return nil, err
	}
	id := m.nextID("APP", m.applicationExists)
	now := nowISO()
	app := ProjectApplication{
		ID: id, ProjectName: in.ProjectName, LeadID: in.LeadID, LeadName: in.LeadName,
		Members: in.Members, MemberCount: len(in.Members), RequestedSeats: in.RequestedSeats,
		PreferredSpace: in.PreferredSpace, StartDate: in.StartDate, EndDate: in.EndDate,
		IdeaSummary: in.IdeaSummary, IdeaHash: hashOf(in.IdeaSummary),
		Milestone: in.Milestone, MilestoneHash: hashOf(in.Milestone),
		Status: "SUBMITTED", CreatedAt: now, UpdatedAt: now,
	}
	m.data.Applications = append(m.data.Applications, app)
	m.log("SUBMIT_APPLICATION", "application", id, in.ProjectName)
	recomputeStats(m.data)
	return &app, nil
}

func (m *memoryStore) ReviewApplication(id string, approved bool, note string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.data.Applications {
		if m.data.Applications[i].ID != id {
			continue
		}
		if m.data.Applications[i].Status != "SUBMITTED" {
			return fmt.Errorf("申请当前状态为 %s，不可重复评审", m.data.Applications[i].Status)
		}
		now := nowISO()
		if approved {
			m.data.Applications[i].Status = "APPROVED"
		} else {
			m.data.Applications[i].Status = "REJECTED"
		}
		m.data.Applications[i].Reviewer = m.actor
		m.data.Applications[i].ReviewNote = note
		m.data.Applications[i].ReviewedAt = now
		m.data.Applications[i].UpdatedAt = now
		m.log("REVIEW_APPLICATION", "application", id, fmt.Sprintf("approved=%v", approved))
		recomputeStats(m.data)
		return nil
	}
	return fmt.Errorf("申请 %s 不存在", id)
}

func (m *memoryStore) ActivateProject(appID string, spaceIDs []string) (*Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var app *ProjectApplication
	for i := range m.data.Applications {
		if m.data.Applications[i].ID == appID {
			app = &m.data.Applications[i]
			break
		}
	}
	if app == nil {
		return nil, fmt.Errorf("申请 %s 不存在", appID)
	}
	if app.Status != "APPROVED" {
		return nil, fmt.Errorf("申请尚未通过评审，当前状态 %s", app.Status)
	}
	if len(spaceIDs) == 0 {
		return nil, fmt.Errorf("必须选择至少一个空间")
	}
	capacity := 0
	seenSpaces := map[string]bool{}
	for _, sid := range spaceIDs {
		if seenSpaces[sid] {
			return nil, fmt.Errorf("空间 %s 重复选择", sid)
		}
		seenSpaces[sid] = true
		sp := m.findSpace(sid)
		if sp == nil {
			return nil, fmt.Errorf("空间 %s 不存在", sid)
		}
		if sp.Status != "OPEN" {
			return nil, fmt.Errorf("空间 %s 当前不可分配", sid)
		}
		if m.hasConflict(sid, app.StartDate, app.EndDate, "") {
			return nil, fmt.Errorf("空间 %s 在 %s 至 %s 已被占用", sid, app.StartDate, app.EndDate)
		}
		capacity += sp.Capacity
	}
	if capacity < app.RequestedSeats {
		return nil, fmt.Errorf("所选空间容量 %d 低于申请工位数 %d", capacity, app.RequestedSeats)
	}
	projectID := m.nextID("PRJ", m.projectExists)
	now := nowISO()
	proj := Project{
		ID: projectID, ApplicationID: app.ID, Name: app.ProjectName,
		LeadID: app.LeadID, LeadName: app.LeadName, Members: app.Members, MemberCount: app.MemberCount,
		Summary: app.IdeaSummary, Status: "ACTIVE", SpaceIDs: spaceIDs,
		StartDate: app.StartDate, EndDate: app.EndDate,
		Milestone: app.Milestone, MilestoneHash: app.MilestoneHash,
		CreatedAt: now, UpdatedAt: now,
	}
	m.data.Projects = append(m.data.Projects, proj)
	for _, sid := range spaceIDs {
		m.data.Allocations = append(m.data.Allocations, SpaceAllocation{
			ID: projectID + ":" + sid, SpaceID: sid, ProjectID: projectID,
			StartDate: app.StartDate, EndDate: app.EndDate, Status: "ACTIVE", CreatedAt: now,
		})
	}
	app.Status, app.ProjectID, app.UpdatedAt = "ACTIVATED", projectID, now
	m.log("ACTIVATE_PROJECT", "project", projectID, "空间: "+strings.Join(spaceIDs, ","))
	recomputeStats(m.data)
	return &proj, nil
}

func (m *memoryStore) ExtendProject(projectID, newEndDate, note string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.data.Projects {
		if m.data.Projects[i].ID != projectID {
			continue
		}
		p := &m.data.Projects[i]
		if p.Status != "ACTIVE" {
			return fmt.Errorf("项目当前状态为 %s，不能续期", p.Status)
		}
		if _, _, err := parsePeriod(p.StartDate, newEndDate); err != nil {
			return err
		}
		for _, sid := range p.SpaceIDs {
			if m.hasConflict(sid, p.EndDate, newEndDate, projectID) {
				return fmt.Errorf("空间 %s 在延期区间内已被其他项目占用", sid)
			}
		}
		old := p.EndDate
		p.EndDate = newEndDate
		p.Extensions++
		p.UpdatedAt = nowISO()
		if note != "" {
			p.Milestone = note
		}
		for j := range m.data.Allocations {
			if m.data.Allocations[j].ProjectID == projectID {
				m.data.Allocations[j].EndDate = newEndDate
			}
		}
		m.log("EXTEND_PROJECT", "project", projectID, old+" → "+newEndDate)
		return nil
	}
	return fmt.Errorf("项目 %s 不存在", projectID)
}

func (m *memoryStore) CloseProject(projectID, note string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.data.Projects {
		if m.data.Projects[i].ID != projectID {
			continue
		}
		p := &m.data.Projects[i]
		if p.Status == "CLOSED" {
			return fmt.Errorf("项目已结项")
		}
		out := 0
		for _, b := range m.data.Borrows {
			if b.ProjectID == projectID {
				switch b.Status {
				case "BORROWED", "RETURN_REQUESTED", "APPROVED", "PENDING", "EXCEPTION":
					out++
				}
			}
		}
		if out > 0 {
			return fmt.Errorf("项目仍有 %d 件设备未归还，无法结项", out)
		}
		for j := range m.data.Allocations {
			if m.data.Allocations[j].ProjectID == projectID {
				m.data.Allocations[j].Status = "RELEASED"
			}
		}
		p.Status, p.CloseNote, p.UpdatedAt = "CLOSED", note, nowISO()
		m.log("CLOSE_PROJECT", "project", projectID, note)
		recomputeStats(m.data)
		return nil
	}
	return fmt.Errorf("项目 %s 不存在", projectID)
}

func (m *memoryStore) SubmitMilestone(projectID, milestone string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.data.Projects {
		if m.data.Projects[i].ID != projectID {
			continue
		}
		if m.data.Projects[i].Status != "ACTIVE" {
			return fmt.Errorf("项目当前状态为 %s，不能提交成果", m.data.Projects[i].Status)
		}
		m.data.Projects[i].Milestone = milestone
		m.data.Projects[i].MilestoneHash = hashOf(milestone)
		m.data.Projects[i].UpdatedAt = nowISO()
		m.log("SUBMIT_MILESTONE", "project", projectID, milestone)
		return nil
	}
	return fmt.Errorf("项目 %s 不存在", projectID)
}

func (m *memoryStore) RequestBorrow(in BorrowInput) (*BorrowRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if strings.TrimSpace(in.Borrower) == "" || strings.TrimSpace(in.Purpose) == "" {
		return nil, fmt.Errorf("借用人与用途不能为空")
	}
	if _, err := time.Parse("2006-01-02", in.DueDate); err != nil {
		return nil, fmt.Errorf("预计归还日期格式应为 YYYY-MM-DD")
	}
	eq := m.findEquipment(in.EquipmentID)
	if eq == nil {
		return nil, fmt.Errorf("设备 %s 不存在", in.EquipmentID)
	}
	if eq.Status != "AVAILABLE" {
		return nil, fmt.Errorf("设备当前状态为 %s，不可申请", eq.Status)
	}
	if strings.TrimSpace(in.ProjectID) == "" {
		return nil, fmt.Errorf("借用设备必须关联已入驻项目")
	}
	found := false
	for _, p := range m.data.Projects {
		if p.ID == in.ProjectID {
			if p.Status != "ACTIVE" {
				return nil, fmt.Errorf("关联项目未处于入驻状态（%s）", p.Status)
			}
			found = true
		}
	}
	if !found {
		return nil, fmt.Errorf("关联项目 %s 不存在", in.ProjectID)
	}
	id := m.nextID("BR", m.borrowExists)
	now := nowISO()
	rec := BorrowRecord{
		ID: id, EquipmentID: in.EquipmentID, EquipmentName: eq.Name, ProjectID: in.ProjectID,
		BorrowerID: in.BorrowerID, Borrower: in.Borrower, Purpose: in.Purpose, PurposeHash: hashOf(in.Purpose),
		DueDate: in.DueDate, Status: "PENDING", CreatedAt: now, UpdatedAt: now,
	}
	m.data.Borrows = append(m.data.Borrows, rec)
	eq.Status, eq.ProjectID, eq.CurrentBorrow, eq.UpdatedAt = "REQUESTED", in.ProjectID, id, now
	m.log("REQUEST_BORROW", "borrow", id, in.EquipmentID+" → "+in.Borrower)
	recomputeStats(m.data)
	return &rec, nil
}

func (m *memoryStore) ReviewBorrow(id string, approved bool, note string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.data.Borrows {
		if m.data.Borrows[i].ID != id {
			continue
		}
		rec := &m.data.Borrows[i]
		if rec.Status != "PENDING" {
			return fmt.Errorf("借用单当前状态为 %s，不可审批", rec.Status)
		}
		eq := m.findEquipment(rec.EquipmentID)
		if eq == nil {
			return fmt.Errorf("设备不存在")
		}
		now := nowISO()
		rec.Reviewer, rec.ReviewNote, rec.ReviewedAt, rec.UpdatedAt = m.actor, note, now, now
		if approved {
			rec.Status, eq.Status = "APPROVED", "APPROVED"
		} else {
			rec.Status, eq.Status, eq.ProjectID, eq.CurrentBorrow = "REJECTED", "AVAILABLE", "", ""
		}
		m.log("REVIEW_BORROW", "borrow", id, fmt.Sprintf("approved=%v", approved))
		recomputeStats(m.data)
		return nil
	}
	return fmt.Errorf("借用单 %s 不存在", id)
}

func (m *memoryStore) ConfirmPickup(id, condition string, photos []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.data.Borrows {
		if m.data.Borrows[i].ID != id {
			continue
		}
		rec := &m.data.Borrows[i]
		if rec.Status != "APPROVED" {
			return fmt.Errorf("借用单当前状态为 %s，不能确认领取", rec.Status)
		}
		eq := m.findEquipment(rec.EquipmentID)
		if eq == nil {
			return fmt.Errorf("设备不存在")
		}
		now := nowISO()
		rec.Status, rec.PickupBy, rec.PickupAt, rec.PickupPhotos = "BORROWED", m.actor, now, photos
		rec.Condition, rec.UpdatedAt = condition, now
		eq.Status, eq.TotalBorrows, eq.UpdatedAt = "BORROWED", eq.TotalBorrows+1, now
		m.log("PICKUP_EQUIPMENT", "borrow", id, rec.EquipmentID)
		recomputeStats(m.data)
		return nil
	}
	return fmt.Errorf("借用单 %s 不存在", id)
}

func (m *memoryStore) RequestReturn(id, note string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.data.Borrows {
		if m.data.Borrows[i].ID != id {
			continue
		}
		rec := &m.data.Borrows[i]
		if rec.Status != "BORROWED" {
			return fmt.Errorf("借用单当前状态为 %s，不能发起归还", rec.Status)
		}
		rec.Status, rec.ReturnBy, rec.ExceptionNote, rec.UpdatedAt = "RETURN_REQUESTED", m.actor, note, nowISO()
		m.log("REQUEST_RETURN", "borrow", id, note)
		return nil
	}
	return fmt.Errorf("借用单 %s 不存在", id)
}

func (m *memoryStore) ConfirmReturn(id, condition, note string, photos []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.data.Borrows {
		if m.data.Borrows[i].ID != id {
			continue
		}
		rec := &m.data.Borrows[i]
		if rec.Status != "RETURN_REQUESTED" {
			return fmt.Errorf("借用单当前状态为 %s，不能验收", rec.Status)
		}
		eq := m.findEquipment(rec.EquipmentID)
		if eq == nil {
			return fmt.Errorf("设备不存在")
		}
		now := nowISO()
		rec.Condition, rec.ExceptionNote, rec.ReturnAt, rec.ReturnPhotos = condition, note, now, photos
		rec.EvidenceHash = hashOf(condition, note)
		rec.UpdatedAt = now
		switch strings.ToUpper(condition) {
		case "NORMAL":
			rec.Status, eq.Status, eq.ProjectID, eq.CurrentBorrow = "RETURNED", "AVAILABLE", "", ""
		case "DAMAGED", "MISSING":
			rec.Status, eq.Status, eq.ProjectID, eq.CurrentBorrow = "EXCEPTION", "MAINTENANCE", "", ""
			eq.Condition = "DAMAGED"
		default:
			return fmt.Errorf("验收结论必须是 NORMAL / DAMAGED / MISSING")
		}
		m.log("CONFIRM_RETURN", "borrow", id, condition)
		recomputeStats(m.data)
		return nil
	}
	return fmt.Errorf("借用单 %s 不存在", id)
}

func (m *memoryStore) ResolveException(id, note string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.data.Borrows {
		if m.data.Borrows[i].ID != id {
			continue
		}
		rec := &m.data.Borrows[i]
		if rec.Status != "EXCEPTION" {
			return fmt.Errorf("借用单当前状态为 %s，不是异常单", rec.Status)
		}
		eq := m.findEquipment(rec.EquipmentID)
		if eq == nil {
			return fmt.Errorf("设备不存在")
		}
		now := nowISO()
		rec.Status, rec.ExceptionNote, rec.UpdatedAt = "RESOLVED", note, now
		eq.Status, eq.Condition, eq.UpdatedAt = "AVAILABLE", "GOOD", now
		m.log("RESOLVE_EXCEPTION", "borrow", id, note)
		recomputeStats(m.data)
		return nil
	}
	return fmt.Errorf("借用单 %s 不存在", id)
}

func (m *memoryStore) History(objectType, id string) ([]HistoryEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	entries := []HistoryEntry{}
	for _, a := range m.data.Activities {
		if a.ObjectType == objectType && a.ObjectID == id {
			payload, _ := json.Marshal(a)
			entries = append(entries, HistoryEntry{TxID: a.TxID, Timestamp: a.Timestamp, Value: string(payload)})
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Timestamp > entries[j].Timestamp })
	return entries, nil
}

func (m *memoryStore) findSpace(id string) *Space {
	for i := range m.data.Spaces {
		if m.data.Spaces[i].ID == id {
			return &m.data.Spaces[i]
		}
	}
	return nil
}

func (m *memoryStore) findEquipment(id string) *Equipment {
	for i := range m.data.Equipment {
		if m.data.Equipment[i].ID == id {
			return &m.data.Equipment[i]
		}
	}
	return nil
}

func (m *memoryStore) hasConflict(spaceID, startDate, endDate, excludeProject string) bool {
	start, end, err := parsePeriod(startDate, endDate)
	if err != nil {
		return true
	}
	for _, a := range m.data.Allocations {
		if a.SpaceID != spaceID || a.Status != "ACTIVE" || a.ProjectID == excludeProject {
			continue
		}
		os, oe, err := parsePeriod(a.StartDate, a.EndDate)
		if err != nil {
			continue
		}
		if start.Before(oe) && end.After(os) {
			return true
		}
	}
	return false
}

func parsePeriod(startDate, endDate string) (time.Time, time.Time, error) {
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("开始日期格式应为 YYYY-MM-DD")
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("结束日期格式应为 YYYY-MM-DD")
	}
	if !end.After(start) {
		return time.Time{}, time.Time{}, fmt.Errorf("结束日期必须晚于开始日期")
	}
	return start, end, nil
}

func isOverdue(rec BorrowRecord) bool {
	if rec.Status != "BORROWED" && rec.Status != "RETURN_REQUESTED" {
		return false
	}
	due, err := time.Parse("2006-01-02", rec.DueDate)
	if err != nil {
		return false
	}
	return time.Now().After(due.Add(24 * time.Hour))
}
