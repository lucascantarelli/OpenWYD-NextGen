-- Gerado por `tmserver/cmd/seedgen` (issue 58) — NÃO EDITAR À MÃO.
-- Tabela: world_event_meta (1 linhas). Regerar: `seedgen -dsn <dsn>`.
-- Dados de referência: nenhum dado de jogador entra aqui (allowlist em internal/store/seed.go).

INSERT INTO world_event_meta (id, version)
SELECT id, version FROM json_populate_record(NULL::world_event_meta, $json${"id": true, "version": 2}$json$::json)
ON CONFLICT DO NOTHING;
