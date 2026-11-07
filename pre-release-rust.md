# ANÁLISE ARQUITETÔNICA, PLANO DE REESCRITA EM RUST E DIRETRIZES DE AUDITORIA E MODERNIZAÇÃO DO PROJETO OPENWYD-NEXTGEN

## 0. AUTÓPSIA TÉCNICA E DIRETRIZES DE ARQUITETURA EM RUST

### 0.1. Diagnóstico Crítico da Arquitetura Legada e Transições Inadequadas
A emulação clássica de WYD (C++ arcaico dos anos 2000) baseia-se em um monolito de estado acoplado (`TMSrv`, `dbServer`, `binServer`), gerenciamento manual de memória frágil (`raw pointers`), sincronização de dados por `structs` empacotadas via binário e concorrência baseada em travas globais (`mutexes`). 

Tentativas ingênuas de modernização comumente caem em dois erros fatais de engenharia:
1. **Microsserviços no Hot Path:** Fatiar a física, o combate, o mapa e o inventário em processos distintos comunicando-se via gRPC/HTTP ou sockets síncronos introduz saltos de rede (*network hops*), *serialization overhead* e *jitter* de latência (incompatíveis com a latência determinística necessária para o *game loop* de 20-50 ticks/segundo).
2. **Pausas de GC e Orientação a Objetos no Core:** Reescrever a engine em linguagens com Garbage Collector (como Go) expõe o jogo a pausas imprevisíveis (*stop-the-world* ou *micro-stutters*). Além disso, manter hierarquias OOP (`Player`, `Mob` derivadas de `Creature` alocadas aleatoriamente na RAM) gera uma alta taxa de *CPU cache misses* (L1/L2/L3), destruindo o desempenho do processador.

### 0.2. A Decisão por Rust e o Blueprint Tecnológico Especializado
A reescrita do núcleo do jogo (**Core Engine**) deve ser executada exclusivamente em **Rust** (edicao 2021+), garantindo segurança de memória em tempo de compilação sem custos de GC, aproveitando concorrência *lock-free* e máxima localidade de cache.

#### Principais Pilares da Arquitetura Rust:
- **Data-Oriented Design (DOD) / ECS (Entity Component System):** Transição total da modelagem OOP para dados contíguos em memória (`Vec<Position>`, `Vec<Stats>`, `Vec<Inventory>`). Varreduras de entidades no *game loop* tornam-se operações vetorizadas contíguas (aproveitando instruções SIMD do hardware).
- **Modelo Thread-per-Core & Spatial Partitioning:** O mundo do jogo é particionado espacialmente. Cada thread gerencia a sua zona em isolamento completo. A comunicação entre threads/zonas ocorre exclusivamente por canais *lock-free* de alta velocidade (`crossbeam-channel` ou *ring buffers* assíncronos), eliminando a contenção por *mutexes*.
- **Async I/O via `io_uring` e Tokio:** A camada de transporte de rede no Linux utilizará I/O assíncrono nativo (`io_uring`) com zero-copy de buffers entre o kernel e o espaço de usuário, garantindo capacidade de vazão para dezenas de milhares de pacotes/segundo.
- **Isolamento de Caminhos (Hot Path vs. Cold Path):**
  - **Hot Path (Motor em Rust):** Monolítico por nó de região, processando movimentação, combate, IA e física com alocação estática (utilizando *Arena Allocators* como `bumpalo` para zerar alocações dinâmicas de heap no *tick loop*).
  - **Cold Path (Serviços Periféricos):** Autenticação, bilhetagem, painéis administrativos e portal web desacoplados em microsserviços periféricos (gRPC/REST) comunicando-se com o core de maneira assíncrona.

---

### 0.3. Mapeamento Geral de Migração: De-Para do Código

