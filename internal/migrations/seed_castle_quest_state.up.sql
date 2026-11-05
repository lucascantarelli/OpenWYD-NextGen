-- Gerado por `tmserver/cmd/seedgen` (issue 58) — NÃO EDITAR À MÃO.
-- Tabela: castle_quest_state (1 linhas). Regerar: `seedgen -dsn <dsn>`.
-- Dados de referência: nenhum dado de jogador entra aqui (allowlist em internal/store/seed.go).

INSERT INTO castle_quest_state (id, level, time_left, clear, leader_name, updated_at)
SELECT id, level, time_left, clear, leader_name, updated_at FROM json_populate_record(NULL::castle_quest_state, $json${"id": 1, "clear": false, "level": 0, "time_left": 0, "updated_at": "2026-11-03T19:31:17.365563+00:00", "leader_name": ""}$json$::json)
ON CONFLICT DO NOTHING;
