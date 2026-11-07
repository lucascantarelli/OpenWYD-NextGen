# Modelo do banco (`wyd-db`) — FASE-04

> Fonte: `roadmap-rust/issues/FASE-04.md` (+ 04-1/04-2/04-3). Implementação:
> `crates/wyd-db/{migrations,seeds,src/{model,repo,pool,transfer}}`.

## 0. Lacuna do arquivo legado (registrado honestamente)

As referências citadas nas issues (`.deprecated-go/internal/migrations/` e
`.deprecated-go/docs/analysis/banco-de-dados.md`) estão **vazias no arquivo**
(diretórios presentes, sem arquivos em 2026-11-07). O modelo abaixo deriva
portanto dos requisitos das issues + domínio do jogo + invariantes do
`AGENTS.md` (autoridade server-side, auditoria, `delivery_queue` como única
ponte web→jogo). Revisar contra o schema Go/legado real quando o arquivo for
preenchido (pendência registrada, não bloqueio).

## 1. ER (texto)

```text
accounts 1───* characters 1───* character_items
   │              │  ╲
   │              │   ╲──* guild_members *───1 guilds
   │              │
   │              └── (pos_x, pos_y, level, exp, hp/mp)
   │
   ├───1 vault_items (PK account_id+slot)
   │
   └───* delivery_queue (grants pendentes do portal/admin)

audit_log (append-only, sem FK para nunca bloquear escrita)
```

## 2. Tabelas e integridade

| Tabela | PK | FKs | CHECKs | Índices quentes |
|---|---|---|---|---|
| `accounts` | `id` | — | `role IN ('player','moderator','admin')`, `login` não-vazio | `UNIQUE(login)` |
| `characters` | `id` | `account_id → accounts` ON DELETE CASCADE | `slot 0–3`, `level ≥ 1`, `exp ≥ 0`, `hp ≥ 0`, `name` não-vazio | `UNIQUE(account_id, slot) WHERE deleted_at IS NULL`, `UNIQUE(name)`, `idx(account_id)` |
| `character_items` | `(character_id, slot)` | `character_id → characters` ON DELETE CASCADE | `slot 0–127`, `qty ≥ 1` | `idx(character_id)` (= PK) |
| `vault_items` | `(account_id, slot)` | `account_id → accounts` ON DELETE CASCADE | `slot 0–127`, `qty ≥ 1` | `idx(account_id)` (= PK) |
| `guilds` | `id` | `master_character_id → characters` DEFERRABLE | `name` não-vazio | `UNIQUE(name)` |
| `guild_members` | `(guild_id, character_id)` | ambas CASCADE | `rank 0–9` | `UNIQUE(character_id)` (um clã por char) |
| `audit_log` | `id` | — (nunca bloqueia) | `action` não-vazia | `idx(at)` |
| `delivery_queue` | `id` | `account_id/character_id` SET NULL (grant sobrevive a delete? não: RESTRICT — ver §4) | coerência por `kind` (item/gold/level) | parcial `WHERE status='pending'` |

Tipos: IDs `BIGINT GENERATED …` (ver `BIGSERIAL` nas migrations — forma
simples e portátil no PG17); dinheiro/EXP `BIGINT ≥ 0`; tempo `TIMESTAMPTZ`;
`role`/`kind`/`status` como `TEXT + CHECK` (enum em tabela separada é
over-engineering nesta escala; CHECK documenta o domínio).

## 3. Decisões

1. **Soft-delete de personagem** (`deleted_at`): nome e slot liberados via
   índice parcial — recriação sem violar unicidade, histórico preservado.
2. **Inventário normalizado, não blob**: `character_items`/`vault_items` com
   PK composta permitem conferência item-a-item no boot (anti-dupe) sem ler
   binário opaco. Efeitos de item continuam catálogo (FASE-10/economia).
3. **`delivery_queue` relacional**: colunas tipadas por `kind` + CHECK de
   coerência, em vez de JSONB opaco — a fila é dinheiro/item, não log.
4. **Sem FK na `audit_log`**: auditoria nunca falha por integridade alheia.
5. **`guilds.master` DEFERRABLE**: permite criar clã + líder na mesma transação.
6. **Grants restritos**: `delivery_queue.account_id/character_id` com
   `ON DELETE RESTRICT` — grant pendente impede apagar a conta/personagem
   (falha explícita em vez de item órfão).

## 4. Migrations (convenção do repo)

- `crates/wyd-db/migrations/NNNN_descricao.sql`, uma por domínio, com
  marcadores `-- migrate:up` e `-- migrate:down` (estilo dbmate/refinery).
- Down reverte exatamente o up (ordem inversa); `audit_log` incluída (ambiente
  descartável — produção futura exige política própria, ver 04-2).
- Runner `sqlx`/`refinery` chega na FASE-05/04-3 (exige rede + ADR); até lá a
  aplicação é por `psql -f` com split nos marcadores (script de validação em
  `tests/constraints.rs` + prova ao vivo documentada na issue).

## 5. Seeds

- `seeds/dev.sql` versionado: conta/personagem/clã de desenvolvimento com
  valores **obviamente falsos** (`DEV-ONLY`, nunca hash real).
- Produção **nunca** versiona seed com segredo (regra da árvore §12); o
  procedimento de seed prod vive no runbook de operação (FASE-05), não aqui.
