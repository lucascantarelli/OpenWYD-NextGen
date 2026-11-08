# Issue 09-2 — Criar/atualizar skills sucessoras Rust ✅ CONCLUÍDA

**Fase:** 9 · **Depende de:** 09-1 ✅ · **Bloqueia:** — · **Esforço:** M · **Status:** ✅ concluída (2026-11-07)

## Objetivo

Publicar as skills sucessoras (padrão frontmatter + regras numeradas) e anotar
as Go com sucessão, sem instruções conflitantes.

## O que foi executado

1. **Criada `wyd-rust-testing`** (sucessora Rust de harness/bots): frontmatter + 5 regras
   numeradas (codec real Strict, goldens, determinismo, carga, higiene), comandos e
   proibições aderentes a `crates/wyd-protocol`, `wyd-core`, `wyd-sim`.
2. **Atualizadas para Rust (mesmo nome, corpo Rust):** `wyd-database` (modelo-banco +
   `wyd-db`/sqlx, delivery_queue), `wyd-client-connect` (handshake 7662/12000, CPSock,
   launcher), `wyd-game-admin` (`wyd-admin` + delivery_queue auditada), `wyd-economy`
   (`wyd-db` + `wyd-sim`, validação por simulação), `wyd-web-admin-ui` (BFF sobre
   `wyd-admin`, `adminCall`, sem Go).
3. **`wyd-testing-bots`** convertida em alias de transição → `wyd-rust-testing`.
4. **`wyd-server-dev`** aposentada: corpo Go removido, stub com sucessoras por domínio
   (`wyd-rust-core`/`-net`/`-db`) e conhecimento atemporal preservado.
5. **Refs obsoletas corrigidas:** `wyd-rust-db` e `wyd-rust-obs` → `docs/arquitetura/modelo-banco.md`;
   `wyd-client-reverso` sem `Source/`; `tools/client/keytable.mjs` →
   `codec/generated/keyword_table.rs`; `tools/client/captura.mjs` →
   `wire/generated/opcode_table.rs` (goldens voltaram a passar 5/5).
6. **Mapas atualizados:** `AGENTS.md` §0 (vigentes + `wyd-rust-testing`, ex-Go resolvidas)
   e §6 (linha `wyd-rust-testing`); `docs/manual-técnico/skills.md` (destinos 09-2 ✅).

## Evidência (passe 2026-11-07)

```text
cargo fmt --all -- --check                                  → FMT-OK
cargo clippy --workspace --all-targets -- -D warnings       → 0 warnings
cargo test --workspace                                     → 20 suites ok, 0 failed
npm --prefix wyd-portal run test -- --run                  → 16 files, 118/118 passed
npm --prefix wyd-portal run typecheck                      → 0 erros
docker compose -f docker-compose.yml -f docker-compose.prod.yml config --quiet → COMPOSE-OK
node tools/protocol/gen-protocol.mjs                       → congelado, nada a regenerar
node tools/client/captura.mjs --check                      → goldens 5/5 válidos
grep tmserver|Source/Code|.deprecated-go|docs/migration em todas as skills → SKILLS-LIMPAS
```

## Critérios de aceite

- [x] Plano 09-1 100% executado; grep limpo; evidência anexada.

## Notas

- Desvios do plano original registrados: `wyd-database`, `wyd-client-connect`,
  `wyd-game-admin`, `wyd-economy`, `wyd-web-admin-ui` atualizadas no mesmo nome em vez
  de renomeadas (domínio continua válido; renomear quebraria referências); `wyd-server-ops`
  já-Rust mantida como vigente (sem duplicata `wyd-rust-ops`); `grpc.ts` do portal tornado
  lazy para proto ausente não quebrar testes.
- Próxima de menor esforço: **FASE-11-2** (guia C4 de pacotes — documentação, sem código).
