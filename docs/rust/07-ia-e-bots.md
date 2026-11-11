# IA de NPC, bots e LLM — estado, plano e fronteiras

> Documento de execução (irmão de `06-matematica.md`). Decisão de arquitetura:
> `docs/arquitetura/adr/0010-ia-npc-e-bots.md`. Árvores: `estrutura-e-padroes.md`
> §10 (`wyd-core`) e §14 (`wyd-sim`). Issues: `FASE-03-6` (IA de NPC) e
> `FASE-15-3` (bots de carga). Painel: `03-progresso.md`.

## 1. Resposta curta

**Não.** Até 2026-11-08 não existia plano de bots com IA nem de IA de NPC neste repositório.
O que existia — e continua existindo — é (a) a fase "IA" do planejamento mestre, que trata
de **skills de agentes de desenvolvimento**, e (b) a skill de **harness de bots de teste**,
sem código de bot em `crates/`. A definição da suíte de bots citada pelo `start-rust.md`
(`pre-release-bots.md`) **não está na árvore**.Este documento fechou a lacuna de planejamento; a **camada A** foi executada em 2026-11-08
(FASE-03-6, ADR-0016) — `crates/wyd-core/src/npc/` + `systems/npc.rs`. As camadas B e C
seguem sem código.

## 2. Estado por item (existente · planejado · lacuna)

| Item | Estado | Evidência |
|---|---|---|
| "FASE 9 — IA" = skills/instruções de agentes | **existente** | `pre-release-rust.md` FASE 9 (l. 173–178), §1.3 (l. 80–83); `roadmap-rust/issues/FASE-09.md` (objetivo: mapeamento Go→Rust de skills) |
| Skill de harness de bots/carga | **existente (doc)** | `docs/manual-técnico/skills.md` l. 25 (`wyd-testing-bots` → `wyd-rust-testing`); `docs/rust/03-progresso.md` l. 166 |
| Simulação massiva de combate (`wyd-sim`) | **existente (estrutura)** | `pre-release-rust.md` FASE 10 §10.3; `estrutura-e-padroes.md` §14; `docs/rust/06-matematica.md` |
| `pre-release-bots.md` (definições da suíte) | **lacuna** | citado em `start-rust.md` l. 6; ausente na raiz (`ls *.md`: `AGENTS.md`, `README.md`, `pre-release-rust.md`, `start-rust.md`) |
| IA de NPC (aggro/perseguição/ataque/fuga/retorno) | **executada (2026-11-08)** | `crates/wyd-core/src/npc/{state,behavior,brain,route,budget}.rs` + `src/systems/npc.rs` (ADR-0016); matemática de ameaça copiada de `crates/wyd-sim/src/formula/threat.rs` e presa por golden em `tests/integracao/npc.rs`; tabela `npc_behavior` continua ausente (perfil vem por parâmetro) |
| Bots jogáveis (cliente falso) | **planejado** | ADR-0010 camada B; crate `crates/wyd-bots` (nova) — `FASE-15-3` |
| LLM para diálogo/quest/nomes | **lacuna (bloqueada por decisão do dono)** | nenhum doc cita LLM; custo/chave exigem aprovação → §7 |
| Tabelas `npc_behavior`, `npc_dialogue`, `llm_job` | **planejado (não criado)** | `banco-catalogo.md` modela 69 tabelas de jogo/conteúdo (§6.5) |
| Stats de monstro (insumo da IA) | **lacuna** | `mob_template_stat`/`mob_template_equip` vazias — `crates/wyd-db/migrations/README.md` §Lacunas; por isso `NpcProfile` é informado pelo chamador, não derivado do banco |

## 3. Por que a resposta parecia "sim"

1. O roadmap tem uma fase chamada "IA" (`roadmap-rust/index.md` l. 37) — é IA de
   desenvolvimento, não de jogo.
2. O `start-rust.md` cita `pre-release-bots.md` como se existisse; ele foi perdido na
   remoção da árvore Go, e a suíte nunca foi reescrita em `docs/`.
3. `docs/wyd/classes.md` l. 183–185 fala de "bot de teste tomar decisão errada" ao explicar
   qualidade de `skill_role` — pressupõe bot, não o descreve.

## 4. Divergências entre os docs legados e o repo (registradas, não silenciadas)

