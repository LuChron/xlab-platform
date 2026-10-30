#!/usr/bin/env bash
# 提交或答辩前的本地验收：格式、后端、链码、前端与部署脚本。

set -euo pipefail

ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
export GOCACHE="$ROOT/.cache/go-build"
mkdir -p "$GOCACHE"

# 非交互 shell 可能没有加载 ~/.zshrc 里的 PATH，补一次常见工具安装位置。
if ! command -v go >/dev/null 2>&1 && [ -x /usr/local/go/bin/go ]; then
  export PATH="/usr/local/go/bin:$PATH"
fi
if ! command -v npm >/dev/null 2>&1; then
  for nodedir in "$HOME"/.nvm/versions/node/*/bin /usr/local/bin; do
    if [ -x "$nodedir/npm" ]; then
      export PATH="$nodedir:$PATH"
      break
    fi
  done
fi

echo "[XLAB] 1/5 检查 Git 补丁格式"
git -C "$ROOT" diff --check

echo "[XLAB] 2/5 检查后端"
(cd "$ROOT/backend" && go test ./...)

echo "[XLAB] 3/5 检查链码"
(cd "$ROOT/chaincode" && go test ./...)

echo "[XLAB] 4/5 构建前端"
cd "$ROOT/frontend"
if [ ! -d node_modules ]; then
  npm ci
fi
npm run build

echo "[XLAB] 5/5 检查部署脚本语法"
bash -n "$ROOT/deploy/dev.sh" "$ROOT/deploy/run.sh" "$ROOT/deploy/up.sh" "$ROOT/deploy/check.sh"

echo "[XLAB] 全部检查通过"
