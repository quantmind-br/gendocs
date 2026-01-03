.PHONY: build test test-all coverage lint deps install uninstall release clean help

BINARY_NAME := gendocs
BUILD_DIR   := build
BIN_DIR     := $(HOME)/.local/bin
GO          := go

UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Linux)
    BINARY := $(BINARY_NAME)-linux-amd64
else ifeq ($(UNAME_S),Darwin)
    BINARY := $(BINARY_NAME)-darwin-amd64
else
    BINARY := $(BINARY_NAME)
endif

help:
	@echo "make build      Compila o binário"
	@echo "make test       Testes unitários"
	@echo "make test-all   Todos os testes"
	@echo "make coverage   Relatório de cobertura"
	@echo "make lint       Linters + formatação"
	@echo "make deps       Dependências"
	@echo "make install    Instala em ~/.local/bin"
	@echo "make uninstall  Remove instalação"
	@echo "make release    Release interativo"
	@echo "make clean      Limpa artefatos"

build:
	@$(GO) build -o $(BUILD_DIR)/$(BINARY) .
	@echo "$(BUILD_DIR)/$(BINARY)"

test:
	@$(GO) test -short -race -timeout 2m ./...

test-all:
	@$(GO) test -race -timeout 5m ./...

coverage:
	@mkdir -p coverage
	@$(GO) test -race -timeout 5m -coverprofile=coverage/coverage.out -covermode=atomic ./...
	@$(GO) tool cover -func=coverage/coverage.out | tail -1
	@$(GO) tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@echo "coverage/coverage.html"

lint:
	@$(GO) fmt ./...
	@golangci-lint run ./...

deps:
	@$(GO) mod download
	@$(GO) mod verify
	@$(GO) mod tidy

install: build
	@mkdir -p $(BIN_DIR)
	@cp $(BUILD_DIR)/$(BINARY) $(BIN_DIR)/$(BINARY_NAME)
	@chmod +x $(BIN_DIR)/$(BINARY_NAME)
	@echo "$(BIN_DIR)/$(BINARY_NAME)"

uninstall:
	@rm -f $(BIN_DIR)/$(BINARY_NAME)

release:
	@./scripts/release.sh

clean:
	@rm -rf $(BUILD_DIR) coverage/
