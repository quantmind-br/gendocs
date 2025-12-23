.PHONY: all build install uninstall clean test help

# Variables
BINARY_NAME=gendocs
BUILD_DIR=build
# Instalação local em ~/.local/bin
BIN_DIR=$(HOME)/.local/bin
CONFIG_DIR=$(HOME)/.gendocs.yaml
PROMPTS_DIR=./prompts
GO=go
GOFLAGS=

# Detect OS
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Linux)
    BINARY=$(BINARY_NAME)-linux-amd64
else ifeq ($(UNAME_S),Darwin)
    BINARY=$(BINARY_NAME)-darwin-amd64
else
    BINARY=$(BINARY_NAME)
endif

all: build

help:
	@echo "Gendocs Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  make build        - Compila o binário"
	@echo "  make install      - Instala o binário em $(BIN_DIR)"
	@echo "  make uninstall    - Remove o binário de $(BIN_DIR)"
	@echo "  make clean        - Remove arquivos de build"
	@echo "  make test         - Executa testes"
	@echo "  make help         - Mostra esta mensagem"

build:
	@echo "Compilando $(BINARY)..."
	$(GO) $(GOFLAGS) build -o $(BUILD_DIR)/$(BINARY) .
	@echo "Binário criado: $(BUILD_DIR)/$(BINARY)"

install: build
	@echo "Instalando $(BINARY) em $(BIN_DIR)..."
	@mkdir -p $(BIN_DIR)
	@cp $(BUILD_DIR)/$(BINARY) $(BIN_DIR)/$(BINARY_NAME)
	@chmod +x $(BIN_DIR)/$(BINARY_NAME)
	@echo "Instalado em: $(BIN_DIR)/$(BINARY_NAME)"
	@echo ""
	@echo "Para configurar, execute:"
	@echo "  $(BINARY_NAME) config"
	@echo ""
	@echo "Ou configure manualmente:"
	@echo "  export ANALYZER_LLM_PROVIDER=\"openai\""
	@echo "  export ANALYZER_LLM_MODEL=\"gpt-4o\""
	@echo "  export ANALYZER_LLM_API_KEY=\"sk-...\""

uninstall:
	@echo "Removendo $(BINARY_NAME) de $(BIN_DIR)..."
	@rm -f $(BIN_DIR)/$(BINARY_NAME)
	@echo "Removido."
	@echo ""
	@echo "Para remover completamente (incluindo configuração):"
	@echo "  rm -f $(CONFIG_DIR)"
	@echo "  rm -rf ~/.gendocs/prompts_backup"

clean:
	@echo "Limpando arquivos de build..."
	@rm -rf $(BUILD_DIR)
	@echo "Limpo."

test:
	@echo "Executando testes..."
	$(GO) test -v ./...

# Development helpers
run: build
	@echo "Executando $(BUILD_DIR)/$(BINARY) analyze --repo-path ../.."
