-- 0029 — canal de servidor (ADR-0022 §6): a lista que o `tmserver` serve no `serv00.htm`
-- e no handshake vem do Postgres, não de arquivo local.
--
-- Um GRUPO é um servidor (instalação independente); cada grupo tem até 10 canais que
-- compartilham o banco e diferem só na instância do mundo (`docs/wyd/canais-e-cidadania.md`).
-- `channel = 0` é a linha de status (URL que o cliente busca no `:80`); `1..10` são os hosts
-- que o cliente disca no `8281` fixo.
--
-- Seed = o registro operacional `tools/client/serverlist-servers.txt` (mesmas células do
-- `serverlist.bin` instalado). Canal novo = linha nova aqui, sem build.

-- migrate:up
CREATE TABLE server_channel (
    group_id   SMALLINT NOT NULL CHECK (group_id >= 0 AND group_id < 10),
    channel    SMALLINT NOT NULL CHECK (channel >= 0 AND channel <= 10),
    host       TEXT NOT NULL CHECK (host <> ''),
    status_url TEXT NOT NULL DEFAULT '' CHECK (char_length(status_url) <= 32),
    enabled    BOOLEAN NOT NULL DEFAULT TRUE,
    note       TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (group_id, channel)
);

INSERT INTO server_channel (group_id, channel, host, status_url, note) VALUES
    (1, 0, '192.168.100.3', 'http://192.168.100.3/serv00.htm', 'linha de status do grupo 1'),
    (1, 1, '192.168.100.3', '', 'canal normal'),
    (1, 2, '192.168.100.4', '', 'canal verde (novato), mesma 8281 em outro IP');

-- migrate:down
DROP TABLE server_channel;
