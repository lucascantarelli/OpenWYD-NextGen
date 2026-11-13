-- 0023_seed_mundo_fixo: cidade, arena e eventos com agenda (tabelas que ficavam vazias).
-- Fonte: docs/wyd/economia.md, docs/wyd/sistemas/eventos.md, docs/wyd/sistemas/battle-royale.md
-- e as decisões registradas em docs/wyd/decisoes-servidor.md §3–§4 (horário, dia e round são nossos).
-- Idempotente: rodar duas vezes não duplica nem altera valor.

-- migrate:up
-- As quatro praças jogáveis. `map_id` fica nulo: o build 7662 não publica o mapa de cada
-- cidade (UNVERIFIED, ver comentário em 0008/0010) e inventar número quebraria o teleporte.
INSERT INTO city (code, name, map_id, is_capital, tax_rate_bps, content_source) VALUES
    ('armia',  'Armia',  NULL, false, 1000, 'docs/wyd/economia.md'),
    ('azran',  'Azran',  NULL, false, 1000, 'docs/wyd/economia.md'),
    ('erion',  'Erion',  NULL, false, 1000, 'docs/wyd/economia.md'),
    -- Noatun é o castelo: não é praça de loja pessoal como as outras, recebe 40% do imposto
    -- das três cidades (decisoes-servidor.md §3).
    ('noatun', 'Noatun', NULL, true,  1000, 'docs/wyd/decisoes-servidor.md §3')
ON CONFLICT (code) DO NOTHING;

-- Arenas com as regras documentadas: 2 equipes, 3 rounds, 5 min por round, 1 respawn,
-- nível mínimo 200 e sem drop (battle-royale.md).
INSERT INTO arena (code, name, kind, team_size, rounds, round_ms, respawns, min_level, no_drop, enabled, content_source) VALUES
    ('arena_real',    'Arena Real',    'royal',         16, 3, 300000, 1, 200, true, true,
     'docs/wyd/sistemas/battle-royale.md'),
    ('battle_royale', 'Battle Royale', 'battle_royale', 16, 3, 300000, 1, 200, true, true,
     'docs/wyd/sistemas/battle-royale.md')
ON CONFLICT (code) DO NOTHING;

INSERT INTO world_event (code, name, kind, enabled, requires_city, description, content_source) VALUES
    ('kefra', 'Kefra', 'kefra', true, false,
     'Respawn fixo no sábado; +100 de fama para a guilda que matar.', 'docs/wyd/sistemas/eventos.md'),
    ('noatun', 'Guerra de Noatun', 'noatun', true, true,
     'Quinzenal, 3 rounds no Altar de Thor; só entra quem tem cidade no servidor.',
     'docs/wyd/decisoes-servidor.md §4'),
    ('city_war', 'Guerra de Cidades', 'city_war', true, false,
     'Aposta na sexta, guerra no domingo; 14 membros por guilda por cidade.',
     'docs/wyd/sistemas/eventos.md')
ON CONFLICT (code) DO NOTHING;

-- Agenda: `weekday` 0 = domingo. Kefra sábado 14:00, Noatun domingo 21:00 a cada 2 semanas,
-- Guerra de Cidades domingo 20:00 (30 min). Tudo configurável: nada é fixo no tick.
INSERT INTO event_schedule (event_code, weekday, start_time, duration_minutes, timezone, every_n_weeks) VALUES
    ('kefra',    6, '14:00', 60, 'America/Sao_Paulo', 1),
    ('noatun',   0, '21:00', 60, 'America/Sao_Paulo', 2),
    ('city_war', 0, '20:00', 30, 'America/Sao_Paulo', 1)
ON CONFLICT (event_code, weekday, start_time) DO NOTHING;

INSERT INTO content_version (id, version, note)
VALUES (1, 6, 'mundo fixo: cidades, arenas e agenda de eventos')
ON CONFLICT (id) DO UPDATE SET version = GREATEST(content_version.version, 6), note = EXCLUDED.note;

-- migrate:down
DELETE FROM event_schedule;
DELETE FROM world_event;
DELETE FROM arena;
DELETE FROM city;
