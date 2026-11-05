# Issue 32 — Contas, papéis e banimento (backend novo)

**Fase:** 6 · **Depende de:** 19, 21 · **Bloqueia:** 30, 31, 38 · **Esforço:** L · **Estado:** ✅ fechada em 2026-11-04 (backend, portal e recusa **vista na tela do cliente**)

## Objetivo

Listar contas, promover/rebaixar papel (`player`/`moderator`/`admin`), bloquear/desbloquear e criar
conta — com auditoria — sem precisar de `psql` na mão.

## Resultado (2026-11-04)

- **Migração `0042_account_admin`** (a 0034 do plano já estava ocupada): `account.role_changed_at`,
  `account.role_changed_by`, `account.blocked_reason` e a tabela `account_admin_audit`
  (`actor_id`, `target_id`, `action`, `old_value`, `new_value`, `reason`, `created_at`, índices por
  alvo e por autor). Aplicada no boot do `webserver`; conferida em `schema_migrations`.
- **`AccountAdminService`** no `web.proto` (regenerado com `protoc-gen-go v1.36.11` /
  `protoc-gen-go-grpc v1.6.2`): `ListAccounts` (keyset + `total_count`), `GetAccount` (conta + 50
  linhas de auditoria), `SetRole`, `SetBlocked`, `CreateAccount`. `AccountRow` não tem campo de
  senha/PIN — o vazamento exigiria mudar o contrato.
- **Regras no store**, dentro da transação (`internal/store/account_admin.go`): ator re-lido do banco,
  ninguém altera o próprio papel/bloqueio, última conta admin **ativa** protegida (`SELECT … FOR
  UPDATE` nas linhas de admin antes de contar), `moderator` só bloqueia `player`. Cada escrita grava
  a auditoria na mesma transação; `blocked_reason` é limpo no desbloqueio.
- **Portal `/contas`**: lista com filtros (prefixo de login, papel, bloqueio) e cursor `apos`, detalhe
  com papel + motivo, bloqueio/desbloqueio, criação de conta já com papel (só `admin`) e a trilha de
  auditoria na tela. `moderator` vê a lista e o detalhe, mas sem criar conta e com o papel travado.
  O aviso "vale no próximo login" está nas duas telas.

### Desvios do plano original (conscientes)

- Migração é **0042**, não 0034 (o número já estava usado por `mob_carry`).
- `actor_id`/`target_id` da auditoria **sem FK**, igual a `character_admin_audit`: o histórico não pode
  depender da existência da conta.
- `CreateAccount` ganhou RPC própria no `AccountAdminService` (delegar ao `AccountWebService` deixaria
  a criação fora da auditoria, porque ele não conhece o moderador).
- `NAME_TAKEN` chega como `ADMIN_RESULT_INVALID` + `message` ("já existe uma conta com esse nome"),
  HTTP 422 — o enum de `AdminResult` não tem código de conflito.
- `adminCall` passou a preferir o `message` de negócio da resposta ao texto genérico do `AdminResult`.
  Sem isso, "esta é a última conta admin ativa" viraria "dados inválidos" na tela.

## Evidência de execução

```text
$ go test ./webserver/internal/accountadmin/ ./webserver/internal/grpcsrv/
ok  .../webserver/internal/accountadmin   1.658s
ok  .../webserver/internal/grpcsrv        0.448s

$ W2PP_E2E_ADMIN_ID=1 W2PP_E2E_WEB_API=127.0.0.1:7600 \
  W2PP_E2E_DSN=postgres://postgres:…@127.0.0.1:5432/postgres \
  go test -tags=e2e -count=1 ./webserver/internal/accountadmin -run TestE2EAccountAdmin -v
=== RUN   TestE2EAccountAdmin
    contas criadas: zqp16930=9 zqm16930=10 zqo16930=11
    cleanup: 3 contas de teste removidas
--- PASS: TestE2EAccountAdmin (0.82s)

$ docker exec -i openwyd-db-1 psql -U postgres -c "select version from schema_migrations order by version desc limit 2;"
 0042_account_admin
 0041_mob_carry

$ docker exec -i openwyd-db-1 psql -U postgres -t -c "select count(*) from account where name like 'zq%'; select count(*) from account_admin_audit;"
 0
 0
```

O e2e (contra o web-api de verdade) cobre: criação já com papel, promoção com
`role_changed_by`/`role_changed_at` e auditoria com o motivo, `moderator` recusado ao alterar papel,
auto-bloqueio recusado (`INVALID` + mensagem), `moderator` bloqueando `moderator` recusado
(`FORBIDDEN`), bloqueio com motivo persistido, desbloqueio limpando o motivo, `VerifyCredentials`
devolvendo `blocked=true` com a conta bloqueada e `false` depois, filtro `role=moderator` achando a
conta promovida e `player` recebendo `FORBIDDEN` na leitura.

