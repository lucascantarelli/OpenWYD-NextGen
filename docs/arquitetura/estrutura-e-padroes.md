# Estrutura de módulos, nomenclatura e padrões (plano normativo)

> **Regra de ouro:** nenhum arquivo novo entra nas `crates/` sem constar na árvore
> deste documento ou na issue que o autoriza. Mudança de árvore exige ADR
> (`docs/arquitetura/adr/`) — ver ADR-0002.
>
> Este documento existe porque a execução anterior criou módulos soltos na raiz da
> crate (`header.rs`, `payload.rs`, `opcodes.rs`…) sem plano de organização. O
> `wyd-protocol` foi **reorganizado na FASE-08-1** exatamente nesta árvore
> (movimento sem mudança de comportamento); nada novo é escrito fora dela.

## 1. Fronteiras entre crates (DAG, sem ciclos)

```
wyd-launcher ──► (nenhuma crate interna)
wyd-bots     ──► wyd-sim, wyd-net, wyd-protocol   # PLANEJADO (ADR-0010, FASE-15-3)
wyd-server   ──► wyd-core, wyd-net, wyd-protocol, wyd-db, wyd-admin   # raiz de composição (§16, ADR-0022)
wyd-admin    ──► wyd-db
wyd-db       ──► (independente)
wyd-sim      ──► wyd-protocol
wyd-net      ──► wyd-protocol
wyd-core     ──► wyd-protocol
wyd-protocol ──► (base, sem dependência interna)
```

Regras:

1. Dependência só aponta para baixo neste diagrama. `wyd-core` **nunca** depende
   de `wyd-db`/`wyd-net`: engine recebe portas (traits) e canais.
2. `wyd-protocol` não conhece regra de jogo: só bytes, layout e erro de parse.
3. Sem ciclo, sem feature flag para "furar" a direção.
4. Cold Path (admin/db) pode depender de contratos do core, nunca do estado vivo.
5. **Só `wyd-server` conhece todas as crates** (raiz de composição, ADR-0022 §1): ele
   monta o monólito de um canal. `wyd-core` continua sem saber o que é Postgres/socket e
   `wyd-db` sem saber o que é transporte; dependência nova entre crates fora deste DAG
   exige ADR.

## 2. Estrutura canônica de uma crate

```
crates/<crate>/
├── Cargo.toml
├── src/
│   ├── lib.rs          # fachada: re-exports, invariantes do crate em //!
│   ├── error.rs        # um enum de erro por crate (XxxError) — exceção: crate
│   │                   # pequena com fronteiras independentes pode ter um erro
│   │                   # por fronteira (ver `wyd-launcher` no §9)
│   ├── main.rs         # SÓ se a crate é binário: CLI fina (parse de args +
│   │                   # códigos de saída); lógica mora na biblioteca
│   ├── <dominio>/      # um diretório por domínio, não por "tipo de arquivo"
│   │   ├── mod.rs      # só declara e re-exporta; sem lógica
│   │   ├── <peca>.rs   # um conceito coeso por arquivo
│   │   └── generated/  # tabelas GERADAS (nunca editadas à mão)
│   └── ...
├── tests/              # integração (usa só a API pública)
│   ├── common/mod.rs   # fixtures compartilhadas
│   └── <fluxo>.rs
└── benches/            # benchmarks (criterion/tokio depois, via ADR)
```

Proibido: `utils.rs`, `helpers.rs`, `misc.rs`, `mod.rs` com lógica, arquivo com
dois domínios, tabela gerada na raiz de `src/`.

## 3. `wyd-protocol` — árvore alvo (FASE-08-1)

```
crates/wyd-protocol/
├── src/
│   ├── lib.rs                 # fachada + invariantes de fio
│   ├── error.rs               # ProtocolError (Header/Payload/Frame/Transform)
│   ├── wire/
│   │   ├── mod.rs
│   │   ├── header.rs          # HEADER 12 bytes (layout §1.1)
│   │   ├── frame.rs           # INITCODE, enquadramento por Size
│   │   └── opcode.rs          # Opcode + Flow + lookup (fonte: gerado)
│   │       └── generated/opcode_table.rs
│   ├── codec/
│   │   ├── mod.rs
│   │   ├── keyword.rs         # transform pKeyWord (semente/trans)
│   │   ├── checksum.rs        # soma claro × cifrado + política
│   │   └── generated/keyword_table.rs
│   └── payload/
│       ├── mod.rs             # trait WirePayload, WireField, offsets
│       ├── auth.rs            # AccountLogin, AccountSecure
│       ├── character.rs       # Create/Delete/CharacterLogin
│       ├── movement.rs        # Action
│       ├── combat.rs          # Attack<N> (13/1/2 alvos)
│       ├── scene.rs           # CreateMob, RemoveMob (entidades na vista — ADR-0025)
│       ├── inventory.rs       # UseItem, TradingItem, ApplyBonus
│       ├── commerce.rs        # Buy, Sell, Trade
│       └── ui.rs              # MessagePanel, UpdateWeather
├── tests/
│   ├── common/mod.rs          # monta pacote (header + corpo + ofuscação)
│   ├── integracao/roundtrip.rs # header+payload+codec pela API pública
│   └── golden/pacotes.rs      # bytes reais dos pacotes (fixtures do cliente 7662)
└── benches/
```

