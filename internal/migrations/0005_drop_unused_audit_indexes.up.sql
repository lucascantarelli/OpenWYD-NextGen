-- DB-02: revisão dos 19 índices de auditoria com `pg_stat_user_indexes` na base
-- viva (06/11/2026, sem reset de estatística desde o boot).
--
-- Critério: fica o índice que serve uma query do código OU que já foi lido; cai o
-- que tem idx_scan = 0, nenhuma query filtrando pela coluna e tabela minúscula
-- (0–7 linhas), onde o seq scan é mais barato que a manutenção do índice.
--
-- Ficam (2):
--   account_admin_audit_target_idx          → `WHERE target_id = $1` (account_admin.go)
--   donate_shop_audit_action_created_idx    → 14 scans; filtro action + created_at
--                                             (donate_revenue.go)
-- Caem (17): idx_scan = 0 e nenhuma SELECT usa a coluna como filtro.
DROP INDEX IF EXISTS account_admin_audit_actor_idx;

DROP INDEX IF EXISTS character_admin_audit_character_idx;
DROP INDEX IF EXISTS character_admin_audit_moderator_idx;

DROP INDEX IF EXISTS content_rate_audit_key_idx;
DROP INDEX IF EXISTS content_rate_audit_moderator_idx;

DROP INDEX IF EXISTS daily_reward_audit_reward_item_id_idx;

DROP INDEX IF EXISTS donate_shop_audit_shop_item_id_idx;
DROP INDEX IF EXISTS donate_shop_audit_subject_idx;

DROP INDEX IF EXISTS drop_rate_audit_slot_idx;
DROP INDEX IF EXISTS drop_rate_audit_moderator_idx;

DROP INDEX IF EXISTS guild_admin_audit_guild_idx;
DROP INDEX IF EXISTS guild_admin_audit_moderator_idx;

DROP INDEX IF EXISTS item_effect_audit_item_idx;
DROP INDEX IF EXISTS item_effect_audit_moderator_idx;

DROP INDEX IF EXISTS mob_carry_audit_template_idx;
DROP INDEX IF EXISTS mob_carry_audit_moderator_idx;

DROP INDEX IF EXISTS npc_audit_npc_id_idx;
