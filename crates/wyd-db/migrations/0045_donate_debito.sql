-- 0045 — razão de débito de donate (FASE-07-6): extrato e idempotência do débito.
--
-- O saldo continua em `account_wallet.coin`; este razão registra cada débito com
-- `idempotency_key` UNIQUE (repetir a chave devolve o mesmo razão, sem debitar duas
-- vezes). A receita do portal é a soma de `donate_credit` (0042) menos estes débitos.
-- Tabela própria em vez de `ALTER TABLE`: a série proíbe `ALTER TABLE` no `up`.

-- migrate:up
CREATE TABLE donate_debit (
    id              BIGSERIAL PRIMARY KEY,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    idempotency_key TEXT NOT NULL UNIQUE,
    account_id      BIGINT NOT NULL REFERENCES accounts (id) ON DELETE RESTRICT,
    credits         BIGINT NOT NULL CHECK (credits > 0),
    -- 'portal' = ajuste manual do staff; 'shop_purchase' = compra na loja de donate.
    reason          TEXT NOT NULL DEFAULT 'portal'
                    CHECK (reason IN ('portal', 'shop_purchase')),
    created_by      BIGINT REFERENCES accounts (id) ON DELETE SET NULL
);

CREATE INDEX donate_debit_account_at_idx ON donate_debit (account_id, created_at);

-- migrate:down
DROP TABLE donate_debit;
