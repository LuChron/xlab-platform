#!/usr/bin/env bash
# 开发模式启动：前后端分离，前端热更新
#
# 终端 1：./deploy/dev.sh backend   → Gin 服务 :8088
# 终端 2：./deploy/dev.sh frontend  → Vite  :5173（自动代理 /api 到 8088）
#
# 只跑演示模式（不需要 Fabric）：
#   XLAB_MODE=demo ./deploy/dev.sh backend

set -euo pipefail

ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
TARGET="${1:-backend}"

export PATH="/usr/local/go/bin:$PATH"

case "$TARGET" in
  backend|fabric)
    cd "$ROOT/backend"
    if [ "$TARGET" = "fabric" ]; then
      export XLAB_MODE=fabric
    else
      export XLAB_MODE="${XLAB_MODE:-auto}"
    fi
    export XLAB_PORT="${XLAB_PORT:-8088}"
    if [ "$XLAB_MODE" = "fabric" ]; then
      export XLAB_CRYPTO_PATH="${XLAB_CRYPTO_PATH:-$ROOT/../../hw7/fabric/scripts/fabric-samples/test-network/organizations/peerOrganizations/org1.example.com}"
      export XLAB_CHANNEL="${XLAB_CHANNEL:-xlabchannel}"
      export XLAB_CHAINCODE="${XLAB_CHAINCODE:-xlab}"
    fi
    echo "[XLAB] 后端启动，模式=$XLAB_MODE 端口=$XLAB_PORT"
    echo "[XLAB] 浏览器打开 http://127.0.0.1:$XLAB_PORT"
    go run .
    ;;
  frontend)
    cd "$ROOT/frontend"
    [ -d node_modules ] || npm ci
    echo "[XLAB] 前端 dev server: http://127.0.0.1:5173"
    npm run dev
    ;;
  build)
    cd "$ROOT/frontend"
    [ -d node_modules ] || npm ci
    npm run build
    echo "[XLAB] 构建产物在 frontend/dist，后端会自动托管"
    ;;
  *)
    echo "用法：$0 [backend|fabric|frontend|build]" >&2
    exit 1
    ;;
esac
