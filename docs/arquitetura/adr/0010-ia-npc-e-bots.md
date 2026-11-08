# ADR-0010 — IA de NPC determinística no Hot Path; bots em crate separada; LLM só no Cold Path

- **Estado:** proposta (planejamento; nada implementado)
- **Data:** 2026-11-08
- **Decisores:** dono do projeto + agente (a dúvida "os bots com IA estão planejados?" abriu a análise)
- **Governa:** `roadmap-rust/issues/FASE-03-6-ia-npc.md`,
  `roadmap-rust/issues/FASE-15-3-bots-carga.md`, `docs/rust/07-ia-e-bots.md`
- **Depende de:** ADR-0001 (monolito Hot Path), ADR-0002 (estrutura de módulos),
  ADR-0004 (ECS), ADR-0006 (async diferido), ADR-0009 (conteúdo no banco)

## Contexto

A pergunta do dono ("nossos bots, com possível inteligência artificial, já estão planejados?")
foi respondida por varredura, não por memória. O que existe hoje:

1. **A "FASE 9 — IA" do planejamento mestre não é IA de jogo.** Em `pre-release-rust.md`
   §FASE 9 (linhas 173–178) o objetivo é "consolidar prompts, Skills e instrução de agentes"
   e "documentar padrões de arquitetura ECS…"; `pre-release-rust.md` §1.3 (linhas 80–83)
   diz o mesmo em uma linha ("Atualize as Skills e instruções dos agentes"). O roadmap ecoa
   isso: `roadmap-rust/index.md` linha 37 descreve a fase 9 como "IA, skills e ferramentas
   auxiliares", e `roadmap-rust/issues/FASE-09.md` fecha o mapeamento Go→Rust de skills.
   Ou seja: "IA" ali é **IA de desenvolvimento**, não comportamento de NPC nem bot jogador.
2. **Bots existentes são de teste.** `docs/manual-técnico/skills.md` linha 25 registra
   `wyd-testing-bots` → `wyd-rust-testing` ("harness/bots/carga em Rust"), e
   `docs/rust/03-progresso.md` linha 166 repete a criação da skill. O harness de carga é
   citado em `pre-release-rust.md` FASE 10 §10.3 (linhas 189–192) e em FASE 15 (carga de
   rede), mas **não existe código de bot** em `crates/` (nenhuma crate além de
   protocol/core/net/db/admin/sim/launcher).
3. **O documento de bots que o `start-rust.md` cita não existe mais.** `start-rust.md`
   linha 6 aponta `D:\dolorestec\wyd2\OpenWYD\pre-release-bots.md - definições da suite de
   bots`. A raiz hoje tem só `AGENTS.md`, `README.md`, `pre-release-rust.md` e
   `start-rust.md`; nenhum `pre-release-bots.md`. A definição da suíte **não foi herdada**
   para `docs/`.
4. **Nenhum módulo de comportamento existe.** A árvore alvo de `wyd-core` em
   `docs/arquitetura/estrutura-e-padroes.md` §10 tem `systems/`, `rules/`, `world/` e
   `ports/`, sem `npc/`/`ai/`; o diretório real (`crates/wyd-core/src/`) contém apenas
   `ecs/`, `ports/`, `spatial/`, `tick/`, `error.rs`, `lib.rs`. `docs/arquitetura/modelo-banco.md`
   §6.5/§6.8 e `banco-catalogo.md` modelam 59 tabelas e nenhuma de comportamento, diálogo ou fila de LLM.
5. **A única menção a bot em conteúdo é sobre qualidade de dado.** `docs/wyd/classes.md`
   linha 183–185: "um `skill_role` errado faz o bot de teste tomar decisão errada e o bug
   aparece como 'IA burra', não como dado ruim" — presume bot de teste, não IA implementada.

O vetor de decisão é o mesmo das ADRs anteriores: `wyd-core` não pode fazer I/O no tick
(ADR-0001), `unsafe`/alocação quente são proibidos, e conteúdo chega pronto do banco
(ADR-0009). Qualquer plano de IA que chame rede, LLM ou banco dentro do tick viola essas
decisões; qualquer LLM sem limite de custo viola a operação do servidor.

## Decisão

Separamos **três camadas** com fronteiras explícitas, e só a primeira roda dentro do
processo do servidor:

| Camada | O que é | Onde vive | Determinismo | I/O |
|---|---|---|---|---|
| **A. IA de NPC (Hot Path)** | aggro, patrulha, fuga, uso de skill, retorno ao spawn | `crates/wyd-core/src/npc/` (novo, nesta ADR) | **total** (RNG semeado e salvo) | nenhum |
| **B. Bots jogáveis / carga** | cliente falso que conecta e joga (QA, carga, economia) | `crates/wyd-bots/` (crate nova) + política pura em `crates/wyd-sim/src/bot/` | política sim, sessão não | rede (cliente), nunca no servidor |
| **C. LLM (texto e conteúdo)** | diálogo de NPC, descrição de quest, nomes | `crates/wyd-admin` (Cold Path) → tabela → revisão humana | não se aplica | HTTP, fora do processo do jogo |

Regras que decorrem da decisão:

- **Nenhum LLM no tick e nenhum no processo do servidor.** Requisição de modelo tem
  latência de segundos e custo por token; o tick é **20 Hz** (50 ms,
  `crates/wyd-core/src/lib.rs:20,25`, `TICK_HZ = 20`). Texto gerado entra no banco pelo Cold
  Path, é **revisado** e só então vira conteúdo carregado no boot (ADR-0009).
