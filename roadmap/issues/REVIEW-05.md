# REVIEW-05 — Banco de dados: roles, retenção, naming e recursos do Postgres

**Fase:** revisão · **Depende de:** 58 (concluída) · **Esforço:** M · **Estado:** ⏳ proposta

## Objetivo

Fechar o que o baseline deixou em aberto: papéis de banco separados, backup com restauração testada,
teste de `down`, naming consistente e uso consciente de views/índices — sem reintroduzir duas fontes
de verdade.

## Contexto/evidência

- Baseline squasheado e adotado: `0001_schema.up.sql` + 12 `seed_*.up.sql`, 49 tabelas de domínio +
  `schema_migrations`, 18.232 linhas de referência validadas em banco vazio (`docs/analysis/banco-de-dados.md` §11).
- Pendências já registradas no §10 do mesmo documento: **roles `openwyd_app`/`openwyd_ro` e DSN sem
  superusuário** (P1), **retenção de backup + teste de restauração** (P1), **`character` →
  `characters`** (P2, quebra código), **views de relatório** (P2), **revisão dos 19 índices de
  auditoria sem leitura** (P2), além de **teste de `down`** e de migração em base vazia no CI (REVIEW-04).
- Todos os serviços usam o mesmo superusuário `postgres` (DSN do compose deriva de
  `POSTGRES_PASSWORD`); nada impede o portal de escrever em tabela de personagem vivo por engano de
  DSN.
- `scripts/db-audit.sql`, `db-backup.sh` e `db-baseline.sh` existem; falta rotina de restore
  verificada.

## Passos

1. **Roles**: `openwyd_app` (DML no que o boot precisa), `openwyd_ro` (portal de leitura) e
   `openwyd_migrator` (DDL); DSNs por serviço no compose; `ALTER DEFAULT PRIVILEGES` no schema.
2. **Teste de restauração**: `scripts/db-restore.sh` que restaura o último dump em um banco
   descartável e compara contagem por tabela; rodar mensal e registrar data em `docs/analysis/banco-de-dados.md`.
3. **Teste de `down`**: em base efêmera, aplicar `0001_schema.up.sql` e depois `down`, validando que
   o schema some sem erro (o `down` é `DROP SCHEMA … CASCADE`, então o teste é de fumaça, não de
   reversibilidade de dados).
4. **Naming**: decidir `character` → `characters` com uma entrega só (migration + código + portal +
   testes) — não deixar meio-caminho.
5. **Índices de auditoria**: para cada um dos 19, medir `pg_stat_user_indexes.idx_scan` após uma
   semana de uso; remover os que nunca são lidos e o dado não justifique.
6. **Views**: criar só quando houver consumo real (portal/relatório), com `security_invoker` para não
   furar RLS/papéis.

## Arquivos

- `internal/migrations/` (nova migration de roles — nunca reutilizando número)
- `docker-compose.yaml`, `.env.example` (DSNs por serviço)
- `scripts/db-restore.sh` (novo)
- `docs/analysis/banco-de-dados.md` (evidência)

## Testes/verificação

```bash
bash scripts/db-baseline.sh                     # dry-run: validação em base vazia
bash scripts/db-restore.sh --check              # restaura em base descartável e compara contagens
docker exec openwyd-db-1 psql -U postgres -c "\du"   # papéis e atributos
```

## Critérios de aceite

- [ ] Nenhum serviço conecta como superusuário; portal só lê o que precisa ler.
- [ ] Restore do último backup validado, com data e contagem registradas.
- [ ] `down` do baseline testado em base efêmera.
- [ ] Decisão sobre `characters` tomada e executada (ou issue própria fechada).
- [ ] Índices de auditoria revisados com `idx_scan` como evidência.

## Riscos/rollback

Risco: trocar DSN por papel sem GRANT completo quebra o boot em produção (foi o padrão dos bugs do
gate da fase 4). Mitigação: aplicar em base efêmera primeiro, com `store.Migrate` + boot completo, e
só então no dev. Rollback: o compose volta ao DSN antigo por variável de ambiente.

## Issues que este review abre

- `DB-01` roles e DSN por serviço
- `DB-02` `db-restore.sh` + rotina mensal
- `DB-03` teste de `down` no CI
- `DB-04` renomear `character` → `characters`
- `DB-05` revisão dos 19 índices de auditoria