| Doc legado | O que ele afirma | Estado real |
|---|---|---|
| `start-rust.md` l. 6 | existe `pre-release-bots.md` com "definições da suite de bots" | arquivo **não existe**; nada equivalente em `docs/` até este documento |
| `start-rust.md` l. 5, 9–13 | roadmap Go em `roadmap/`, projeto antigo em `.deprecated-go`, restando "apenas `.deprecated-go` e README.md" | `AGENTS.md` §0: `roadmap/` (Go), `.deprecated-go/` e `tmserver/` **removidos** em 2026-11; árvore atual é `crates/` + `docs/` + `roadmap-rust/` |
| `pre-release-rust.md` FASE 9 | título "IA, skills e ferramentas" sugere IA de produto | conteúdo é skills de agente (l. 175–178) |
| `start-rust.md` | instrução "não escrever código até tudo documentado" | cumprida: `crates/wyd-bots` e `npc/` seguem **sem código** |

## 5. As três camadas e seus limites

**A. IA de NPC (Hot Path, `crates/wyd-core/src/npc/`).** Máquina de estados + utilidade
inteira (a parte matemática — tabela de ameaça, alvo e `NpcMode` — já existe em
`crates/wyd-sim/src/formula/threat.rs`, ADR-0011): `idle → patrol → aggro → chase → attack → flee → return`. Decide a **5 Hz**
(proposta a medir), com RNG semeado e salvo, sem I/O e sem alocação no quente. Emite
*intenção* (`MoveTo`, `UseSkill`, `Attack`) que os `systems/` validam — o servidor segue
autoridade (ADR-0001, `AGENTS.md` invariante 3).

**B. Bots jogáveis (`crates/wyd-bots/` + política em `crates/wyd-sim/src/bot/`).** Cliente
falso para QA, carga e economia (50–500 conexões, perfis por classe/nível). Roda **fora** do
processo do servidor; usa `wyd-protocol` + `wyd-net`; a política é função pura testada no
laboratório, para o mesmo perfil gerar a mesma sequência de ações.

**C. LLM (Cold Path, `crates/wyd-admin`).** Só texto: diálogo de NPC, descrição de quest,
nomes. Fluxo: staff pede → job na tabela `llm_job` → worker chama o provedor → resultado
**revisado** pelo portal → vira linha em `npc_dialogue`/`quest` → entra no boot seguinte.
Nunca no tick, nunca no cliente, chave só em segredo de ambiente.

## 6. Módulos e arquivos (planejados, com caminho real)

```
crates/wyd-core/src/npc/            # FASE-03-6 (ADR-0010)
├── mod.rs            # fachada do módulo (sem lógica)
├── state.rs          # NpcState em SoA: alvo, rota, cooldown, âncoras, humor
├── behavior.rs       # FSM (transições explícitas, uma função por estado)
├── brain.rs          # pontuação inteira de alvo/ação (utilidade)
├── route.rs          # rota do generator/npc_definition (UNVERIFIED até trace)
└── budget.rs         # cadência de decisão, teto por tick, contadores p/ telemetria
crates/wyd-core/src/systems/npc.rs  # integração com o tick (ordem documentada)
crates/wyd-core/tests/integracao/npc.rs        # determinismo: mesmo seed → mesmas decisões

crates/wyd-sim/src/bot/             # FASE-15-3
├── mod.rs
├── persona.rs        # perfil (classe, nível, objetivo) — dado, sem lógica
└── policy.rs         # estado → ação, pura e testável (mesma FSM do core)

crates/wyd-bots/                    # crate nova (ADR-0010 altera o DAG §1)
├── src/lib.rs        # sessão: conecta, autentica, joga
├── src/scenario.rs   # carga: N bots × perfil × duração
└── src/bin/wyd-bots.rs  # CLI (contagem, perfil, alvo, duração, saída de métricas)
```

`crates/wyd-bots` **não** entra no compose de produção (`docs/rust/05-stack-docker.md`:
serviço = binário do jogo). Nada depende dela.

## 7. Fluxo de dados

```
                   ┌──────────── Cold Path ────────────┐
 staff (portal) ──►│ llm_job ──► worker LLM ──► revisão │──► npc_dialogue / quest
                   └───────────────────────────────────┘
                                   │ (migração/seed)
                                   ▼
   PostgreSQL ──boot──► ContentTables (SoA imutável) ──► wyd-core
                                                        │
   tick (50 Hz) ──► systems/ ──► npc::budget (5 Hz) ────┘
                        │             └─► intenção (MoveTo/Attack/UseSkill)
                        ▼
                  estado do mundo ──► snapshot (RNG incluído) ──► wyd-db
```