Layout UNVERIFIED (`MSG_Trade`, `MSG_SetHpMp`, `_AUTH_GAME`) fica fora de
`payload/` até captura real (FASE-08-2): sem tipo inventado.

## 4. Nomenclatura

| Elemento | Regra | Exemplo |
|---|---|---|
| Arquivo/diretório | `snake_case`, domínio no singular | `combat.rs`, `movement.rs` |
| Struct/enum/trait | `PascalCase`, substantivo do domínio | `AccountLogin`, `WirePayload` |
| Função/método | `snake_case`, verbo | `decode`, `encode_into`, `frame_len` |
| Constante | `SCREAMING_SNAKE_CASE` | `HEADER_SIZE`, `KEYWORD_OFFSET` |
| Erro | `XxxError` (enum) + variantes descritivas | `ProtocolError::BadLength` |
| Gerado | diretório `generated/` + cabeçalho "GERADO" | `opcode_table.rs` |
| Teste | `<comportamento>_<condição>` | `attack_round_trip_with_two_targets` |
| Byte de protocolo | nome do campo C em `snake_case` | `client_tick`, `skill_parm` |

Um conceito público por arquivo; campos de fio mantêm o nome do legado para
rastreabilidade (nada de "renomear bonito" e perder a ponte com `Basedef.h`).

## 5. Padrões adotados (e por quê)

| Padrão | Onde | Motivo |
|---|---|---|
| Facade | `lib.rs` de cada crate | API pública mínima e estável; módulos internos livres |
| Newtype | `Opcode`, `ConnectionId`, `Slot`, `Tick` | impede trocar unidades/tipos por engano |
| Enum + `match` exaustivo | dispatch de opcode, estados | sem `dyn` no Hot Path; compilador cobra caso novo |
| Typestate | sessão (`Connected → Handshaked → Authenticated`) | impossível processar login antes do INITCODE |
| Strategy | política de checksum (`Strict`/`Tolerant`) | decisão explícita, testável, sem `if` espalhado |
| Repository (porta) | `wyd-db` expõe traits; `wyd-core` consome | core não conhece SQL; testável com fake |
| State machine | fases do tick (input → simulação → saída) | ordem determinística, auditável |
| Builder | configuração de servidor/sim | default seguro + validação no `build()` |
| Adapter | `wyd-net` sobre `tokio`/TCP | transporte substituível sem tocar regra |

Não usados agora: `dyn` em caminho quente, DI framework, macros de DSL, OOP com
herança. Genéricos só quando o ganho é medido.

## 6. SOLID aplicado

| Princípio | Aplicação concreta |
|---|---|
| S — responsabilidade única | `wire::header` só lê/escreve 12 bytes; `codec` só transforma; `payload` só layout |
| O — aberto/fechado | novo opcode entra como variante + arquivo de domínio, sem `match` espalhado |
| L — substituição | traits de repositório/codec com contrato testado por fake |
| I — segregação | traits pequenas (`WirePayload`, `WireField`), nada de "God trait" |
| D — inversão | core depende de porta (`trait`), não de `sqlx`/`tokio` |

## 7. Testes — onde cada um mora

Taxonomia completa, regras de higiene, matriz por crate e auditoria caso a caso:
**`docs/arquitetura/testes.md`**. Resumo:

1. **Unidade** (`unidade`): `#[cfg(test)] mod tests` no arquivo que contém a regra;
   sem I/O, rede, banco, processo ou relógio real. Roda em todo `cargo test`.
2. **Integração** (`integracao`): `crates/<crate>/tests/integracao/<tema>.rs`, usando
   só a API pública; sem infraestrutura externa (nem Postgres, nem socket, nem processo).
3. **Contrato** (`contrato`): `tests/contrato/<tema>.rs` — documentação ↔ código,
   schema ↔ migrations, rota ↔ cliente TS, identidade ↔ assets. Falha se o contrato
   publicado divergir; não prova comportamento de jogo.
4. **Golden** (`golden`): `tests/golden/<tema>.rs` — bytes/valores exatos com a fonte
   citada no nome ou no comentário (`include_bytes!`, seed SQL, fixture).
5. **Sistema** (`sistema`): `tests/sistema/<tema>.rs` — Postgres real, socket CPSock,
   processo filho ou filesystem. Opt-in por variável de ambiente, com `SKIP` explícito.
6. **Bench**: `benches/` por crate crítica, comparável entre commits.

