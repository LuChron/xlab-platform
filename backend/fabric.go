package main

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	common "github.com/hyperledger/fabric-protos-go-apiv2/common"
	gwproto "github.com/hyperledger/fabric-protos-go-apiv2/gateway"
	msp "github.com/hyperledger/fabric-protos-go-apiv2/msp"
	peer "github.com/hyperledger/fabric-protos-go-apiv2/peer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// fabricStore 通过 Fabric Gateway 访问真实账本。
// 连接参数与 hw7 的 SDK 一致：gRPC + TLS + X509 身份 + 私钥签名。
type fabricStore struct {
	mu        sync.Mutex
	contract  *client.Contract
	conn      *grpc.ClientConn
	gateway   *client.Gateway
	channel   string
	chaincode string

	// 多组织签名：每个组织一份 Gateway 连接，按调用方身份选合同对象。
	// actor 只决定本次交易由谁签名，读接口不受影响。
	cfg    FabricConfig
	peers  map[string]*orgPeer
	peerMu *sync.RWMutex
	actor  string
	shared bool
}

// orgPeer 一个组织到 peer 的已建立连接
type orgPeer struct {
	mspID    string
	contract *client.Contract
	conn     *grpc.ClientConn
	gateway  *client.Gateway
}

// FabricConfig Fabric 连接配置
type FabricConfig struct {
	MSPID         string
	CryptoPath    string
	CertDir       string
	KeyDir        string
	TLSCertPath   string
	PeerEndpoint  string
	GatewayPeer   string
	ChannelName   string
	ChaincodeName string
}

// fabricConfigFromEnv 从环境变量读取配置，默认值对齐 hw7 的 test-network 布局
func fabricConfigFromEnv() FabricConfig {
	crypto := envOr("XLAB_CRYPTO_PATH",
		"../../hw7/fabric/scripts/fabric-samples/test-network/organizations/peerOrganizations/org1.example.com")
	return FabricConfig{
		MSPID:         envOr("XLAB_MSP_ID", "Org1MSP"),
		CryptoPath:    crypto,
		CertDir:       envOr("XLAB_CERT_DIR", crypto+"/users/User1@org1.example.com/msp/signcerts"),
		KeyDir:        envOr("XLAB_KEY_DIR", crypto+"/users/User1@org1.example.com/msp/keystore"),
		TLSCertPath:   envOr("XLAB_TLS_CERT", crypto+"/peers/peer0.org1.example.com/tls/ca.crt"),
		PeerEndpoint:  envOr("XLAB_PEER_ENDPOINT", "localhost:7051"),
		GatewayPeer:   envOr("XLAB_GATEWAY_PEER", "peer0.org1.example.com"),
		ChannelName:   envOr("XLAB_CHANNEL", "xlabchannel"),
		ChaincodeName: envOr("XLAB_CHAINCODE", "xlab"),
	}
}

// org2Config 由 org1 的配置推导出 Org2 团队的签名身份。
// 链码里的 actorMsp 直接取自交易证书，如果所有请求都用 Org1 私钥签名，
// 账本上就只剩一个调用方，「谁在什么时候拿了什么」也就无从核验。
func org2Config(base FabricConfig) FabricConfig {
	org1 := "organizations/peerOrganizations/org1.example.com"
	org2 := envOr("XLAB_ORG2_CRYPTO_PATH", strings.Replace(base.CryptoPath, org1, "organizations/peerOrganizations/org2.example.com", 1))
	cfg := base
	cfg.MSPID = "Org2MSP"
	cfg.CryptoPath = org2
	cfg.CertDir = org2 + "/users/User1@org2.example.com/msp/signcerts"
	cfg.KeyDir = org2 + "/users/User1@org2.example.com/msp/keystore"
	cfg.TLSCertPath = org2 + "/peers/peer0.org2.example.com/tls/ca.crt"
	cfg.PeerEndpoint = envOr("XLAB_ORG2_PEER_ENDPOINT", "localhost:9051")
	cfg.GatewayPeer = "peer0.org2.example.com"
	return cfg
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func newFabricStore(cfg FabricConfig) (*fabricStore, error) {
	conn, err := newGrpcConnection(cfg)
	if err != nil {
		return nil, fmt.Errorf("建立 gRPC 连接失败: %w", err)
	}

	gw, err := client.Connect(
		newIdentity(cfg),
		client.WithSign(newSign(cfg)),
		client.WithClientConnection(conn),
		client.WithEvaluateTimeout(10*time.Second),
		client.WithEndorseTimeout(15*time.Second),
		client.WithSubmitTimeout(10*time.Second),
		client.WithCommitStatusTimeout(60*time.Second),
	)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("连接 Gateway 失败: %w", err)
	}

	contract := gw.GetNetwork(cfg.ChannelName).GetContract(cfg.ChaincodeName)
	return &fabricStore{
		contract: contract, conn: conn, gateway: gw,
		channel: cfg.ChannelName, chaincode: cfg.ChaincodeName,
		cfg:    cfg,
		actor:  cfg.MSPID,
		peerMu: new(sync.RWMutex),
		peers: map[string]*orgPeer{
			cfg.MSPID: {mspID: cfg.MSPID, contract: contract, conn: conn, gateway: gw},
		},
	}, nil
}

