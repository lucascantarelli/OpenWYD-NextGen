-- 0001_contas: contas de acesso (login, credencial, papel).
-- Convenção: -- migrate:up / -- migrate:down (ver docs/arquitetura/modelo-banco.md §4).

-- migrate:up
CREATE TABLE accounts (
    id          BIGSERIAL PRIMARY KEY,
    login       TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    pin_hash    TEXT,
    -- Estado do bloqueio do PIN (persistido: reiniciar o processo não zera tentativas).
    pin_failures SMALLINT NOT NULL DEFAULT 0,
    pin_locked_until TIMESTAMPTZ,
    role        TEXT NOT NULL DEFAULT 'player',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_login_at TIMESTAMPTZ,
    CONSTRAINT accounts_login_nonempty CHECK (char_length(login) > 0),
    CONSTRAINT accounts_role CHECK (role IN ('player', 'moderator', 'admin')),
    CONSTRAINT accounts_pin_failures CHECK (pin_failures >= 0)
);
CREATE UNIQUE INDEX accounts_login_unique ON accounts (login);

-- migrate:down
DROP INDEX accounts_login_unique;
DROP TABLE accounts;
