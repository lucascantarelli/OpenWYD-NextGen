-- Gerado por `tmserver/cmd/seedgen` (issue 58) — NÃO EDITAR À MÃO.
-- Tabela: kefra_state (1 linhas). Regerar: `seedgen -dsn <dsn>`.
-- Dados de referência: nenhum dado de jogador entra aqui (allowlist em internal/store/seed.go).

INSERT INTO kefra_state (id, defeated, next_spawn_unix, last_spawn_unix, revision)
SELECT id, defeated, next_spawn_unix, last_spawn_unix, revision FROM json_populate_record(NULL::kefra_state, $json${"id": 1, "defeated": false, "revision": 1, "last_spawn_unix": 0, "next_spawn_unix": 1794312000}$json$::json)
ON CONFLICT DO NOTHING;
