# Bots e IA — modelo de decisão, determinismo e fronteiras

> Documento de **arquitetura** do subsistema de IA. Complementa (não substitui) a decisão em
> `docs/arquitetura/adr/0010-ia-npc-e-bots.md` e a execução em `docs/rust/07-ia-e-bots.md`.
> Cada afirmação abaixo leva `CONFIRMADO` (com caminho:linha) ou `UNVERIFIED/provisório`.
> **Camada A executada em 2026-11-08** (FASE-03-6, ADR-0016): `crates/wyd-core/src/npc/` e
> `crates/wyd-core/src/systems/npc.rs` rodam a FSM descrita abaixo. Bots jogáveis (camada B)
> e LLM (camada C) seguem plano.

## 1. Resposta à pergunta "os bots com IA já estão planejados e documentados?"

**Sim, estão planejados e documentados — e não estão implementados.** A resposta tem três
partes, porque a palavra "IA" aparece na árvore com dois sentidos diferentes:

| Afirmação | Estado | Evidência |
|---|---|---|
| Existe decisão de arquitetura para IA de NPC, bots jogáveis e LLM | **CONFIRMADO** | `docs/arquitetura/adr/0010-ia-npc-e-bots.md` (estado: "proposta (planejamento; nada implementado)") |
| Existe documento de execução com estado por item, orçamento e ordem | **CONFIRMADO** | `docs/rust/07-ia-e-bots.md` §1–§12 |
| Existem issues de roadmap para as duas camadas de jogo | **CONFIRMADO** | `roadmap-rust/issues/FASE-03-6-ia-npc.md`, `roadmap-rust/issues/FASE-15-3-bots-carga.md` |
| As árvores de módulos já reservam os diretórios | **CONFIRMADO** | `docs/arquitetura/estrutura-e-padroes.md:16` (`wyd-bots` no DAG), `:199` (`systems/npc.rs`), `:206` (`npc/`), `:358` (`bot/`), `:381` ("reservados") |
| A matemática de aggro/alvo existe e é pura | **CONFIRMADO** | `crates/wyd-sim/src/formula/threat.rs` (275 linhas: `ThreatTable`, `MAX_THREAT_ENTRIES = 8`, `select_target`, `NpcMode::should_aggro`) |
| Existe comportamento de NPC no tick (FSM + utilidade) | **CONFIRMADO** | `crates/wyd-core/src/npc/{state,behavior,brain,route,budget}.rs` + `src/systems/npc.rs` (ADR-0016); `tests/integracao/npc.rs` com 12 testes, golden FNV-1a e recusa atômica |
| Existe crate de bots jogáveis | **NÃO** — `crates/` não tem `wyd-bots`; a crate está só no DAG (§2) | `estrutura-e-padroes.md:16` |
| Existe LLM em qualquer lugar | **NÃO** — nenhum código, nenhuma tabela; camada C depende de decisão do dono (custo/chave/provedor) | `07-ia-e-bots.md` §9, §11.3 |
| "FASE 9 — IA" do planejamento mestre é IA de jogo | **NÃO** — trata de skills/instruções de agentes de desenvolvimento | `pre-release-rust.md:173–178` e `:80–83` |
| A suíte de bots citada pelo `start-rust.md` existe na árvore | **NÃO** — `start-rust.md:8` aponta `pre-release-bots.md`; a raiz só tem `AGENTS.md`, `README.md`, `pre-release-rust.md`, `start-rust.md` | `ls *.md` |

O que **não** estava fechado (e este documento fecha): o modelo de decisão com unidades, o
requisito de determinismo em forma verificável, e o contrato de estado/observação/persistência
por camada. A ADR diz "determinismo total"; aqui está o que isso significa em bits.

## 2. As três camadas (fronteira, não sugestão)

| Camada | O que é | Onde vive (planejado) | Determinismo | I/O |
|---|---|---|---|---|
| **A. IA de NPC** | aggro, patrulha, fuga, skill, retorno | `crates/wyd-core/src/npc/` + `systems/npc.rs` | **total** (RNG semeado e salvo) | nenhum no tick |
| **B. Bots jogáveis/carga** | cliente falso que conecta e joga | `crates/wyd-bots/` + política pura em `crates/wyd-sim/src/bot/` | política sim; sessão não | rede, **fora** do servidor |
| **C. LLM (texto)** | diálogo, descrição, nomes | `crates/wyd-admin` (Cold Path) → tabela → revisão | não se aplica | HTTP, fora do processo do jogo |