Bots (camada B) falam com o servidor **pelo mesmo protocolo do cliente** (`wyd-net`), sem
atalho interno: se o protocolo não permite, o bot não faz. Isso mantém o bot como teste de
verdade do contrato de rede.

## 8. Contrato com `wyd-sim`

- A política (`behavior.rs`/`brain.rs`) é **função pura** de `NpcState × visão do mundo →
  intenção`; `wyd-sim` reusa a mesma função para cenários de balanceamento, sem duplicar
  regra (dívida de duplicação registrada em `06-matematica.md` §4 vale aqui também).
- Golden de decisão: seed + estado inicial + 100 ticks → lista de intenções. Mudança de
  comportamento quebra o golden e exige justificativa, como em `tests/golden/formulas.rs`.
- `scenario/` pode explorar hipótese (agressividade, alcance); `formula/` e `npc/` não —
  mesma regra do laboratório matemático.

## 9. Banco (planejado; nada criado nesta entrega)

| Tabela | Papel | Por que ainda não existe |
|---|---|---|
| `npc_behavior` | perfil por `npc_template`: agressividade, alcance de aggro, cadência, skills, fuga, retorno | faltam stats de monstro (`mob_template_stat` vazia) e a semântica de `route_type` é UNVERIFIED |
| `npc_dialogue` | falas revisadas (LLM ou escritas à mão) por NPC/idioma | depende da decisão de LLM (§10) |
| `llm_job` | fila de geração com estado, custo e revisor | idem; criar antes seria schema sem produtor |

Regra mantida: esses dados são **conteúdo carregado no boot** (ADR-0009), nunca consultados
no tick.

## 10. Orçamento, limites e telemetria

Propostas iniciais (a medir, não fatos):

| Grandeza | Alvo inicial | Como será provado |
|---|---|---|
| custo da IA no tick | ≤ 10% do orçamento do tick com 500 NPCs ativos | **medido** (5 execuções): `cargo bench -p wyd-core --bench tick` → 88,6–204,5 µs/tick (p99 até 367,8) = 0,18%–0,41% do tick de 50 ms com 500 NPCs + 50 jogadores (pior p99 0,74%); 0,71–1,64 µs/decisão; sem alvos 12,5–31,8 µs |
| cadência de decisão | 5 Hz por NPC, escalonada por `entity_id % 4` para não concentrar | **feito**: `npc::budget::should_decide` + `DecisionBudget`, teste de cadência em `tests/integracao/npc.rs` |
| decisões por tick | teto fixo; excedente espera o próximo ciclo | teste de saturação |
| bots de carga | 200 conexões, P99 de pacote registrado | relatório da `FASE-15-3` |
| custo de LLM | 0 por padrão; teto por job e por dia quando habilitado | tabela `llm_job` (tokens/custo por linha) |

Telemetria (FASE-06, `docs/rust/…`, skill `wyd-rust-obs`): decisões/s, profundidade da
fila, NPCs em cada estado, tempo por sistema no tick, erros de política. Sem métrica, "IA
funciona" é opinião.

## 11. Ordem de execução

1. `FASE-03-6` — IA de NPC no core (FSM + utilidade + determinismo + bench). **Entregue em
   2026-11-08** (ADR-0016) sem depender de `mob_template_stat`: o perfil de comportamento é
   parâmetro validado no boot. Bench de custo por tick executável e medido (§10).
2. `FASE-15-3` — crate `wyd-bots`, política em `wyd-sim`, carga e relatório.
3. LLM (camada C) — sem issue até o dono decidir provedor, custo e quem revisa.

## 12. O que este plano **não** faz

- Não coloca LLM no tick nem no processo do servidor.
- Não cria modelo treinado, embeddings ou dependência de ML.
- Não promete "NPC que aprende": o escopo é FSM + utilidade determinística.
- Não implementa bot jogável nem LLM — `crates/wyd-bots`, `npc_dialogue` e `llm_job`
  seguem sem código; a camada A (`npc/`) foi entregue depois de documentada, como manda
  `start-rust.md` ("não escrever código até tudo documentado").
