/*
SPDX-License-Identifier: Apache-2.0

X-Lab 资源协作平台链码
一个通道、一个链码，承载两类业务：
  1) 项目入驻与空间分配  PROJECT / APPLICATION / ALLOCATION / SPACE
  2) 设备申请与交接      EQUIPMENT / BORROW

设计要点
  - 链上保存业务字段、状态、摘要哈希、操作身份与时间；文件存储尚未接入。
  - 权限在链码层强制：审核、分配、交接确认只允许管理方组织（Org1MSP）执行。
  - 每次状态变更追加一条 ACTIVITY 记录，形成不可篡改的操作流。
*/

package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

const (
	equipmentPrefix   = "EQUIPMENT:"
	spacePrefix       = "SPACE:"
	projectPrefix     = "PROJECT:"
	applicationPrefix = "APPLICATION:"
	borrowPrefix      = "BORROW:"
	allocationPrefix  = "ALLOCATION:"
	activityPrefix    = "ACTIVITY:"
	initializedKey    = "META:INITIALIZED"
)

// 组织与角色
const (
	// 管理方：X-Lab 资源管理委员会 / 未来学习中心运营方
	AdminMSP = "Org1MSP"
	// 使用方：入驻项目团队 / 实验室成员
	UserMSP = "Org2MSP"
)

// ---------------- 数据模型 ----------------

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

// Space 空间单元（工位 / 工程台 / 功能区）
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

// SpaceAllocation 空间占用区间
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

// Activity 操作流水（链上审计轨迹）
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