- **IA de NPC é aritmética inteira e determinística.** Nada de `f32` no caminho de decisão
  (mesma regra de `docs/rust/06-matematica.md`); pontuação em inteiro, RNG semeado por
  entidade e persistido no snapshot, para que o mesmo save produza a mesma sequência de
  decisões.
- **Cadência de decisão separada da cadência de simulação.** A IA decide a 5 Hz — uma decisão
  a cada **4 ticks** de 20 Hz, com fase `entity_id % 4` — enquanto movimento/combate rodam no
  tick cheio; `budget.rs` fixa teto de decisões por tick e o bench
  (`crates/wyd-core/benches/tick.rs`) prova o custo. Unidades, estágios e contrato de
  determinismo: `docs/arquitetura/ia-bots.md` §3–§4.
- **Bots não entram no servidor.** `crates/wyd-bots` é binário de teste/carga, fora do
  compose de produção (`docs/rust/05-stack-docker.md`: só entra no compose o que é
  serviço). Ele depende de `wyd-protocol` + `wyd-net` + `wyd-sim` e **nada depende dele**.
- **Sem `unsafe` e sem dependência nova** para IA: utilidade/FSM são `match` e soma inteira.
  Se algum dia houver modelo treinado embarcado, exige ADR própria.
- **LLM é opcional e desligado por padrão.** Sem `OPENAI_API_KEY`/provedor configurado, o
  Cold Path não gera nada e o jogo funciona igual; chave nunca em `crates/`, nunca no
  cliente, nunca em log (`AGENTS.md` invariante 6).
- **Conteúdo de comportamento é dado, não código.** Perfil de NPC vem de tabela
  (`npc_behavior`, planejada) carregada no boot como SoA; ajustar agressividade não
  recompila o servidor.

## Consequências

- **Positivas:** a pergunta passa a ter resposta rastreável; o custo por tick fica medido,
  não estimado; bots de carga deixam de ser promessa ("harness") e viram crate com
  contrato; LLM fica onde não pode derrubar o jogo; reusa a mesma FSM no core e no
  laboratório, então balanceamento e comportamento não divergem.
- **Negativas:** **crate nova** (`wyd-bots`) muda o DAG de `estrutura-e-padroes.md` §1 e
  exige issue própria; `npc/` cresce a árvore do core (§10); tabelas novas
  (`npc_behavior`, `npc_dialogue`, `llm_job`) exigem migração futura e validador no boot;
  IA determinística limita "esperteza" (nada de planejamento adaptativo caro por decisão).
- **Invariantes afetados:** nenhum I/O/alocação quente; nenhum mutex global; servidor
  continua autoridade (IA só emite intenção que os `systems/` validam); `ESCENE_FIELD`
  inalterado; entrada de rede nunca causa `panic!` (vale para os bots também).

## Alternativas consideradas

| Alternativa | Por que foi descartada |
|---|---|
| LLM respondendo NPC em tempo real dentro do tick | latência de segundos × tick de 20–50 Hz; custo por token imprevisível; indeterminismo quebra replay/golden |
| Serviço de IA separado consultado por gRPC no tick | reintroduz o salto de rede que a ADR-0001 elimina |
| Bots dentro de `wyd-core` | mistura teste com engine; o servidor carregaria código de cliente falso |
| Bots em `tools/` (Node) | §15 de `estrutura-e-padroes.md` fecha `tools/` como análise/geração, nunca runtime |
| Behaviors em script embutido (Lua/WASM) por NPC | dependência nova + avaliação dinâmica no quente; sem ganho sobre tabela + FSM |
| Modelo de ML treinado embarcado | sem dado de treino, sem orçamento de CPU provado, exige ADR própria |

## Rastreabilidade

- Documento de execução: `docs/rust/07-ia-e-bots.md` (estado por item, fluxo de dados,
  orçamento, divergências dos docs legados).
- Modelo de decisão e determinismo verificável: `docs/arquitetura/ia-bots.md` (unidades,
  cadência de 4 ticks, golden de decisão, estado/observação, mapa de pastas, D-IA01..D-IA06).
- Issues: `roadmap-rust/issues/FASE-03-6-ia-npc.md`,
  `roadmap-rust/issues/FASE-15-3-bots-carga.md`.
- Árvores atualizadas nesta ADR: `docs/arquitetura/estrutura-e-padroes.md` §1 (DAG),
  §10 (`wyd-core/src/npc/`), §14 (`wyd-sim/src/bot/`), §15 (inalterado: `tools/` segue
  sem runtime).
- Evidência da ausência: `ls *.md` na raiz (sem `pre-release-bots.md`); `grep -rInE
  "bot|IA|LLM" docs roadmap-rust crates` (só skills de agente, launcher e a nota de
  `docs/wyd/classes.md`); `docs/arquitetura/estrutura-e-padroes.md` §10 sem `npc/`.

## Revisão

- Revisar quando: a FASE-03-6 medir o custo real da IA no tick; o dono decidir sobre LLM
  (custo/chave/provedor); `mob_template_stat` for populado (a IA depende de stats de
  monstro que hoje estão vazios — `crates/wyd-db/migrations/README.md`, "Lacunas
  declaradas").
- Sinais de que a decisão envelheceu: alguém pedindo "só uma chamadinha de LLM no tick";
  custo de IA passando de 10% do orçamento do tick sem bench; bot precisando de `unsafe`.
