-- 0040 — capacidade do canal: lotação e manutenção sem parar o processo.
--
-- Regra nova (QA-06): cada canal recusa entrada nova quando atinge
-- `max_players` sessões vivas, e recusa tudo em `maintenance`, sem derrubar
-- quem já está dentro. Tabela própria em vez de `ALTER TABLE server_channel`:
-- a série proíbe `ALTER TABLE` no `up` (coluna nova nasce na criação).
--
-- Seed = mesmos canais do 0029 com os defaults operacionais (1000 = MAX_USER do
-- legado, sem manutenção). Canal novo = linha nova aqui.

-- migrate:up
CREATE TABLE channel_capacity (
    group_id    SMALLINT NOT NULL CHECK (group_id >= 0 AND group_id < 10),
    channel     SMALLINT NOT NULL CHECK (channel >= 1 AND channel <= 10),
    max_players INTEGER NOT NULL DEFAULT 1000 CHECK (max_players >= 0),
    maintenance BOOLEAN NOT NULL DEFAULT FALSE,
    note        TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (group_id, channel),
    FOREIGN KEY (group_id, channel) REFERENCES server_channel (group_id, channel)
);

INSERT INTO channel_capacity (group_id, channel, note) VALUES
    (1, 1, 'canal normal: 1000 sessões, sem manutenção'),
    (1, 2, 'canal verde: 1000 sessões, sem manutenção');

-- migrate:down
DROP TABLE channel_capacity;