// HistoryEntry 单个对象的变更历史
// Value 用字符串承载原始 JSON：contractapi 的 schema 校验无法表达"任意形状的对象"，
// 直接用 json.RawMessage 会因 "Expected array, given object" 校验失败。
type HistoryEntry struct {
	TxID      string `json:"txId"`
	Timestamp string `json:"timestamp"`
	IsDelete  bool   `json:"isDelete"`
	Value     string `json:"value"`
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

// ---------------- 合约 ----------------

type XLabContract struct{ contractapi.Contract }

// ------------------------------ 初始化 ------------------------------

// InitLedger 写入演示用的空间与设备台账
func (c *XLabContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
	if err := requireAdmin(ctx); err != nil {
		return err
	}
	marker, err := ctx.GetStub().GetState(initializedKey)
	if err != nil {
		return err
	}
	if marker != nil {
		return fmt.Errorf("账本已初始化，禁止重复执行 InitLedger")
	}
	for _, prefix := range []string{spacePrefix, equipmentPrefix, projectPrefix, applicationPrefix, borrowPrefix, allocationPrefix, activityPrefix} {
		exists, err := statePrefixExists(ctx, prefix)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("账本已存在业务数据，禁止执行 InitLedger")
		}
	}
	now, err := transactionTime(ctx)
	if err != nil {
		return err
	}
	actor := clientID(ctx)

	spaces := []Space{
		{ID: "A1", Name: "协作工位 A1", Zone: "团队协作站", Kind: "DESK", Floor: "3F", Capacity: 2, Status: "OPEN", Description: "双屏工位，适合软件与设计协作", Amenities: []string{"双显示器", "千兆网络", "独立电源"}, MapX: 1, MapY: 1, MapW: 2, MapH: 1},
		{ID: "A2", Name: "协作工位 A2", Zone: "团队协作站", Kind: "DESK", Floor: "3F", Capacity: 2, Status: "OPEN", Description: "靠窗双人工位，自然采光", Amenities: []string{"双显示器", "白板"}, MapX: 3, MapY: 1, MapW: 2, MapH: 1},
		{ID: "A3", Name: "协作工位 A3", Zone: "团队协作站", Kind: "DESK", Floor: "3F", Capacity: 2, Status: "OPEN", Description: "紧邻灵感讨论角", Amenities: []string{"双显示器"}, MapX: 5, MapY: 1, MapW: 2, MapH: 1},
		{ID: "A4", Name: "协作工位 A4", Zone: "团队协作站", Kind: "DESK", Floor: "3F", Capacity: 2, Status: "OPEN", Description: "适合短周期项目", Amenities: []string{"单显示器"}, MapX: 7, MapY: 1, MapW: 2, MapH: 1},
		{ID: "A5", Name: "协作工位 A5", Zone: "团队协作站", Kind: "DESK", Floor: "3F", Capacity: 2, Status: "OPEN", Description: "临窗长桌，适合两两结对", Amenities: []string{"双显示器", "扩展坞"}, MapX: 9, MapY: 1, MapW: 2, MapH: 1},
		{ID: "A6", Name: "协作工位 A6", Zone: "团队协作站", Kind: "DESK", Floor: "3F", Capacity: 2, Status: "OPEN", Description: "角落工位，安静适合写作", Amenities: []string{"单显示器", "台灯"}, MapX: 11, MapY: 1, MapW: 2, MapH: 1},
		{ID: "B1", Name: "工程台 B1", Zone: "工程实践坊", Kind: "BENCH", Floor: "4F", Capacity: 3, Status: "OPEN", Description: "硬件装配与焊接调试台", Amenities: []string{"焊接台", "示波器", "防静电垫"}, MapX: 1, MapY: 1, MapW: 3, MapH: 2},
		{ID: "B2", Name: "工程台 B2", Zone: "工程实践坊", Kind: "BENCH", Floor: "4F", Capacity: 3, Status: "OPEN", Description: "3D 打印与耗材收纳", Amenities: []string{"3D 打印机", "工具墙"}, MapX: 4, MapY: 1, MapW: 3, MapH: 2},
		{ID: "B3", Name: "工程台 B3", Zone: "工程实践坊", Kind: "BENCH", Floor: "4F", Capacity: 3, Status: "OPEN", Description: "SMD 焊接与返修工作位", Amenities: []string{"热风焊台", "万用表", "防静电垫"}, MapX: 7, MapY: 1, MapW: 3, MapH: 2},
		{ID: "B4", Name: "工程台 B4", Zone: "工程实践坊", Kind: "BENCH", Floor: "4F", Capacity: 3, Status: "OPEN", Description: "嵌入式开发与树莓派实验台", Amenities: []string{"树莓派集群", "逻辑分析仪"}, MapX: 10, MapY: 1, MapW: 3, MapH: 2},
		{ID: "C1", Name: "大屏协作区", Zone: "大屏区", Kind: "ZONE", Floor: "3F", Capacity: 8, Status: "OPEN", Description: "展示、评审与多人协作", Amenities: []string{"激光大屏", "无线投屏", "音响"}, MapX: 1, MapY: 3, MapW: 5, MapH: 2},
		{ID: "C3", Name: "路演小厅", Zone: "大屏区", Kind: "ZONE", Floor: "3F", Capacity: 6, Status: "OPEN", Description: "小型路演与答辩彩排", Amenities: []string{"LED 屏", "翻页器", "补光灯"}, MapX: 6, MapY: 3, MapW: 3, MapH: 2},
		{ID: "C2", Name: "影视制作区", Zone: "影视区", Kind: "ZONE", Floor: "4F", Capacity: 4, Status: "OPEN", Description: "拍摄、录音与后期", Amenities: []string{"绿幕", "补光灯", "录音棚"}, MapX: 1, MapY: 3, MapW: 6, MapH: 2},
		{ID: "C4", Name: "录音棚", Zone: "影视区", Kind: "ZONE", Floor: "4F", Capacity: 2, Status: "OPEN", Description: "隔音录音与配音", Amenities: []string{"隔音墙", "电容麦", "声卡"}, MapX: 7, MapY: 3, MapW: 6, MapH: 2},
		{ID: "D1", Name: "灵感讨论角", Zone: "灵感讨论角", Kind: "ZONE", Floor: "3F", Capacity: 6, Status: "OPEN", Description: "沙发区，适合头脑风暴", Amenities: []string{"移动白板", "便签墙"}, MapX: 9, MapY: 3, MapW: 2, MapH: 2},
		{ID: "D2", Name: "头脑风暴室", Zone: "灵感讨论角", Kind: "ZONE", Floor: "3F", Capacity: 4, Status: "OPEN", Description: "封闭讨论间，可远程连线", Amenities: []string{"交互白板", "视频会议"}, MapX: 11, MapY: 3, MapW: 2, MapH: 2},
		{ID: "E1", Name: "休闲补给站", Zone: "休闲补给站", Kind: "ZONE", Floor: "3F", Capacity: 10, Status: "OPEN", Description: "咖啡、简餐与交流", Amenities: []string{"咖啡机", "冰箱", "制冰机"}, MapX: 1, MapY: 5, MapW: 6, MapH: 2},
		{ID: "E2", Name: "共享厨房", Zone: "休闲补给站", Kind: "ZONE", Floor: "3F", Capacity: 6, Status: "OPEN", Description: "自热简餐与茶歇", Amenities: []string{"微波炉", "饮水机", "餐具柜"}, MapX: 7, MapY: 5, MapW: 6, MapH: 2},
	}

	equipment := []Equipment{
		{ID: "EQ-001", Name: "MacBook Pro 16", Category: "计算设备", Brand: "Apple", Model: "M3 Max / 36G", SerialNo: "C02XL0XLAB01", SpaceID: "A1", Location: "团队协作站 A1", Custodian: "资源管理委员会", Status: "AVAILABLE", Condition: "GOOD", Accessories: []string{"140W 电源适配器", "USB-C 线"}, Note: "预装开发环境与设计软件"},
		{ID: "EQ-002", Name: "Dell U2723QE 显示器", Category: "显示设备", Brand: "Dell", Model: "U2723QE 4K", SerialNo: "CN-0XLAB002", SpaceID: "A1", Location: "团队协作站 A1", Custodian: "资源管理委员会", Status: "AVAILABLE", Condition: "GOOD", Accessories: []string{"DP 线", "HDMI 线", "电源线"}, Note: "27 寸 4K，支持 USB-C 一线连"},
		{ID: "EQ-003", Name: "Dell U2424H 竖屏", Category: "显示设备", Brand: "Dell", Model: "U2424H", SerialNo: "CN-0XLAB003", SpaceID: "A2", Location: "团队协作站 A2", Custodian: "资源管理委员会", Status: "AVAILABLE", Condition: "GOOD", Accessories: []string{"HDMI 线"}, Note: "可旋转竖屏，适合看代码与文档"},
		{ID: "EQ-004", Name: "索尼 A7M4 微单", Category: "影音设备", Brand: "Sony", Model: "ILCE-7M4", SerialNo: "S01XLAB004", SpaceID: "C2", Location: "影视区器材柜", Custodian: "传播管理委员会", Status: "AVAILABLE", Condition: "GOOD", Accessories: []string{"机身盖", "电池 ×2", "充电器", "背带"}, Note: "含 SD 卡，拍摄前请格式化"},
		{ID: "EQ-005", Name: "索尼 24-70mm F2.8 镜头", Category: "影音设备", Brand: "Sony", Model: "SEL2470GM2", SerialNo: "L02XLAB005", SpaceID: "C2", Location: "影视区器材柜", Custodian: "传播管理委员会", Status: "AVAILABLE", Condition: "GOOD", Accessories: []string{"前后盖", "遮光罩", "镜头袋"}, Note: "G 大师二代，贵重器材需当面验收"},
		{ID: "EQ-006", Name: "无线领夹麦克风", Category: "影音设备", Brand: "Rode", Model: "Wireless GO II", SerialNo: "R03XLAB006", SpaceID: "C2", Location: "影视区器材柜", Custodian: "传播管理委员会", Status: "MAINTENANCE", Condition: "FAIR", Accessories: []string{"接收器", "发射器 ×2", "防风毛套"}, Note: "电池仓接触不良，待更换"},
		{ID: "EQ-007", Name: "便携投影仪", Category: "展示设备", Brand: "Epson", Model: "EF-12", SerialNo: "P04XLAB007", SpaceID: "C1", Location: "大屏区储物柜", Custodian: "资源管理委员会", Status: "AVAILABLE", Condition: "GOOD", Accessories: []string{"遥控器", "HDMI 线", "电源", "便携包"}, Note: "1080P 激光投影"},
		{ID: "EQ-008", Name: "Bambu Lab P1S 3D 打印机", Category: "工程设备", Brand: "Bambu Lab", Model: "P1S", SerialNo: "B05XLAB008", SpaceID: "B2", Location: "工程实践坊 B2", Custodian: "硬件团队", Status: "AVAILABLE", Condition: "GOOD", Accessories: []string{"耗材刮刀", "备用喷嘴", "PLA 耗材 ×2"}, Note: "使用前需通过操作培训"},
		{ID: "EQ-009", Name: "Arduino 创客套件", Category: "工程设备", Brand: "Arduino", Model: "UNO R4 套件", SerialNo: "A06XLAB009", SpaceID: "B1", Location: "工程实践坊 B1", Custodian: "硬件团队", Status: "AVAILABLE", Condition: "GOOD", Accessories: []string{"主板", "传感器 20+", "面包板", "杜邦线"}, Note: "适合入门与教学演示"},
		{ID: "EQ-010", Name: "数字示波器", Category: "工程设备", Brand: "Rigol", Model: "DHO814", SerialNo: "O07XLAB010", SpaceID: "B1", Location: "工程实践坊 B1", Custodian: "硬件团队", Status: "AVAILABLE", Condition: "GOOD", Accessories: []string{"探头 ×2", "电源线"}, Note: "100MHz 四通道"},
		{ID: "EQ-011", Name: "移动白板套装", Category: "协作工具", Brand: "得力", Model: "120×90", SerialNo: "-", SpaceID: "D1", Location: "灵感讨论角", Custodian: "资源管理委员会", Status: "AVAILABLE", Condition: "GOOD", Accessories: []string{"白板笔 ×3", "板擦", "磁扣"}, Note: "可移动，适合讨论"},
		{ID: "EQ-012", Name: "Meta Quest 3 头显", Category: "计算设备", Brand: "Meta", Model: "Quest 3 512G", SerialNo: "Q08XLAB012", SpaceID: "C1", Location: "大屏区储物柜", Custodian: "软件团队", Status: "AVAILABLE", Condition: "GOOD", Accessories: []string{"手柄 ×2", "充电线", "面罩"}, Note: "用于交互原型测试"},
		{ID: "EQ-013", Name: "Mac Studio 工作站", Category: "计算设备", Brand: "Apple", Model: "M2 Ultra / 128G", SerialNo: "S09XLAB013", SpaceID: "B2", Location: "工程实践坊 B2", Custodian: "资源管理委员会", Status: "AVAILABLE", Condition: "GOOD", Accessories: []string{"电源线", "网线"}, Note: "本地大模型推理与渲染"},
		{ID: "EQ-014", Name: "Rode 指向性麦克风", Category: "影音设备", Brand: "Rode", Model: "NTG4+", SerialNo: "R10XLAB014", SpaceID: "C2", Location: "影视区器材柜", Custodian: "传播管理委员会", Status: "AVAILABLE", Condition: "GOOD", Accessories: []string{"防风罩", "XLR 线", "支架"}, Note: "采访用枪麦"},
		{ID: "EQ-015", Name: "移动硬盘 2TB", Category: "存储设备", Brand: "Samsung", Model: "T7 Shield", SerialNo: "D11XLAB015", SpaceID: "C2", Location: "影视区器材柜", Custodian: "资源管理委员会", Status: "AVAILABLE", Condition: "GOOD", Accessories: []string{"USB-C 线"}, Note: "素材盘，归还前请勿格式化"},
		{ID: "EQ-016", Name: "Logitech 4K 摄像头", Category: "影音设备", Brand: "Logitech", Model: "Brio 4K", SerialNo: "L12XLAB016", SpaceID: "C1", Location: "大屏区储物柜", Custodian: "传播管理委员会", Status: "AVAILABLE", Condition: "GOOD", Accessories: []string{"三脚架", "USB-C 线"}, Note: "直播与线上评审"},
		{ID: "EQ-017", Name: "ThinkPad X1 Carbon", Category: "计算设备", Brand: "Lenovo", Model: "X1 Carbon G11", SerialNo: "T13XLAB017", SpaceID: "A3", Location: "团队协作站 A3", Custodian: "资源管理委员会", Status: "AVAILABLE", Condition: "GOOD", Accessories: []string{"65W 电源", "USB-C 扩展坞"}, Note: "轻量办公，适合外出演示"},
		{ID: "EQ-018", Name: "罗技 MX Master 3S 鼠标", Category: "协作工具", Brand: "Logitech", Model: "MX Master 3S", SerialNo: "M14XLAB018", SpaceID: "A2", Location: "团队协作站 A2", Custodian: "资源管理委员会", Status: "AVAILABLE", Condition: "GOOD", Accessories: []string{"接收器", "USB-C 线"}, Note: "静音按键，多设备切换"},
		{ID: "EQ-019", Name: "Keychron K8 Pro 键盘", Category: "协作工具", Brand: "Keychron", Model: "K8 Pro", SerialNo: "K15XLAB019", SpaceID: "A1", Location: "团队协作站 A1", Custodian: "资源管理委员会", Status: "AVAILABLE", Condition: "GOOD", Accessories: []string{"键帽拔取器", "USB-C 线"}, Note: "热插拔，可换轴"},
		{ID: "EQ-020", Name: "USB 采集卡 4K", Category: "影音设备", Brand: "Elgato", Model: "Cam Link 4K", SerialNo: "E16XLAB020", SpaceID: "C2", Location: "影视区器材柜", Custodian: "传播管理委员会", Status: "AVAILABLE", Condition: "GOOD", Accessories: []string{"HDMI 线", "USB-C 线"}, Note: "相机直播信号采集"},
		{ID: "EQ-021", Name: "爱图仕 200D 补光灯", Category: "影音设备", Brand: "Aputure", Model: "Amaran 200d", SerialNo: "A17XLAB021", SpaceID: "C2", Location: "影视区器材柜", Custodian: "传播管理委员会", Status: "AVAILABLE", Condition: "GOOD", Accessories: []string{"灯架", "柔光箱", "电源线"}, Note: "200W 常亮灯，色温可调"},
		{ID: "EQ-022", Name: "DJI RS 3 稳定器", Category: "影音设备", Brand: "DJI", Model: "RS 3", SerialNo: "D18XLAB022", SpaceID: "C2", Location: "影视区器材柜", Custodian: "传播管理委员会", Status: "AVAILABLE", Condition: "GOOD", Accessories: []string{"快装板", "跟焦器", "电池 ×2"}, Note: "三轴稳定，承重 3kg"},
		{ID: "EQ-023", Name: "树莓派 5 集群", Category: "计算设备", Brand: "Raspberry Pi", Model: "Pi 5 / 8G ×4", SerialNo: "R19XLAB023", SpaceID: "B4", Location: "工程实践坊 B4", Custodian: "硬件团队", Status: "AVAILABLE", Condition: "GOOD", Accessories: []string{"电源 ×4", "网线 ×4", "散热片"}, Note: "边缘计算与集群实验"},
		{ID: "EQ-024", Name: "热风焊台", Category: "工程设备", Brand: "Quick", Model: "861DW", SerialNo: "Q20XLAB024", SpaceID: "B3", Location: "工程实践坊 B3", Custodian: "硬件团队", Status: "AVAILABLE", Condition: "GOOD", Accessories: []string{"风嘴 ×4", "镊子", "焊锡丝"}, Note: "SMD 返修用，需培训"},
		{ID: "EQ-025", Name: "万用表 Fluke 87V", Category: "工程设备", Brand: "Fluke", Model: "87V", SerialNo: "F21XLAB025", SpaceID: "B1", Location: "工程实践坊 B1", Custodian: "硬件团队", Status: "AVAILABLE", Condition: "GOOD", Accessories: []string{"表笔", "温度探头"}, Note: "工业级真有效值万用表"},
		{ID: "EQ-026", Name: "绿幕套装", Category: "影音设备", Brand: "Neewer", Model: "2.6×3m", SerialNo: "N22XLAB026", SpaceID: "C2", Location: "影视区器材柜", Custodian: "传播管理委员会", Status: "AVAILABLE", Condition: "GOOD", Accessories: []string{"背景布", "支架", "夹子 ×6"}, Note: "可折叠，含收纳袋"},
		{ID: "EQ-027", Name: "NAS 网络存储 16TB", Category: "存储设备", Brand: "Synology", Model: "DS923+", SerialNo: "S23XLAB027", SpaceID: "C4", Location: "录音棚机柜", Custodian: "资源管理委员会", Status: "AVAILABLE", Condition: "GOOD", Accessories: []string{"硬盘 ×4", "网线"}, Note: "素材与项目资料集中备份"},
		{ID: "EQ-028", Name: "电容麦克风", Category: "影音设备", Brand: "Audio-Technica", Model: "AT2020", SerialNo: "A24XLAB028", SpaceID: "C4", Location: "录音棚", Custodian: "传播管理委员会", Status: "AVAILABLE", Condition: "GOOD", Accessories: []string{"防喷罩", "XLR 线", "减震架"}, Note: "录音棚固定设备，不外借"},
	}

	for _, s := range spaces {
		if err := putJSON(ctx, spacePrefix+s.ID, s); err != nil {
			return err
		}
	}
	for _, e := range equipment {
		e.CreatedAt = now
		e.UpdatedAt = now
		e.UpdatedBy = actor
		if err := putJSON(ctx, equipmentPrefix+e.ID, e); err != nil {
			return err
		}
	}
	if err := ctx.GetStub().PutState(initializedKey, []byte(now)); err != nil {
		return err
	}
	return c.logActivity(ctx, "INIT_LEDGER", "LEDGER", "xlab", fmt.Sprintf("初始化 %d 个空间与 %d 台设备", len(spaces), len(equipment)))
}

