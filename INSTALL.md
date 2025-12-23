# Guia de Instalação e Configuração

Este guia fornece instruções passo a passo para instalar e configurar o Gendocs Go.

## Pré-requisitos

- **Go 1.22 ou posterior**
- **API key de um provedor LLM** (OpenAI, Anthropic, ou Google Gemini)
- (Opcional) **GitLab** com token OAuth para funcionalidade de cronjob

## 1. Instalação

### Opção A: Usar Makefile (Recomendado)

```bash
# Compilar
make build

# Instalar (requer sudo)
make install

# Verificar instalação
gendocs --version
```

### Opção B: Scripts de Instalação

```bash
# Instalar (requer sudo)
sudo ./install.sh

# Desinstalar
sudo ./uninstall.sh
```

### Opção C: Compilar Manualmente

```bash
git clone https://github.com/divar-ir/ai-doc-gen.git
cd ai-doc-gen-feature-go-version/gendocs

# Compilar
go build -o gendocs .

# Opcional: mover para PATH global
sudo mv gendocs /usr/local/bin/
```

### Opção D: Binário pré-compilado (quando disponível)

```bash
# Baixar binário
wget https://github.com/divar-ir/ai-doc-gen/releases/latest/download/gendocs-linux-amd64
chmod +x gendocs-linux-amd64
sudo mv gendocs-linux-amd64 /usr/local/bin/gendocs
```

## 2. Configuração Rápida

### Método 1: Wizard Interativo (Recomendado)

```bash
# Inicia o wizard de configuração
./gendocs config
```

O wizard vai te guiar através de:
1. Seleção do provedor (OpenAI, Anthropic, ou Gemini)
2. Configuração da API key
3. Seleção do modelo
4. (Opcional) Base URL para APIs compatíveis com OpenAI

A configuração é salva em `~/.gendocs.yaml`.

### Método 2: Variáveis de Ambiente

```bash
# Configurar provedor e API key
export ANALYZER_LLM_PROVIDER="openai"
export ANALYZER_LLM_MODEL="gpt-4o"
export ANALYZER_LLM_API_KEY="sk-sua-chave-aqui"

# Para Anthropic Claude
# export ANALYZER_LLM_PROVIDER="anthropic"
# export ANALYZER_LLM_MODEL="claude-3-5-sonnet-20241022"
# export ANALYZER_LLM_API_KEY="sk-ant-sua-chave-aqui"

# Para Google Gemini
# export ANALYZER_LLM_PROVIDER="gemini"
# export ANALYZER_LLM_MODEL="gemini-1.5-pro"
# export ANALYZER_LLM_CONFIG_API_KEY="sua-chave-aqui"
```

Adicione ao seu `~/.bashrc` ou `~/.zshrc`:

```bash
echo 'export ANALYZER_LLM_PROVIDER="openai"' >> ~/.bashrc
echo 'export ANALYZER_LLM_MODEL="gpt-4o"' >> ~/.bashrc
echo 'export ANALYZER_LLM_API_KEY="sk-sua-chave"' >> ~/.bashrc
source ~/.bashrc
```

### Método 3: Arquivo de Configuração `.ai/config.yaml`

Crie um arquivo `.ai/config.yaml` no seu projeto:

```yaml
analyzer:
  llm:
    provider: openai
    model: gpt-4o
    api_key: ${ANALYZER_LLM_API_KEY}
    base_url: ""  # Opcional, para APIs compatíveis
    retries: 2
    timeout: 180
    max_tokens: 8192
    temperature: 0.0
  max_workers: 0  # 0 = auto-detectar CPUs
  exclude_code_structure: false
  exclude_data_flow: false
  exclude_dependencies: false
  exclude_request_flow: false
  exclude_api_analysis: false
```

## 3. Verificar Instalação

```bash
# Verificar versão
./gendocs --version

# Verificar ajuda
./gendocs --help

# Verificar configuração (se usou wizard)
cat ~/.gendocs.yaml
```

## 4. Primeiro Uso

### Analisar um Projeto

```bash
# Analisar o diretório atual
./gendocs analyze --repo-path .

# Analisar outro diretório
./gendocs analyze --repo-path /caminho/para/projeto

# Com flags de exclusão
./gendocs analyze --repo-path . --exclude-api-analysis --exclude-dependencies

# Com depuração
./gendocs analyze --repo-path . --debug
```

