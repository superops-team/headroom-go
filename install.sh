#!/bin/bash
set -e

# headroom-go 一键安装脚本
# Usage: curl -sSL https://raw.githubusercontent.com/superops-team/headroom-go/main/install.sh | bash
# Or with specific version: curl -sSL https://raw.githubusercontent.com/superops-team/headroom-go/main/install.sh | bash -s -- v0.1.1

RELEASE_URL="https://github.com/superops-team/headroom-go/releases/download"
INSTALL_DIR="/usr/local/bin"
VERSION="${1:-latest}"

if [ "$VERSION" = "latest" ]; then
    # 获取最新版本
    echo "Detecting latest version..."
    VERSION=$(curl -sSL https://api.github.com/repos/superops-team/headroom-go/releases/latest | grep -o '"tag_name": "[^"]*"' | cut -d'"' -f4)
    if [ -z "$VERSION" ]; then
        echo "Failed to detect latest version, using v0.1.1 as fallback"
        VERSION="v0.1.1"
    fi
fi

echo "Installing headroom-go $VERSION..."

# 检测操作系统和架构
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $ARCH in
    x86_64)
        ARCH="amd64"
        ;;
    arm64|aarch64)
        ARCH="arm64"
        ;;
    *)
        echo "Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

# 确定文件名
case $OS in
    linux)
        FILENAME="headroom-linux-${ARCH}"
        ;;
    darwin)
        FILENAME="headroom-darwin-${ARCH}"
        ;;
    *)
        echo "Unsupported OS: $OS"
        exit 1
        ;;
esac

echo "Downloading $FILENAME..."

# 下载二进制文件
DOWNLOAD_URL="${RELEASE_URL}/${VERSION}/${FILENAME}"
echo "Downloading from: $DOWNLOAD_URL"

if command -v curl &>/dev/null; then
    curl -sSL -o /tmp/headroom "$DOWNLOAD_URL"
elif command -v wget &>/dev/null; then
    wget -q -O /tmp/headroom "$DOWNLOAD_URL"
else
    echo "Error: neither curl nor wget is available"
    exit 1
fi

# 检查下载是否成功
if [ ! -f /tmp/headroom ]; then
    echo "Failed to download headroom"
    exit 1
fi

# 赋予执行权限
chmod +x /tmp/headroom

# 安装到目标目录
if [ -w "$INSTALL_DIR" ]; then
    mv /tmp/headroom "$INSTALL_DIR/headroom"
else
    echo "Need sudo to install to $INSTALL_DIR"
    sudo mv /tmp/headroom "$INSTALL_DIR/headroom"
fi

# 验证安装
if command -v headroom &>/dev/null; then
    echo "headroom-go $VERSION installed successfully!"
    echo ""
    headroom version
    echo ""
    echo "Usage:"
    echo "  headroom compress --input=input.txt --output=output.txt"
    echo "  headroom proxy --port=8787"
else
    echo "Installation completed, but headroom command not found in PATH"
    echo "Please add $INSTALL_DIR to your PATH:"
    echo "  export PATH=\$PATH:$INSTALL_DIR"
fi