# Issue 03 — Subir Postgres, migrações e contas seed

**Fase:** 0 · **Depende de:** 02 · **Bloqueia:** 04, 07, 09, 12, 15 · **Esforço:** S · **Estado:** ✅ concluída (2026-11-02)

## Objetivo

Banco PostgreSQL no ar, com **todas as migrações aplicadas** e a conta `test`/`test123` semeada
(argon2id, sem personagens), pronta para o login.

## Contexto / evidência

- Compose define `db: postgres:17-alpine`, volume `pgdata`, DSN
  `postgres://postgres:dev@db:5432/postgres` injetado como `W2PP_DB_DSN`.
- As migrações são embutidas (`internal/migrations/*.up.sql`, até `0033_celestial_circlet_adds`) e
  aplicadas no boot do primeiro serviço que conectar (`store.Migrate`) — **não** existe passo manual
  de `psql -f`.
- `dbserver` tem o subcomando `seed-account` (idempotente, argon2id, conta sem personagens).
- O `run-local.sh` sobe `db binserver dbserver webserver` **antes** do `tmserver` e espera o dbserver
  ficar `Up`, porque os serviços disputam `Migrate` em banco novo e um login pode pendurar se o
  dbserver ainda não estiver pronto.

## Passos

```bash
# (comandos rodam na raiz do repositório)

# 1. Banco + serviços internos (sem tmserver ainda)
docker compose up -d --build db binserver dbserver webserver

# 2. Esperar o dbserver subir de verdade
docker compose ps
docker compose logs --since 2m dbserver | tail -30
#   esperado: conexão ok + migrações aplicadas, sem erro de DSN

# 3. Semear a conta
docker compose run --rm dbserver seed-account -name test -pass test123

# 4. Conferir
docker compose exec db psql -U postgres -c "\dt" | head -20
docker compose exec db psql -U postgres -c \
  "select name, role, is_blocked from account order by name;"
```

Conta extra para teste de dois clientes simultâneos (visão entre players):

```bash
docker compose run --rm dbserver seed-account -name test2 -pass test123
```

## Arquivos

- Alterados: nenhum (operação de ambiente).
- Referência de schema: `internal/migrations/` e `internal/domain/`.
- Proibido: criar tabela/coluna `à mão` via `psql`; qualquer mudança de schema pertence a uma
  migração nova.

## Testes / verificação

```bash
# 1. a conta existe e não está bloqueada
docker compose exec -T db psql -U postgres -tAc \
  "select count(*) from account where name='test' and is_blocked=false;"   # esperado: 1

# 2. a senha está hasheada (nunca em texto)
docker compose exec -T db psql -U postgres -tAc \
  "select left(pass_hash,10) from account where name='test';"                # esperado: prefixo argon2

# 3. a conta não tem personagem (a criação é feita no cliente)
docker compose exec -T db psql -U postgres -tAc \
  "select count(*) from character c join account a on a.id=c.account_id where a.name='test';"
# esperado: 0

# 4. migração aplicada até a última versão
docker compose exec -T db psql -U postgres -c "select version from schema_migrations order by version desc limit 3;"
```

## Critérios de aceite

- [ ] `db` e `dbserver` em `Up`, logs sem erro de migração/DSN
- [ ] `seed-account` roda sem erro e é idempotente (rodar duas vezes não duplica conta)
- [ ] Hash argon2id confirmado, sem senha em texto plano em lugar nenhum
- [ ] Contas `test` e `test2` disponíveis; `role` = usuário comum, `is_blocked` = false
- [ ] Saída dos 4 comandos de verificação anexada como evidência

## Riscos / rollback

- **Risco:** banco antigo de outra tentativa com schema parcial → migração falha. **Mitigação:** ler
  o erro exato antes de qualquer reset; `docker compose down -v` só com autorização explícita.
- **Risco:** `seed-account` sem `W2PP_DB_DSN` no `run` → tenta DSN local e falha. **Mitigação:** usar
  `docker compose run --rm dbserver …` (herda o env do serviço), não `docker run`.
- **Risco:** dois serviços aplicando migração em paralelo. **Mitigação:** mesmo padrão do
  `run-local.sh` (dbserver primeiro, esperar `Up`).
- **Rollback:** `docker compose stop dbserver webserver binserver` mantém os dados; recriar o volume
  é o último recurso e apaga tudo.