Pasta por tipo exige **alvo explícito**: o Cargo não descobre `tests/<subpasta>/*.rs`.
Cada arquivo tem um `[[test]]` no `Cargo.toml` (`name = "<tipo>_<tema>"`,
`path = "tests/<tipo>/<tema>.rs"`); o nome do alvo mantém o prefixo do tipo, então
`cargo test --test sistema_login` continua válido. `tests/common/mod.rs` NÃO é alvo de
teste (sem `fn main`), é o lugar de helpers compartilhados.

**Proibido**: teste que reimplementa a lógica para "provar" a si mesmo; teste sem
asserção de valor concreto; `#[ignore]` para esconder falha; `return;` silencioso como
gate (o skip precisa dizer qual variável falta); `sleep` fixo para sincronizar (poll
com prazo); asserção enfraquecida para "passar no CI".

## 8. Fluxo obrigatório antes de escrever código

1. A árvore do módulo já existe neste documento (ou numa issue).
2. Se for fronteira nova → ADR aceita primeiro (ADR-0002 para esta estrutura).
3. Listar arquivos criados/alterados na issue **antes** de criar.
4. Escrever o menor código que fecha o contrato, com teste no mesmo commit.
5. Atualizar `docs/rust/03-progresso.md` e a issue com evidência.

## 9. `wyd-launcher` — árvore executada (FASE-08-7/08-3)

```
crates/wyd-launcher/
├── Cargo.toml            # sem dependências externas (decisão offline, ADR-0003)
├── src/
│   ├── lib.rs            # Facade: re-exports + invariantes em //!
│   ├── main.rs           # binário: CLI fina (check/launch/update + exit codes)
│   ├── manifest.rs       # parse estrito + verify (ManifestError)
│   ├── patch.rs          # apply/rollback atômico (PatchError)
│   ├── bootstrap.rs      # plan/describe/spawn (LaunchError)
│   ├── sha256.rs         # digest em std puro (integridade, não anti-adversário)
│   ├── distribuicao.rs   # pacote + HMAC + resume + update (DistError)
│   ├── serverlist.rs     # serverlist.bin/txt da grade 10×11×64 (ServerlistError)
│   ├── snbin.rs          # sn.bin: nomes vindos de `identidade.tokens` (143 B medidos)
│   ├── identidade.rs     # parser da fonte única `docs/arquitetura/identidade.tokens`
│   ├── branding.rs       # Launcher.bmp/.hat gerados da identidade (escrita atômica)
│   └── gui.rs            # LauncherGUI.html (contrato visual da GUI Tauri, só da identidade)
└── tests/                # golden/serverlist_paridade.rs (bytes Rust × Node);
                          # contrato/identidade.rs (identidade × globals.css;
                          # assets instalados × gerados)
```

Sem subdirs — crate pequena. Um erro por fronteira (`ManifestError`,
`PatchError`, `LaunchError`, `DistError` com `From` entre eles) em vez de
`error.rs` único: exceção deliberada a §2, coberta pela skill
`solid-design-patterns` ("um erro por fronteira").

## 10. `wyd-core` — árvore alvo (FASE-03)

```
crates/wyd-core/
├── Cargo.toml            # deps ECS/arena só via ADR (ver 01-dependencias.md)
├── src/
│   ├── lib.rs            # fachada: Tick, TICK_HZ, invariantes Hot Path em //!
│   ├── error.rs          # CoreError
│   ├── ecs/
│   │   ├── mod.rs
│   │   ├── components.rs # Position, Stats, Inventory + spawn/despawn em lote (03-2) — dados, sem lógica
│   │   └── resources.rs  # recursos globais (tempo, seed) + DomainEvents (fila limitada)
│   ├── tick/
│   │   ├── mod.rs
│   │   └── scheduler.rs  # intervalo fixo, ordem documentada dos sistemas
│   ├── spatial/
│   │   ├── mod.rs
│   │   └── grid.rs       # visibilidade / interest management O(1)
│   ├── systems/
│   │   ├── mod.rs        # executado (03-5): só declara e re-exporta
│   │   ├── movement.rs   # executado (03-5): orçamento de velocidade + limites do mundo
│   │   ├── combat.rs     # executado (03-5): dano por `DamageModel` (ADR-0014), não o inventa
│   │   ├── skill.rs      # executado (03-5): skill aprendida, MP, cooldown e alcance
│   │   ├── inventory.rs  # executado (03-5): equipar/desequipar/mover (ADR-0015)
│   │   ├── visibility.rs
│   │   └── npc.rs        # executado (03-6): integra npc/ ao tick (ADR-0016)
│   ├── rules/            # fórmulas puras aplicadas (dano/EXP aprovados)
│   │   └── mod.rs
│   ├── world/
│   │   ├── mod.rs
│   │   ├── state.rs      # dono do estado do mundo (um por nó)
│   │   └── snapshot.rs   # save/load do estado
│   ├── npc/              # executado (03-6, ADR-0016) — IA determinística
│   │   ├── mod.rs
│   │   ├── state.rs      # NpcMode, NpcState/NpcStates SoA, ThreatTable, RNG semeado — dado
│   │   ├── behavior.rs   # FSM: idle→patrol→aggro→chase→attack→flee→return (precedência)
│   │   ├── brain.rs      # percepção + alvo + intenção inteira (NpcProfile por parâmetro)
│   │   ├── route.rs      # rota ancorada (route_type UNVERIFIED até o trace)
│   │   └── budget.rs     # cadência de decisão (5 Hz, tick % 4) e teto por tick
│   └── ports/
│       ├── mod.rs
│       ├── persistence.rs # trait: o core pede, wyd-db entrega (DIP)
│       ├── content.rs     # trait: tabelas de conteúdo carregadas do banco (ADR-0009)
│       └── clock.rs       # trait: tempo injetável (determinismo em teste)
├── tests/
│   ├── common/mod.rs
│   ├── integracao/content.rs    # conteúdo inconsistente aborta o boot (ADR-0009)
│   ├── integracao/determinism.rs # mesmo seed → mesmo resultado
│   ├── integracao/autoridade.rs / integracao/inventario.rs / integracao/npc.rs
│   ├── integracao/compose.rs / integracao/gem.rs
│   ├── golden/gema_seed.rs      # efeito de gema × seed SQL
│   └── sistema/telemetry.rs     # tracing/métricas com relógio e stdout reais
└── benches/
    └── tick.rs           # executado (03-6 + 03-2): custo da IA (500 NPCs) e baseline SoA (10k entidades)
```