| Subsistema / Domínio | Arquitetura Legada (C++ / Binários) | Arquitetura Destino (`OpenWYD-NextGen` em Rust) | Benefício de Engenharia / Resolução de Problema |
| :--- | :--- | :--- | :--- |
| **Game Loop & Core** | `TMSrv` (Main.cpp, Map.cpp, Actor.cpp) com ponteiros brutos e Mutexes globais. | **Rust Engine Core** (`wyd-core`): Thread-per-Core + ECS (`bevy_ecs` ou `hecs`) + Spatial Grid. | Fim de *cache misses* e trava de threads; determinismo de ticks (20-50 FPS nativos sem pausas de GC). |
| **Camada de Rede** | `select`/`poll` legados com pacotes brutos desprotegidos. | **Rust Async Net Layer** (`wyd-net`): `tokio` / `io_uring` + Parsers binários com `nom` / `rkyv`. | Prevenção absoluta contra *buffer overflows*, estouro de memória e variação de latência. |
| **Persistência de Dados** | `dbServer` lendo/escrevendo arquivos `.dat` binários instáveis em disco. | **Data Access Layer** (`wyd-db`): PostgreSQL + `sqlx` (Rust async SQL) + Redis Cache transacional. | Consistência ACID, integridade referencial, auditoria transacional e segurança contra perda de dados. |
| **Definições de Domínio** | `Structures.h` com *struct packing* propenso a desalinhamento de CPU. | **Domain Contracts** (`wyd-protocol`): Layouts de memória explícitos (`#[repr(C)]` / `zerocopy`). | Garantia de alinhamento binário exato entre cliente e servidor com validação de limites. |
| **Lógica de Eventos/Quests** | Scripts `.lua` acoplados via C-API arcaica e XMLs expostos. | **Rust Modular System / Safe Scripting Sandbox** (`wasmtime` ou motores compilados). | Isolamento de falhas de scripts de regras sem perigo de derrubar o processo da engine principal. |
| **Gestão Administrativa** | `binServer` acoplado com comandos GM expostos em TCP cru. | **Control Plane & Web Gateway** (`wyd-admin` / Next.js): APIs gRPC/REST desacopladas do *Cold Path*. | Operações de backend e web totalmente isoladas do ciclo de execução principal do jogo. |

---

# 1. IDENTIFICAÇÃO DA MISSÃO E DIRETRIZES GERAIS

**Nome:** Análise de Roadmap, Auditoria Técnica e Modernização Completa do Projeto via Reescrita em Rust.

**Objetivo principal:** Realizar uma auditoria técnica abrangente de todo o projeto, revisar e reorganizar integralmente o roadmap, identificar todas as pendências, inconsistências, problemas arquiteturais e oportunidades de melhoria e, posteriormente, executar a transição progressiva para a arquitetura moderna baseada em Rust.

Esta missão envolve todas as áreas do ecossistema:
- Backend e servidor de jogo em Rust.
- Cliente do jogo e engenharia reversa do protocolo.
- Banco de dados relacional e estratégias de persistência.
- Infraestrutura, DevOps e CI/CD automatizado.
- Portal web (Next.js/TypeScript).
- Sistemas, mecânicas e balanceamento de jogo.
- Observabilidade, métricas e segurança nativa.
- Suíte de testes (unitários, integração, E2E, simulações de combate).
- Skills, agentes de IA e documentação técnica.

**Diretriz fundamental:** Não se limite às issues já existentes. Faça uma investigação completa do projeto, identificando problemas, lacunas, código legado, implementações ausentes e falhas de arquitetura. Toda decisão deve ter justificativa técnica baseada em performance, segurança de memória e manutenibilidade.

---

## 1.1. Regras de Execução e Qualidade Técnica

1. **Análise antes da implementação:** Investigue o funcionamento real antes de alterar qualquer código. Não substitua componentes sem diagnosticar gargalos com profiling ou provas de conceito.
2. **Preservação do Conhecimento:** O conhecimento acumulado em código antigo, rotinas de engenharia reversa e regras de negócio do WYD deve ser extraído e documentado antes de qualquer desativação.
3. **Rastreabilidade:** Relacione problema $\rightarrow$ issue $\rightarrow$ diagnóstico $\rightarrow$ solução em Rust $\rightarrow$ testes $\rightarrow$ validação.
4. **Qualidade do Código Rust:** 
   - Respeitar padrões idomáticos (`clippy`, `rustfmt`).
   - Zerar o uso de `unsafe` desnecessário (isolando `unsafe` estritamente a interop de baixo nível ou otimizações críticas comprovadas).
   - Aplicar `RAII`, tratamento rigoroso de erros com `Result`/`Option` e `Error` tipados (`thiserror` / `anyhow`).
   - Priorizar concorrência segura por design (evitar *data races* garantido pelo *borrow checker*).

---

# FASE 1 — AUDITORIA COMPLETA DO ROADMAP E ORGANIZAÇÃO DAS ISSUES

## 1.1. Auditoria do estado atual
Examine todas as Issues (abertas, fechadas, em andamento), Milestones e diretórios de histórico (`done/`, `test/`). Verifique a evidência real de implementação em vez de status manuais.

## 1.2. Reorganização e Identificação de Lacunas
Elimine duplicidades, agrupe por domínio e faça varreduras por `TODO`, `FIXME`, `HACK` e código morto no repositório.

## 1.3. Skills e Agentes de IA
Atualize as Skills e instruções dos agentes para refletirem a transição para **Rust** (`cargo`, `clippy`, padrões ECS, `tokio`, `io_uring`) e a stack web moderna (Next.js/TypeScript).

---