Fonte: ADR-0010 §Decisão. `TICK_HZ = 20` (50 ms por tick) em `crates/wyd-core/src/lib.rs:20,25`
— é essa a base de todo orçamento abaixo (CONFIRMADO no código; a ADR citava "20–50 Hz", valor
que este documento corrige).

## 3. Modelo de decisão (percepção → alvo → ação)

Cada decisão é uma **função pura** de `VisãoDoMundo × NpcState → Intenção`. Nenhum estágio
consulta banco, rede, relógio de parede ou `HashMap`.

| Estágio | Entrada (unidades) | Saída | Regra vigente | Custo-alvo |
|---|---|---|---|---|
| **P1. Percepção** | posição do NPC e dos candidatos (tiles 0..4095), HP (`u32`), `aggro_range` (tiles) | lista de candidatos visíveis | distância comparada com alcance, **inclusiva** (`NpcMode::should_aggro`, `threat.rs:184–188`) | O(k) com k ≤ candidatos da célula do grid |
| **P2. Seleção de alvo** | tabela de ameaça (`ThreatTable`, teto **8** entradas) + tick atual | `entity: u32` ou nenhum | maior ameaça; **empate pelo menor `entity`** (`threat.rs:127–146`) — nunca por ordem de chegada | O(8) fixo, sem alocação |
| **P3. Ação** | `NpcMode` (7 estados, `threat.rs:155–180`) + alvo + cooldowns | `Intenção` (`Idle`, `MoveTo`, `Attack`, `UseSkill`, `Flee`, `ReturnSpawn`) | uma intenção por decisão; `systems/` valida e aplica | O(1) |

Ameaça hoje é `threat_from_damage(dano) = dano` (`threat.rs:149–151`, marcada *provisório* no
próprio código): 1 ponto de ameaça por 1 de dano. Expiração por TTL de ticks (`select_target`/
`prune`), não por tempo de parede.

**Cadência.** A decisão roda a **5 Hz**, isto é, uma vez a cada **4 ticks** de 20 Hz
(`tick % 4 == fase`), com `fase = entity_id % 4` para espalhar os NPCs em vez de concentrar
todos no mesmo tick. Movimento e combate continuam no tick cheio. Correção de proposta: a ADR
dizia fase `entity_id % 10`, que a 20 Hz daria 2 Hz, não 5 Hz — ver **D-IA01**.

**Orçamento por tick.** Teto de decisões por tick em `npc/budget.rs` (`DecisionBudget`); ao
estourar, o NPC espera o próximo ciclo em vez de atrasar o tick. Meta de custo: IA ≤ 10% do
orçamento do tick com 500 NPCs ativos — **medida** em `crates/wyd-core/benches/tick.rs`
(`cargo bench -p wyd-core --bench tick`, release): 500 NPCs + 50 jogadores custam
**88,6–204,5 µs/tick** em cinco execuções (p99 119,4–367,8 µs), isto é **0,18%–0,41%** dos
50 ms de um tick a 20 Hz (p99 0,24%–0,74%) e 0,71–1,64 µs por decisão; sem nenhum jogador,
12,5–31,8 µs/tick (0,025%–0,064%) — o piso de percepção. O teto de 10% tem folga mínima de
~13×, então a cadência de 5 Hz e o teto de 133 decisões por tick ficam como estão (números de
2026-11-08, host Windows, build release; variação é ruído de máquina).

## 4. Determinismo — o que significa, em termos verificáveis

**Contrato.** Dados o mesmo snapshot inicial (entidades, posições, HP, tabelas de ameaça,
estado do RNG), a mesma semente, as mesmas tabelas de conteúdo carregadas no boot e a mesma
sequência de entradas externas por tick (dano recebido), o subsistema produz **a mesma
sequência de intenções**, na mesma ordem, com o mesmo tick de origem.

Formato verificável (planejado):