Isso vai gerar arquivos em `.ai/docs/`:
- `structure_analysis.md`
- `dependency_analysis.md`
- `data_flow_analysis.md`
- `request_flow_analysis.md`
- `api_analysis.md`

### Gerar Documentação

```bash
# Gerar README.md a partir das análises
./gendocs generate readme --repo-path .

# Gerar arquivos de configuração para IA
./gendocs generate ai-rules --repo-path .
```

Isso vai criar:
- `README.md` no diretório raiz
- `CLAUDE.md` (instruções para Claude)
- `AGENTS.md` (convenções de agentes)

### Processamento em Lote GitLab

```bash
# Configurar GitLab
export GITLAB_API_URL="https://gitlab.com"
export GITLAB_OAUTH_TOKEN="glpat-sua-token-aqui"
export GITLAB_USER_EMAIL="seu-email@example.com"

# Processar todos os projetos de um grupo
./gendocs cronjob analyze --group-project-id 123 --max-days-since-last-commit 14
```

Isso vai:
1. Buscar todos os projetos do grupo
2. Filtrar (pular arquivados, sem commits recentes)
3. Clonar cada projeto
4. Rodar análise
5. Criar branch `ai-analyzer-YYYY-MM-DD`
6. Fazer commit com resultados
7. Criar Merge Request

## 5. Exemplos de Configuração

### OpenAI com Modelo Customizado

```yaml
# ~/.gendocs.yaml
analyzer:
  llm:
    provider: openai
    model: gpt-4o-mini
    max_tokens: 4096
```

### APIs Compatíveis com OpenAI

```yaml
analyzer:
  llm:
    provider: openai
    model: llama-3.1-70b-instruct
    base_url: https://api.deepinfra.com/v1/openai
```

### Anthropic Claude

```yaml
analyzer:
  llm:
    provider: anthropic
    model: claude-3-5-sonnet-20241022
```

### Google Gemini via Vertex AI

```yaml
analyzer:
  llm:
    provider: gemini
    model: gemini-1.5-pro
gemini:
  use_vertex_ai: true
  project_id: seu-project-id
  location: us-central1
```

## 6. Troubleshooting

### Erro: "Required environment variable 'ANALYZER_LLM_API_KEY' is not set"

**Solução**: Configure a API key:
```bash
export ANALYZER_LLM_API_KEY="sk-sua-chave"
```

### Erro: "failed to load prompts"

**Solução**: Certifique-se de estar no diretório correto:
```bash
cd gendocs
./gendocs analyze --repo-path ../projeto-analisar
```

### Erro: "API error: status 401"

**Solução**: Verifique sua API key. Para testar:
```bash
curl https://api.openai.com/v1/models \
  -H "Authorization: Bearer $ANALYZER_LLM_API_KEY"
```

### Ver Logs de Depuração

```bash
# Ativar debug
./gendocs analyze --repo-path . --debug

# Logs são salvos em
cat .ai/logs/gendocs.log
```

## 7. Estrutura de Diretórios

```
projeto-analisado/
├── .ai/
│   ├── config.yaml          # Config do projeto (opcional)
│   ├── docs/
│   │   ├── structure_analysis.md
│   │   ├── dependency_analysis.md
│   │   ├── data_flow_analysis.md
│   │   ├── request_flow_analysis.md
│   │   └── api_analysis.md
│   └── logs/
│       └── gendocs.log        # Logs estruturados (JSON)
├── README.md                 # Gerado por `gendocs generate readme`
├── CLAUDE.md                 # Gerado por `gendocs generate ai-rules`
└── AGENTS.md                 # Gerado por `gendocs generate ai-rules`
```

## 8. Integração CI/CD

### GitHub Actions

```yaml
- name: Run gendocs analyze
  run: |
    go install github.com/divar-ir/ai-doc-gen/gendocs@latest
    gendocs analyze --repo-path .
```

### GitLab CI

```yaml
analyze:
  script:
    - go install github.com/divar-ir/ai-doc-gen/gendocs@latest
    - gendocs analyze --repo-path .
```

## 9. Atualização

```bash
cd ai-doc-gen-feature-go-version/gendocs
git pull
go build -o gendocs .
```

## 10. Suporte

- **Issues**: https://github.com/divar-ai-doc-gen/issues
- **Documentação**: Leia PLAN.md para detalhes da arquitetura
- **Python vs Go**: A versão Go mantém paridade de recursos com a Python