// For 返回以指定组织身份签名的视图：读接口共享同一批连接，写接口改用该组织的证书与私钥。
// 链码里的 actorMsp 取自交易证书，所以团队提交的申请必须由 Org2 签名，
// 否则账本上所有操作都记在委员会名下，审计就失去意义。
func (f *fabricStore) For(mspID string) (Store, error) {
	if mspID == "" {
		return nil, fmt.Errorf("组织身份不能为空")
	}
	if mspID == f.actor {
		return f, nil
	}
	f.peerMu.RLock()
	_, exists := f.peers[mspID]
	f.peerMu.RUnlock()
	if !exists {
		if err := f.dialOrg(mspID); err != nil {
			return nil, fmt.Errorf("%s 身份不可用: %w", mspID, err)
		}
	}
	return &fabricStore{
		contract: f.contract, conn: f.conn, gateway: f.gateway,
		channel: f.channel, chaincode: f.chaincode,
		cfg: f.cfg, actor: mspID, peers: f.peers, peerMu: f.peerMu, shared: true,
	}, nil
}

// dialOrg 为其它组织建立 Gateway 连接（复用同一份 peer 配置，仅换证书与私钥）
func (f *fabricStore) dialOrg(mspID string) error {
	f.peerMu.Lock()
	defer f.peerMu.Unlock()
	if _, ok := f.peers[mspID]; ok {
		return nil
	}

	var cfg FabricConfig
	switch mspID {
	case "Org2MSP":
		cfg = org2Config(f.cfg)
	default:
		return fmt.Errorf("未配置的组织 %s", mspID)
	}
	conn, err := newGrpcConnection(cfg)
	if err != nil {
		return fmt.Errorf("建立 gRPC 连接失败: %w", err)
	}
	gw, err := client.Connect(
		newIdentity(cfg),
		client.WithSign(newSign(cfg)),
		client.WithClientConnection(conn),
		client.WithEvaluateTimeout(10*time.Second),
		client.WithEndorseTimeout(15*time.Second),
		client.WithSubmitTimeout(10*time.Second),
		client.WithCommitStatusTimeout(60*time.Second),
	)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("连接 Gateway 失败: %w", err)
	}
	f.peers[mspID] = &orgPeer{
		mspID:    mspID,
		contract: gw.GetNetwork(cfg.ChannelName).GetContract(cfg.ChaincodeName),
		conn:     conn, gateway: gw,
	}
	return nil
}

// writeContract 当前身份对应的合同对象
func (f *fabricStore) writeContract() (*client.Contract, error) {
	f.peerMu.RLock()
	defer f.peerMu.RUnlock()
	if p, ok := f.peers[f.actor]; ok {
		return p.contract, nil
	}
	return nil, fmt.Errorf("组织 %s 的 Fabric 签名连接不可用", f.actor)
}

func (f *fabricStore) Mode() string { return "fabric" }

