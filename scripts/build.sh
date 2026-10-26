#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WEB_DIR="$ROOT_DIR/web"
SERVER_DIR="$ROOT_DIR/server"
SERVER_BIN="$SERVER_DIR/mini-wechat"

SKIP_WEB=0
SKIP_SERVER=0

usage() {
  cat <<'EOF'
用法: ./scripts/build.sh [选项]

选项:
  --skip-web      跳过前端构建
  --skip-server   跳过后端构建
  -h, --help      显示帮助
EOF
}

for arg in "$@"; do
  case "$arg" in
    --skip-web) SKIP_WEB=1 ;;
    --skip-server) SKIP_SERVER=1 ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "未知参数: $arg" >&2
      usage >&2
      exit 1
      ;;
  esac
done

if [ "$SKIP_WEB" -eq 0 ] && [ "$SKIP_SERVER" -eq 0 ]; then
  echo "==> 构建 Mini WeChat"
fi

if [ "$SKIP_WEB" -eq 0 ]; then
  command -v npm >/dev/null 2>&1 || {
    echo "未找到 npm，请先安装 Node.js" >&2
    exit 1
  }

  if [ ! -d "$WEB_DIR/node_modules" ]; then
    echo "==> 安装前端依赖"
    (cd "$WEB_DIR" && npm ci)
  fi

  echo "==> 构建前端"
  (cd "$WEB_DIR" && npm run build)
fi

if [ "$SKIP_SERVER" -eq 0 ]; then
  command -v go >/dev/null 2>&1 || {
    echo "未找到 go，请先安装 Go" >&2
    exit 1
  }

  echo "==> 构建后端"
  (cd "$SERVER_DIR" && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "$SERVER_BIN" .)
fi

echo
echo "构建完成"
if [ "$SKIP_WEB" -eq 0 ]; then
  echo "  前端产物: $WEB_DIR/dist"
fi
if [ "$SKIP_SERVER" -eq 0 ]; then
  echo "  后端二进制: $SERVER_BIN"
fi

if [ "$SKIP_WEB" -eq 0 ] && [ "$SKIP_SERVER" -eq 0 ]; then
  echo
  echo "运行: cd server && ./mini-wechat"
  echo "访问: http://localhost:8090"
fi
