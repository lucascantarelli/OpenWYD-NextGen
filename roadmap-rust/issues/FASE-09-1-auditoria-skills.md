# Issue 09-1 — Auditoria profunda das skills Go restantes

**Fase:** 9 · **Depende de:** 01-3 · **Bloqueia:** 09-2 · **Esforço:** S

## Objetivo

Auditar cada skill Go restante (responsabilidade, sobreposição, trechos que
citam código arquivado) e decidir o destino com motivo.

## Contexto / evidência

`pre-release-rust.md` FASE 9: consolidar para a pilha Rust. Insumo: tabela 01-3.
Alvo: `wyd-testing-bots`, `wyd-server-ops`, `wyd-web-admin-ui`,
`wyd-game-admin`, `wyd-economy`, `wyd-client-connect`, `wyd-database`.

## Passos

1. Ler cada SKILL.md restante e extrair responsabilidades e referências de código.
2. Marcar trechos obsoletos (Go arquivado citado como vigente) linha a linha.
3. Decidir por skill: suceder (nova Rust), atualizar (manter nome) ou aposentar.
4. Publicar o plano de execução da 09-2.

## Arquivos

- Criados: relatório de auditoria (nesta issue).
- Alterados: nenhum (auditoria não edita).
- Proibido tocar: `.agents/skills/*/SKILL.md` (edição é da 09-2).

## Testes / verificação

```bash
make check
```

Saída esperada: gates verdes; relatório com 100% das skills destinadas.

## Skills Relacionadas do freebuff

- .agents/skills/skill-creator - padrão de qualidade.
- .agents/skills/documentation-writer - relatório.

## Documentação de referencia

- pre-release-rust.md (FASE 9).
- roadmap-rust/issues/FASE-01-3-inventario-skills.md - inventário base.

## Relatório de auditoria (2026-11-07 — arquivo Go/C++ removido; fonte canônica `docs/` + `crates/`)

| Skill Go | Destino (09-2) | Motivo | Trechos obsoletos (caminho vigente → substituir por) |
|---|---|---|---|
| `wyd-server-dev` | **aposentar** | Escrever Go do tmserver; Hot Path agora é `wyd-rust-core`/`-net`/`-db` por domínio | `tmserver/`, `dbserver/`, `binserver/`, `webserver/`, `internal/`, `tmserver/content/`, `docs/migration/` → `crates/wyd-*/`, `docs/arquitetura/` |
| `wyd-database` | **suceder** → `wyd-rust-db` + apêndice Postgres | Schema/migrations/backup seguem válidos; ferramenta passa a `sqlx`/`refinery` | `internal/store/seed.go`, `internal/textenc`, `internal/migrations`, `W2PP_DB_DSN`, `dbserver` → `crates/wyd-db/migrations/`, `docs/arquitetura/modelo-banco.md` |
| `wyd-client-connect` | **atualizar** (mesmo nome) | serverlist/versão/diagnóstico seguem reais; servidor alvo passa a Rust | `tmserver/content/.../serv00.htm`, `make run-local`, `W2PP_CLIENT_VERSION`, `Source/Code/ClientPatch_v7662`, `W2PP_E2E_* + go test -tags=e2e .../tmserver/internal/world/` → `docs/cliente/` + `crates/wyd-net/` + `crates/wyd-launcher/` |
| `wyd-testing-bots` | **suceder** → `wyd-rust-testing` | Harness/bots/carga seguem necessários, em Rust (`cargo test`, benches, `wyd-sim`) | `go test ./tmserver/internal/...`, `go test -bench`, `W2PP_E2E_*`, `MSYS_NO_PATHCONV ... golang:1.26-alpine` → `cargo test --workspace`, `cargo bench --workspace` |
| `wyd-game-admin` | **atualizar** | Comandos GM/papéis são regra atemporal; backend passa a `wyd-admin` | `tmserver/internal/handler/gm.go`, `docs/migration/web-platform-plan.md §binServer` → `crates/wyd-admin/src/` + `docs/arquitetura/` |
| `wyd-economy` | **atualizar** | Preços/drops/refino/EXP são domínio atemporal; ferramenta passa a `wyd-sim` | `tmserver/content/.../*.txt`, `tmserver/internal/content/catalog.go`, `docs/migration/config-ops.md`, `dbserver import-npcs` → `crates/wyd-db/` + `crates/wyd-sim/` + `docs/arquitetura/modelo-banco.md` |
| `wyd-web-admin-ui` | **atualizar** | Portal segue Next.js; BFF passa a falar com `wyd-admin` | `docs/migration/web-platform-plan.md`, `tmserver` loop único, `tmserver/content/` `:ro`, `webserver :7600`, `internal/secure` → `crates/wyd-admin/src/` + `docs/rust/05-stack-docker.md` |

Regras atemporais a extrair na 09-2 (não perder): autoridade server-side, `delivery_queue` como única ponte web→jogo, auditoria de toda ação admin, `moderator_id` da sessão, `cache-control: no-store`, backup antes de escrever em `client/`, paridade de RNG como contrato, UNVERIFIED nunca vira gate.

Plano de execução da 09-2: (1) criar `wyd-rust-ops`/`wyd-rust-testing` no padrão vigente; (2) atualizar `wyd-database`, `wyd-client-connect`, `wyd-game-admin`, `wyd-economy`, `wyd-web-admin-ui` para Rust sem instruções conflitantes; (3) anotar `wyd-server-dev` como aposentada com link para sucessoras; (4) grep de conferência + `make check`.

## Subtasks

- [x] Codificar issue
- [x] Testar issue
- [x] Revisar issue
- [x] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] Cada skill com destino + motivo + trechos obsoletos listados.

## Riscos / rollback

- Risco: perder conhecimento embutido nas skills Go → mitigação: extrair regras
  atemporais para a sucessora antes de aposentar.
- Rollback: relatório é documentação (sem alteração em disco).
