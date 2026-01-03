.PHONY: all build install uninstall clean test help release

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
	@echo "  make build          - Compila o binário"
	@echo "  make install        - Instala o binário em $(BIN_DIR)"
	@echo "  make uninstall      - Remove o binário de $(BIN_DIR)"
	@echo "  make clean          - Remove arquivos de build"
	@echo "  make test           - Executa todos os testes"
	@echo "  make test-verbose   - Executa testes com saída detalhada"
	@echo "  make test-coverage  - Executa testes com relatório de coverage"
	@echo "  make test-short     - Executa apenas testes curtos"
	@echo "  make lint           - Executa linters"
	@echo "  make release        - Cria uma nova release no GitHub"
	@echo "  make help           - Mostra esta mensagem"

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
	@rm -rf coverage/
	@echo "Limpo."

test:
	@echo "Executando testes..."
	$(GO) test -race -timeout 5m ./...
	@echo "✓ Testes concluídos"

test-verbose:
	@echo "Executando testes (verbose)..."
	$(GO) test -v -race -timeout 5m ./...

test-coverage:
	@echo "Executando testes com coverage..."
	@mkdir -p coverage
	$(GO) test -race -timeout 5m -coverprofile=coverage/coverage.out -covermode=atomic ./...
	@$(GO) tool cover -func=coverage/coverage.out | tail -1
	@echo ""
	@echo "Para ver relatório HTML:"
	@echo "  go tool cover -html=coverage/coverage.out"

test-short:
	@echo "Executando testes curtos (sem integração)..."
	$(GO) test -short -race -timeout 2m ./...
	@echo "✓ Testes curtos concluídos"

lint:
	@echo "Executando linters..."
	@which golangci-lint > /dev/null || (echo "golangci-lint não instalado. Instale em https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run ./...
	@echo "✓ Linting concluído"

# Development helpers
run: build
	@echo "Executando $(BUILD_DIR)/$(BINARY) analyze --repo-path ..."

release:
	@./scripts/release.sh
