# REVIEW-13 — Revisão permanente: banco de dados e persistência

**Fase:** revisão contínua · **Depende de:** — · **Bloqueia:** — · **Esforço:** contínuo · **Estado:** 🔁 viva

## Objetivo

Ponto de controle do schema, das migrações e dos dados do jogo: versionamento, reprodutibilidade,
índices, integridade e paridade com o servidor e o cliente.

## Escopo

- Migrações: `up` e `down` para cada uma, ordem, idempotência, squash e ambientes.
- Modelagem: chaves, FKs, constraints, tipos, normalização, redundância.
- Índices: o que existe e o que é usado (`idx_scan`), sem índice "de enfeite".
- Conteúdo: `item_catalog`, `skill_catalog`, `npc_template`, `npc_definition`, `world_item_seed`,
  seeds de referência e overlays.
- Roles e permissões: `openwyd_app`, `openwyd_migrator`, `openwyd_ro`, `db-roles`.
- Backup/restore testado e contagem por tabela.

## O que registrar a cada rodada

| Item | Como medir |
|---|---|
| Migrações | `ls internal/migrations \| wc -l` e conferência de `down` por arquivo |
| Tabelas e linhas | `scripts/db-audit.sql` |
| Índices sem uso | `pg_stat_user_indexes` com `idx_scan = 0` |
| Índices faltando | `pg_stat_user_tables` com `seq_scan` alto em tabela grande |
| Qualidade de texto | `scripts/check-data-quality.sql` |
| Restore | `scripts/db-restore.sh` + contagem idêntica |
| Achados novos | uma linha por achado com tabela e consulta |

## Critérios de aceite (por rodada)

- [ ] Base vazia migra do zero e volta (`down`) sem erro
- [ ] Restore reproduz o schema e as contagens
- [ ] Nenhuma tabela de personagem vivo escrita pela web
- [ ] Divergência entre catálogo do banco e conteúdo do cliente registrada como issue

## Histórico

| Data | Migrações | Achados | Destino |
|---|---|---|---|
| 06/11/2026 | 23 arquivos em `internal/migrations` | catálogo de item/skill/NPC já vem do banco (issue 58); faltam guard/level item/mob merc/game settings (issue 59) e a decisão sobre os mapas (issue 60) | issues 59 e 60 já abertas |