## 11. `wyd-net` — árvore alvo (FASE-08)

```
crates/wyd-net/
├── Cargo.toml            # tokio/zerocopy só via ADR
├── src/
│   ├── lib.rs            # fachada + MAX_PACKET_BYTES
│   ├── error.rs          # NetError (tamanho → opcode → payload)
│   ├── session/
│   │   ├── mod.rs
│   │   ├── state.rs      # typestate Connected → Handshaked → Authenticated
│   │   └── handshake.rs  # INITCODE + rate-limit (com FASE-06)
│   ├── transport/
│   │   ├── mod.rs
│   │   ├── tcp.rs        # Adapter sobre o socket (substituível)
│   │   └── framing.rs    # corte por Size, zero-copy
│   ├── dispatch/
│   │   ├── mod.rs
│   │   └── router.rs     # opcode → handler (match exaustivo)
│   ├── limits/
│   │   ├── mod.rs
│   │   └── rate.rs       # tetos por conexão/IP, backpressure
│   └── telemetry/
│       ├── mod.rs
│       └── metrics.rs    # pacotes/s, filas, rejeições (sem println!)
└── tests/
    ├── common/mod.rs
    ├── integracao/flood.rs       # truncado/desconhecido/limite → erro tipado e rate-limit
    └── integracao/metrics.rs     # contadores da feature `metrics`
```

## 12. `wyd-db` — árvore alvo (FASE-04)

```
crates/wyd-db/
├── Cargo.toml            # sqlx/sha2 só via ADR (0021/0024)
├── migrations/
│   └── NNNN_descricao.sql # versionadas, reversíveis (up/down testado)
├── seeds/                # PREVISTO: dev.sql versionado; prod nunca versiona segredo
├── src/
│   ├── lib.rs            # fachada (scheme, versão do schema)
│   ├── error.rs          # DbError (sem DSN/segredo em Display)
│   ├── migrate.rs        # executado (ADR-0024): série no boot, ledger e advisory lock
│   ├── model/
│   │   ├── mod.rs
│   │   ├── account.rs
│   │   ├── character.rs
│   │   ├── guild.rs
│   │   ├── content.rs    # item/skill/npc/level/settings (ADR-0009)
│   │   └── audit.rs
│   ├── repo/             # traits (portas) + implementação pg
│   │   ├── mod.rs
│   │   ├── accounts.rs   # porta: create_account/find_by_login/find_by_id + moderação (ADR-0023)
│   │   ├── characters.rs # porta: list/create_character/top_by_level/soft_delete
│   │   ├── grants.rs     # leitura de delivery_queue (Cold Path)
│   │   ├── content.rs    # carga única no boot + validação (ADR-0009)
│   │   ├── audit.rs      # trilha append-only: record/recent (ADR-0023)
│   │   ├── economy.rs    # imposto + concessões com fonte real (ADR-0023)
│   │   └── pg.rs         # executado (04-3): impl das portas sobre sqlx
│   ├── pool/
│   │   ├── mod.rs
│   │   ├── config.rs     # pool dimensionado, timeouts, transações curtas
│   │   └── pg.rs         # executado (04-3, ADR-0021): pool sqlx + mapeamento de erro
│   └── transfer/
│       ├── mod.rs
│       ├── delivery_queue.rs # grants do portal → dreno do servidor
│       └── save.rs       # executado (04-3): fila de save, política e dreno Cold Path
└── tests/
    ├── contrato/catalogo.rs    # doc ↔ migrations ↔ SCHEMA_VERSION (sem Postgres)
    ├── contrato/migrations.rs  # estrutura das migrations (sem Postgres)
    ├── sistema/pg.rs           # portas + save contra Postgres real (opt-in)
    ├── sistema/sessoes.rs      # presença por canal (opt-in)
    └── sistema/stack_caps.rs   # view item_stack_cap (opt-in)
```

