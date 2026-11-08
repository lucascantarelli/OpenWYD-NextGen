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
│       ├── inventory.rs       # UseItem, TradingItem, ApplyBonus
│       ├── commerce.rs        # Buy, Sell, Trade
│       └── ui.rs              # MessagePanel, UpdateWeather
├── tests/
│   ├── common/mod.rs          # monta pacote (header + corpo + ofuscação)
│   └── roundtrip.rs           # integração: header+payload+codec
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

1. **Unitário**: `#[cfg(test)] mod tests` no fim do arquivo que contém a regra.
2. **Integração**: `crates/<crate>/tests/*.rs`, usando apenas a API pública.
3. **Golden byte-exato**: `tests/` com fixtures em `tests/fixtures/` (bytes reais).
4. **Bench**: `benches/` por crate crítica, comparável entre commits.
5. **Proibido**: teste que reimplementa a lógica para "provar" a si mesmo; teste
   sem asserção de valor concreto; `#[ignore]` para esconder falha.

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
│   └── distribuicao.rs   # pacote + HMAC + resume + update (DistError)
└── tests/                # PREVISTO (integração da CLI; hoje testes em cada módulo)
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
│   │   ├── components.rs # Position, Stats, Inventory — dados, sem lógica
│   │   └── resources.rs  # recursos globais (tempo, seed de RNG)
│   ├── tick/
│   │   ├── mod.rs
│   │   └── scheduler.rs  # intervalo fixo, ordem documentada dos sistemas
│   ├── spatial/
│   │   ├── mod.rs
│   │   └── grid.rs       # visibilidade / interest management O(1)
│   ├── systems/
│   │   ├── mod.rs
│   │   ├── movement.rs
│   │   ├── combat.rs     # aplica fórmula aprovada (FASE-10), não a inventa
│   │   └── visibility.rs
│   ├── rules/            # fórmulas puras aplicadas (dano/EXP aprovados)
│   │   └── mod.rs
│   ├── world/
│   │   ├── mod.rs
│   │   ├── state.rs      # dono do estado do mundo (um por nó)
│   │   └── snapshot.rs   # save/load do estado
│   └── ports/
│       ├── mod.rs
│       ├── persistence.rs # trait: o core pede, wyd-db entrega (DIP)
│       ├── content.rs     # trait: tabelas de conteúdo carregadas do banco (ADR-0009)
│       └── clock.rs       # trait: tempo injetável (determinismo em teste)
├── tests/
│   ├── common/mod.rs
│   ├── content.rs        # conteúdo inconsistente aborta o boot (ADR-0009)
│   └── determinism.rs    # mesmo seed → mesmo resultado
└── benches/
    └── tick.rs           # regressão de performance do loop
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
    └── reject_malformed.rs # truncado/desconhecido/limite → erro tipado
```

## 12. `wyd-db` — árvore alvo (FASE-04)

```
crates/wyd-db/
├── Cargo.toml            # sqlx/refinery só via ADR
├── migrations/
│   └── NNNN_descricao.sql # versionadas, reversíveis (up/down testado)
├── seeds/                # PREVISTO: dev.sql versionado; prod nunca versiona segredo
├── src/
│   ├── lib.rs            # fachada (scheme, versão do schema)
│   ├── error.rs          # DbError (sem DSN/segredo em Display)
│   ├── model/
│   │   ├── mod.rs
│   │   ├── account.rs
│   │   ├── character.rs
│   │   ├── guild.rs
│   │   ├── content.rs    # item/skill/npc/level/settings (ADR-0009)
│   │   └── audit.rs
│   ├── repo/             # traits (portas) + implementação pg
│   │   ├── mod.rs
│   │   ├── accounts.rs
│   │   ├── characters.rs
│   │   ├── grants.rs     # leitura de delivery_queue (Cold Path)
│   │   └── content.rs    # carga única no boot + validação (ADR-0009)
│   ├── pool/
│   │   ├── mod.rs
│   │   └── config.rs     # pool dimensionado, timeouts, transações curtas
│   └── transfer/
│       ├── mod.rs
│       └── delivery_queue.rs # grants do portal → dreno do servidor
└── tests/
    └── constraints.rs    # contra Postgres descartável: FK violada falha
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
│   │   ├── mod.rs
│   │   ├── accounts.rs
│   │   └── grants.rs     # intenções → delivery_queue (idempotente)
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
    ├── unauthorized.rs   # sem sessão → 401; sem papel → 403
    └── contract.rs       # rota→método, envelope AdminResult, limites (ADR-0008)
```

Transporte decidido na **ADR-0008** (HTTP/1.1 + JSON em loopback); `.proto` não
entra na árvore.

## 14. `wyd-sim` — árvore alvo (FASE-10)

```
crates/wyd-sim/
├── Cargo.toml            # deps numéricas só via ADR
├── src/
│   ├── lib.rs            # fachada (seed explícita, determinismo)
│   ├── formula/          # funções puras: fato × hipótese separados
│   │   ├── mod.rs
│   │   ├── damage.rs     # ordem: base→atrib→equip→skill→buff→crit→defesa
│   │   ├── speed.rs      # inclui a anomalia ~200% (cenário reproduzível)
│   │   └── crit.rs
│   ├── scenario/         # cenários numerados (tabelas velocidade×hits…)
│   │   └── mod.rs
│   └── report/
│       ├── mod.rs
│       └── table.rs      # saída comparável (atual × proposto)
└── tests/
    └── goldens.rs        # regressões matemáticas de valor exato
```

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
