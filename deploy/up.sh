#!/usr/bin/env bash
# X-Lab 平台一键部署到 Fabric
#
# 用法：
#   ./deploy/up.sh              # 启动网络 + 通道 + 链码 + 初始化账本
#   ./deploy/up.sh current      # 查看当前链码定义与包 ID
#   ./deploy/up.sh down         # 关闭网络并清理链码容器
#
# 为什么用 CCAAS（外部链码服务）而不是 peer 直接 docker build：
#   Docker Engine >= 25（本环境 29.x）要求 API >= 1.44，而 Fabric peer 内置的
#   Docker 客户端版本较旧，peer 直接构建会报 "write unix @->/run/docker.sock: broken pipe"。
#   CCAAS 由我们先构建镜像、peer 只负责连接，可完全避开该问题，也更适合离线演示。

set -euo pipefail

ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
FABRIC_SAMPLES="${XLAB_FABRIC_SAMPLES:-$ROOT/../../hw7/fabric/scripts/fabric-samples}"
TEST_NETWORK="$FABRIC_SAMPLES/test-network"
CHANNEL="${XLAB_CHANNEL:-xlabchannel}"
CC_NAME="${XLAB_CHAINCODE:-xlab}"
CC_LABEL="xlab_1.0"
CC_VERSION="${XLAB_CC_VERSION:-1.0}"
CC_SEQUENCE="${XLAB_CC_SEQUENCE:-1}"
CC_IMAGE="xlab-cc:${CC_VERSION}"
CC_CONTAINER="xlab-cc.org1.example.com"

export PATH="/usr/local/go/bin:$PATH"
export FABRIC_CFG_PATH="$TEST_NETWORK/../config"
export CONTAINER_CLI_COMPOSE="${CONTAINER_CLI_COMPOSE:-docker compose}"

info() { printf '\033[36m[XLAB]\033[0m %s\n' "$*"; }
ok()   { printf '\033[32m[XLAB]\033[0m %s\n' "$*"; }
fail() { printf '\033[31m[XLAB]\033[0m %s\n' "$*" >&2; exit 1; }

ACTION="${1:-up}"

org_env() {
  local org="$1"
  export CORE_PEER_TLS_ENABLED=true
  export PATH="$TEST_NETWORK/../bin:$PATH"
  if [ "$org" = "1" ]; then
    export CORE_PEER_LOCALMSPID="Org1MSP"
    export CORE_PEER_ADDRESS=localhost:7051
    export CORE_PEER_TLS_ROOTCERT_FILE="$TEST_NETWORK/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt"
    export CORE_PEER_MSPCONFIGPATH="$TEST_NETWORK/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp"
  else
    export CORE_PEER_LOCALMSPID="Org2MSP"
    export CORE_PEER_ADDRESS=localhost:9051
    export CORE_PEER_TLS_ROOTCERT_FILE="$TEST_NETWORK/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt"
    export CORE_PEER_MSPCONFIGPATH="$TEST_NETWORK/organizations/peerOrganizations/org2.example.com/users/Admin@org2.example.com/msp"
  fi
}

ORDERER_CA="$TEST_NETWORK/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem"
ORG1_TLS="$TEST_NETWORK/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt"
ORG2_TLS="$TEST_NETWORK/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt"

[ -d "$TEST_NETWORK" ] || fail "找不到 test-network：$TEST_NETWORK
请设置 XLAB_FABRIC_SAMPLES 指向 fabric-samples 目录"

case "$ACTION" in
  down)
    docker rm -f "$CC_CONTAINER" >/dev/null 2>&1 || true
    cd "$TEST_NETWORK" && ./network.sh down
    ok "网络已关闭"
    exit 0
    ;;
  current)
    org_env 1
    peer lifecycle chaincode querycommitted -C "$CHANNEL" -n "$CC_NAME" 2>&1 | head -5
    echo
    peer lifecycle chaincode queryinstalled 2>&1 | grep "Package ID" | tail -3
    echo
    docker ps -a --format 'table {{.Names}}\t{{.Status}}' | grep -E "xlab|peer0|orderer" || true
    exit 0
    ;;
  up) ;;
  *) fail "未知参数：$ACTION（可用：up | down | current）" ;;
esac

# ============================ 部署流程 ============================

info "1/7 构建链码镜像 $CC_IMAGE"
cd "$ROOT/chaincode"
[ -d vendor ] || { info "  生成 vendor 目录"; go mod vendor; }
docker build -f "$ROOT/deploy/docker/Dockerfile.ccaas" -t "$CC_IMAGE" . >/dev/null
ok "  镜像构建完成"

info "2/7 启动 Fabric 网络"
cd "$TEST_NETWORK"
./network.sh down >/dev/null 2>&1 || true
./network.sh up

info "3/7 创建通道 $CHANNEL"
./network.sh createChannel -c "$CHANNEL" || info "  通道可能已存在，继续"

