# C4 + especificações de pacotes (FASE-11-2)

> **Gate de arquitetura** (FASE-11-2, 2026-11-07). Regra: **zero componente inventado** —
> cada caixa cita o código que a implementa (tabela de rastreabilidade no §5).
> Fonte de pacotes: `protocolo.md` §4–§6 (normativo de fio); este documento não duplica
> o catálogo, só ancora cada fluxo no módulo que o produz/consome.
> Convenções Rust vigentes: `AGENTS.md` §0–§3; árvore de módulos em
> `estrutura-e-padroes.md` (ADR-0002); stack em `docs/rust/05-stack-docker.md`.

## 1. Nível 1 — Contexto

```text
                    +------------------+
                    | Cliente WYD 7662 |  (fechado, fora do git; dossiê em docs/cliente/)
                    |  WYD.exe i386    |  envia ClientVersion 12000, INITCODE 0x1F11F311
                    +--------+---------+
                             | TCP 8281 (jogo) + HTTP 80 (status serv00.htm)
                             v
+------------+   +----------------------+   +------------------+
|  Jogador   +--->+ Servidor Rust (Hot   +-->+  PostgreSQL      |
|  (humano)  |   | Path monolítico)     |   |  (8 tabelas,     |
+------------+   | crates/wyd-*         |   |  delivery_queue)  |
                 +----------+-----------+   +------------------+
                            |
                            v
                 +----------------------+
                 | Portal Next.js (Cold  |
                 | Path, BFF sobre      |
                 | wyd-admin; nunca toca |
                 | estado vivo)         |
                 +----------------------+
```

Fora do caminho de jogo: `wyd-launcher` (valida/patcha `client/` via manifesto) e
`wyd-sim` (laboratório offline de fórmulas, sem I/O de jogo).

## 2. Nível 2 — Containers (o que sobe)

| Container | Código que o implementa | Sobe onde |
|---|---|---|
| Servidor de jogo (Hot Path) | workspace `crates/` (`Cargo.toml` members: `wyd-protocol`, `wyd-core`, `wyd-net`, `wyd-db`, `wyd-admin`, `wyd-sim`, `wyd-launcher`) | binário futuro da FASE-07 (hoje: testes + harness; `docker-compose.yml` reserva o nome `gateway:7600` p/ FASE-07-1) |
| Banco | `crates/wyd-db/migrations/` + `seeds/dev.sql` (modelo em `docs/arquitetura/modelo-banco.md`) | compose service `db` (Postgres + volume `pgdata`) |
| Portal | `wyd-portal/` (BFF; `src/server/admin.ts` → `adminCall`) | compose service `portal` (`:3000`) |
| Cliente | `client/` (WYD.exe 7662, fora do git) | máquina do jogador; `serverlist.bin` aponta p/ `IP:8281` |
| Launcher | `crates/wyd-launcher/src/main.rs` + `manifest.rs`/`patch.rs` | lado do jogador (`cargo run -p wyd-launcher -- check`) |

## 3. Nível 3 — Componentes críticos (só o que existe)

### 3.1 Tick / ECS / espacial (`wyd-core`)

| Componente | Código |
|---|---|
| Agendador de tick | `crates/wyd-core/src/tick/` (`Scheduler`, `MAX_HZ`, `MIN_HZ`) |
| ECS SoA | `crates/wyd-core/src/ecs/` |
| Grade espacial | `crates/wyd-core/src/spatial/` (`Grid`, `GridConfig`) |
| Portas p/ persistência (traits, sem SQL no tick) | `crates/wyd-core/src/ports/` |
| Pacote autoritativo de cena (`ESCENE_FIELD` = 30000) | `crates/wyd-core/src/lib.rs` (reexporta de `wyd-protocol`) |

### 3.2 Codec / wire (`wyd-protocol`, sem deps externas)

| Componente | Código |
|---|---|
| HEADER 12 bytes LE + política Strict/Tolerant | `crates/wyd-protocol/src/wire/header.rs`, `src/error.rs` |
| Tabela de opcodes congelada (198 `_MSG_*`) | `crates/wyd-protocol/src/wire/generated/opcode_table.rs` |
| Tabela `pKeyWord[512]` congelada | `crates/wyd-protocol/src/codec/generated/keyword_table.rs` |
| Transformação + checksum | `crates/wyd-protocol/src/codec/keyword.rs`, `src/codec/checksum.rs` |
| Payloads portados (auth, character, combat, commerce, inventory, movement, ui) | `crates/wyd-protocol/src/payload/` |
| Goldens de fio (5 sintéticos, nunca capturados) | `crates/wyd-protocol/tests/fixtures/cliente/*.json` via `tools/client/captura.mjs --check` |

### 3.3 Transporte / limites (`wyd-net`)

| Componente | Código |
|---|---|
| Rate-limit por conexão | `crates/wyd-net/src/limits/` (`TokenBucket`, `ConnectionGate`) |
| Teste de inundação | `crates/wyd-net/tests/integracao/flood.rs` |

> Transporte completo (framing, sessão, `8281/tcp`) é FASE-07 — não desenhado aqui
> para não inventar componente.

### 3.4 Persistência (`wyd-db`)

| Componente | Código |
|---|---|
| Modelo (Account, Character, Guild, …) | `crates/wyd-db/src/model/` |
| Repositórios async | `crates/wyd-db/src/repo/` |
| Pool dimensionado (fora do tick) | `crates/wyd-db/src/pool/` (`PoolConfig`) |
| Ponte web→jogo (única escrita externa) | `crates/wyd-db/src/transfer/` (`DeliveryQueue`, `QueuedGrant`) |

### 3.5 Control plane (`wyd-admin`, Cold Path)

| Componente | Código |
|---|---|
| API p/ o portal | `crates/wyd-admin/src/api/` |
| Sessão/papéis | `crates/wyd-admin/src/auth/` |
| Auditoria | `crates/wyd-admin/src/audit/` |
| Serviços (grants idempotentes) | `crates/wyd-admin/src/service/` |

## 4. Fluxos de pacotes (âncora, não transcrição)

O catálogo normativo está em `protocolo.md` §4 (opcodes), §5 (payloads portados) e
§6 (UNVERIFIED — não vira tipo). Fluxos ponta a ponta:

| Fluxo | Pacotes (Type) | Produzido / consumido em |
|---|---|---|
| Login C2G | `AccountLogin 0x020D` (ClientVersion 12000) | `payload/auth.rs` + goldens `crates/wyd-protocol/tests/fixtures/cliente/account_login.json` |
| Confirmação S2C | opcodes `0x010A`–`0x011D` (`CNFAccountLogin`…`StillPlaying`) | `wire/generated/opcode_table.rs` (`Opcode::CNF*`) |
| Entrada em campo | cena autoritativa `ESCENE_FIELD` (30000) | `wyd-core` (`lib.rs` reexport) |
| Comércio/combina | `0x02C3`–`0x02C4` (`CombineItemLindy/Shany`) e família | `payload/commerce.rs`, `payload/combat.rs` |
| Grants web→jogo | nunca pacote direto; via `delivery_queue` drenada no login | `wyd-db/src/transfer/`, `wyd-admin/src/service/` |

Diagnóstico de fio: `docs/cliente/pacotes.md` + `crates/wyd-net/tests/`.

## 5. Rastreabilidade (cada caixa → código)

Todas as caixas dos §1–§3 citam caminho existente acima; exceções declaradas como
futuras estão marcadas com a FASE dona (07). Regra de manutenção: qualquer mudança
de árvore exige ADR (rito FASE-13-1) e este documento muda na mesma entrega
(`docs/rust/02-execucao.md` §3).