Nenhum acesso a banco dentro do tick; DSN real nunca versionado.

## 13. `wyd-admin` — árvore alvo (FASE-07)

```
crates/wyd-admin/
├── Cargo.toml
├── src/
│   ├── lib.rs            # fachada + ADMIN_SCOPE
│   ├── error.rs          # AdminError (401/403/422 por variante)
│   ├── api/              # endpoints por domínio (nunca estado vivo)
│   │   ├── mod.rs        # só declara e reexporta; cada domínio no seu módulo
│   │   ├── accounts.rs
│   │   ├── catalog.rs    # `list_items` da foto do banco (ADR-0022)
│   │   ├── drop.rs       # pesos de drop: leitura sem auditoria, escrita auditada
│   │   ├── economy.rs    # estatísticas com fonte auditada (ADR-0023)
│   │   ├── grants.rs     # intenções → delivery_queue (idempotente)
│   │   ├── item_effect.rs # `list_effects`: símbolo EF_*, sem código inventado (07-5)
│   │   ├── mob.rs        # `list_mob_templates` de `npc_template` (07-5)
│   │   ├── moderation.rs # papel/bloqueio + trilha (ADR-0023)
│   │   ├── npc.rs        # `get_npc`/`set_npc` + `list_npcs` paginado (07-5)
│   │   ├── panel.rs      # painel do jogador (conta, personagens, ranking)
│   │   ├── presence.rs   # presença por canal (só leitura)
│   │   ├── rates.rs      # `ConfigStore` (porta) + taxas/editáveis
│   │   └── teleport.rs   # `list_routes` + política de rota
│   ├── transport/        # Cold Path: HTTP/1.1 + JSON em loopback (ADR-0008)
│   │   ├── mod.rs        # roteador servico/metodo → api/*, envelope + trace_id
│   │   ├── http.rs       # HTTP/1.1 mínimo em std (limite de corpo, close)
│   │   └── json.rs       # JSON do subconjunto do contrato (escape/números)
│   ├── auth/
│   │   ├── mod.rs
│   │   ├── session.rs    # sessão de staff (moderator_id da sessão!)
│   │   └── roles.rs
│   ├── audit/
│   │   ├── mod.rs
│   │   └── log.rs        # quem/o quê/quando em toda ação
│   └── service/
│       ├── mod.rs
│       └── grant_service.rs # regra + idempotência por transação rastreável
└── tests/
    ├── contrato/autorizacao.rs # sem sessão → 401; sem papel → 403
    ├── contrato/transporte.rs  # rota→método, envelope AdminResult, limites (ADR-0008)
    ├── contrato/conteudo.rs    # 07-5: listas paginadas (fake + Postgres semeado)
    ├── contrato/painel.rs      # painel do jogador
    ├── contrato/login.rs       # credenciais/pin pelo Cold Path
    ├── contrato/operacao.rs    # operação diária/donate/eventos
    └── sistema/seed.rs         # regressão do seed de staff em boot real (opt-in)
```

Transporte decidido na **ADR-0008** (HTTP/1.1 + JSON em loopback); `.proto` não
entra na árvore.

## 14. `wyd-sim` — árvore alvo (FASE-10, ADR-0011)

Um arquivo por **sistema de jogo**, não por "tipo de arquivo": a árvore cai da matemática
(`docs/arquitetura/matematica-sim.md` §2), não o contrário.