// ------------------------------ 设备台账 ------------------------------

// CreateEquipment 新增设备（仅管理方）
func (c *XLabContract) CreateEquipment(ctx contractapi.TransactionContextInterface, id, name, category, brand, model, serialNo, spaceID, custodian, note string) error {
	if err := requireAdmin(ctx); err != nil {
		return err
	}
	if err := requireMissing(ctx, equipmentPrefix+id); err != nil {
		return err
	}
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("设备名称不能为空")
	}
	now, err := transactionTime(ctx)
	if err != nil {
		return err
	}
	location := ""
	if spaceID != "" {
		var sp Space
		if err := getJSON(ctx, spacePrefix+spaceID, &sp); err != nil {
			return fmt.Errorf("存放空间 %s 不存在", spaceID)
		}
		location = sp.Name
	}
	eq := Equipment{
		ID: id, Name: name, Category: category, Brand: brand, Model: model, SerialNo: serialNo,
		SpaceID: spaceID, Location: location, Custodian: custodian, Status: "AVAILABLE",
		Condition: "GOOD", Note: note, CreatedAt: now, UpdatedAt: now, UpdatedBy: clientID(ctx),
	}
	if err := putJSON(ctx, equipmentPrefix+id, eq); err != nil {
		return err
	}
	return c.logActivity(ctx, "CREATE_EQUIPMENT", "equipment", id, name)
}