info "4/7 打包 CCAAS 链码"
PKG_DIR="$ROOT/deploy/ccaas"
mkdir -p "$PKG_DIR"
cat > "$PKG_DIR/connection.json" <<JSON
{
  "address": "${CC_CONTAINER}:9999",
  "dial_timeout": "30s",
  "tls_required": false
}
JSON
cat > "$PKG_DIR/metadata.json" <<JSON
{
  "type": "ccaas",
  "label": "${CC_LABEL}"
}
JSON
( cd "$PKG_DIR" && tar -czf code.tar.gz connection.json && tar -czf "$ROOT/chaincode/${CC_NAME}-ccaas.tar.gz" metadata.json code.tar.gz )
PKG_FILE="$ROOT/chaincode/${CC_NAME}-ccaas.tar.gz"

info "5/7 安装并批准链码"
org_env 2
peer lifecycle chaincode install "$PKG_FILE" >/dev/null 2>&1 || true
org_env 1
peer lifecycle chaincode install "$PKG_FILE" >/dev/null 2>&1 || true

PACKAGE_ID=$(peer lifecycle chaincode queryinstalled --output json \
  | python3 -c "import json,sys;d=json.load(sys.stdin);ids=[c['package_id'] for c in d.get('installed_chaincodes',[]) if c['label']=='${CC_LABEL}'];print(ids[-1] if ids else '')")
[ -n "$PACKAGE_ID" ] || fail "未能获取链码包 ID"
ok "  包 ID：$PACKAGE_ID"

EXISTING_SEQ=$(peer lifecycle chaincode querycommitted -C "$CHANNEL" -n "$CC_NAME" 2>/dev/null \
  | grep -oE "Sequence: [0-9]+" | grep -oE "[0-9]+" || true)
if [ -n "$EXISTING_SEQ" ]; then
  CC_SEQUENCE=$((EXISTING_SEQ + 1))
  info "  已有定义 sequence=$EXISTING_SEQ，本次升级为 $CC_SEQUENCE"
fi

org_env 2
peer lifecycle chaincode approveformyorg -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com \
  --channelID "$CHANNEL" --name "$CC_NAME" --version "$CC_VERSION" \
  --package-id "$PACKAGE_ID" --sequence "$CC_SEQUENCE" --tls --cafile "$ORDERER_CA" >/dev/null
org_env 1
peer lifecycle chaincode approveformyorg -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com \
  --channelID "$CHANNEL" --name "$CC_NAME" --version "$CC_VERSION" \
  --package-id "$PACKAGE_ID" --sequence "$CC_SEQUENCE" --tls --cafile "$ORDERER_CA" >/dev/null
ok "  两个组织均已批准"

peer lifecycle chaincode commit -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com \
  --channelID "$CHANNEL" --name "$CC_NAME" --version "$CC_VERSION" --sequence "$CC_SEQUENCE" \
  --tls --cafile "$ORDERER_CA" \
  --peerAddresses localhost:7051 --tlsRootCertFiles "$ORG1_TLS" \
  --peerAddresses localhost:9051 --tlsRootCertFiles "$ORG2_TLS" >/dev/null
ok "  链码定义已提交"

info "6/7 启动链码服务容器"
docker rm -f "$CC_CONTAINER" >/dev/null 2>&1 || true
docker run -d --name "$CC_CONTAINER" --network fabric_test \
  -e CORE_CHAINCODE_ID_NAME="$PACKAGE_ID" \
  -e CHAINCODE_SERVER_ADDRESS=0.0.0.0:9999 \
  "$CC_IMAGE" >/dev/null
sleep 3
docker ps --format '{{.Names}}' | grep -q "^${CC_CONTAINER}$" || fail "链码容器未启动，请查看：docker logs $CC_CONTAINER"
ok "  链码容器运行中"

info "7/7 初始化账本"
if peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com \
  --tls --cafile "$ORDERER_CA" -C "$CHANNEL" -n "$CC_NAME" \
  --peerAddresses localhost:7051 --tlsRootCertFiles "$ORG1_TLS" \
  --peerAddresses localhost:9051 --tlsRootCertFiles "$ORG2_TLS" \
  -c '{"function":"InitLedger","Args":[]}' --waitForEvent >/dev/null 2>&1; then
  ok "  InitLedger 完成"
else
  info "  InitLedger 可能已执行过，跳过"
fi

echo
ok "部署完成"
cat <<EOF

  通道      : $CHANNEL
  链码      : $CC_NAME (v$CC_VERSION, sequence $CC_SEQUENCE)
  包 ID     : $PACKAGE_ID
  Peer      : peer0.org1.example.com:7051 / peer0.org2.example.com:9051
  Orderer   : localhost:7050

启动后端（Fabric 模式）：

  ./deploy/dev.sh fabric

或手动：

  cd backend && XLAB_MODE=fabric \\
  XLAB_CRYPTO_PATH=$TEST_NETWORK/organizations/peerOrganizations/org1.example.com \\
  XLAB_CHANNEL=$CHANNEL XLAB_CHAINCODE=$CC_NAME go run .

（Fabric 不可用时后端会自动回退到演示模式，页面功能不受影响）

EOF