```text
$ curl http://127.0.0.1:3000/contas                     → 307 /login   (sem cookie)
$ curl -H "Cookie: wyd_session=<admin>" …/contas        → 200  ("Contas e papéis", botão "criar conta")
$ curl -H "Cookie: wyd_session=<admin>" …/contas/1      → 200  ("Auditoria", aviso do próximo login)
$ curl -H "Cookie: wyd_session=<moderator>" …/contas    → 200  (sem botão "criar conta")
$ curl -H "Cookie: wyd_session=<moderator>" …/contas/1  → 200  ("Só um admin altera papéis. Você pode bloquear contas de jogador.")
$ cd portal && npx tsc --noEmit → 0 · npx vitest run → 76/76 · npx next build → 0
```

## Critérios de aceite

- [x] Listar/buscar contas sem qualquer campo sensível (`AccountRow` não tem hash; teste de `Get`)
- [x] Papel e bloqueio alteráveis com motivo, gravando auditoria na mesma transação
- [x] Auto-rebaixamento e remoção do último admin bloqueados **no store**, não só na UI
- [x] `moderator` não altera papel nem bloqueia `admin`/`moderator`
- [x] Bloqueio refletido no login **do jogo** (teste real com o cliente Windows) — 04/11: o cliente
  exibiu “Conta bloqueada Grade X, entre em contato com Suporte.” e fechou; `VerifyCredentials` já
  refletia o bloqueio
- [x] Migração aplica e reverte limpa; `go test` verde

## Riscos / rollback

- **Escalada de privilégio:** regra no store (não só no gRPC) + auditoria de toda escrita; `role` só
  muda com ator `admin` re-lido do banco dentro da transação.
- **Perder a gestão:** a última conta admin ativa não é rebaixada nem bloqueada; a conta `test`
  continua admin de recuperação.
- **Rollback:** `0042_account_admin.down.sql` remove tabela e colunas. Papéis podem ser redefinidos por
  SQL com o comando de emergência da skill `wyd-game-admin` (sem auditoria do BFF, e sabendo disso).

## Evidência de execução (2026-11-04) — bloqueio recusado + bug da mensagem

```sql
-- antes
select id, name, role, is_blocked, blocked_reason from account order by id;
 id | name | role  | is_blocked | blocked_reason
  1 | test | admin | f          |

update account set is_blocked=true, blocked_reason='teste QA issue 32' where name='test';
UPDATE 1

-- depois: test | admin | is_blocked=t | teste QA issue 32
```

- **Servidor:** `TestLoginNoAccountAndBlocked` e `TestLoginBadVersionClosed` **PASS** — o
  `LoginBlocked` vindo do dbServer vira recusa e a conexão é fechada.
- **Bug encontrado (o que você viu na tela):** `notify()` mandava `MSG_MessageBoxOk` com um código
  numérico local de 4 bytes. O struct legado é
  `{int Useless1; int Useless2; char String[96]}` (`Basedef.h:1534-1541`): com body de 4 bytes a
  string vai vazia e o cliente abre uma **caixa em branco** — nenhuma mensagem de erro aparece.
- **Causa do formato:** o legado manda **texto**, não código — `SendClientMessage(conn,
  g_pMessageStringTable[...])` = `_MSG_MessagePanel` com `HEADER.ID = 0`
  (`ProcessDBMessage.cpp:518-570`).
- **Correção:** as cinco recusas de login com texto confirmado no `Language.txt` passaram a sair por
  `sendClientMessage`: versão (36), 3 senhas erradas (37), conta inexistente (131), bloqueada (132,
  `Parm 0` → “Grade X”), senha incorreta (133). Bytes Windows-1252, como o resto da tabela.
- **Testes ajustados** para exigir `MsgMessagePanel` + texto exato (antes aceitavam
  `MsgMessageBoxOk` + código, o que mascarava o bug).
- **Confirmado no cliente** depois do rebuild (`docker compose up -d --build tmserver`).

### Confirmação no cliente (2026-11-04, 19:51–19:52)

```text
19:51:56.607  recv packet conn=1 type=0x020d len=104 routed=true
19:51:56.607  account login: relaying to dbServer conn=1 account=test
19:51:56.613  session send stats conn=1 frames=1 bytes=140 by_type=0x0101:1
19:51:56.613  session last sends conn=1 tail=0x0101(0)/128/-0ms
19:51:56.613  connection closed conn=1
19:52:03.173  … mesma sequência na 2ª tentativa
```

- O cliente exibiu **“Conta bloqueada Grade X, entre em contato com Suporte.”** e foi fechado em
  seguida (a recusa encerra a conexão, como no legado).
- 140 B no fio = header 12 + `MSG_MessagePanel.String[128]` com `HEADER.ID = 0`.
- **Conta liberada** ao fim do teste: `update account set is_blocked=false, blocked_reason='' where
  name='test';` → `is_blocked=f`.
- Fora de escopo (seguem no placeholder, documentado em `notice.go`): `LoginNow` e falha de backend,
  que não têm texto confirmado no `Language.txt`.