// UpdateEquipmentStatus 维护状态（仅管理方）
func (c *XLabContract) UpdateEquipmentStatus(ctx contractapi.TransactionContextInterface, id, status, condition, note string) error {
	if err := requireAdmin(ctx); err != nil {
		return err
	}
	var eq Equipment
	if err := getJSON(ctx, equipmentPrefix+id, &eq); err != nil {
		return err
	}
	if eq.Status == "BORROWED" {
		return fmt.Errorf("设备处于借出状态，请先完成归还再调整")
	}
	// 维护不得释放仍挂在有效借用链上的设备（REQUESTED/PENDING/APPROVED/异常等），
	// 否则会允许对同一设备再开第二条借用单，破坏设备独占约束。
	active, err := c.hasActiveBorrowForEquipment(ctx, id)
	if err != nil {
		return err
	}
	if active {
		return fmt.Errorf("设备存在未结清的借用或异常单，请先完成归还与验收再调整")
	}
	valid := map[string]bool{"AVAILABLE": true, "MAINTENANCE": true, "RETIRED": true}
	if !valid[status] {
		return fmt.Errorf("非法设备状态 %s", status)
	}
	now, err := transactionTime(ctx)
	if err != nil {
		return err
	}
	eq.Status, eq.Condition, eq.Note = status, condition, note
	eq.UpdatedAt, eq.UpdatedBy = now, clientID(ctx)
	if err := putJSON(ctx, equipmentPrefix+id, eq); err != nil {
		return err
	}
	return c.logActivity(ctx, "UPDATE_EQUIPMENT_STATUS", "equipment", id, status)
}

// ------------------------------ 空间台账 ------------------------------

// CreateSpace 新增空间（仅管理方）
func (c *XLabContract) CreateSpace(ctx contractapi.TransactionContextInterface, id, name, zone, kind, floor, capacityText, description string) error {
	if err := requireAdmin(ctx); err != nil {
		return err
	}
	if err := requireMissing(ctx, spacePrefix+id); err != nil {
		return err
	}
	capacity, err := strconv.Atoi(capacityText)
	if err != nil || capacity < 1 {
		return fmt.Errorf("容量必须为正整数")
	}
	sp := Space{ID: id, Name: name, Zone: zone, Kind: kind, Floor: floor, Capacity: capacity, Status: "OPEN", Description: description}
	if err := putJSON(ctx, spacePrefix+id, sp); err != nil {
		return err
	}
	return c.logActivity(ctx, "CREATE_SPACE", "space", id, name)
}

// ------------------------------ 项目入驻 ------------------------------

// SubmitProjectApplication 提交 idea 入驻申请（使用方）
func (c *XLabContract) SubmitProjectApplication(ctx contractapi.TransactionContextInterface, id, projectName, leadID, leadName, membersJSON, requestedSeatsText, preferredSpaceJSON, startDate, endDate, ideaSummary, ideaHash, milestone, milestoneHash string) error {
	if err := requireMember(ctx); err != nil {
		return err
	}
	ownerIdentity, err := verifiedClientID(ctx)
	if err != nil {
		return err
	}
	if err := requireMissing(ctx, applicationPrefix+id); err != nil {
		return err
	}
	if strings.TrimSpace(projectName) == "" || strings.TrimSpace(leadID) == "" {
		return fmt.Errorf("项目名称与负责人不能为空")
	}
	if strings.TrimSpace(leadName) == "" {
		return fmt.Errorf("负责人姓名不能为空")
	}
	var members []string
	if strings.TrimSpace(membersJSON) != "" {
		if err := json.Unmarshal([]byte(membersJSON), &members); err != nil {
			return fmt.Errorf("成员列表必须是 JSON 数组")
		}
	}
	if len(members) == 0 {
		members = []string{leadName}
	}
	seats, err := strconv.Atoi(requestedSeatsText)
	if err != nil || seats < 1 {
		return fmt.Errorf("申请工位数必须为正整数")
	}
	var preferred []string
	if strings.TrimSpace(preferredSpaceJSON) != "" {
		if err := json.Unmarshal([]byte(preferredSpaceJSON), &preferred); err != nil {
			return fmt.Errorf("空间偏好必须是 JSON 数组")
		}
	}
	if _, _, err := parsePeriod(startDate, endDate); err != nil {
		return err
	}
	now, err := transactionTime(ctx)
	if err != nil {
		return err
	}
	app := ProjectApplication{
		ID: id, ProjectName: projectName, LeadID: leadID, LeadName: leadName,
		Members: members, MemberCount: len(members), RequestedSeats: seats,
		PreferredSpace: preferred, StartDate: startDate, EndDate: endDate,
		IdeaSummary: ideaSummary, IdeaHash: ideaHash, Milestone: milestone, MilestoneHash: milestoneHash,
		Status: "SUBMITTED", CreatedAt: now, UpdatedAt: now, OwnerIdentity: ownerIdentity,
	}
	if err := putJSON(ctx, applicationPrefix+id, app); err != nil {
		return err
	}
	return c.logActivity(ctx, "SUBMIT_APPLICATION", "application", id, projectName)
}

// ReviewProjectApplication 评审入驻申请（仅管理方）
func (c *XLabContract) ReviewProjectApplication(ctx contractapi.TransactionContextInterface, id, approvedText, note string) error {
	if err := requireAdmin(ctx); err != nil {
		return err
	}
	var app ProjectApplication
	if err := getJSON(ctx, applicationPrefix+id, &app); err != nil {
		return err
	}
	if app.Status != "SUBMITTED" {
		return fmt.Errorf("申请 %s 当前状态为 %s，不可重复评审", id, app.Status)
	}
	approved, err := strconv.ParseBool(approvedText)
	if err != nil {
		return fmt.Errorf("approved 必须是 true 或 false")
	}
	now, err := transactionTime(ctx)
	if err != nil {
		return err
	}
	if approved {
		app.Status = "APPROVED"
	} else {
		app.Status = "REJECTED"
	}
	app.Reviewer, app.ReviewNote, app.ReviewedAt, app.UpdatedAt = clientID(ctx), note, now, now
	if err := putJSON(ctx, applicationPrefix+id, app); err != nil {
		return err
	}
	return c.logActivity(ctx, "REVIEW_APPLICATION", "application", id, fmt.Sprintf("approved=%v", approved))
}

// ActivateProject 分配空间并激活项目（仅管理方）
func (c *XLabContract) ActivateProject(ctx contractapi.TransactionContextInterface, projectID, applicationID, spaceIDsJSON string) error {
	if err := requireAdmin(ctx); err != nil {
		return err
	}
	if err := requireMissing(ctx, projectPrefix+projectID); err != nil {
		return err
	}
	var app ProjectApplication
	if err := getJSON(ctx, applicationPrefix+applicationID, &app); err != nil {
		return err
	}
	if app.Status != "APPROVED" {
		return fmt.Errorf("申请尚未通过评审，当前状态 %s", app.Status)
	}
	var spaceIDs []string
	if err := json.Unmarshal([]byte(spaceIDsJSON), &spaceIDs); err != nil || len(spaceIDs) == 0 {
		return fmt.Errorf("必须选择至少一个空间")
	}
	capacity := 0
	seenSpaces := map[string]bool{}
	for _, sid := range spaceIDs {
		if seenSpaces[sid] {
			return fmt.Errorf("空间 %s 重复选择", sid)
		}
		seenSpaces[sid] = true
		var sp Space
		if err := getJSON(ctx, spacePrefix+sid, &sp); err != nil {
			return err
		}
		if sp.Status != "OPEN" {
			return fmt.Errorf("空间 %s 当前不可分配（%s）", sid, sp.Status)
		}
		conflict, err := c.hasSpaceConflict(ctx, sid, app.StartDate, app.EndDate)
		if err != nil {
			return err
		}
		if conflict {
			return fmt.Errorf("空间 %s 在 %s 至 %s 已被占用", sid, app.StartDate, app.EndDate)
		}
		capacity += sp.Capacity
	}
	if capacity < app.RequestedSeats {
		return fmt.Errorf("所选空间容量 %d 低于申请工位数 %d", capacity, app.RequestedSeats)
	}
	now, err := transactionTime(ctx)
	if err != nil {
		return err
	}
	project := Project{
		ID: projectID, ApplicationID: applicationID, Name: app.ProjectName,
		LeadID: app.LeadID, LeadName: app.LeadName, Members: app.Members, MemberCount: app.MemberCount,
		Summary: app.IdeaSummary, Status: "ACTIVE", SpaceIDs: spaceIDs,
		StartDate: app.StartDate, EndDate: app.EndDate,
		Milestone: app.Milestone, MilestoneHash: app.MilestoneHash,
		CreatedAt: now, UpdatedAt: now, OwnerIdentity: app.OwnerIdentity,
	}
	if err := putJSON(ctx, projectPrefix+projectID, project); err != nil {
		return err
	}
	for _, sid := range spaceIDs {
		allocID := projectID + ":" + sid
		alloc := SpaceAllocation{ID: allocID, SpaceID: sid, ProjectID: projectID, StartDate: app.StartDate, EndDate: app.EndDate, Status: "ACTIVE", CreatedAt: now}
		if err := putJSON(ctx, allocationPrefix+allocID, alloc); err != nil {
			return err
		}
	}
	app.Status, app.ProjectID, app.UpdatedAt = "ACTIVATED", projectID, now
	if err := putJSON(ctx, applicationPrefix+applicationID, app); err != nil {
		return err
	}
	return c.logActivity(ctx, "ACTIVATE_PROJECT", "project", projectID, fmt.Sprintf("空间: %s", strings.Join(spaceIDs, ",")))
}