// Chain 通过 qscc 系统链码读取真实链状态：高度、当前/前一区块哈希、最近若干区块的解码信息。
// 区块哈希按 Fabric 官方算法（ASN.1 编码 header 三元组后 sha256）计算，保证"前后哈希首尾相接"可验证。
func (f *fabricStore) Chain() (*ChainInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	qscc := f.gateway.GetNetwork(f.channel).GetContract("qscc")

	// 1) 链状态
	infoBytes, err := qscc.EvaluateTransaction("GetChainInfo", f.channel)
	if err != nil {
		return nil, fmt.Errorf("查询链信息失败: %s", chaincodeReason(err))
	}
	info := &common.BlockchainInfo{}
	if err := proto.Unmarshal(infoBytes, info); err != nil {
		return nil, fmt.Errorf("解析链信息失败: %w", err)
	}

	out := &ChainInfo{
		Height:      info.Height,
		CurrentHash: hex.EncodeToString(info.CurrentBlockHash),
		PrevHash:    hex.EncodeToString(info.PreviousBlockHash),
		Blocks:      []Block{},
	}

	// 2) 最近 N 个区块（0 是创世块，一般不包含业务交易，但保留完整链条）
	const maxBlocks = 8
	start := uint64(0)
	if info.Height > maxBlocks {
		start = info.Height - maxBlocks
	}
	for n := start; n < info.Height; n++ {
		blk, err := f.fetchBlock(qscc, n)
		if err != nil {
			// 单个区块失败不阻断整页，跳过并在日志留痕
			log.Printf("[XLAB] 读取区块 %d 失败: %v", n, err)
			continue
		}
		out.Blocks = append(out.Blocks, *blk)
	}
	return out, nil
}

// fetchBlock 用 qscc GetBlockByNumber 拉一个区块并解码交易。
func (f *fabricStore) fetchBlock(qscc *client.Contract, number uint64) (*Block, error) {
	raw, err := qscc.EvaluateTransaction("GetBlockByNumber", f.channel, strconv.FormatUint(number, 10))
	if err != nil {
		return nil, fmt.Errorf("查询区块失败: %s", chaincodeReason(err))
	}
	b := &common.Block{}
	if err := proto.Unmarshal(raw, b); err != nil {
		return nil, fmt.Errorf("解析区块失败: %w", err)
	}

	blk := &Block{
		Number:   b.Header.Number,
		Hash:     hex.EncodeToString(blockHeaderHash(b.Header)),
		PrevHash: hex.EncodeToString(b.Header.PreviousHash),
		DataHash: hex.EncodeToString(b.Header.DataHash),
		TxCount:  len(b.Data.Data),
		Txs:      []BlockTx{},
	}

	for _, envelopeBytes := range b.Data.Data {
		tx := decodeTx(envelopeBytes)
		if tx != nil {
			blk.Txs = append(blk.Txs, *tx)
		}
	}
	return blk, nil
}

