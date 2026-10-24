#!/usr/bin/env bash
# 生产/演示运行：构建前端 → 编译后端 → 托管 dist 于单端口
#
#   ./deploy/run.sh           端口 18088，自动探测 Fabric（连不上退回演示数据）
#   ./deploy/run.sh fabric    强制连 Fabric
#   ./deploy/run.sh stop      停止
#
# 日志：/tmp/xlab-backend.log

set -euo pipefail

ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
PORT="${XLAB_PORT:-18088}"
BIN="/tmp/xlab-backend-new"
LOG="/tmp/xlab-backend.log"
export PATH="/usr/local/go/bin:$PATH"

stop() {
  pkill -f "$BIN" 2>/dev/null || true
  sleep 0.6
}

if [ "${1:-}" = "stop" ]; then
  stop
  echo "[XLAB] 已停止"
  exit 0
fi

stop

cd "$ROOT/frontend"
[ -d node_modules ] || npm ci
npm run build

cd "$ROOT/backend"
go build -buildvcs=false -o "$BIN" .

cd "$ROOT/backend"
export XLAB_PORT="$PORT"
export XLAB_MODE="${XLAB_MODE:-${1:-auto}}"
export XLAB_WEB_DIR="$ROOT/frontend/dist"
export XLAB_CHANNEL="${XLAB_CHANNEL:-xlabchannel}"
export XLAB_CHAINCODE="${XLAB_CHAINCODE:-xlab}"
export XLAB_CRYPTO_PATH="${XLAB_CRYPTO_PATH:-$ROOT/../../hw7/fabric/scripts/fabric-samples/test-network/organizations/peerOrganizations/org1.example.com}"

nohup "$BIN" >"$LOG" 2>&1 &
sleep 2.2

MODE=$(curl -s "http://127.0.0.1:$PORT/api/health" | sed -n 's/.*"mode":"\([a-z]*\)".*/\1/p')
echo "[XLAB] 已在 http://127.0.0.1:$PORT 启动（模式: ${MODE:-未知}，日志: $LOG）"
