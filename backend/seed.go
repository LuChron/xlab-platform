package main

import (
	"fmt"
	"strings"
)

// 种子数据：与链码 InitLedger 保持一致。
// 演示模式（Fabric 未启动）下使用这份数据，保证答辩现场不受容器环境影响。

// 组织：管理方 = X-Lab 资源管理委员会；使用方 = 入驻项目团队
const (
	AdminMSP = "Org1MSP"
	UserMSP  = "Org2MSP"
)

func demoSpaces() []Space {
	return []Space{
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
}

func demoEquipment() []Equipment {
	return []Equipment{
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
}

// seedDemo 构造一份带历史记录的演示账本
func seedDemo() *Ledger {
	now := nowISO()
	l := &Ledger{Mode: "demo", Channel: "xlabchannel", Chaincode: "xlab"}

	l.Spaces = demoSpaces()
	l.Equipment = demoEquipment()
	for i := range l.Equipment {
		l.Equipment[i].CreatedAt = now
		l.Equipment[i].UpdatedAt = now
		l.Equipment[i].UpdatedBy = "Org1MSP/Admin"
	}

	// 一个已入驻的示例项目：交互式学习装置
	app := ProjectApplication{
		ID: "APP-2026-001", ProjectName: "交互式学习装置", LeadID: "linxy", LeadName: "林同学",
		Members: []string{"林同学", "周同学", "沈同学"}, MemberCount: 3, RequestedSeats: 2,
		PreferredSpace: []string{"A2", "B1"}, StartDate: "2026-10-12", EndDate: "2026-11-09",
		IdeaSummary: "通过体感交互与可视化反馈，帮助学习者理解抽象的空间几何概念。",
		IdeaHash:    "9f2c41d7a8e3b05c6d19f4a2b7e8c031d5a6f9b2e4c7d1a8f3b6c9e2d5a8f1b4",
		Milestone:   "完成可操作原型，并在大屏区做一次现场测试。",
		Status:      "ACTIVATED", Reviewer: "Org1MSP/Admin", ReviewedAt: "2026-10-11T02:15:00Z",
		ReviewNote: "idea 清晰，空间需求合理，批准入驻。", ProjectID: "PRJ-2026-001",
		CreatedAt: "2026-10-10T08:30:00Z", UpdatedAt: "2026-10-12T01:00:00Z",
	}
	l.Applications = append(l.Applications, app)

	proj := Project{
		ID: "PRJ-2026-001", ApplicationID: app.ID, Name: app.ProjectName,
		LeadID: app.LeadID, LeadName: app.LeadName, Members: app.Members, MemberCount: app.MemberCount,
		Summary: app.IdeaSummary, Status: "ACTIVE", SpaceIDs: []string{"A2", "B1"},
		StartDate: app.StartDate, EndDate: app.EndDate, Milestone: app.Milestone,
		MilestoneHash: "c4d8e2f6a1b5c9d3e7f2a6b1c5d9e3f7a2b6c1d5e9f3a7b2c6d1e5f9a3b7c2d6",
		Extensions:    0, CreatedAt: "2026-10-12T01:00:00Z", UpdatedAt: "2026-10-16T06:20:00Z",
	}
	l.Projects = append(l.Projects, proj)

	l.Allocations = append(l.Allocations,
		SpaceAllocation{ID: "PRJ-2026-001:A2", SpaceID: "A2", ProjectID: "PRJ-2026-001", StartDate: "2026-10-12", EndDate: "2026-11-09", Status: "ACTIVE", CreatedAt: "2026-10-12T01:00:00Z"},
		SpaceAllocation{ID: "PRJ-2026-001:B1", SpaceID: "B1", ProjectID: "PRJ-2026-001", StartDate: "2026-10-12", EndDate: "2026-11-09", Status: "ACTIVE", CreatedAt: "2026-10-12T01:00:00Z"},
	)

	// 一台设备正在借出中，演示“责任关联到项目”
	rec := BorrowRecord{
		ID: "BR-2026-001", EquipmentID: "EQ-002", EquipmentName: "Dell U2723QE 显示器",
		ProjectID: "PRJ-2026-001", Borrower: "林同学",
		Purpose: "原型开发需要双屏对照调试", PurposeHash: "a1b2c3d4e5f6",
		DueDate: "2026-11-05", Status: "BORROWED",
		Reviewer: "Org1MSP/Admin", ReviewedAt: "2026-10-13T02:00:00Z",
		PickupBy: "Org1MSP/Admin", PickupAt: "2026-10-13T02:10:00Z",
		Condition: "GOOD", CreatedAt: "2026-10-13T01:40:00Z", UpdatedAt: "2026-10-13T02:10:00Z",
	}
	l.Borrows = append(l.Borrows, rec)
	markBorrowed(l, "EQ-002", "PRJ-2026-001", rec.ID)

	// 一条已完成的归还，展示闭环
	rec2 := BorrowRecord{
		ID: "BR-2026-002", EquipmentID: "EQ-009", EquipmentName: "Arduino 创客套件",
		ProjectID: "PRJ-2026-001", Borrower: "周同学",
		Purpose: "传感器选型验证", PurposeHash: "f6e5d4c3b2a1", DueDate: "2026-10-20",
		Status: "RETURNED", Reviewer: "Org1MSP/Admin", ReviewedAt: "2026-10-14T03:00:00Z",
		PickupBy: "Org1MSP/Admin", PickupAt: "2026-10-14T03:10:00Z",
		ReturnBy: "Org2MSP/TeamUser", ReturnAt: "2026-10-16T07:00:00Z",
		Condition: "NORMAL", EvidenceHash: "b7c8d9e0f1a2", CreatedAt: "2026-10-14T02:50:00Z", UpdatedAt: "2026-10-16T07:00:00Z",
	}
	l.Borrows = append(l.Borrows, rec2)

	// 一条待审批申请，让管理员工作台有内容
	app2 := ProjectApplication{
		ID: "APP-2026-002", ProjectName: "无障碍校园地图", LeadID: "zhaox", LeadName: "赵同学",
		Members: []string{"赵同学", "钱同学"}, MemberCount: 2, RequestedSeats: 2,
		PreferredSpace: []string{"A3", "A4"}, StartDate: "2026-10-25", EndDate: "2026-11-22",
		IdeaSummary: "为行动不便的同学标注校园无障碍路线与设施，提供可离线使用的导航。",
		IdeaHash:    "3e8b1f5a9c2d6e0b4f7a1c5e9d2b6f0a4c8e1d5b9f3a7c2e6d0b4f8a2c6e1d5b",
		Milestone:   "完成 20 个点位的实地采集与路线验证。", Status: "SUBMITTED",
		CreatedAt: "2026-10-17T05:00:00Z", UpdatedAt: "2026-10-17T05:00:00Z",
	}
	l.Applications = append(l.Applications, app2)

	// 一条待审批借用
	rec3 := BorrowRecord{
		ID: "BR-2026-003", EquipmentID: "EQ-007", EquipmentName: "便携投影仪",
		ProjectID: "PRJ-2026-001", Borrower: "沈同学", Purpose: "中期评审现场演示",
		PurposeHash: "c9d0e1f2a3b4", DueDate: "2026-11-02", Status: "PENDING",
		CreatedAt: "2026-10-17T09:30:00Z", UpdatedAt: "2026-10-17T09:30:00Z",
	}
	l.Borrows = append(l.Borrows, rec3)
	markRequested(l, "EQ-007", "PRJ-2026-001", rec3.ID)

	// 操作流水
	l.Activities = []Activity{
		{TxID: "8f2a1c4e6b9d0f3a5c7e2b4d6f8a1c3e5b7d9f2a4c6e8b1d3f5a7c9e2b4d6f8a", Action: "SUBMIT_APPLICATION", ObjectType: "application", ObjectID: "APP-2026-002", Actor: "Org2MSP/TeamUser", ActorMSP: "Org2MSP", Detail: "无障碍校园地图", Timestamp: "2026-10-17T05:00:00Z"},
		{TxID: "a3b5d7f9c1e3a5b7d9f1c3e5a7b9d1f3c5e7a9b1d3f5c7e9a1b3d5f7c9e1a3b5", Action: "REQUEST_BORROW", ObjectType: "borrow", ObjectID: "BR-2026-003", Actor: "Org2MSP/TeamUser", ActorMSP: "Org2MSP", Detail: "EQ-007 → 沈同学", Timestamp: "2026-10-17T09:30:00Z"},
		{TxID: "b2c4e6a8d0f2b4c6e8a0d2f4b6c8e0a2d4f6b8c0e2a4d6f8b0c2e4a6d8f0b2c4", Action: "CONFIRM_RETURN", ObjectType: "borrow", ObjectID: "BR-2026-002", Actor: "Org1MSP/Admin", ActorMSP: "Org1MSP", Detail: "NORMAL", Timestamp: "2026-10-16T07:00:00Z"},
		{TxID: "d1e3f5a7c9b1d3f5a7c9e1b3d5f7a9c1e3b5d7f9a1c3e5b7d9f1a3c5e7b9d1e3", Action: "SUBMIT_MILESTONE", ObjectType: "project", ObjectID: "PRJ-2026-001", Actor: "Org2MSP/TeamUser", ActorMSP: "Org2MSP", Detail: "完成可操作原型，并在大屏区做一次现场测试。", Timestamp: "2026-10-16T06:20:00Z"},
		{TxID: "f5a7c9e1b3d5f7a9c1e3b5d7f9a1c3e5b7d9f1a3c5e7b9d1e3f5a7c9b1d3f5a7", Action: "PICKUP_EQUIPMENT", ObjectType: "borrow", ObjectID: "BR-2026-002", Actor: "Org1MSP/Admin", ActorMSP: "Org1MSP", Detail: "EQ-009", Timestamp: "2026-10-14T03:10:00Z"},
		{TxID: "c9e1b3d5f7a9c1e3b5d7f9a1c3e5b7d9f1a3c5e7b9d1e3f5a7c9b1d3f5a7c9e1", Action: "REVIEW_BORROW", ObjectType: "borrow", ObjectID: "BR-2026-002", Actor: "Org1MSP/Admin", ActorMSP: "Org1MSP", Detail: "approved=true", Timestamp: "2026-10-14T03:00:00Z"},
		{TxID: "e7b9d1f3a5c7e9b1d3f5a7c9e1b3d5f7a9c1e3b5d7f9a1c3e5b7d9f1a3c5e7b9", Action: "REQUEST_BORROW", ObjectType: "borrow", ObjectID: "BR-2026-002", Actor: "Org2MSP/TeamUser", ActorMSP: "Org2MSP", Detail: "EQ-009 → 周同学", Timestamp: "2026-10-14T02:50:00Z"},
		{TxID: "a1c3e5b7d9f1a3c5e7b9d1f3a5c7e9b1d3f5a7c9e1b3d5f7a9c1e3b5d7f9a1c3", Action: "PICKUP_EQUIPMENT", ObjectType: "borrow", ObjectID: "BR-2026-001", Actor: "Org1MSP/Admin", ActorMSP: "Org1MSP", Detail: "EQ-002", Timestamp: "2026-10-13T02:10:00Z"},
		{TxID: "b3d5f7a9c1e3b5d7f9a1c3e5b7d9f1a3c5e7b9d1e3f5a7c9b1d3f5a7c9e1b3d5", Action: "REVIEW_BORROW", ObjectType: "borrow", ObjectID: "BR-2026-001", Actor: "Org1MSP/Admin", ActorMSP: "Org1MSP", Detail: "approved=true", Timestamp: "2026-10-13T02:00:00Z"},
		{TxID: "c5e7a9b1d3f5c7e9a1b3d5f7c9e1a3b5d7f9a1c3e5b7d9f1a3c5e7b9d1e3f5c7", Action: "REQUEST_BORROW", ObjectType: "borrow", ObjectID: "BR-2026-001", Actor: "Org2MSP/TeamUser", ActorMSP: "Org2MSP", Detail: "EQ-002 → 林同学", Timestamp: "2026-10-13T01:40:00Z"},
		{TxID: "d7f9a1c3e5b7d9f1a3c5e7b9d1e3f5a7c9b1d3f5a7c9e1b3d5f7a9c1e3b5d7f9", Action: "ACTIVATE_PROJECT", ObjectType: "project", ObjectID: "PRJ-2026-001", Actor: "Org1MSP/Admin", ActorMSP: "Org1MSP", Detail: "空间: A2,B1", Timestamp: "2026-10-12T01:00:00Z"},
		{TxID: "e9b1d3f5a7c9e1b3d5f7a9c1e3b5d7f9a1c3e5b7d9f1a3c5e7b9d1e3f5a7c9e1", Action: "REVIEW_APPLICATION", ObjectType: "application", ObjectID: "APP-2026-001", Actor: "Org1MSP/Admin", ActorMSP: "Org1MSP", Detail: "approved=true", Timestamp: "2026-10-11T02:15:00Z"},
		{TxID: "f1a3c5e7b9d1e3f5a7c9b1d3f5a7c9e1b3d5f7a9c1e3b5d7f9a1c3e5b7d9f1a3", Action: "SUBMIT_APPLICATION", ObjectType: "application", ObjectID: "APP-2026-001", Actor: "Org2MSP/TeamUser", ActorMSP: "Org2MSP", Detail: "交互式学习装置", Timestamp: "2026-10-10T08:30:00Z"},
		{TxID: "a5c7e9b1d3f5a7c9e1b3d5f7a9c1e3b5d7f9a1c3e5b7d9f1a3c5e7b9d1e3f5a7", Action: "INIT_LEDGER", ObjectType: "ledger", ObjectID: "xlab", Actor: "Org1MSP/Admin", ActorMSP: "Org1MSP", Detail: "初始化 18 个空间与 28 台设备", Timestamp: "2026-10-09T00:00:00Z"},
	}

	recomputeStats(l)
	return l
}

func markBorrowed(l *Ledger, eqID, projectID, borrowID string) {
	for i := range l.Equipment {
		if l.Equipment[i].ID == eqID {
			l.Equipment[i].Status = "BORROWED"
			l.Equipment[i].ProjectID = projectID
			l.Equipment[i].CurrentBorrow = borrowID
			l.Equipment[i].TotalBorrows++
		}
	}
}

func markRequested(l *Ledger, eqID, projectID, borrowID string) {
	for i := range l.Equipment {
		if l.Equipment[i].ID == eqID {
			l.Equipment[i].Status = "REQUESTED"
			l.Equipment[i].ProjectID = projectID
			l.Equipment[i].CurrentBorrow = borrowID
		}
	}
}

// recomputeStats 重新计算看板统计
func recomputeStats(l *Ledger) {
	s := Stats{ByCategory: map[string]int{}, ByProjectStatus: map[string]int{}}
	for _, e := range l.Equipment {
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
	s.SpaceTotal = len(l.Spaces)
	// 与链码保持一致：空间是否可分配取决于生效中的分配记录
	occupied := map[string]bool{}
	for _, a := range l.Allocations {
		if a.Status == "ACTIVE" {
			occupied[a.SpaceID] = true
		}
	}
	for _, sp := range l.Spaces {
		if sp.Status == "OPEN" && !occupied[sp.ID] {
			s.SpaceFree++
		}
	}
	for _, p := range l.Projects {
		s.ByProjectStatus[p.Status]++
		if p.Status == "ACTIVE" {
			s.ProjectActive++
		}
	}
	for _, b := range l.Borrows {
		if b.Status == "PENDING" {
			s.BorrowPending++
		}
	}
	for _, a := range l.Applications {
		if a.Status == "SUBMITTED" {
			s.AppPending++
		}
	}
	l.Stats = s
}

// 防止 SQL 注入类问题：所有 ID 统一清洗
func sanitizeID(s string) error {
	if strings.TrimSpace(s) == "" {
		return fmt.Errorf("标识不能为空")
	}
	if len(s) > 64 {
		return fmt.Errorf("标识过长")
	}
	return nil
}