// decodeTx 解码区块里的一笔交易（Envelope → Payload → ChannelHeader + Transaction → 链码调用参数）。
// 失败时返回 nil，不阻断整个区块的展示。
func decodeTx(envelopeBytes []byte) *BlockTx {
	env := &common.Envelope{}
	if err := proto.Unmarshal(envelopeBytes, env); err != nil {
		return nil
	}
	payload := &common.Payload{}
	if err := proto.Unmarshal(env.Payload, payload); err != nil {
		return nil
	}
	chHeader := &common.ChannelHeader{}
	if err := proto.Unmarshal(payload.Header.ChannelHeader, chHeader); err != nil {
		return nil
	}

	tx := &BlockTx{TxID: chHeader.TxId}
	if chHeader.Timestamp != nil {
		tx.Timestamp = chHeader.Timestamp.AsTime().Format(time.RFC3339)
	}

	// 发起组织从 SignatureHeader.Creator → SerializedIdentity.Mspid 解出
	sh := &common.SignatureHeader{}
	if err := proto.Unmarshal(payload.Header.SignatureHeader, sh); err == nil && len(sh.Creator) > 0 {
		sid := &msp.SerializedIdentity{}
		if err := proto.Unmarshal(sh.Creator, sid); err == nil {
			tx.Msp = sid.Mspid
		}
	}

	// 链码名与函数名从 Transaction → ChaincodeActionPayload → ChaincodeInvocationSpec 解出
	t := &peer.Transaction{}
	if err := proto.Unmarshal(payload.Data, t); err != nil || len(t.Actions) == 0 {
		return tx
	}
	cap := &peer.ChaincodeActionPayload{}
	if err := proto.Unmarshal(t.Actions[0].Payload, cap); err != nil {
		return tx
	}
	cpp := &peer.ChaincodeProposalPayload{}
	if err := proto.Unmarshal(cap.ChaincodeProposalPayload, cpp); err != nil {
		return tx
	}
	cis := &peer.ChaincodeInvocationSpec{}
	if err := proto.Unmarshal(cpp.Input, cis); err != nil || cis.ChaincodeSpec == nil || cis.ChaincodeSpec.Input == nil {
		return tx
	}
	if cis.ChaincodeSpec.ChaincodeId != nil {
		tx.Chaincode = cis.ChaincodeSpec.ChaincodeId.Name
	}
	if len(cis.ChaincodeSpec.Input.Args) > 0 {
		tx.Fn = string(cis.ChaincodeSpec.Input.Args[0])
	}
	return tx
}

// blockHeaderHash 按 Fabric 官方算法计算区块头哈希：ASN.1 编码 Number/PreviousHash/DataHash 后 sha256。
// 直接 sha256(proto.Marshal(header)) 算出来对不上 currentBlockHash。
func blockHeaderHash(h *common.BlockHeader) []byte {
	type asn1Header struct {
		Number       *big.Int
		PreviousHash []byte
		DataHash     []byte
	}
	encoded, err := asn1.Marshal(asn1Header{new(big.Int).SetUint64(h.Number), h.PreviousHash, h.DataHash})
	if err != nil {
		return nil
	}
	sum := sha256.Sum256(encoded)
	return sum[:]
}
func (f *fabricStore) Close() {
	if f.shared {
		return // 视图共享主连接，由主实例负责关闭
	}
	f.peerMu.RLock()
	peers := make([]*orgPeer, 0, len(f.peers))
	for _, p := range f.peers {
		peers = append(peers, p)
	}
	f.peerMu.RUnlock()
	for _, p := range peers {
		if p.gateway != nil {
			p.gateway.Close()
		}
		if p.conn != nil {
			_ = p.conn.Close()
		}
	}
}

// Ledger 从链上拉取全量数据并组装成与演示模式一致的快照
func (f *fabricStore) Ledger() (*Ledger, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	l := &Ledger{Mode: "fabric", Channel: f.channel, Chaincode: f.chaincode}

	if err := f.query("GetAllEquipment", &l.Equipment); err != nil {
		return nil, err
	}
	if err := f.query("GetAllSpaces", &l.Spaces); err != nil {
		return nil, err
	}
	if err := f.query("GetAllProjects", &l.Projects); err != nil {
		return nil, err
	}
	if err := f.query("GetAllApplications", &l.Applications); err != nil {
		return nil, err
	}
	if err := f.query("GetAllBorrows", &l.Borrows); err != nil {
		return nil, err
	}
	if err := f.query("GetAllocations", &l.Allocations); err != nil {
		return nil, err
	}
	if err := f.query("GetActivityLog", &l.Activities); err != nil {
		// 活动日志失败不应阻断主流程
		l.Activities = []Activity{}
	}
	if err := f.query("GetStats", &l.Stats); err != nil {
		return nil, err
	}
	if l.Stats.ByCategory == nil {
		l.Stats.ByCategory = map[string]int{}
	}
	if l.Stats.ByProjectStatus == nil {
		l.Stats.ByProjectStatus = map[string]int{}
	}
	return l, nil
}

func (f *fabricStore) query(fn string, target interface{}) error {
	payload, err := f.contract.EvaluateTransaction(fn)
	if err != nil {
		return fmt.Errorf("调用 %s 失败: %s", fn, chaincodeReason(err))
	}
	if len(payload) == 0 {
		return nil
	}
	return json.Unmarshal(payload, target)
}