# FASE 2 — EXECUÇÃO CONTROLADA E EVOLUÇÃO SEQUENCIAL DO ROADMAP

1. **Execução unitária:** Trabalhar em uma Issue por vez com ciclo estrito (Análise $\rightarrow$ Design $\rightarrow$ Código Rust $\rightarrow$ Teste $\rightarrow$ Docs).
2. **Revisão Consolidada:** A cada 10 Issues concluídas, realizar uma auditoria técnica de regressão, performance (benchmarks) e integração de módulos.
3. **Priorização Orientada à Arquitetura:** Respeitar a ordem de dependência: Contratos de Dados/Rede $\rightarrow$ Engine Core em Rust $\rightarrow$ Persistência $\rightarrow$ APIs/Portal.

---

# FASE 3 — MODERNIZAÇÃO E REESCRITA DO SERVIDOR EM RUST

## 3.1. Arquitetura da Engine (`wyd-core`)
- Implementar o ciclo de tick em frequência fixa determinística.
- Adotar o padrão ECS para gestão de estado de todas as entidades do mapa (`Players`, `NPCs`, `Monsters`, `Items` no chão).
- Aplicar particionamento espacial de mapa em grades (*Spatial Grid*) para busca de visibilidade e gerenciamento de interesse (*Interest Management*) em tempo $O(1)$.

## 3.2. Eliminação do Código Legado e Arquivos TXT/CSV
- Substituir arquivos de texto plano arcaicos por carregamento seguro de tabelas tratadas ou banco de dados relacional.
- Estruturar a validação de schema na inicialização do servidor, abortando a execução com relatórios claros se os dados estáticos de jogo estiverem inconsistentes.

## 3.3. Garantia de Compatibilidade de Estado
- O servidor Rust deve manter autoridade total sobre a física, cálculo de dano, uso de skills e validação de inventários, garantindo paridade funcional com o comportamento original do jogo, mas sem suas falhas de segurança.

---

# FASE 4 — MODERNIZAÇÃO E EVOLUÇÃO DO BANCO DE DADOS

## 4.1. Modelagem Relacional e Integridade
- Estruturar o banco de dados (PostgreSQL) para armazenar contas, personagens, inventários, baús, guildas e logs de auditoria.
- Garantir restrições de integridade referencial (`FOREIGN KEYS`, `CHECK constraints`) e índices adequados para consultas de alta frequência.

## 4.2. Migrações e Persistência Assíncrona (`sqlx`)
- Utilizar ferramenta de versionamento de banco (`sqlx-cli` / `refinery`) integrada ao pipeline de build Rust.
- Implementar a persistência no servidor Rust com operações assíncronas em segundo plano, evitando I/O bloqueante no *tick loop* do jogo.

---

# FASE 5 — MODERNIZAÇÃO DO DEVOPS E CI/CD (RUST STACK)

## 5.1. GitHub Actions e Automation
- Configurar pipelines de integração contínua para Rust:
  - `cargo fmt --check`
  - `cargo clippy -- -D warnings`
  - `cargo test --all-targets`
  - `cargo bench` (testes de regressão de desempenho)
  - Security Scanning (`cargo audit`, `cargo deny`).

## 5.2. Empacotamento e Proveniência
- Gerar compilações otimizadas para release (`cargo build --release` com LTO habilitado).
- Empacotar os serviços em containers Docker minimalistas (multi-stage builds baseados em imagens *distroless* ou Alpine).

---

# FASE 6 — OBSERVABILIDADE, MONITORAMENTO E SEGURANÇA NATIA

## 6.1. Telemetria e Métricas em Rust
- Integrar a biblioteca `tracing` e `prometheus` ao servidor em Rust para instrumentação estruturada de código.
- Expor métricas nativas do motor: latência de tick (P50, P95, P99), pacotes/seg processados, contagem de entidades ativas e consumo de memória.

## 6.2. Segurança e Hardening de Conexão
- Implementar proteção contra ataques de *Denial of Service* (DoS) no handshake TCP.
- Aplicar *rate limiting* e sanitização estrita no parser de pacotes binários em Rust.

---

# FASE 7 — PORTAL WEB, EXPERIÊNCIA DO USUÁRIO E ECOSSISTEMA DIGITAL

## 7.1. Next.js / TypeScript Gateway
- Estruturar a aplicação web em Next.js (App Router, Server Components) integrada com a base de dados central e APIs gRPC/REST de controle.

## 7.2. Funcionalidades de Jogador e Painel Administrativo
- Painel do usuário (gestão de conta, recuperação, rankings em tempo real, visualização de personagens).
- Painel de administração com auditoria de logs, estatísticas de economia e ferramentas de moderação.
- Suíte de testes E2E com **Playwright** cobrindo fluxos críticos (autenticação, loja, painel).