// ExtendProject 续期（仅管理方）
func (c *XLabContract) ExtendProject(ctx contractapi.TransactionContextInterface, projectID, newEndDate, note string) error {
	if err := requireAdmin(ctx); err != nil {
		return err
	}
	var p Project
	if err := getJSON(ctx, projectPrefix+projectID, &p); err != nil {
		return err
	}
	if p.Status != "ACTIVE" {
		return fmt.Errorf("项目当前状态为 %s，不能续期", p.Status)
	}
	_, curEnd, err := parsePeriod(p.StartDate, p.EndDate)
	if err != nil {
		return err
	}
	newEnd, err := time.Parse("2006-01-02", newEndDate)
	if err != nil {
		return fmt.Errorf("结束日期格式应为 YYYY-MM-DD")
	}
	if !newEnd.After(curEnd) {
		return fmt.Errorf("新结束日期必须晚于当前期限")
	}
	for _, sid := range p.SpaceIDs {
		conflict, err := c.hasSpaceConflictExcluding(ctx, sid, p.EndDate, newEndDate, projectID)
		if err != nil {
			return err
		}
		if conflict {
			return fmt.Errorf("空间 %s 在延期区间内已被其他项目占用", sid)
		}
	}
	now, err := transactionTime(ctx)
	if err != nil {
		return err
	}
	oldEnd := p.EndDate
	p.EndDate, p.Extensions, p.UpdatedAt = newEndDate, p.Extensions+1, now
	// 续期理由只进入流水 detail，不覆盖阶段成果 milestone（milestone 由 SubmitMilestone 单独管理）
	for _, sid := range p.SpaceIDs {
		allocID := p.ID + ":" + sid
		var alloc SpaceAllocation
		if err := getJSON(ctx, allocationPrefix+allocID, &alloc); err == nil {
			alloc.EndDate = newEndDate
			if err := putJSON(ctx, allocationPrefix+allocID, alloc); err != nil {
				return err
			}
		}
	}
	if err := putJSON(ctx, projectPrefix+projectID, p); err != nil {
		return err
	}
	return c.logActivity(ctx, "EXTEND_PROJECT", "project", projectID, fmt.Sprintf("%s → %s", oldEnd, newEndDate))
}

// CloseProject 结项退出：先校验无未归还设备，再释放空间（仅管理方）
func (c *XLabContract) CloseProject(ctx contractapi.TransactionContextInterface, projectID, note string) error {
	if err := requireAdmin(ctx); err != nil {
		return err
	}
	var p Project
	if err := getJSON(ctx, projectPrefix+projectID, &p); err != nil {
		return err
	}
	if p.Status == "CLOSED" {
		return fmt.Errorf("项目已结项")
	}
	outstanding, err := c.countOutstandingBorrows(ctx, projectID)
	if err != nil {
		return err
	}
	if outstanding > 0 {
		return fmt.Errorf("项目仍有 %d 件设备未归还，无法结项", outstanding)
	}
	now, err := transactionTime(ctx)
	if err != nil {
		return err
	}
	for _, sid := range p.SpaceIDs {
		allocID := p.ID + ":" + sid
		var alloc SpaceAllocation
		if err := getJSON(ctx, allocationPrefix+allocID, &alloc); err == nil {
			alloc.Status = "RELEASED"
			if err := putJSON(ctx, allocationPrefix+allocID, alloc); err != nil {
				return err
			}
		}
	}
	p.Status, p.CloseNote, p.UpdatedAt = "CLOSED", note, now
	if err := putJSON(ctx, projectPrefix+projectID, p); err != nil {
		return err
	}
	return c.logActivity(ctx, "CLOSE_PROJECT", "project", projectID, note)
}

// SubmitMilestone 提交阶段成果（使用方）
func (c *XLabContract) SubmitMilestone(ctx contractapi.TransactionContextInterface, projectID, milestone, milestoneHash string) error {
	if err := requireMember(ctx); err != nil {
		return err
	}
	var p Project
	if err := getJSON(ctx, projectPrefix+projectID, &p); err != nil {
		return err
	}
	if p.Status != "ACTIVE" {
		return fmt.Errorf("项目当前状态为 %s，不能提交成果", p.Status)
	}
	if err := requireOwner(ctx, p.OwnerIdentity); err != nil {
		return err
	}
	if strings.TrimSpace(milestone) == "" {
		return fmt.Errorf("成果内容不能为空")
	}
	now, err := transactionTime(ctx)
	if err != nil {
		return err
	}
	p.Milestone, p.MilestoneHash, p.UpdatedAt = milestone, milestoneHash, now
	if err := putJSON(ctx, projectPrefix+projectID, p); err != nil {
		return err
	}
	return c.logActivity(ctx, "SUBMIT_MILESTONE", "project", projectID, milestone)
}

// ------------------------------ 设备借用 ------------------------------

// RequestEquipmentBorrow 提交借用申请（使用方）
func (c *XLabContract) RequestEquipmentBorrow(ctx contractapi.TransactionContextInterface, borrowID, equipmentID, projectID, borrowerID, borrower, purpose, purposeHash, dueDate string) error {
	if err := requireMember(ctx); err != nil {
		return err
	}
	borrowerIdentity, err := verifiedClientID(ctx)
	if err != nil {
		return err
	}
	if err := requireMissing(ctx, borrowPrefix+borrowID); err != nil {
		return err
	}
	if strings.TrimSpace(borrower) == "" || strings.TrimSpace(purpose) == "" {
		return fmt.Errorf("借用人与用途不能为空")
	}
	due, err := time.Parse("2006-01-02", dueDate)
	if err != nil {
		return fmt.Errorf("预计归还日期格式应为 YYYY-MM-DD")
	}
	txTime, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return err
	}
	today, _ := time.Parse("2006-01-02", txTime.AsTime().Format("2006-01-02"))
	if due.Before(today) {
		return fmt.Errorf("预计归还日期不能早于今天")
	}
	var eq Equipment
	if err := getJSON(ctx, equipmentPrefix+equipmentID, &eq); err != nil {
		return err
	}
	if eq.Status != "AVAILABLE" {
		return fmt.Errorf("设备 %s 当前状态为 %s，不可申请", equipmentID, eq.Status)
	}
	if strings.TrimSpace(projectID) == "" {
		return fmt.Errorf("借用设备必须关联已入驻项目")
	}
	var p Project
	if err := getJSON(ctx, projectPrefix+projectID, &p); err != nil {
		return err
	}
	if p.Status != "ACTIVE" {
		return fmt.Errorf("关联项目未处于入驻状态（%s）", p.Status)
	}
	if err := requireOwner(ctx, p.OwnerIdentity); err != nil {
		return err
	}
	projectEnd, err := time.Parse("2006-01-02", p.EndDate)
	if err != nil {
		return fmt.Errorf("关联项目结束日期无效")
	}
	if due.After(projectEnd) {
		return fmt.Errorf("预计归还日期不能晚于项目期限 %s", p.EndDate)
	}
	now, err := transactionTime(ctx)
	if err != nil {
		return err
	}
	rec := BorrowRecord{
		ID: borrowID, EquipmentID: equipmentID, EquipmentName: eq.Name, ProjectID: projectID,
		BorrowerID: borrowerID, Borrower: borrower, BorrowerIdentity: borrowerIdentity,
		Purpose: purpose, PurposeHash: purposeHash, DueDate: dueDate,
		Status: "PENDING", PickupPhotos: []string{}, ReturnPhotos: []string{},
		CreatedAt: now, UpdatedAt: now,
	}
	eq.Status, eq.ProjectID, eq.CurrentBorrow, eq.UpdatedAt, eq.UpdatedBy = "REQUESTED", projectID, borrowID, now, borrower
	if err := putJSON(ctx, borrowPrefix+borrowID, rec); err != nil {
		return err
	}
	if err := putJSON(ctx, equipmentPrefix+equipmentID, eq); err != nil {
		return err
	}
	return c.logActivity(ctx, "REQUEST_BORROW", "borrow", borrowID, fmt.Sprintf("%s → %s", equipmentID, borrower))
}