```
crates/wyd-sim/
├── Cargo.toml            # deps numéricas só via ADR (hoje: nenhuma)
├── src/
│   ├── lib.rs            # fachada: declara e reexporta; sem regra
│   ├── error.rs          # SimError (um erro por fronteira)
│   ├── units.rs          # Bps, Tenths, mul_bps — truncamento único, sem f64
│   ├── roll/             # sorteio determinístico (fronteira, não fórmula)
│   │   ├── mod.rs        # trait Die + escala de chance
│   │   └── xorshift64.rs # XorShift64 (seed explícita, estado auditável)
│   ├── formula/          # funções puras: fato × hipótese separados
│   │   ├── mod.rs        # só declara/reexporta
│   │   ├── attributes.rs # derivados: nível → atributo → equipamento
│   │   ├── progression.rs# pontos, tetos de maestria, degraus, ExpTable
│   │   ├── damage.rs     # base→skill→buff→crítico→defesa/perfuração→resist→absorção
│   │   ├── hit.rs        # acerto, evasão, crítico
│   │   ├── effect.rs     # efeitos com duração em ticks
│   │   ├── resist.rs     # resistência elemental
│   │   ├── drop.rs       # roll por item (1..1.000.000)
│   │   ├── regen.rs      # HP/MP por tick com resto acumulado
│   │   ├── threat.rs     # aggro e estado de criatura (matemática da IA)
│   │   ├── refine.rs     # +0..+15 em bps, composição +10 e gemas de efeito
│   │   ├── speed.rs      # ⛔ BLOQUEADO até FASE-10-2 (anomalia ~200% sem causa)
│   │   └── economia.rs   # ⬜ planejado (imposto, preço, drop de gold)
│   ├── scenario/         # ordem do tick e do golpe
│   │   ├── mod.rs
│   │   └── combat.rs     # TICK_PHASES + resolve_attack (acerto→crítico→dano)
│   ├── bot/              # PLANEJADO (ADR-0010, FASE-15-3) — política de bot
│   │   ├── mod.rs
│   │   ├── persona.rs    # perfil (classe, nível, objetivo) — dado, sem lógica
│   │   └── policy.rs     # decisão pura: estado → ação (mesma FSM do core)
│   └── report/
│       ├── mod.rs
│       └── table.rs      # saída comparável (atual × proposto)
└── tests/
    ├── golden/formulas.rs    # regressões matemáticas de valor exato (inalteradas)
    ├── golden/refino_seed.rs # refino × seed SQL
    └── integracao/determinism.rs # mesma seed → mesma partida; replay com rolls gravados
```

Mapa de sistemas, unidades, ordem no tick e decisões provisórias:
**`docs/arquitetura/matematica-sim.md`**. Inventário curto e dívidas:
**`docs/rust/06-matematica.md`**.

Regras que a árvore carrega:

1. Fórmula sem prova fica ⛔ no plano; `scenario/` pode explorar hipótese, `formula/` não.
2. Nenhuma lógica em `lib.rs`/`mod.rs`/`main.rs`; fórmula nova é módulo novo (OCP).
3. Percentual é `Bps`, decimal de item é `Tenths`, duração é tick inteiro; `f64` proibido em
   `formula/` (relatório é texto).
4. Sortear é papel de `roll::Die`; fórmula que consome `rand` ou relógio reprova revisão.
5. `bot/` e `formula/speed.rs` são **reservados**: existem no plano, entram por issue própria.

Equações documentadas fora do código, no manual técnico (`docs/`).

## 15. `tools/` e raiz — árvore fechada

```
tools/
├── client/               # pipeline de análise estática do cliente (Node)
│   ├── README.md         # ordem do pipeline + como reproduzir
│   ├── pe.mjs            # seções/imports/entry (08-5)
│   ├── strings.mjs       # indexação de strings
│   ├── analyse.mjs       # relatório consolidado
│   ├── evidence.mjs      # prova por âncora (endereço+bytes)
│   ├── formats.mjs       # .wyt/.msa/sn.bin (parcial)
│   ├── keytable.mjs      # tabela de keywords
│   └── captura.mjs       # --gen/--check de capturas (08-6)
├── protocol/
│   ├── gen-protocol.mjs  # tabelas a partir do C++ arquivado
│   └── README.md         # PREVISTO: fonte canônica + reprodução
├── content/
│   ├── gen-seed.mjs      # legado (origin/main) → migração SQL de seed (ADR-0009)
│   └── README.md         # comando + sha256 do insumo; nada vai para runtime
└── dev/                  # placeholder: só .gitkeep até ter propósito declarado
```

**Nenhum dado de jogo é lido de arquivo em runtime** (ADR-0009): a fonte é o PostgreSQL,
carregado por `wyd-db` no boot e validado antes do primeiro tick (`ports/content.rs`).
`tools/content/` só converte o legado (`tmserver/content/**`, `internal/migrations/seed_*`)
em SQL de migração — o arquivo é insumo do gerador e base de conhecimento, não fonte.

Regra: `tools/` é análise/geração, nunca runtime; script novo entra no
`README.md` do seu diretório. Raiz segue `visao-pastas.md` §1 (só manifestos e
contratos) — referência normativa, sem duplicação aqui. Diretório vazio sem
dono continua proibido: `.gitkeep` só via issue.

## 16. `wyd-server` — árvore executada (ADR-0022, FASE-03/07/08)

> **ADR-0026 implementada (2026-11-11).** `boot/`, `auth/`, `cpsock/`, `world/`,
> `character/`, `gameplay/`, `config/` e `status/` são diretórios com `mod.rs` de fachada;
> a raiz fica só com `channels.rs`, `metrics.rs` e `logs.rs`. Quando o domínio ainda não
> teve consumidor que justifique subdivisão fina, as regras ficam em `service.rs`.

Raiz de composição do monólito **de um canal**. Não é "mais uma lib": é o único lugar que
pode conhecer `wyd-core` + `wyd-net` + `wyd-db` + `wyd-admin` ao mesmo tempo. Lógica de jogo
continua no `wyd-core`; regra de persistência no `wyd-db`; este crate só **liga** as peças e
expõe as superfícies de processo.