func (f *fabricStore) submit(fn string, args ...string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	contract, err := f.writeContract()
	if err != nil {
		return nil, err
	}
	payload, err := contract.SubmitTransaction(fn, args...)
	if err != nil {
		return nil, fmt.Errorf("提交 %s 失败: %s", fn, chaincodeReason(err))
	}
	return payload, nil
}

// chaincodeReason 把 gRPC 的封装错误还原成链码真正抛出的那句话。
// 默认的 Error() 只有 "failed to endorse transaction, see attached details"，
// 业务原因（例如"项目仍有 1 件设备未归还"）藏在 status details 里，必须解出来。
func chaincodeReason(err error) string {
	st, ok := status.FromError(err)
	if !ok {
		return err.Error()
	}
	for _, d := range st.Details() {
		if e, ok := d.(*gwproto.ErrorDetail); ok && e.Message != "" {
			return trimChaincodePrefix(e.Message)
		}
	}
	return trimChaincodePrefix(st.Message())
}

// trimChaincodePrefix 去掉 peer 附加的 "chaincode response 500, " 前缀，
// 只把链码自己写的那句话交给前端展示。
func trimChaincodePrefix(msg string) string {
	if i := strings.Index(msg, ", "); i > 0 && strings.HasPrefix(msg, "chaincode response ") {
		return msg[i+2:]
	}
	return msg
}

func (f *fabricStore) CreateApplication(in ApplicationInput) (*ProjectApplication, error) {
	id := fmt.Sprintf("APP-%d", time.Now().UnixMilli())
	membersJSON, _ := json.Marshal(in.Members)
	preferredJSON, _ := json.Marshal(in.PreferredSpace)
	_, err := f.submit("SubmitProjectApplication",
		id, in.ProjectName, in.LeadID, in.LeadName,
		string(membersJSON), fmt.Sprint(in.RequestedSeats), string(preferredJSON),
		in.StartDate, in.EndDate, in.IdeaSummary, hashOf(in.IdeaSummary), in.Milestone, hashOf(in.Milestone),
	)
	if err != nil {
		return nil, err
	}
	// 提交成功后从账本回读，返回真实落账数据
	l, err := f.Ledger()
	if err == nil {
		for i := range l.Applications {
			if l.Applications[i].ID == id {
				return &l.Applications[i], nil
			}
		}
	}
	return &ProjectApplication{ID: id, ProjectName: in.ProjectName, Status: "SUBMITTED"}, nil
}

func (f *fabricStore) ReviewApplication(id string, approved bool, note string) error {
	_, err := f.submit("ReviewProjectApplication", id, fmt.Sprint(approved), note)
	return err
}

func (f *fabricStore) ActivateProject(appID string, spaceIDs []string) (*Project, error) {
	projectID := fmt.Sprintf("PRJ-%d", time.Now().UnixMilli())
	spaceJSON, _ := json.Marshal(spaceIDs)
	if _, err := f.submit("ActivateProject", projectID, appID, string(spaceJSON)); err != nil {
		return nil, err
	}
	// 回读该项目（ReadProject 需要项目 ID 作为参数）
	if l, err := f.Ledger(); err == nil {
		for i := range l.Projects {
			if l.Projects[i].ID == projectID {
				return &l.Projects[i], nil
			}
		}
	}
	return &Project{ID: projectID, ApplicationID: appID, Status: "ACTIVE"}, nil
}

func (f *fabricStore) ExtendProject(projectID, newEndDate, note string) error {
	_, err := f.submit("ExtendProject", projectID, newEndDate, note)
	return err
}

func (f *fabricStore) CloseProject(projectID, note string) error {
	_, err := f.submit("CloseProject", projectID, note)
	return err
}

func (f *fabricStore) SubmitMilestone(projectID, milestone string) error {
	_, err := f.submit("SubmitMilestone", projectID, milestone, hashOf(milestone))
	return err
}

