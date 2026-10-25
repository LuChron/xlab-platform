package main

// 领域模型：字段与链码 contract.go 中的结构保持一致，
// 后端不在链上模型之外另造一套，避免联调时出现映射错位。

// Equipment 设备台账
type Equipment struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Category      string   `json:"category"`
	Brand         string   `json:"brand"`
	Model         string   `json:"model"`
	SerialNo      string   `json:"serialNo"`
	SpaceID       string   `json:"spaceId"`
	Location      string   `json:"location"`
	Custodian     string   `json:"custodian"`
	Status        string   `json:"status"`
	Condition     string   `json:"condition"`
	Accessories   []string `json:"accessories"`
	ImageURL      string   `json:"imageUrl"`
	ProjectID     string   `json:"projectId"`
	CurrentBorrow string   `json:"currentBorrowId"`
	TotalBorrows  int      `json:"totalBorrows"`
	Note          string   `json:"note"`
	CreatedAt     string   `json:"createdAt"`
	UpdatedAt     string   `json:"updatedAt"`
	UpdatedBy     string   `json:"updatedBy"`
}

// Space 空间单元
type Space struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Zone        string   `json:"zone"`
	Kind        string   `json:"kind"`
	Floor       string   `json:"floor"`
	Capacity    int      `json:"capacity"`
	Status      string   `json:"status"`
	Description string   `json:"description"`
	Amenities   []string `json:"amenities"`
	MapX        int      `json:"mapX"`
	MapY        int      `json:"mapY"`
	MapW        int      `json:"mapW"`
	MapH        int      `json:"mapH"`
}

// ProjectApplication 入驻申请
type ProjectApplication struct {
	ID             string   `json:"id"`
	ProjectName    string   `json:"projectName"`
	LeadID         string   `json:"leadId"`
	LeadName       string   `json:"leadName"`
	Members        []string `json:"members"`
	MemberCount    int      `json:"memberCount"`
	RequestedSeats int      `json:"requestedSeats"`
	PreferredSpace []string `json:"preferredSpace"`
	StartDate      string   `json:"startDate"`
	EndDate        string   `json:"endDate"`
	IdeaSummary    string   `json:"ideaSummary"`
	IdeaHash       string   `json:"ideaHash"`
	Milestone      string   `json:"milestone"`
	MilestoneHash  string   `json:"milestoneHash"`
	Status         string   `json:"status"`
	Reviewer       string   `json:"reviewer"`
	ReviewNote     string   `json:"reviewNote"`
	ReviewedAt     string   `json:"reviewedAt"`
	ProjectID      string   `json:"projectId"`
	CreatedAt      string   `json:"createdAt"`
	UpdatedAt      string   `json:"updatedAt"`
	OwnerIdentity  string   `json:"ownerIdentity"`
}

// Project 已入驻项目
type Project struct {
	ID            string   `json:"id"`
	ApplicationID string   `json:"applicationId"`
	Name          string   `json:"name"`
	LeadID        string   `json:"leadId"`
	LeadName      string   `json:"leadName"`
	Members       []string `json:"members"`
	MemberCount   int      `json:"memberCount"`
	Summary       string   `json:"summary"`
	Status        string   `json:"status"`
	SpaceIDs      []string `json:"spaceIds"`
	StartDate     string   `json:"startDate"`
	EndDate       string   `json:"endDate"`
	Milestone     string   `json:"milestone"`
	MilestoneHash string   `json:"milestoneHash"`
	Extensions    int      `json:"extensions"`
	CloseNote     string   `json:"closeNote"`
	CreatedAt     string   `json:"createdAt"`
	UpdatedAt     string   `json:"updatedAt"`
	OwnerIdentity string   `json:"ownerIdentity"`
}

// SpaceAllocation 空间占用
type SpaceAllocation struct {
	ID        string `json:"id"`
	SpaceID   string `json:"spaceId"`
	ProjectID string `json:"projectId"`
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
}

