-- Gerado por `tmserver/cmd/seedgen` (issue 58) — NÃO EDITAR À MÃO.
-- Tabela: npc_config_meta (1 linhas). Regerar: `seedgen -dsn <dsn>`.
-- Dados de referência: nenhum dado de jogador entra aqui (allowlist em internal/store/seed.go).

INSERT INTO npc_config_meta (id, version)
SELECT id, version FROM json_populate_record(NULL::npc_config_meta, $json${"id": true, "version": 19}$json$::json)
ON CONFLICT DO NOTHING;