func (f *fabricStore) RequestBorrow(in BorrowInput) (*BorrowRecord, error) {
	id := fmt.Sprintf("BR-%d", time.Now().UnixMilli())
	if _, err := f.submit("RequestEquipmentBorrow",
		id, in.EquipmentID, in.ProjectID, in.BorrowerID, in.Borrower, in.Purpose, hashOf(in.Purpose), in.DueDate,
	); err != nil {
		return nil, err
	}
	if l, err := f.Ledger(); err == nil {
		for i := range l.Borrows {
			if l.Borrows[i].ID == id {
				return &l.Borrows[i], nil
			}
		}
	}
	return &BorrowRecord{ID: id, EquipmentID: in.EquipmentID, Status: "PENDING"}, nil
}

func (f *fabricStore) ReviewBorrow(id string, approved bool, note string) error {
	_, err := f.submit("ReviewEquipmentBorrow", id, fmt.Sprint(approved), note)
	return err
}

func (f *fabricStore) ConfirmPickup(id, condition string, photos []string) error {
	photosJSON, _ := json.Marshal(photos)
	_, err := f.submit("ConfirmEquipmentPickup", id, condition, string(photosJSON))
	return err
}

func (f *fabricStore) RequestReturn(id, note string) error {
	_, err := f.submit("RequestEquipmentReturn", id, note)
	return err
}

func (f *fabricStore) ConfirmReturn(id, condition, note string, photos []string) error {
	photosJSON, _ := json.Marshal(photos)
	_, err := f.submit("ConfirmEquipmentReturn", id, condition, hashOf(condition, note), note, string(photosJSON))
	return err
}

func (f *fabricStore) ResolveException(id, note string) error {
	_, err := f.submit("ResolveException", id, note)
	return err
}

func (f *fabricStore) History(objectType, id string) ([]HistoryEntry, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	payload, err := f.contract.EvaluateTransaction("GetHistory", objectType, id)
	if err != nil {
		return nil, fmt.Errorf("查询历史失败: %w", err)
	}
	entries := []HistoryEntry{}
	if len(payload) == 0 {
		return entries, nil
	}
	if err := json.Unmarshal(payload, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// ---------------- Gateway 连接辅助（复用 hw7 写法） ----------------

func newGrpcConnection(cfg FabricConfig) (*grpc.ClientConn, error) {
	certificate, err := loadCertificate(cfg.TLSCertPath)
	if err != nil {
		return nil, err
	}
	certPool := x509.NewCertPool()
	certPool.AddCert(certificate)
	transportCredentials := credentials.NewClientTLSFromCert(certPool, cfg.GatewayPeer)
	return grpc.Dial(cfg.PeerEndpoint, grpc.WithTransportCredentials(transportCredentials))
}

func newIdentity(cfg FabricConfig) *identity.X509Identity {
	certificate, err := loadCertificate(firstFile(cfg.CertDir))
	if err != nil {
		panic(fmt.Errorf("读取用户证书失败: %w", err))
	}
	id, err := identity.NewX509Identity(cfg.MSPID, certificate)
	if err != nil {
		panic(err)
	}
	return id
}

func newSign(cfg FabricConfig) identity.Sign {
	keyPEM, err := os.ReadFile(firstFile(cfg.KeyDir))
	if err != nil {
		panic(fmt.Errorf("读取私钥失败: %w", err))
	}
	privateKey, err := identity.PrivateKeyFromPEM(keyPEM)
	if err != nil {
		panic(err)
	}
	sign, err := identity.NewPrivateKeySign(privateKey)
	if err != nil {
		panic(err)
	}
	return sign
}

func loadCertificate(filename string) (*x509.Certificate, error) {
	pem, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("读取证书 %s 失败: %w", filename, err)
	}
	return identity.CertificateFromPEM(pem)
}

func firstFile(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		panic(fmt.Errorf("读取目录 %s 失败: %w", dir, err))
	}
	for _, e := range entries {
		if !e.IsDir() {
			return filepath.Join(dir, e.Name())
		}
	}
	panic(fmt.Errorf("目录 %s 中没有可用文件", dir))
}