// ReviewEquipmentBorrow 审批借用申请（仅管理方）
func (c *XLabContract) ReviewEquipmentBorrow(ctx contractapi.TransactionContextInterface, borrowID, approvedText, note string) error {
	if err := requireAdmin(ctx); err != nil {
		return err
	}
	var rec BorrowRecord
	if err := getJSON(ctx, borrowPrefix+borrowID, &rec); err != nil {
		return err
	}
	if rec.Status != "PENDING" {
		return fmt.Errorf("借用单当前状态为 %s，不可审批", rec.Status)
	}
	approved, err := strconv.ParseBool(approvedText)
	if err != nil {
		return fmt.Errorf("approved 必须是 true 或 false")
	}
	var eq Equipment
	if err := getJSON(ctx, equipmentPrefix+rec.EquipmentID, &eq); err != nil {
		return err
	}
	now, err := transactionTime(ctx)
	if err != nil {
		return err
	}
	rec.Reviewer, rec.ReviewNote, rec.ReviewedAt, rec.UpdatedAt = clientID(ctx), note, now, now
	eq.UpdatedAt, eq.UpdatedBy = now, clientID(ctx)
	if approved {
		rec.Status, eq.Status = "APPROVED", "APPROVED"
	} else {
		rec.Status, eq.Status, eq.ProjectID, eq.CurrentBorrow = "REJECTED", "AVAILABLE", "", ""
	}
	if err := putJSON(ctx, borrowPrefix+borrowID, rec); err != nil {
		return err
	}
	if err := putJSON(ctx, equipmentPrefix+rec.EquipmentID, eq); err != nil {
		return err
	}
	return c.logActivity(ctx, "REVIEW_BORROW", "borrow", borrowID, fmt.Sprintf("approved=%v", approved))
}

// ConfirmEquipmentPickup 领取交接：管理方记录现场核对的设备与附件状态。
func (c *XLabContract) ConfirmEquipmentPickup(ctx contractapi.TransactionContextInterface, borrowID, condition, photosJSON string) error {
	if err := requireAdmin(ctx); err != nil {
		return err
	}
	var rec BorrowRecord
	if err := getJSON(ctx, borrowPrefix+borrowID, &rec); err != nil {
		return err
	}
	if rec.Status != "APPROVED" {
		return fmt.Errorf("借用单当前状态为 %s，不能确认领取", rec.Status)
	}
	if condition != "GOOD" && condition != "FAIR" {
		return fmt.Errorf("领取成色必须是 GOOD / FAIR")
	}
	var eq Equipment
	if err := getJSON(ctx, equipmentPrefix+rec.EquipmentID, &eq); err != nil {
		return err
	}
	now, err := transactionTime(ctx)
	if err != nil {
		return err
	}
	photos := []string{}
	if strings.TrimSpace(photosJSON) != "" {
		if err := json.Unmarshal([]byte(photosJSON), &photos); err != nil {
			return fmt.Errorf("照片列表必须是 JSON 数组")
		}
	}
	rec.Status, rec.PickupBy, rec.PickupAt, rec.PickupPhotos = "BORROWED", clientID(ctx), now, photos
	rec.Condition, rec.UpdatedAt = condition, now
	eq.Status, eq.TotalBorrows, eq.UpdatedAt, eq.UpdatedBy = "BORROWED", eq.TotalBorrows+1, now, clientID(ctx)
	if err := putJSON(ctx, borrowPrefix+borrowID, rec); err != nil {
		return err
	}
	if err := putJSON(ctx, equipmentPrefix+rec.EquipmentID, eq); err != nil {
		return err
	}
	return c.logActivity(ctx, "PICKUP_EQUIPMENT", "borrow", borrowID, rec.EquipmentID)
}

// RequestEquipmentReturn 发起归还（使用方）
func (c *XLabContract) RequestEquipmentReturn(ctx contractapi.TransactionContextInterface, borrowID, note string) error {
	if err := requireMember(ctx); err != nil {
		return err
	}
	var rec BorrowRecord
	if err := getJSON(ctx, borrowPrefix+borrowID, &rec); err != nil {
		return err
	}
	if rec.Status != "BORROWED" {
		return fmt.Errorf("借用单当前状态为 %s，不能发起归还", rec.Status)
	}
	if err := requireOwner(ctx, rec.BorrowerIdentity); err != nil {
		return err
	}
	now, err := transactionTime(ctx)
	if err != nil {
		return err
	}
	rec.Status, rec.ReturnBy, rec.ExceptionNote, rec.UpdatedAt = "RETURN_REQUESTED", clientID(ctx), note, now
	if err := putJSON(ctx, borrowPrefix+borrowID, rec); err != nil {
		return err
	}
	return c.logActivity(ctx, "REQUEST_RETURN", "borrow", borrowID, note)
}

// ConfirmEquipmentReturn 归还验收（仅管理方）
// condition: NORMAL 正常归还 / DAMAGED 损坏 / MISSING 缺件
func (c *XLabContract) ConfirmEquipmentReturn(ctx contractapi.TransactionContextInterface, borrowID, condition, evidenceHash, note, photosJSON string) error {
	if err := requireAdmin(ctx); err != nil {
		return err
	}
	var rec BorrowRecord
	if err := getJSON(ctx, borrowPrefix+borrowID, &rec); err != nil {
		return err
	}
	if rec.Status != "RETURN_REQUESTED" {
		return fmt.Errorf("借用单当前状态为 %s，不能验收", rec.Status)
	}
	var eq Equipment
	if err := getJSON(ctx, equipmentPrefix+rec.EquipmentID, &eq); err != nil {
		return err
	}
	now, err := transactionTime(ctx)
	if err != nil {
		return err
	}
	photos := []string{}
	if strings.TrimSpace(photosJSON) != "" {
		if err := json.Unmarshal([]byte(photosJSON), &photos); err != nil {
			return fmt.Errorf("照片列表必须是 JSON 数组")
		}
	}
	// 先校验并归一化结论，再写任何字段：与演示模式（store.go 统一 ToUpper）
	// 口径一致，也避免测试桩（无 MVCC 回滚）把非法结论写进状态。
	verdict := strings.ToUpper(strings.TrimSpace(condition))
	switch verdict {
	case "NORMAL", "DAMAGED", "MISSING":
	default:
		return fmt.Errorf("验收结论必须是 NORMAL / DAMAGED / MISSING")
	}
	rec.Condition, rec.EvidenceHash, rec.ExceptionNote, rec.ReturnAt, rec.ReturnPhotos, rec.UpdatedAt = verdict, evidenceHash, note, now, photos, now
	eq.UpdatedAt, eq.UpdatedBy = now, clientID(ctx)
	switch verdict {
	case "NORMAL":
		rec.Status, eq.Status, eq.ProjectID, eq.CurrentBorrow = "RETURNED", "AVAILABLE", "", ""
	case "DAMAGED", "MISSING":
		rec.Status, eq.Status = "EXCEPTION", "MAINTENANCE"
		eq.ProjectID, eq.CurrentBorrow = "", ""
		eq.Condition = "DAMAGED"
	}
	if err := putJSON(ctx, borrowPrefix+borrowID, rec); err != nil {
		return err
	}
	if err := putJSON(ctx, equipmentPrefix+rec.EquipmentID, eq); err != nil {
		return err
	}
	return c.logActivity(ctx, "CONFIRM_RETURN", "borrow", borrowID, verdict)
}

