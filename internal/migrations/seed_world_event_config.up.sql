-- Gerado por `tmserver/cmd/seedgen` (issue 58) — NÃO EDITAR À MÃO.
-- Tabela: world_event_config (1 linhas). Regerar: `seedgen -dsn <dsn>`.
-- Dados de referência: nenhum dado de jogador entra aqui (allowlist em internal/store/seed.go).

INSERT INTO world_event_config (id, enabled, item_index, rate, start_index, current_index, end_index, indexed, notice_enabled, double_exp_enabled, newbie_event_enabled, updated_by, updated_at)
SELECT id, enabled, item_index, rate, start_index, current_index, end_index, indexed, notice_enabled, double_exp_enabled, newbie_event_enabled, updated_by, updated_at FROM json_populate_record(NULL::world_event_config, $json${"id": true, "rate": 0, "enabled": false, "indexed": false, "end_index": 0, "item_index": 0, "updated_at": "2026-11-04T15:57:28.913357+00:00", "updated_by": null, "start_index": 0, "current_index": 0, "notice_enabled": true, "double_exp_enabled": false, "newbie_event_enabled": false}$json$::json)
ON CONFLICT DO NOTHING;
