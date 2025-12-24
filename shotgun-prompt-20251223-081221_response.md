```markdown
### 1) Contexto Inferido do Projeto
O Gendocs é uma ferramenta CLI escrita em Go que automatiza a documentação de bases de código utilizando modelos de linguagem de grande escala (LLMs). O sistema utiliza uma arquitetura de múltiplos agentes especializados (estrutura, dependências, fluxo de dados, etc.) para analisar repositórios locais ou remotos (GitLab) e gerar arquivos como README.md e configurações para assistentes de IA (CLAUDE.md). O projeto está em estágio avançado de maturidade funcional, focando agora em expansão de recursos e robustez.

---

### 2) Funcionalidades Propostas (Priorizadas)

- **Nome:** Suporte a Repositórios GitHub no Cronjob
- **Problema/Oportunidade:** Atualmente, a automação em lote (cronjob) é restrita ao GitLab, limitando o uso para organizações que utilizam o GitHub.
- **Descrição da Funcionalidade:** Implementar um novo cliente e handler para a API do GitHub, permitindo clonagem, criação de branches e Pull Requests de forma automatizada.
- **Valor Esperado (usuário/negócio):** Expansão massiva da base de usuários potenciais e suporte a projetos Open Source.
- **Complexidade:** M
- **Justificativa da Complexidade:** Requer implementação de uma nova interface de cliente baseada na API REST/GraphQL do GitHub, similar à do GitLab.
- **Riscos e Dependências:** Dependência das bibliotecas de cliente do GitHub ou implementação manual via HTTP; limites de taxa (rate limiting) da API.
- **Evidência (arquivos/pastas):** `internal/gitlab/client.go`, `cmd/cronjob.go` e `internal/handlers/cronjob.go`.
- **Impacto Técnico (prováveis módulos/arquivos):** Criar `internal/github/client.go`, abstrair uma interface `GitProvider` em `internal/git/interface.go` e atualizar `handlers/cronjob.go`.
- **Plano de Implementação (passos):** 1. Definir interface comum para Git Providers. 2. Implementar cliente GitHub. 3. Adicionar flags de configuração para GitHub em `cmd/cronjob.go`. 4. Refatorar handler para aceitar qualquer provider.
- **Critérios de Aceite:** O comando `cronjob` deve ser capaz de abrir um Pull Request no GitHub após a análise.
- **Testes Recomendados:** Testes de integração com a API do GitHub (usando mocks ou repositórios de teste).

- **Nome:** Sistema de Plugins para Ferramentas de Análise (Tools)
- **Problema/Oportunidade:** Os agentes possuem apenas ferramentas de leitura de arquivo e listagem, o que limita a análise de metadados complexos (ex: AST, métricas de complexidade).
- **Descrição da Funcionalidade:** Criar um framework para que novas ferramentas (tools) possam ser registradas dinamicamente no `BaseAgent`, permitindo que o LLM execute comandos como `grep`, `find-references` ou `get-git-blame`.
- **Valor Esperado (usuário/negócio):** Análises muito mais profundas e precisas sobre a evolução e qualidade do código.
- **Complexidade:** M
- **Justificativa da Complexidade:** Requer a criação de uma estrutura de registro de ferramentas mais flexível no factory de agentes.
- **Riscos e Dependências:** Risco de segurança ao permitir execução de comandos; necessidade de sanitização de inputs do LLM.
- **Evidência (arquivos/pastas):** `internal/tools/base.go`, `internal/agents/base.go`.
- **Impacto Técnico (prováveis módulos/arquivos):** `internal/tools/`, `internal/agents/sub_agents.go`.
- **Plano de Implementação (passos):** 1. Criar interface de Tool extensível. 2. Implementar ferramentas de shell seguras. 3. Atualizar o loop de execução em `base.go` para suportar múltiplas ferramentas dinâmicas.
- **Critérios de Aceite:** Um novo agente deve conseguir usar uma ferramenta de busca (grep) sem alteração no código do agente base.
- **Testes Recomendados:** Testes unitários para cada nova ferramenta; testes de integração simulando chamadas de ferramentas pelo LLM.

- **Nome:** Cache de Análise de Arquivos (Incremental Analysis)
- **Problema/Oportunidade:** Analisar repositórios grandes repetidamente consome muitos tokens e tempo, mesmo quando poucos arquivos mudaram.
- **Descrição da Funcionalidade:** Implementar um sistema de cache baseado em hashes (SHA-256) dos arquivos analisados. O agente só reanalisará arquivos cujos hashes mudaram desde a última execução.
- **Valor Esperado (usuário/negócio):** Redução drástica de custos com API e tempo de execução em processos de CI/CD.
- **Complexidade:** G
- **Justificativa da Complexidade:** Requer persistência de estado da análise anterior e lógica de "merge" de resultados parciais em documentos Markdown finais.
- **Riscos e Dependências:** Risco de a documentação ficar inconsistente se a mudança em um arquivo afetar o contexto global não reanalisado.
- **Evidência (arquivos/pastas):** `internal/agents/analyzer.go`, `internal/tools/file_read.go`.
- **Impacto Técnico (prováveis módulos/arquivos):** Novo módulo `internal/cache/`, alterações em `AnalyzerAgent` e ferramentas de leitura.
- **Plano de Implementação (passos):** 1. Criar gerenciador de cache local em `.ai/cache.json`. 2. Implementar lógica de verificação de hash no `FileReadTool`. 3. Ajustar prompts para lidar com "contexto delta".
- **Critérios de Aceite:** Uma segunda execução sem mudanças no código deve durar menos de 10% do tempo original e não consumir tokens de processamento de conteúdo.
- **Testes Recomendados:** Testes de performance comparando execuções; testes de integridade do cache.

- **Nome:** Exportação para Formatos Adicionais (HTML/PDF/Wiki)
- **Problema/Oportunidade:** A documentação gerada é estritamente Markdown, o que pode não ser ideal para apresentações executivas ou portais de desenvolvedor.
- **Descrição da Funcionalidade:** Adicionar um comando `gendocs generate export` que converte os arquivos de `.ai/docs/` em um site estático (HTML) ou documento PDF formatado.
- **Valor Esperado (usuário/negócio):** Facilidade de compartilhamento da documentação com stakeholders não técnicos.
- **Complexidade:** P
- **Justificativa da Complexidade:** Existem bibliotecas Go prontas para conversão de Markdown para HTML/PDF.
- **Riscos e Dependências:** Dependência de bibliotecas externas como `goldmark` ou geradores de PDF.
- **Evidência (arquivos/pastas):** `cmd/generate.go`, `internal/handlers/readme.go`.
- **Impacto Técnico (prováveis módulos/arquivos):** Novo handler `internal/handlers/export.go`.
- **Plano de Implementação (passos):** 1. Integrar biblioteca de renderização Markdown. 2. Criar templates CSS básicos. 3. Implementar comando de exportação no Cobra.
- **Critérios de Aceite:** O comando deve gerar um arquivo `.html` ou `.pdf` visualmente agradável a partir do `README.md`.
- **Testes Recomendados:** Testes de geração de arquivo e validação de formato.

- **Nome:** Customização de Prompts via Projeto (Custom Rules)
- **Problema/Oportunidade:** Atualmente os prompts estão embutidos ou em uma pasta fixa, dificultando a adaptação para convenções específicas de uma empresa.
- **Descrição da Funcionalidade:** Permitir que o usuário coloque arquivos `.yaml` em `.ai/prompts/` dentro do repositório alvo para sobrescrever ou estender os prompts padrão do sistema.
- **Valor Esperado (usuário/negócio):** Flexibilidade total para adaptar o tom e os pontos focais da documentação.
- **Complexidade:** M
- **Justificativa da Complexidade:** Requer alteração na lógica de busca do `prompts/manager.go` para suportar precedência de caminhos.
- **Riscos e Dependências:** [SUPOSIÇÃO] Prompts mal escritos pelo usuário podem quebrar o formato de saída esperado pelos agentes.
- **Evidência (arquivos/pastas):** `internal/prompts/manager.go`, `internal/config/loader.go`.
- **Impacto Técnico (prováveis módulos/arquivos):** `internal/prompts/manager.go`.
- **Plano de Implementação (passos):** 1. Atualizar `NewManager` para aceitar múltiplos diretórios. 2. Implementar lógica de merge/override de chaves YAML. 3. Documentar a estrutura de customização no `INSTALL.md`.
- **Critérios de Aceite:** Um arquivo YAML no repositório analisado deve ser capaz de mudar o `system_prompt` de um agente sem alterar o binário do Gendocs.
- **Testes Recomendados:** Testes unitários de prioridade de carregamento de prompts.

- **Nome:** Agente de Análise de Segurança e Vulnerabilidades
- **Problema/Oportunidade:** A análise foca em estrutura e fluxo, mas ignora padrões óbvios de insegurança ou exposição de segredos.
- **Descrição da Funcionalidade:** Criar um novo sub-agente `SecurityAnalyzer` focado em detectar hardcoded secrets, falta de sanitização e vulnerabilidades conhecidas em dependências.
- **Valor Esperado (usuário/negócio):** Adição de valor crítico de segurança preventivo diretamente na documentação do projeto.
- **Complexidade:** M
- **Justificativa da Complexidade:** Requer criação de prompts específicos e possivelmente integração com ferramentas de varredura existentes.
- **Riscos e Dependências:** Risco de falsos positivos; dependência da capacidade do LLM de identificar padrões de segurança.
- **Evidência (arquivos/pastas):** `internal/agents/factory.go`, `prompts/analyzer.yaml`.
- **Impacto Técnico (prováveis módulos/arquivos):** `internal/agents/analyzer.go`, `internal/agents/factory.go`, novo arquivo de prompt.
- **Plano de Implementação (passos):** 1. Criar prompt de sistema para Segurança. 2. Registrar sub-agente no `AnalyzerAgent`. 3. Adicionar flag `--exclude-security-analysis`.
- **Critérios de Aceite:** Geração do arquivo `security_analysis.md` com achados de segurança classificados por severidade.
- **Testes Recomendados:** Testes em repositórios com vulnerabilidades conhecidas (ex: OWASP Juice Shop).

---

### 3) Vitórias Rápidas (Quick Wins)

1.  **Suporte a Variáveis de Ambiente no TUI Config:** Permitir que o wizard de configuração (`cmd/config.go`) detecte e sugira chaves de API já presentes no ambiente do usuário, evitando redigitação.
2.  **Validação de Sintaxe YAML Pós-Geração:** Adicionar um passo de validação após a geração de arquivos de regras da IA para garantir que o LLM não gerou Markdown inválido que quebre o uso em IDEs como Cursor. (Evidência: `internal/agents/analyzer.go`).
3.  **Melhoria na Detecção Automática de Workers:** Ajustar o `worker_pool/pool.go` para limitar workers baseado não apenas em CPUs, mas também em limites de taxa conhecidos dos provedores (ex: Anthropic tem limites mais rígidos que OpenAI). (Evidência: `internal/config/loader.go`).
```