// ResolveException 异常处理完毕，设备恢复可借（仅管理方）
func (c *XLabContract) ResolveException(ctx contractapi.TransactionContextInterface, borrowID, note string) error {
	if err := requireAdmin(ctx); err != nil {
		return err
	}
	var rec BorrowRecord
	if err := getJSON(ctx, borrowPrefix+borrowID, &rec); err != nil {
		return err
	}
	if rec.Status != "EXCEPTION" {
		return fmt.Errorf("借用单当前状态为 %s，不是异常单", rec.Status)
	}
	var eq Equipment
	if err := getJSON(ctx, equipmentPrefix+rec.EquipmentID, &eq); err != nil {
		return err
	}
	now, err := transactionTime(ctx)
	if err != nil {
		return err
	}
	rec.Status, rec.ExceptionNote, rec.UpdatedAt = "RESOLVED", note, now
	eq.Status, eq.Condition, eq.UpdatedAt, eq.UpdatedBy = "AVAILABLE", "GOOD", now, clientID(ctx)
	if err := putJSON(ctx, borrowPrefix+borrowID, rec); err != nil {
		return err
	}
	if err := putJSON(ctx, equipmentPrefix+rec.EquipmentID, eq); err != nil {
		return err
	}
	return c.logActivity(ctx, "RESOLVE_EXCEPTION", "borrow", borrowID, note)
}

// ------------------------------ 查询 ------------------------------

