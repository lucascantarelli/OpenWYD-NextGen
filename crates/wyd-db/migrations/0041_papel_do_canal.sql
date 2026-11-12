-- 0041 — papel do canal (GAME-10): normal × verde, e o bônus de cidadania no drop.
--
-- Regra do legado (`docs/wyd/canais-e-cidadania.md` §2–§3): cidadania é o vínculo com **um**
-- canal (`Citizen = ServerIndex + 1`) e o canal verde/novato é propriedade fixa do processo —
-- sem PvP e com loja pessoal aberta. Os dois processos continuam idênticos em build: o papel
-- vem daqui (ou de `WYD_SERVER_CHANNEL_ROLE`, que sobrescreve para teste/operação).
--
-- Tabela própria em vez de `ALTER TABLE server_channel`: a série proíbe `ALTER TABLE` no `up`.
-- Seed = os mesmos canais do 0029/0040; o 0029 já os descreve como "normal" e "verde (novato)".

-- migrate:up
CREATE TABLE channel_role (
    group_id SMALLINT NOT NULL CHECK (group_id >= 0 AND group_id < 10),
    channel  SMALLINT NOT NULL CHECK (channel >= 1 AND channel <= 10),
    role     TEXT NOT NULL DEFAULT 'normal' CHECK (role IN ('normal', 'verde')),
    note     TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (group_id, channel),
    FOREIGN KEY (group_id, channel) REFERENCES server_channel (group_id, channel)
);

INSERT INTO channel_role (group_id, channel, role, note) VALUES
    (1, 1, 'normal', 'canal normal: PvP conforme o mapa, regra completa'),
    (1, 2, 'verde', 'canal verde (novato): sem PvP, loja pessoal permitida');

-- Bônus de drop do canal de cidadania, em basis points, como **parâmetro** do servidor:
-- 0 = desligado (default) porque o legado não traz a taxa (`Language.txt:423/432` só a
-- descreve). `source_line = 0` marca linha nossa: não vem do `gameconfig.txt`, por isso não
-- carrega linha do arquivo-fonte.
INSERT INTO game_setting (section, key, value_int, value_text, source_line) VALUES
    ('Channel Settings', 'citizen_bonus_bps', 0, '0', 0);

-- migrate:down
DELETE FROM game_setting WHERE section = 'Channel Settings' AND key = 'citizen_bonus_bps';
DROP TABLE channel_role;
