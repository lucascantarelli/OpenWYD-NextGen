# ADR-0023 — Moderação, trilha de auditoria e economia auditada no Cold Path

- **Estado:** aceita
- **Data:** 2026-11-09
- **Decisores:** dono do projeto + agente
- **Governa:** WEB-03 (painel de staff), WEB-04 (E2E do painel), QA-08 (painéis)

## Contexto

A WEB-03 exige painel de staff com três coisas que o control plane ainda não tinha:

1. **moderação** de conta (papel e bloqueio de acesso) com trilha do autor;
2. **auditoria** consultável — `auth.denied` e recibos de grant já existiam em memória/stdout,
   mas a tabela `audit_log` (0005) não tinha porta de escrita nem de leitura;
3. **estatísticas de economia** com fonte real, nunca número inventado.

O `AGENTS.md` §7 exige ADR para arquivo novo fora da árvore da ADR-0002, e o catálogo do banco
(`crates/wyd-db/tests/contrato/catalogo.rs`) proíbe `ALTER TABLE` no bloco `up` das migrations: coluna nova
nasce com a tabela. Blocking em `accounts` exigiria `ALTER`; bloquear sem motivo auditável não
atende ao critério de aceite.

## Decisão

### 1. Nova tabela `account_moderation` (migration `0031_moderacao.sql`)

| Coluna | Papel |
|---|---|
| `account_id` | PK, FK para `accounts` (ON DELETE CASCADE) |
| `blocked` | estado atual; ausência de linha = liberada |
| `reason` | motivo obrigatório quando `blocked` (CHECK) |
| `blocked_by` | FK `accounts` (autor da última decisão, ON DELETE SET NULL) |
| `blocked_at` | instante do bloqueio; CHECK coerente com `blocked` |

Estado disciplinar separado da identidade: `accounts` não engorda, e um dia de bloqueio não
reescreve a linha de autenticação. O login (portal e jogo) consulta o estado no mesmo `SELECT` da
credencial (`CredentialRecord.blocked`) e recusa **antes** de comparar senha/PIN.

### 2. Portas novas em `wyd-db`

| Arquivo | Porta | Por quê |
|---|---|---|
| `src/repo/audit.rs` | `AuditTrail` (`record`, `recent`) | trilha append-only: gateway grava, painel lê |
| `src/repo/economy.rs` | `Economy` (`summary`) | `city_tax_ledger` + `delivery_queue`, cada número com fonte |
| `src/repo/accounts.rs` | `moderation_state`/`set_role`/`set_blocked` na porta `Accounts` | moderação é operação de conta, não domínio paralelo |

`AccountModeration` fica em `src/model/account.rs` (conta + estado) e `AuditRow`/`NewAuditEntry`
em `src/repo/audit.rs`. As duas portas de leitura usam futuro empacotado (`Pin<Box<dyn Future>>`)
porque o roteador do gateway as guarda como `Arc<dyn Trait + Send + Sync>`: o transporte não tem
runtime (ADR-0006) e não pode ser genérico sobre o pool.

### 3. Escrita da trilha

`wyd-gateway` substitui o `StdoutSink` por `PersistentSink`: grava `audit_log` e continua logando no
stdout (Loki). O `actor` usa o formato `id:role` sobre a coluna TEXT da 0005; a leitura separa os
campos por `split_part` com máscara para o valor legado `system`. Não há retry: falha de escrita é
logada como `audit_write_failed`, nunca engolida.

### 4. Permissões

- `accounts/set_role`: só admin; ninguém altera o próprio papel (409).
- `accounts/set_blocked`: staff; motivo obrigatório ao bloquear; staff alvo só por admin;
  ninguém bloqueia a própria conta (409).
- `audit_admin/recent`: só admin (a trilha contém ações de todo o staff).
- `economy_admin/summary`: staff.

Toda função escala o autor da **sessão** (`StaffSession`), audita a ação e registra a negação
(`auth.denied:<ação>`), como o resto de `api/*`.

## Consequências

- **Positivas:** bloqueio vale no jogo e no portal; toda escrita de staff tem linha em `audit_log`
  com autor/alvo/motivo; `/auditoria` e `/economia` mostram dado real; `catalogo.rs` e
  `check-migrations.mjs` continuam verdes (tabela criada, não alterada); o banco antigo só precisa
  do `0031` aplicado (tabela aditiva).
- **Negativas:** a trilha cresce sem política de retenção (append-only por desenho; retenção é
  issue futura); `audit_log.actor` continua TEXT por compatibilidade com a 0005; leitura de
  auditoria exige admin (moderador não revisa o próprio histórico).
- **Rollback:** `DROP TABLE account_moderation` (down da 0031) + `git checkout -- wyd-portal/
  crates/wyd-{db,admin,server}/`.

## Arquivos criados por esta decisão

```
crates/wyd-db/migrations/0031_moderacao.sql
crates/wyd-db/src/repo/audit.rs
crates/wyd-db/src/repo/economy.rs
crates/wyd-admin/src/api/moderation.rs
crates/wyd-admin/src/api/economy.rs
wyd-portal/src/app/(admin)/auditoria/page.tsx
wyd-portal/src/app/(admin)/economia/page.tsx
```