```
crates/wyd-server/
├── Cargo.toml            # deps: protocol, core, net (feature metrics), db, admin, tokio (borda)
├── src/
│   ├── lib.rs            # fachada: invariantes do processo em //!
│   ├── main.rs           # só CLI: chama `boot::main()`
│   ├── boot/
│   │   ├── mod.rs        # run(): env → migrations → pool → listeners → loops
│   │   ├── env.rs        # binds e defaults de escuta
│   │   └── healthcheck.rs# `--healthcheck` (GET /health)
│   ├── cpsock/
│   │   ├── mod.rs        # accept loop, `Stage`/`Scene`, respostas e testes de fio
│   │   ├── handshake.rs  # INITCODE, frame, checksum e versão do cliente
│   │   ├── login.rs      # senha/PIN e cerca de sessão
│   │   ├── selection.rs  # seleção e Create/Delete/CharacterLogin
│   │   ├── action.rs     # dispatch de UseItem/Attack/Action/chat e cena
│   │   └── limits.rs     # limite de conexões por IP
│   ├── auth/
│   │   ├── mod.rs        # Authenticator/AuthError/PinOutcome (fachada do domínio)
│   │   ├── password.rs   # senha de login (Argon2id)
│   │   └── pin.rs        # PIN 4..=6, cadastro/troca e bloqueio persistido
│   ├── character/
│   │   ├── mod.rs        # fachada do domínio
│   │   ├── crud.rs       # Create/Delete/CharacterLogin com autoridade no Postgres
│   │   ├── snapshot.rs   # STRUCT_SELCHAR/STRUCT_MOB a partir do banco
│   │   └── session.rs    # sessão única por conta entre canais (0030)
│   ├── world/
│   │   ├── mod.rs        # World/Engine, tipos, comandos e porta `EngineHandle`
│   │   ├── tick.rs       # EngineThread/run_loop, movimento e avanço do relógio
│   │   ├── scene.rs      # entrada/saída, cena, ids `MobID`/`ESCENE_FIELD`
│   │   ├── npc.rs        # `npc_specs`/`sync_npcs` e diálogo
│   │   ├── combat.rs     # golpe autoritativo e dano por template
│   │   └── save.rs       # `SaveSink`, `SaveQueueSink` e snapshots
│   ├── channels.rs       # World: canais habilitados e papéis (normal/verde)
│   ├── status/
│   │   ├── mod.rs        # fachada do domínio
│   │   └── service.rs    # `/serv00.htm` + `/health` + sonda UDP :8281
│   ├── config/
│   │   ├── mod.rs        # fachada do domínio
│   │   └── service.rs    # snapshot/revisão/carga fora do tick; falha mantém o último
│   │                     # snapshot bom; NPC carrega o STRUCT_MOB do `npc_template`
│   ├── gameplay/
│   │   ├── mod.rs        # fachada do domínio
│   │   └── service.rs    # refino/loot/EXP/quest/portões/inventário; lê o snapshot na
│   │                     # chamada; `use_item` é o handler do `_MSG_UseItem`; loot no abate
│   ├── logs.rs           # log estruturado do processo (event=...)
│   └── metrics.rs        # métricas do processo (uptime, canais, status, IP-limit)
└── tests/                  # prefixo = tipo (docs/arquitetura/testes.md)
    ├── contrato/monitoramento.rs # doc ↔ rotas/consultas do /monitoramento
    ├── integracao/canal_verde.rs / integracao/config_gameplay.rs
    ├── integracao/config_recarga.rs / integracao/tick_processo.rs
    ├── sistema/handshake.rs  # INITCODE, versão, checksum (inclusive servidor vivo)
    ├── sistema/login.rs      # senha, PIN, bloqueio, sem segredo no log
    ├── sistema/pin_persistencia.rs / sistema/pin_troca.rs / sistema/portal_pin.rs
    ├── sistema/personagens_crud.rs / sistema/selchar.rs / sistema/selecao.rs
    ├── sistema/sessao_unica.rs / sistema/limite_ip.rs / sistema/status_udp.rs
    ├── integracao/campo_treinamento.rs / sistema/campo_vivo.rs
    ├── sistema_cena_*.rs / sistema/gold_inicial.rs
    └── sistema/loja_viva.rs / sistema/lotacao.rs / sistema/conta_de_teste.rs
```

**EXP ligada ao fio (2026-11-10).** `_MSG_Attack`/`AttackOne`/`AttackTwo` chegam por
`cpsock::dispatch` → `SelectionStore::attack` → `Engine::attack_at` (dano do servidor por
`wyd_core::systems::combat`) e, no abate, `gain_exp` → `Gameplay::exp_reward` (multiplicador
`exp` do snapshot). `_MSG_Action` abre diálogo com o `display_name` de `npc_definition`
(`MessagePanel`). NPCs nascem de `npc_specs(&GameConfig, mapa)` e são reconciliados a cada
revisão. O que **continua** sem resolvedor é caixa/pickup e o sorteio de loot no abate.

