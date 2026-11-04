# Issue 32 — Contas, papéis e banimento (backend novo)

**Fase:** 6 · **Depende de:** 19, 21 · **Bloqueia:** 30, 31, 38 · **Esforço:** L · **Estado:** ⏳ aberta

## Objetivo

Listar contas, promover/rebaixar papel (`player`/`moderator`/`admin`), bloquear/desbloquear e criar
conta — com auditoria — sem precisar de `psql` na mão.

## Contexto / evidência

- **Não existe RPC de papel/ban.** Hoje é SQL manual: `web.proto:200` ("Não há RPC para promover") e
  `docs/integrations/npc-admin-nextjs.md §2`; a skill `wyd-game-admin` usa `update account set
  role=… where name=…`. O próprio `AGENTS.md` avisa que o `role` é carregado no login.
- `account`: `id, name, pass_hash, role, is_blocked, donate_balance, …`. O login do jogo e o
  `VerifyCredentials` consultam `is_blocked` (`web.proto:70`).
- `AccountWebService.CreateAccount` já existe (argon2id no Go, `NAME_TAKEN`/`INVALID`
  — `web.proto:42-63`), então criar conta **não** exige backend novo; papel/ban exige.
- `DonateRevenueAdminService.SearchAccounts` já busca contas por nome (leitura) — reaproveitar como
  base de busca, mas não para escrita.
- Promover/rebaixar em `account` só tem efeito no próximo login; sessão web aberta mantém o papel
  antigo (por isso TTL curto na issue 19).

## Passos

1. **Migração** `internal/migrations/0034_account_admin.up.sql` (e `.down.sql` simétrico):
   ```sql
   ALTER TABLE account ADD COLUMN IF NOT EXISTS role_changed_at timestamptz;
   ALTER TABLE account ADD COLUMN IF NOT EXISTS role_changed_by bigint REFERENCES account(id);
   ALTER TABLE account ADD COLUMN IF NOT EXISTS blocked_reason text;
   CREATE TABLE IF NOT EXISTS account_admin_audit (
     id          bigserial PRIMARY KEY,
     actor_id    bigint NOT NULL REFERENCES account(id),
     target_id   bigint NOT NULL REFERENCES account(id),
     action      text   NOT NULL,   -- 'set_role' | 'set_blocked' | 'create'
     old_value   text,
     new_value   text,
     reason      text,
     created_at  timestamptz NOT NULL DEFAULT now()
   );
   CREATE INDEX IF NOT EXISTS account_admin_audit_target_idx ON account_admin_audit(target_id, id DESC);
   ```
2. **Proto** (`api/web/v1/web.proto`): `AccountAdminService` com
   `ListAccounts(filter, cursor)`, `GetAccount(account_id)`, `SetRole(account_id, role, reason)`,
   `SetBlocked(account_id, blocked, reason)`, `CreateAccount` (delegando ao já existente).
   Regras no comentário do serviço: `moderator_id` obrigatório, `admin` não pode rebaixar a si mesmo,
   `admin` não pode rebaixar o **último** admin, papel inválido → `ADMIN_RESULT_INVALID`.
3. **Store** `webserver/internal/accountadmin`: `List` (busca por nome, filtro de papel/block,
   paginação por `id`), `SetRole`, `SetBlocked`, cada escrita gravando `account_admin_audit` na mesma
   transação. Nunca devolver `pass_hash`.
4. **gRPC** `webserver/internal/grpcsrv/accountadmin.go` + registro em
   `webserver/cmd/webserver/main.go`. Autorização: `admin` para papel; `moderator` só pode
   bloquear/desbloquear `player`; ninguém altera papel de `admin` exceto `admin`.
5. **Teste Go** `accountadmin_test.go`: promoção, auto-rebaixamento recusado, último admin protegido,
   `player` recebe `FORBIDDEN`, auditoria criada, `pass_hash` nunca serializado.
6. **Portal** `/contas`: tabela (`id`, nome, papel, bloqueada, saldo donate, criada em), busca,
   filtros; detalhe com form de papel (Select + motivo obrigatório) e toggle de bloqueio (motivo
   obrigatório); botão "criar conta" usando `CreateAccount` (mostra `NAME_TAKEN` como 409).
7. Aviso na tela: "papel vale no próximo login; sessões web abertas mantêm o papel anterior".

## Arquivos

- Criados: `internal/migrations/0034_account_admin.{up,down}.sql`,
  `webserver/internal/accountadmin/{service.go,service_test.go}`,
  `webserver/internal/grpcsrv/accountadmin.go`,
  `portal/src/app/(admin)/contas/page.tsx`, `portal/src/app/(admin)/contas/[id]/page.tsx`,
  `portal/src/server/actions/accounts.ts`, `portal/src/lib/account.ts`.
- Alterados: `api/web/v1/web.proto`, `api/web/v1/web.pb.go` (regen),
  `webserver/cmd/webserver/main.go`, `portal/src/components/nav.ts`,
  `docs/integrations/` (novo guia `accounts-nextjs.md`).
- Proibido: expor `pass_hash`; permitir que a web redefina senha nesta issue (fica para depois, com
  fluxo próprio); apagar contas.

## Testes / verificação

```bash
# (comandos rodam na raiz do repositório)
MSYS_NO_PATHCONV=1 docker run --rm -v "D:/dolorestec/wyd2/OpenWYD:/src" -w /src golang:1.26-alpine \
  sh -c "go build ./... && go test ./webserver/internal/accountadmin/ ./webserver/internal/grpcsrv/"
```

```bash
docker compose up -d webserver   # aplica a migração no boot
docker compose exec -T db psql -U postgres -d postgres -c \
  "select name, role, role_changed_at is not null as auditado from account order by id;"
```

Evidência de ponta a ponta: promover `teste` a `moderator` pela tela, logar com `teste` no cliente do
jogo e usar um `/gm` de moderador; bloquear `teste2` e confirmar que o login do jogo é recusado;
tentar se auto-rebaixar como `test` → erro de negócio.

## Critérios de aceite

- [ ] Listar/buscar contas sem qualquer campo sensível
- [ ] Papel e bloqueio alteráveis com motivo, gravando auditoria na mesma transação
- [ ] Auto-rebaixamento e remoção do último admin bloqueados no servidor (não só na UI)
- [ ] `moderator` não altera papel nem bloqueia `admin`/`moderator`
- [ ] Bloqueio refletido no login do jogo (teste real)
- [ ] Migração aplica e reverte limpa; `go test` verde

## Riscos / rollback

- **Risco:** escalada de privilégio por bug de autorização. **Mitigação:** regra no store (não só no
  gRPC) + testes cobrindo cada combinação de papel.
- **Risco:** bloquear a única conta admin e perder a gestão. **Mitigação:** guarda do "último admin" e
  conta `test` como admin de recuperação documentada.
- **Rollback:** `.down.sql` remove as colunas/tabela; papéis podem ser redefinidos por SQL (comando na
  skill `wyd-game-admin`).
