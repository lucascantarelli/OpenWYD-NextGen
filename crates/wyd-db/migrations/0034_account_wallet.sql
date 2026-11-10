-- 0034_account_wallet: moeda da conta (`MSG_DBCNFAccountLogin.Coin`).
-- A moeda é da **conta** (o `Cargo`/`Coin` do pacote de login), separada do ouro do
-- personagem (`characters.gold`, 0031): são saldos distintos no legado. Tabela 1:1 em vez de
-- coluna em `accounts` porque a série proíbe `ALTER TABLE` no up; conta sem linha responde
-- `Coin = 0`, nunca valor sintético.
--
-- `Coin` é `int` no fio (`STRUCT_MOB.Coin@28` e `MSG_DBCNFAccountLogin.Coin@1884`), por isso
-- o teto do CHECK é `i32::MAX`.

-- migrate:up
CREATE TABLE account_wallet (
    account_id BIGINT PRIMARY KEY REFERENCES accounts (id) ON DELETE CASCADE,
    coin       BIGINT NOT NULL DEFAULT 0,
    CONSTRAINT account_wallet_coin CHECK (coin >= 0 AND coin <= 2147483647)
);

-- migrate:down
DROP TABLE account_wallet;
