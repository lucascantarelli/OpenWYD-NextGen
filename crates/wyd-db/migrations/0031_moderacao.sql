-- 0031_moderacao: estado de moderação da conta (bloqueio de acesso).
-- Tabela própria de propósito: a coluna nasce na criação da tabela (o catálogo do
-- banco proíbe `ALTER TABLE` no up) e a identidade em `accounts` não engorda.
-- Bloqueio é decisão de staff, sempre com motivo: o CHECK cobra o motivo quando
-- `blocked` é verdadeiro.

-- migrate:up
CREATE TABLE account_moderation (
    account_id  BIGINT PRIMARY KEY REFERENCES accounts (id) ON DELETE CASCADE,
    blocked     BOOLEAN NOT NULL DEFAULT false,
    reason      TEXT NOT NULL DEFAULT '',
    blocked_by  BIGINT REFERENCES accounts (id) ON DELETE SET NULL,
    blocked_at  TIMESTAMPTZ,
    CONSTRAINT account_moderation_reason CHECK (NOT blocked OR char_length(reason) > 0),
    CONSTRAINT account_moderation_blocked_at CHECK (
        (blocked AND blocked_at IS NOT NULL) OR (NOT blocked AND blocked_at IS NULL)
    )
);

CREATE INDEX account_moderation_blocked_idx ON account_moderation (account_id) WHERE blocked;

-- migrate:down
DROP INDEX account_moderation_blocked_idx;
DROP TABLE account_moderation;
