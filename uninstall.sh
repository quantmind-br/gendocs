#!/bin/bash
# Script de desinstalação para Gendocs
# Uso: sudo ./uninstall.sh

set -e

BINARY_NAME="gendocs"
BIN_DIR="/usr/local/bin"
CONFIG_DIR="$HOME/.gendocs.yaml"

echo "=== Gendocs Uninstallation Script ==="
echo ""

# Remove binary
echo "Removendo binário de $BIN_DIR..."
if [ -f "$BIN_DIR/$BINARY_NAME" ]; then
    sudo rm -f "$BIN_DIR/$BINARY_NAME"
    echo "✅ Binário removido"
else
    echo "⚠️  Binário não encontrado em $BIN_DIR/$BINARY_NAME"
fi

# Ask about config
echo ""
read -p "Remover configuração em $CONFIG_DIR? (y/N) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    rm -f "$CONFIG_DIR"
    echo "✅ Configuração removida"
fi

echo ""
echo "Desinstalação completa!"
echo ""
echo "Para reinstalar:"
echo "  make install"
echo "  ou"
echo "  ./install.sh"
