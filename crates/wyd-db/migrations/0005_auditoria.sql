-- 0005_auditoria: trilha append-only (sem FK para nunca bloquear a escrita).
-- Convenção: -- migrate:up / -- migrate:down (ver docs/arquitetura/modelo-banco.md §4).

-- migrate:up
CREATE TABLE audit_log (
    id      BIGSERIAL PRIMARY KEY,
    at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    actor   TEXT NOT NULL DEFAULT 'system',
    action  TEXT NOT NULL,
    target  TEXT NOT NULL DEFAULT '',
    detail  TEXT NOT NULL DEFAULT '',
    CONSTRAINT audit_log_action_nonempty CHECK (char_length(action) > 0)
);
CREATE INDEX audit_log_at_idx ON audit_log (at);

-- migrate:down
DROP INDEX audit_log_at_idx;
DROP TABLE audit_log;