```text
saída por decisão: <tick>,<entity>,<kind>,<arg0>,<arg1>
golden:            fixtures/npc/decision_golden.txt
verificação:       hash FNV-1a 64 do fluxo + diff da primeira divergência
teste:             crates/wyd-core/tests/integracao/npc.rs  (2 execuções → mesmo hash)
                   + execução com entidades inseridas em ordem invertida → mesmas decisões
                   + replay a partir de snapshot → mesmo fluxo
```

**O que quebra o contrato** (cada item é proibição, não recomendação): iteração de `HashMap`,
`f32`/`f64` no caminho de decisão (`docs/rust/06-matematica.md`: inteiro), relógio de parede,
agendamento de thread, RNG sem semente ou semeado por tempo, e qualquer I/O no tick. O
desempate por menor `entity` existe exatamente para que a ordem de inserção não vire decisão —
já coberto por `tie_breaks_by_lowest_entity_for_determinism` (`threat.rs`).

**RNG.** Estado por NPC (`u64`), semeado no spawn a partir de `(seed_do_mundo, entity_id)` e
**salvo no snapshot**, para que religar o servidor continue a mesma sequência em vez de
recomeçar (D-IA04).

## 5. Estado, persistência e observação

| Dado | Onde vive | Persistência | Fonte |
|---|---|---|---|
| `NpcMode`, alvo, cooldowns, âncora de spawn, RNG | SoA no core (`npc/state.rs`) | snapshot do mundo (save), não tabela por NPC | ADR-0004 (ECS), ADR-0010 §5 |
| Perfil de comportamento (agressividade, alcance, cadência, skills, fuga) | tabela de conteúdo carregada no boot | **tabela `npc_behavior` — planejada, NÃO criada** | `07-ia-e-bots.md` §9 |
| Falas revisadas | tabela `npc_dialogue` — **planejada** | — | idem |
| Fila de geração LLM | tabela `llm_job` — **planejada** | — | idem |
| Métricas (decisões/s, NPCs por estado, tempo por sistema, fila) | contadores no processo | não persiste; exposto por telemetria | `roadmap-rust/issues/FASE-06-1-telemetria.md` |

Regra de schema: **nada de tabela nova sem produtor no código.** Hoje nenhum código consulta
`npc_behavior`/`npc_dialogue`/`llm_job`, então elas seguem como pendência declarada no
`banco-catalogo.md` (§3, decisões D-Bxx) e não entram nas migrations. O catálogo continua com
59 tabelas; a lacuna é nomeada, não silenciosa.

## 6. Integração com o que já existe (sem acoplamento novo)

| Sistema existente | Como a IA conversa com ele | O que **não** acontece |
|---|---|---|
| Tick/`systems/` | IA emite **intenção**; o sistema valida e aplica (servidor é autoridade) | IA não move nem causa dano direto |
| `ComponentStore`/inventário | skill de NPC consome o mesmo caminho já modelado | IA não ganha porta de escrita paralela |
| `DomainEvents` (`ecs/resources.rs`) | mudança de alvo/estado pode publicar evento próprio, drenado fora do tick | IA não faz `println`/log no quente |
| `delivery_queue` | só Cold Path → Hot Path (grants); IA **não** escreve nela | nenhum caminho de jogo → web |
| Conteúdo (ADR-0009) | tabelas carregadas no boot viram SoA imutável | nenhuma consulta SQL no tick |
| `wyd-sim` | reusa `formula/threat.rs`; golden de decisão mora no core | regra não é duplicada entre sim e core |

## 7. Mapa de pastas e arquivos (planejado; nada é movido nem renomeado)

```text
crates/wyd-core/src/npc/            # FASE-03-6, camada A
├── mod.rs            # fachada (sem lógica)
├── state.rs          # NpcState em SoA: modo, alvo, rota, âncoras, cooldown, rng
├── behavior.rs       # FSM: uma função por transição, sem I/O
├── brain.rs          # pontuação inteira (percepção + alvo + ação) — pura
├── route.rs          # rota do generator (semântica UNVERIFIED até trace)
└── budget.rs         # cadência (4 ticks), teto por tick, contadores
crates/wyd-core/src/systems/npc.rs  # integração com o tick (ordem documentada)
crates/wyd-core/tests/integracao/npc.rs        # determinismo e golden de decisão
crates/wyd-core/benches/tick.rs     # custo da IA no orçamento do tick

crates/wyd-sim/src/bot/             # FASE-15-3, camada B (política pura)
├── mod.rs
├── persona.rs        # perfil: classe, nível, objetivo (dado)
└── policy.rs         # estado → ação, mesma FSM, testável sem rede

crates/wyd-bots/                    # crate nova, binário de teste/carga
├── src/lib.rs        # sessão (conecta, autentica, joga) via wyd-net
├── src/scenario.rs   # N bots × perfil × duração
└── src/bin/wyd-bots.rs  # CLI
```

