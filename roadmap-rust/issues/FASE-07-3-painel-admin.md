# Issue 07-3 — Painel administrativo (auditoria, economia, moderação)

**Fase:** 7 · **Depende de:** 07-1 · **Bloqueia:** 07-4 · **Esforço:** M

## Objetivo

Entregar o painel de staff com auditoria de logs, estatísticas de economia e
ferramentas de moderação, tudo rastreável.

## Contexto / evidência

`pre-release-rust.md` FASE 7 §7.2. Auditoria nasce no gateway (07-1); aqui é
visualização + ações moderadas com confirmação.

## Passos

1. Listar ações de moderação e relatórios (economia, logs) com dados fonte.
2. Implementar telas com confirmação para ações destrutivas e trilha de auditoria.
3. Estatísticas de economia a partir de dados auditados (sem número inventado).
4. `portal-test` + `make check`.

## Arquivos

- Criados: telas admin + testes no portal.
- Alterados: `wyd-portal/src/**`, `crates/wyd-admin/src/` (endpoint faltante, com teste).
- Proibido tocar: bypass de auditoria, ação sem confirmação destrutiva.

## Testes / verificação

```bash
cargo test -p wyd-admin
make check
```

Saída esperada: `test result: ok`; `portal-test` verde; ação de moderação gera
registro auditável.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-web-admin-ui - auditoria e papéis.
- .agents/skills/frontend-design - direção visual.
- .agents/skills/ui-ux-practices - confirmações e UX de risco.

## Documentação de referencia

- wyd-portal/README.md - overlays e auditoria.
- pre-release-rust.md (FASE 7 §7.2).

## Subtasks

- [x] Codificar issue
- [x] Testar issue
- [x] Revisar issue
- [x] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [x] Toda ação de moderação auditada e confirmada; economia com fonte auditada.

> Entrega (2026-11-09): `account_moderation` (0031) guarda bloqueio/motivo/autor e o login do
> portal e do jogo recusa antes de comparar senha/PIN; `Accounts::moderation_state/set_role/
> set_blocked` com travas (só admin troca papel, ninguém altera o próprio papel nem bloqueia a
> própria conta, moderador não bloqueia staff, bloqueio exige motivo); `PersistentSink` grava
> `audit_log`; portas `AuditTrail`/`Economy` (ADR-0023); rotas `accounts/get_moderation`,
> `accounts/set_role`, `accounts/set_blocked`, `audit_admin/recent`, `economy_admin/summary`;
> telas `/contas` (painel de moderação com diálogo de confirmação), `/auditoria` e `/economia`.
> Economia usa `city_tax_ledger` + `delivery_queue` — sem número inventado.

## Evidência de execução (2026-11-09, stack viva)

- **Build:** `docker compose build gateway` do fonte atual (imagem `wyd/gateway:dev` criada
  `18:20:02Z`) e `up -d` sem tocar nos canais (16 sessões, último save `16:14:29Z`);
  `/health` → `{"ok":true,"database":"up","schema":31}`. Migration `0031` já aplicada não foi
  reexecutada.
- **Contrato por HTTP real** (token do portal + `x-wyd-actor`): `accounts/get_moderation`,
  `accounts/set_role` (`test4` player↔moderator), `accounts/set_blocked` (moderador bloqueia e
  libera `test5`), `audit_admin/recent`, `economy_admin/summary` — todos `result:1`.
- **Travas:** jogador → `result:2`; moderador bloqueando staff → `result:2` ("só admin bloqueia
  staff"); bloqueio sem motivo → `result:3`; admin trocando o próprio papel → `result:5`;
  moderador trocando papel → `result:2`; admin bloqueando a própria conta → `result:5`;
  moderador lendo a trilha → `result:2`; `x-wyd-actor` malformado → HTTP `400`.
- **Banco:** `account_moderation` com `blocked=true`, motivo, `blocked_by=3` e `blocked_at`
  coerentes (e `blocked=false` após liberar); `audit_log` com `1:admin accounts.set_role`,
  `3:moderator accounts.block/accounts.unblock` e `3:moderator auth.denied:accounts.set_role`;
  `economy_admin/summary` refletiu `city_tax_ledger` (`armia` 700, `noatun` 1500) e
  `delivery_queue` (`gold`, 1, 100).
- **Login:** conta-sonda (`--seed-staff`) → `ok:true`; bloqueada → `ok:false, blocked:true`;
  senha errada em conta bloqueada → `blocked:false` (não vaza); desbloqueada → `ok:true`.
  No jogo, `cargo test -p wyd-server --test login` → **7/7** contra banco de teste
  (`wyd_test_login`, 31 migrations), incluindo bloqueio/recusa/desbloqueio.
- **Defeitos corrigidos na execução:** `set_blocked` devolvia o estado anterior (snapshot do
  `WITH`; agora lê o `RETURNING` do upsert); `log_denied` perdia o papel real do ator
  (`id:player` fixo); gate de login entrava em panic sem DSN e colidia `login` em paralelo
  (`skip:` explícito + sequência atômica).
- **Gates:** `cargo fmt --check` 0; `clippy -D warnings` em `wyd-db`/`wyd-admin`/`wyd-server` 0;
  `cargo test` das três crates verde (com DSN no `wyd-server`); `check-migrations` 31/69/147 ok;
  `tsc --noEmit` 0; `vitest` 86/86.

## Riscos / rollback

- Risco: ação irreversível sem confirmação → mitigação: confirmação obrigatória
  + soft-action quando possível.
- Rollback: `git checkout -- wyd-portal/ crates/wyd-admin/`.