**Loot sem call site (2026-11-10).** `Gameplay::loot` só aparece em
`crates/wyd-server/tests/integracao/config_gameplay.rs`; nenhum caminho de produção o chama. Inspeção
nominal do que **não** resolve morte/caixa/pickup:

- `wyd_sim::resolve_attack` / `AttackOutcome` — só `hit`/`crit`/`damage`; não recebe HP e não
  devolve morte nem callback de loot.
- `wyd_sim::TICK_PHASES` — `"deaths"` é um rótulo da ordem do tick, não uma função.
- `wyd_sim::formula::drop::{drops, effective_rate, gem_bonus}` — `drops(rate, roll)` é só o
  predicado `roll <= rate` por item; não conhece monstro, tabela, kill ou caixa.
- `wyd_sim::formula::accessory::attempt` — refino de acessório, não abertura de caixa.
- `wyd_sim::formula::progression::{grant_exp, exp_to_next}` — EXP, não loot.
- `wyd_core::systems::combat::attack` — devolve `Hit { remaining_hp, .. }`, sem evento de
  morte e sem tabela de drop.
- `wyd_core::systems::npc::decide` — intenção de movimento/ataque de NPC, não diálogo, gift
  ou pickup.
- `wyd_server::world::{Command, Engine::step}` — comandos `Enter`/`Leave`/`Snapshots`,
  `SyncNpcs`/`Attack`/`Talk`; o `step` só drena eventos, não resolve morte/loot.
- `cpsock::dispatch` — trata `CreateCharacter`, `DeleteCharacter`, `CharacterLogin`,
  `CharacterLogout`, `UseItem`, `ApplyBonus`, `Attack*` e `Action`; caixa, pickup e teleporte
  caem em `outro` tolerado. O abate credita EXP (`gain_exp` → `exp_reward`) mas **não**
  rola loot: a tabela de drop continua sem call site de produção.

Código legado C++/Go não está na árvore compilada; `origin/main` é referência histórica.

Regras que esta árvore carrega:

1. **Sem regra de jogo aqui.** `main.rs`/`cpsock/` chamam sistemas do core; não calculam
   dano, drop, cooldown nem inventário.
2. **Um processo por canal** (ADR-0022 §7): canal vem de `WYD_SERVER_CHANNEL` + `server_channel`,
   nunca de build nem de porta nova (o cliente fixa 8281).
3. **Sem I/O no tick.** Listeners e banco vivem em threads/tokio próprios; o tick (FASE-03-7)
   recebe intenção por fila e devolve evento por fila.
4. **Migrations no boot** (ADR-0024) antes do pool; DDL nunca no tick.
5. **Teste que exige banco pula com aviso** quando `WYD_TEST_DATABASE_URL`/`DATABASE_URL`
   não existe — `cargo test --workspace` tem de passar em máquina limpa.

## 17. `wyd-bots` — árvore executada (ADR-0027, FASE-15-4)

Cliente de teste/QA que fala o protocolo real do canal; não é runtime de servidor. A
fachada pública (`Bot`, `BotConfig`, `Race`, `SceneMob`, `parse_etapa`, `nome_do_campo` e
`tela::*`) é consumida pelo `wyd-launcher` e pelos testes de sistema do `wyd-server`.

```
crates/wyd-bots/
├── Cargo.toml
└── src/
    ├── lib.rs              # fachada: mod bot + pub mod tela + re-exports + testes
    ├── main.rs             # CLI: BotConfig por raça e execução da campanha
    ├── bot/
    │   ├── mod.rs          # socket, handshake, autenticação, seleção e helpers de frame
    │   ├── config.rs       # Race (classes 0–3) + BotConfig (conta/slot/treinadores)
    │   ├── cena.rs         # SceneMob + nome_do_campo (CreateMob)
    │   ├── campanha.rs     # campaign + quest + parse_etapa
    │   ├── comercio.rs     # loja, compra e SendItem/Coin
    │   ├── combate.rs      # Attack/AttackOne/AttackTwo e kill
    │   ├── movimento.rs    # Action, walk_to e open_gate
    │   └── recusa.rs       # motivo_da_recusa (texto acionável ao jogador)
    └── tela/
        └── mod.rs          # tela de conexão (renderização e entrada)
```

Regras:

1. `lib.rs` só declara/re-exporta; testes de unidade ficam nele.
2. Cada arquivo de `bot/` contém apenas os métodos de `impl Bot` do seu domínio; o header
   de imports compartilhados fica em `bot/mod.rs` como `pub(crate) use`.
3. `tela/` é UI de conexão, não protocolo de jogo; `main.rs` e os consumidores externos
   continuam usando `wyd_bots::tela::*`.
4. Nenhuma regra de servidor mora aqui: o bot apenas envia pacotes e lê a cena/painéis.