---

# FASE 8 — ENGENHARIA REVERSA E MODERNIZAÇÃO DO CLIENTE WYD

## 8.1. Mapeamento de Protocolo Binário
- Mapear a estrutura exata dos pacotes transmitidos pelo cliente legados (cabeçalhos, opcodes, tamanho, payload).
- Escrever o módulo `wyd-protocol` em Rust com validação de limites zero-copy (`zerocopy` / `rkyv`).

## 8.2. Investigação de Patchs e Launcher Independente
- Analisar executáveis e DLLs (`ClientPatch_v7662`).
- Desenvolver um Launcher moderno e seguro em Rust/Tauri ou CLI para verificação de hashs de integridade dos arquivos do cliente, patching automático e inicialização.

---

# FASE 9 — ESTRUTURA DE IA, SKILLS E FERRAMENTAS AUXILIARES

- Consolidar prompts, Skills e instrução de agentes para a pilha de engenharia moderna em Rust.
- Documentar padrões de arquitetura ECS, tratamento de erros idomáticos em Rust e concorrência para que os assistentes operem com máxima precisão técnica.

---

# FASE 10 — ENGENHARIA REVERSA MATEMÁTICA, BUG HUNTING E BALANCEAMENTO

## 10.1. Mapeamento de Atributos e Combate
- Documentar e codificar as equações exatas de combate: ataque base, atributos (Força, Inteligência, Destreza, Constituição), bônus de equipamentos, modificadores percentuais, redução por defesa/absorção e taxas críticas.

## 10.2. Investigação do Comportamento de Velocidade de Ataque (Hits & Críticos)
- Investigar minuciosamente a anomalia de velocidade de ataque ($\sim 200\%$) e ocorrência de hits duplicados ou extrapolação de DPS.
- Modelar o comportamento desejado utilizando funções matemáticas determinísticas e seguras no servidor em Rust, prevenindo exploits de manipulação de pacotes pelo cliente.

## 10.3. Laboratório Matemático e Simulações em Rust
- Criar uma ferramenta CLI/suíte de testes em Rust (`wyd-sim`) capaz de simular milhões de interações de combate em milissegundos para validar balanceamento entre classes, escalonamento de atributos e detecção de *overflows* ou *underflows*.

---

# FASE 11 — DOCUMENTAÇÃO E PADRONIZAÇÃO COMPLETA

- Manter documentação técnica MkDocs atualizada com diagramas de arquitetura C4, especificações de pacotes de rede e guias de contribuição para o código em Rust.

---

# FASE 12 — DASHBOARD E VISUALIZAÇÃO DO ESTADO DO PROJETO

- Construir painéis de monitoramento (Grafana + Prometheus) exibindo a saúde do servidor Rust, estado das migrations de banco e telemetria de produção.

---

# FASE 13 — PROPOSTAS TÉCNICAS E ARQUITETURA DE DECISÃO (ADRs)

- Registrar qualquer mudança substancial de design através de *Architecture Decision Records* (ADRs) antes da execução de refatorações de grande escala.

---

# FASE 14 — PESQUISA TÉCNICA CONTÍNUA

- Acompanhar evoluções na pilha Rust para sistemas de jogos (ecossistema ECS, otimizações de I/O em Linux, bibliotecas de serialização de alta performance).

---

# FASE 15 — AUDITORIA FINAL E PREPARAÇÃO PARA RELEASE

- Executar a validação fim a fim: suíte completa de testes Rust, verificação de concorrência (`cargo-tarpaulin`, `miri`), testes de estresse de carga de rede, validação de integridade do banco e build de produção.

---

# 16. ORDEM ESTRATÉGICA DE EXECUÇÃO E CRITÉRIOS DE ACEITE

### Ordem Recomendada:
1. Auditoria e Reorganização do Roadmap.
2. Definição do Contrato de Protocolo e Tipos em Rust (`wyd-protocol`).
3. Implementação da Engine Core em Rust com ECS e Spatial Grid (`wyd-core`).
4. Desenho e Integração da Persistência Assíncrona (`wyd-db` / PostgreSQL).
5. Configuração do CI/CD Rust e Observabilidade (Prometheus/Grafana).
6. Integração do Portal Web (Next.js) e Launcher.
7. Suíte de Simulação Matemática de Combate e Balanceamento (`wyd-sim`).
8. Validação Final, Testes de Carga e Release.

### Critério de Conclusão Global:
A missão só é considerada encerrada após a comprovação de que o ecossistema em Rust está estável, compilando sem warnings de clippy, livre de data-races, com cobertura de testes unitários/integrados das fórmulas críticas e operando com determinismo e alta performance.