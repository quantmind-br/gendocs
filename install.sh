#!/bin/bash
# Script de instalação para Gendocs
# Uso: sudo ./install.sh

set -e

BINARY_NAME="gendocs"
BUILD_DIR="build"
BIN_DIR="/usr/local/bin"
CONFIG_DIR="$HOME/.gendocs.yaml"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "=== Gendocs Installation Script ==="
echo ""

# Detect OS
OS="$(uname -s)"
case "$OS" in
    Linux*)
        BINARY="gendocs-linux-amd64"
        ;;
    Darwin*)
        BINARY="gendocs-darwin-amd64"
        ;;
    *)
        BINARY="gendocs"
        ;;
esac

echo "Detectado: $OS"
echo ""

# Check Go installation
if ! command -v go &> /dev/null; then
    echo "Erro: Go não está instalado."
    echo ""
    echo "Instale Go 1.22+:"
    echo "  https://go.dev/dl/"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}')
echo "Go encontrado: $GO_VERSION"
echo ""

# Build
echo "Compilando..."
cd "$SCRIPT_DIR"
go build -o "$BUILD_DIR/$BINARY" .

# Install binary
echo "Instalando binário em $BIN_DIR..."
sudo mkdir -p "$BIN_DIR"
sudo cp "$BUILD_DIR/$BINARY" "$BIN_DIR/$BINARY_NAME"
sudo chmod +x "$BIN_DIR/$BINARY_NAME"

echo ""
echo "✅ Instalação completa!"
echo ""
echo "Binário instalado em: $BIN_DIR/$BINARY_NAME"
echo ""
echo "📖 Para configuração, execute:"
echo "  $BINARY_NAME config"
echo ""
echo "Ou configure manualmente:"
echo "  export ANALYZER_LLM_PROVIDER=\"openai\""
echo "  export ANALYZE_LLM_MODEL=\"gpt-4o\""
echo "  export ANALYZE_LLM_API_KEY=\"sk-...\""
echo ""
echo "🚀 Para analisar um projeto:"
echo "  $BINARY_NAME analyze --repo-path /caminho/para/projeto"