// BorrowRecord 设备交接单
type BorrowRecord struct {
	ID               string   `json:"id"`
	EquipmentID      string   `json:"equipmentId"`
	EquipmentName    string   `json:"equipmentName"`
	ProjectID        string   `json:"projectId"`
	BorrowerID       string   `json:"borrowerId"`
	Borrower         string   `json:"borrower"`
	BorrowerIdentity string   `json:"borrowerIdentity"`
	Purpose          string   `json:"purpose"`
	PurposeHash      string   `json:"purposeHash"`
	DueDate          string   `json:"dueDate"`
	Status           string   `json:"status"`
	Reviewer         string   `json:"reviewer"`
	ReviewedAt       string   `json:"reviewedAt"`
	ReviewNote       string   `json:"reviewNote"`
	PickupBy         string   `json:"pickupBy"`
	PickupAt         string   `json:"pickupAt"`
	PickupPhotos     []string `json:"pickupPhotos"`
	ReturnBy         string   `json:"returnBy"`
	ReturnAt         string   `json:"returnAt"`
	ReturnPhotos     []string `json:"returnPhotos"`
	Condition        string   `json:"condition"`
	EvidenceHash     string   `json:"evidenceHash"`
	ExceptionNote    string   `json:"exceptionNote"`
	Overdue          bool     `json:"overdue"`
	CreatedAt        string   `json:"createdAt"`
	UpdatedAt        string   `json:"updatedAt"`
}

// Activity 链上操作流水
type Activity struct {
	TxID       string `json:"txId"`
	Action     string `json:"action"`
	ObjectType string `json:"objectType"`
	ObjectID   string `json:"objectId"`
	Actor      string `json:"actor"`
	ActorMSP   string `json:"actorMsp"`
	Detail     string `json:"detail"`
	Timestamp  string `json:"timestamp"`
}

// Stats 看板统计
type Stats struct {
	EquipmentTotal  int            `json:"equipmentTotal"`
	EquipmentAvail  int            `json:"equipmentAvailable"`
	EquipmentOut    int            `json:"equipmentOut"`
	EquipmentFix    int            `json:"equipmentMaintenance"`
	SpaceTotal      int            `json:"spaceTotal"`
	SpaceFree       int            `json:"spaceFree"`
	ProjectActive   int            `json:"projectActive"`
	BorrowPending   int            `json:"borrowPending"`
	AppPending      int            `json:"applicationPending"`
	ByCategory      map[string]int `json:"byCategory"`
	ByProjectStatus map[string]int `json:"byProjectStatus"`
}

// Ledger 全量账本快照
type Ledger struct {
	Equipment    []Equipment          `json:"equipment"`
	Spaces       []Space              `json:"spaces"`
	Projects     []Project            `json:"projects"`
	Applications []ProjectApplication `json:"applications"`
	Borrows      []BorrowRecord       `json:"borrows"`
	Allocations  []SpaceAllocation    `json:"allocations"`
	Activities   []Activity           `json:"activities"`
	Stats        Stats                `json:"stats"`
	Mode         string               `json:"mode"`
	Channel      string               `json:"channel"`
	Chaincode    string               `json:"chaincode"`
}

// BlockTx 区块内的一笔交易（从区块载荷解码得到，链码函数名与调用方组织是核心）
type BlockTx struct {
	TxID      string `json:"txId"`
	Fn        string `json:"fn"`        // 链码函数名，例如 ConfirmEquipmentPickup
	Chaincode string `json:"chaincode"` // 链码名，例如 xlab
	Msp       string `json:"msp"`       // 交易发起组织，例如 Org1MSP
	Timestamp string `json:"timestamp"` // 交易提交时间（RFC3339）
}

// Block 一个真实区块（由 qscc GetBlockByNumber 解码得到）
type Block struct {
	Number   uint64    `json:"number"`
	Hash     string    `json:"hash"`     // 本区块哈希（ASN.1 编码 header 后 sha256）
	PrevHash string    `json:"prevHash"` // 前一区块哈希，首尾相接可验证
	DataHash string    `json:"dataHash"` // 交易数据的 Merkle 根
	TxCount  int       `json:"txCount"`
	Txs      []BlockTx `json:"txs"`
}

// ChainInfo 链状态（qscc GetChainInfo + 最近若干区块）
type ChainInfo struct {
	Height      uint64  `json:"height"`
	CurrentHash string  `json:"currentHash"`
	PrevHash    string  `json:"prevHash"`
	Blocks      []Block `json:"blocks"`
}
