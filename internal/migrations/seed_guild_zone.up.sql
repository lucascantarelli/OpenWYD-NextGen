-- Gerado por `tmserver/cmd/seedgen` (issue 58) — NÃO EDITAR À MÃO.
-- Tabela: guild_zone (5 linhas). Regerar: `seedgen -dsn <dsn>`.
-- Dados de referência: nenhum dado de jogador entra aqui (allowlist em internal/store/seed.go).

INSERT INTO guild_zone (zone, charge_guild, challenge_guild, clan, victory, city_tax, challenge_money, tax_vault, updated_at)
SELECT zone, charge_guild, challenge_guild, clan, victory, city_tax, challenge_money, tax_vault, updated_at FROM json_populate_record(NULL::guild_zone, $json${"clan": 0, "zone": 0, "victory": 0, "city_tax": 0, "tax_vault": 0, "updated_at": "2026-11-03T19:31:17.365563+00:00", "charge_guild": 0, "challenge_guild": 0, "challenge_money": 0}$json$::json)
ON CONFLICT DO NOTHING;
INSERT INTO guild_zone (zone, charge_guild, challenge_guild, clan, victory, city_tax, challenge_money, tax_vault, updated_at)
SELECT zone, charge_guild, challenge_guild, clan, victory, city_tax, challenge_money, tax_vault, updated_at FROM json_populate_record(NULL::guild_zone, $json${"clan": 0, "zone": 1, "victory": 0, "city_tax": 0, "tax_vault": 0, "updated_at": "2026-11-03T19:31:17.365563+00:00", "charge_guild": 0, "challenge_guild": 0, "challenge_money": 0}$json$::json)
ON CONFLICT DO NOTHING;
INSERT INTO guild_zone (zone, charge_guild, challenge_guild, clan, victory, city_tax, challenge_money, tax_vault, updated_at)
SELECT zone, charge_guild, challenge_guild, clan, victory, city_tax, challenge_money, tax_vault, updated_at FROM json_populate_record(NULL::guild_zone, $json${"clan": 0, "zone": 2, "victory": 0, "city_tax": 0, "tax_vault": 0, "updated_at": "2026-11-03T19:31:17.365563+00:00", "charge_guild": 0, "challenge_guild": 0, "challenge_money": 0}$json$::json)
ON CONFLICT DO NOTHING;
INSERT INTO guild_zone (zone, charge_guild, challenge_guild, clan, victory, city_tax, challenge_money, tax_vault, updated_at)
SELECT zone, charge_guild, challenge_guild, clan, victory, city_tax, challenge_money, tax_vault, updated_at FROM json_populate_record(NULL::guild_zone, $json${"clan": 0, "zone": 3, "victory": 0, "city_tax": 0, "tax_vault": 0, "updated_at": "2026-11-03T19:31:17.365563+00:00", "charge_guild": 0, "challenge_guild": 0, "challenge_money": 0}$json$::json)
ON CONFLICT DO NOTHING;
INSERT INTO guild_zone (zone, charge_guild, challenge_guild, clan, victory, city_tax, challenge_money, tax_vault, updated_at)
SELECT zone, charge_guild, challenge_guild, clan, victory, city_tax, challenge_money, tax_vault, updated_at FROM json_populate_record(NULL::guild_zone, $json${"clan": 0, "zone": 4, "victory": 0, "city_tax": 0, "tax_vault": 0, "updated_at": "2026-11-03T19:31:17.365563+00:00", "charge_guild": 0, "challenge_guild": 0, "challenge_money": 0}$json$::json)
ON CONFLICT DO NOTHING;