// ReadEquipment 查询单台设备
func (c *XLabContract) ReadEquipment(ctx contractapi.TransactionContextInterface, id string) (*Equipment, error) {
	var result Equipment
	if err := getJSON(ctx, equipmentPrefix+id, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ReadProject 查询单个项目
func (c *XLabContract) ReadProject(ctx contractapi.TransactionContextInterface, id string) (*Project, error) {
	var result Project
	if err := getJSON(ctx, projectPrefix+id, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ReadBorrow 查询借用单
func (c *XLabContract) ReadBorrow(ctx contractapi.TransactionContextInterface, id string) (*BorrowRecord, error) {
	var result BorrowRecord
	if err := getJSON(ctx, borrowPrefix+id, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetAllEquipment 设备台账
func (c *XLabContract) GetAllEquipment(ctx contractapi.TransactionContextInterface) ([]Equipment, error) {
	items := []Equipment{}
	err := scanPrefix(ctx, equipmentPrefix, func(value []byte) error {
		var item Equipment
		if err := json.Unmarshal(value, &item); err != nil {
			return err
		}
		item.Accessories = nonNil(item.Accessories)
		items = append(items, item)
		return nil
	})
	return items, err
}

// nonNil 保证 JSON 序列化出数组而不是 null，满足 contractapi 的 schema 校验
func nonNil(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}

// GetAllSpaces 空间台账
func (c *XLabContract) GetAllSpaces(ctx contractapi.TransactionContextInterface) ([]Space, error) {
	items := []Space{}
	err := scanPrefix(ctx, spacePrefix, func(value []byte) error {
		var item Space
		if err := json.Unmarshal(value, &item); err != nil {
			return err
		}
		item.Amenities = nonNil(item.Amenities)
		items = append(items, item)
		return nil
	})
	return items, err
}

// GetAllProjects 全部项目
func (c *XLabContract) GetAllProjects(ctx contractapi.TransactionContextInterface) ([]Project, error) {
	items := []Project{}
	err := scanPrefix(ctx, projectPrefix, func(value []byte) error {
		var item Project
		if err := json.Unmarshal(value, &item); err != nil {
			return err
		}
		item.SpaceIDs = nonNil(item.SpaceIDs)
		item.Members = nonNil(item.Members)
		items = append(items, item)
		return nil
	})
	return items, err
}

// GetAllApplications 全部入驻申请
func (c *XLabContract) GetAllApplications(ctx contractapi.TransactionContextInterface) ([]ProjectApplication, error) {
	items := []ProjectApplication{}
	err := scanPrefix(ctx, applicationPrefix, func(value []byte) error {
		var item ProjectApplication
		if err := json.Unmarshal(value, &item); err != nil {
			return err
		}
		item.Members = nonNil(item.Members)
		item.PreferredSpace = nonNil(item.PreferredSpace)
		items = append(items, item)
		return nil
	})
	return items, err
}

// GetAllBorrows 全部借用单
func (c *XLabContract) GetAllBorrows(ctx contractapi.TransactionContextInterface) ([]BorrowRecord, error) {
	items := []BorrowRecord{}
	err := scanPrefix(ctx, borrowPrefix, func(value []byte) error {
		var item BorrowRecord
		if err := json.Unmarshal(value, &item); err != nil {
			return err
		}
		item.Overdue = isOverdue(item)
		if item.PickupPhotos == nil {
			item.PickupPhotos = []string{}
		}
		if item.ReturnPhotos == nil {
			item.ReturnPhotos = []string{}
		}
		items = append(items, item)
		return nil
	})
	return items, err
}

// GetAllocations 全部空间占用
func (c *XLabContract) GetAllocations(ctx contractapi.TransactionContextInterface) ([]SpaceAllocation, error) {
	items := []SpaceAllocation{}
	err := scanPrefix(ctx, allocationPrefix, func(value []byte) error {
		var item SpaceAllocation
		if err := json.Unmarshal(value, &item); err != nil {
			return err
		}
		items = append(items, item)
		return nil
	})
	return items, err
}

// GetActivityLog 全局操作流水（按时间倒序）
func (c *XLabContract) GetActivityLog(ctx contractapi.TransactionContextInterface) ([]Activity, error) {
	items := []Activity{}
	err := scanPrefix(ctx, activityPrefix, func(value []byte) error {
		var item Activity
		if err := json.Unmarshal(value, &item); err != nil {
			return err
		}
		items = append(items, item)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Timestamp == items[j].Timestamp {
			return items[i].TxID > items[j].TxID
		}
		return items[i].Timestamp > items[j].Timestamp
	})
	if len(items) > 200 {
		items = items[:200]
	}
	return items, nil
}

// GetHistory 单对象变更史
func (c *XLabContract) GetHistory(ctx contractapi.TransactionContextInterface, objectType, id string) ([]HistoryEntry, error) {
	prefixes := map[string]string{
		"equipment": equipmentPrefix, "space": spacePrefix, "project": projectPrefix,
		"application": applicationPrefix, "borrow": borrowPrefix, "allocation": allocationPrefix,
	}
	prefix, ok := prefixes[strings.ToLower(objectType)]
	if !ok {
		return nil, fmt.Errorf("不支持的对象类型 %s", objectType)
	}
	iterator, err := ctx.GetStub().GetHistoryForKey(prefix + id)
	if err != nil {
		return nil, err
	}
	defer iterator.Close()
	entries := []HistoryEntry{}
	for iterator.HasNext() {
		item, err := iterator.Next()
		if err != nil {
			return nil, err
		}
		entry := HistoryEntry{TxID: item.TxId, Timestamp: item.Timestamp.AsTime().UTC().Format(time.RFC3339), IsDelete: item.IsDelete}
		if !item.IsDelete {
			entry.Value = string(item.Value)
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// GetStats 看板统计
func (c *XLabContract) GetStats(ctx contractapi.TransactionContextInterface) (*Stats, error) {
	s := &Stats{ByCategory: map[string]int{}, ByProjectStatus: map[string]int{}}

	eqs, err := c.GetAllEquipment(ctx)
	if err != nil {
		return nil, err
	}
	for _, e := range eqs {
		s.EquipmentTotal++
		s.ByCategory[e.Category]++
		switch e.Status {
		case "AVAILABLE":
			s.EquipmentAvail++
		case "BORROWED", "APPROVED", "REQUESTED":
			s.EquipmentOut++
		case "MAINTENANCE":
			s.EquipmentFix++
		}
	}

	spaces, err := c.GetAllSpaces(ctx)
	if err != nil {
		return nil, err
	}
	s.SpaceTotal = len(spaces)
	// 空间是否可分配取决于是否有生效中的分配记录，而不是空间自身的状态字段
	occupied := map[string]bool{}
	err = scanPrefix(ctx, allocationPrefix, func(value []byte) error {
		var alloc SpaceAllocation
		if err := json.Unmarshal(value, &alloc); err != nil {
			return err
		}
		if alloc.Status == "ACTIVE" {
			occupied[alloc.SpaceID] = true
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for _, sp := range spaces {
		if sp.Status == "OPEN" && !occupied[sp.ID] {
			s.SpaceFree++
		}
	}

	projects, err := c.GetAllProjects(ctx)
	if err != nil {
		return nil, err
	}
	for _, p := range projects {
		s.ByProjectStatus[p.Status]++
		if p.Status == "ACTIVE" {
			s.ProjectActive++
		}
	}

	borrows, err := c.GetAllBorrows(ctx)
	if err != nil {
		return nil, err
	}
	for _, b := range borrows {
		if b.Status == "PENDING" {
			s.BorrowPending++
		}
	}

	apps, err := c.GetAllApplications(ctx)
	if err != nil {
		return nil, err
	}
	for _, a := range apps {
		if a.Status == "SUBMITTED" {
			s.AppPending++
		}
	}
	return s, nil
}

// ------------------------------ 内部工具 ------------------------------

func (c *XLabContract) hasSpaceConflict(ctx contractapi.TransactionContextInterface, spaceID, startDate, endDate string) (bool, error) {
	return c.hasSpaceConflictExcluding(ctx, spaceID, startDate, endDate, "")
}

func (c *XLabContract) hasSpaceConflictExcluding(ctx contractapi.TransactionContextInterface, spaceID, startDate, endDate, excludeProjectID string) (bool, error) {
	start, end, err := parsePeriod(startDate, endDate)
	if err != nil {
		return false, err
	}
	conflict := false
	err = scanPrefix(ctx, allocationPrefix, func(value []byte) error {
		var item SpaceAllocation
		if err := json.Unmarshal(value, &item); err != nil {
			return err
		}
		if item.SpaceID != spaceID || item.Status != "ACTIVE" || item.ProjectID == excludeProjectID {
			return nil
		}
		otherStart, otherEnd, err := parsePeriod(item.StartDate, item.EndDate)
		if err != nil {
			return err
		}
		if start.Before(otherEnd) && end.After(otherStart) {
			conflict = true
		}
		return nil
	})
	return conflict, err
}

func (c *XLabContract) countOutstandingBorrows(ctx contractapi.TransactionContextInterface, projectID string) (int, error) {
	count := 0
	err := scanPrefix(ctx, borrowPrefix, func(value []byte) error {
		var item BorrowRecord
		if err := json.Unmarshal(value, &item); err != nil {
			return err
		}
		if item.ProjectID != projectID {
			return nil
		}
		switch item.Status {
		case "BORROWED", "RETURN_REQUESTED", "APPROVED", "PENDING", "EXCEPTION":
			count++
		}
		return nil
	})
	return count, err
}

// hasActiveBorrowForEquipment 判断设备是否存在未结清的借用单（含未处理异常）。
// 与 countOutstandingBorrows 同一状态集合，保证维护入口与结项检查口径一致。
func (c *XLabContract) hasActiveBorrowForEquipment(ctx contractapi.TransactionContextInterface, equipmentID string) (bool, error) {
	active := false
	err := scanPrefix(ctx, borrowPrefix, func(value []byte) error {
		var item BorrowRecord
		if err := json.Unmarshal(value, &item); err != nil {
			return err
		}
		if item.EquipmentID != equipmentID {
			return nil
		}
		switch item.Status {
		case "BORROWED", "RETURN_REQUESTED", "APPROVED", "PENDING", "EXCEPTION":
			active = true
		}
		return nil
	})
	return active, err
}

func (c *XLabContract) logActivity(ctx contractapi.TransactionContextInterface, action, objectType, objectID, detail string) error {
	now, err := transactionTime(ctx)
	if err != nil {
		return err
	}
	act := Activity{
		TxID: ctx.GetStub().GetTxID(), Action: action, ObjectType: objectType,
		ObjectID: objectID, Actor: clientID(ctx), ActorMSP: mspID(ctx), Detail: detail, Timestamp: now,
	}
	return putJSON(ctx, activityPrefix+ctx.GetStub().GetTxID(), act)
}

// requireAdmin 仅允许管理方组织执行
func requireAdmin(ctx contractapi.TransactionContextInterface) error {
	msp := mspID(ctx)
	if msp != AdminMSP {
		return fmt.Errorf("权限不足：该操作仅允许 %s 执行，当前身份 %s（%s）", AdminMSP, clientID(ctx), msp)
	}
	return nil
}

// requireMember 仅允许入驻项目团队组织执行。
func requireMember(ctx contractapi.TransactionContextInterface) error {
	msp := mspID(ctx)
	if msp != UserMSP {
		return fmt.Errorf("权限不足：该操作仅允许 %s 执行，当前身份 %s（%s）", UserMSP, clientID(ctx), msp)
	}
	return nil
}

func requireOwner(ctx contractapi.TransactionContextInterface, ownerIdentity string) error {
	currentIdentity, err := verifiedClientID(ctx)
	if err != nil {
		return err
	}
	if strings.TrimSpace(ownerIdentity) == "" || ownerIdentity != currentIdentity {
		return fmt.Errorf("对象归属校验失败：当前身份无权操作该记录")
	}
	return nil
}

func mspID(ctx contractapi.TransactionContextInterface) string {
	msp, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return "unknown"
	}
	return msp
}

func clientID(ctx contractapi.TransactionContextInterface) string {
	id, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return "unknown"
	}
	return id
}

func verifiedClientID(ctx contractapi.TransactionContextInterface) (string, error) {
	id, err := ctx.GetClientIdentity().GetID()
	if err != nil || strings.TrimSpace(id) == "" {
		return "", fmt.Errorf("无法验证 Fabric 客户身份")
	}
	return id, nil
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

func transactionTime(ctx contractapi.TransactionContextInterface) (string, error) {
	ts, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return "", err
	}
	return ts.AsTime().UTC().Format(time.RFC3339), nil
}

func getJSON(ctx contractapi.TransactionContextInterface, key string, target interface{}) error {
	value, err := ctx.GetStub().GetState(key)
	if err != nil {
		return err
	}
	if value == nil {
		return fmt.Errorf("%s 不存在", key)
	}
	return json.Unmarshal(value, target)
}

func putJSON(ctx contractapi.TransactionContextInterface, key string, value interface{}) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return ctx.GetStub().PutState(key, payload)
}

func statePrefixExists(ctx contractapi.TransactionContextInterface, prefix string) (bool, error) {
	iterator, err := ctx.GetStub().GetStateByRange(prefix, prefix+"\uffff")
	if err != nil {
		return false, err
	}
	defer iterator.Close()
	return iterator.HasNext(), nil
}

func requireMissing(ctx contractapi.TransactionContextInterface, key string) error {
	value, err := ctx.GetStub().GetState(key)
	if err != nil {
		return err
	}
	if value != nil {
		return fmt.Errorf("%s 已存在", key)
	}
	return nil
}

func scanPrefix(ctx contractapi.TransactionContextInterface, prefix string, visit func([]byte) error) error {
	iterator, err := ctx.GetStub().GetStateByRange(prefix, prefix+"\uffff")
	if err != nil {
		return err
	}
	defer iterator.Close()
	for iterator.HasNext() {
		item, err := iterator.Next()
		if err != nil {
			return err
		}
		if err := visit(item.Value); err != nil {
			return err
		}
	}
	return nil
}