Dependências (confirmadas no DAG de `estrutura-e-padroes.md:16`): `wyd-bots → {wyd-sim,
wyd-net, wyd-protocol}` e **nada depende de `wyd-bots`**; `wyd-core` consome `wyd-sim`
(ADR-0011), nunca o contrário. `crates/wyd-bots` fica fora do compose de produção
(`docs/rust/05-stack-docker.md`: só serviço entra).

## 8. Decisões provisórias (padrão `D-xx` de `matematica-sim.md`)

| ID | Decisão | Escolha | Por quê | Como confirmar |
|---|---|---|---|---|
| D-IA01 | Cadência de decisão | 5 Hz = 1 decisão a cada 4 ticks (20 Hz), `fase = entity_id % 4` | 5 Hz foi a proposta da ADR; a fase `% 10` daria 2 Hz e concentraria carga | bench com 500 NPCs: decisões/s e custo por tick |
| D-IA02 | Ameaça por dano | 1:1, teto 8 entradas, TTL em ticks, empate pelo menor `entity` | única proporção com respaldo no código (`threat.rs`), marcada provisória | trace do cliente: quem o mob persegue após vários atacantes |
| D-IA03 | Métrica de distância da percepção | a do `spatial/` (FASE-03-3), recebida pronta por `should_aggro` | a fórmula não decide a métrica; o grid já é a autoridade de vizinhança | comparar alcance de aggro com captura do cliente |
| D-IA04 | RNG | estado `u64` por NPC, semeado por `(seed, entity_id)`, salvo no snapshot | replay/religamento precisam da mesma sequência | golden: salvar → recarregar → mesmo fluxo |
| D-IA05 | LLM | fora do processo, no Cold Path, desligado por padrão, sem tabela até o dono decidir | latência/custo por token × tick de 50 ms; chave nunca no jogo | decisão do dono sobre provedor/custo/revisor |
| D-IA06 | Persistência do comportamento | conteúdo em tabela no boot; estado vivo só no snapshot | NPC não é linha de banco; salvar por NPC a 5 Hz violaria o Hot Path | FASE-04-3 (carga) + save/restore do snapshot |

## 9. Lacunas declaradas (o que o plano **não** resolve)

1. **Stats de monstro vazios** (`mob_template_stat`/`mob_template_equip`): a FASE-03-6 não
   depende mais delas para rodar — o perfil chega por `NpcProfile` (parâmetro do chamador,
   ADR-0016) —, mas sem os números do banco o perfil ainda é informado à mão.
2. **`route_type`/`generator_data` UNVERIFIED**: patrulha depende de semântica de rota do
   `NPCGener` que só o trace do cliente resolve; até lá, patrulha fica limitada a âncora.
3. **Tabelas de comportamento/diálogo/LLM não criadas**: pendência declarada (D-IA05, §5), sem
   migração especulativa.
4. **Golden de decisão existe para o fluxo de 24 ticks** (`tests/integracao/npc.rs`, hash FNV-1a): cobre
   percepção, alvo e intenção; o cenário de 100 ticks do §4 e o bench de custo continuam
   pendentes.
5. **Nenhum bot de carga existe**: `FASE-15-3` está aberta; o harness citado em
   `docs/manual-técnico/skills.md:25` é skill de teste, não código.
6. **LLM não tem provedor, orçamento nem revisor definidos** — decisão do dono, não do agente.

## 10. Como este plano é cobrado

- `docs/rust/07-ia-e-bots.md` §11 fixa a ordem: FASE-03-6 (core) → FASE-15-3 (bots) → LLM.
- Cada item da camada A nasce com teste de determinismo (golden) e bench antes de entrar no
  tick; sem bench, o "≤ 10% do orçamento" continua UNVERIFIED.
- Schema só muda quando houver código consultando o dado; até lá, `banco-catalogo.md` §3
  registra a pendência em vez de criar tabela órfã.
