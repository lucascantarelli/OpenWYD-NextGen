-- 0042 — razão de crédito de donate (WEB-06): idempotência de `credit_donate_balance`.
--
-- O saldo mora em `account_wallet.coin` (conta, não personagem); a idempotência por chave
-- mora aqui: `idempotency_key` UNIQUE garante que repetir a mesma chave devolve o mesmo
-- razão (`replay = true`) sem creditar duas vezes. Tabela própria em vez de `ALTER TABLE`:
-- a série proíbe `ALTER TABLE` no `up`.
-- Dinheiro real nunca entra aqui: só crédito já conciliado (nota de `donate_shop_item`).

-- migrate:up
CREATE TABLE donate_credit (
    id              BIGSERIAL PRIMARY KEY,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    idempotency_key TEXT NOT NULL UNIQUE,
    account_id      BIGINT NOT NULL REFERENCES accounts (id) ON DELETE RESTRICT,
    credits         BIGINT NOT NULL CHECK (credits > 0),
    created_by      BIGINT REFERENCES accounts (id) ON DELETE SET NULL
);

-- migrate:down
DROP TABLE donate_credit;
