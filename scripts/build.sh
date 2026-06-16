#!/usr/bin/env bash
# build.sh - 跨平台构建脚本
#
# 用途：一次性构建 Linux / Windows / macOS 的 release 二进制
# 用法：
#   ./scripts/build.sh                     # 用 VERSION 文件中的版本
#   VERSION=v1.0.1 ./scripts/build.sh      # 显式指定版本
#   BUILD_DIR=dist ./scripts/build.sh      # 自定义输出目录
#
# 依赖：Go 1.26.4+，bash
# 产物：build/ 或 $BUILD_DIR 下三个二进制
# 注意：Windows 上请用 Git Bash / WSL / MSYS2 运行

set -euo pipefail

# ---------- 参数 ----------
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"
cd "$PROJECT_ROOT"

VERSION="${VERSION:-$(cat VERSION)}"
BUILD_TIME="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
BUILD_DIR="${BUILD_DIR:-build}"
LDFLAGS="-s -w -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME}"

# ---------- 工具检查 ----------
if ! command -v go >/dev/null 2>&1; then
    echo "error: go not found in PATH" >&2
    exit 1
fi

# ---------- 信息 ----------
echo "==> Project root: $PROJECT_ROOT"
echo "==> Version:      $VERSION"
echo "==> Build time:   $BUILD_TIME"
echo "==> Output dir:   $BUILD_DIR"
echo

# ---------- 准备 ----------
mkdir -p "$BUILD_DIR"
rm -f "$BUILD_DIR"/report-*

# ---------- 平台列表 ----------
PLATFORMS=(
    "linux  amd64"
    "windows amd64"
    "darwin  amd64"
)

# ---------- 构建 ----------
for PLATFORM in "${PLATFORMS[@]}"; do
    GOOS="${PLATFORM% *}"
    GOARCH="${PLATFORM#* }"
    EXT=""
    [ "$GOOS" = "windows" ] && EXT=".exe"

    OUTPUT="${BUILD_DIR}/report-${GOOS}-${GOARCH}${EXT}"
    echo "==> Building $GOOS/$GOARCH -> $OUTPUT"

    GOOS="$GOOS" GOARCH="$GOARCH" CGO_ENABLED=0 \
        go build -trimpath -ldflags "$LDFLAGS" \
        -o "$OUTPUT" \
        ./cmd/report
done

# ---------- 产物摘要 ----------
echo
echo "==> Build complete!"
ls -lh "$BUILD_DIR"/report-*
