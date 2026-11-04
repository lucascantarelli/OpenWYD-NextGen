-- Contas, papéis e banimento (issue 32).
--
-- Até aqui papel e bloqueio só mudavam por SQL manual. O web-api ganha escrita
-- auditada: cada alteração grava o valor antigo e o novo na MESMA transação, para
-- que a pergunta "quem rebaixou esta conta e por quê" tenha resposta.
--
-- `blocked_reason` fica no account para o login (e a tela) explicarem o motivo;
-- `role_changed_at/by` respondem "desde quando" e "quem". A auditoria guarda todo
-- o histórico, inclusive de contas que deixariam de existir (por isso target_id
-- e actor_id NÃO têm FK).
ALTER TABLE account ADD COLUMN IF NOT EXISTS role_changed_at TIMESTAMPTZ;
ALTER TABLE account ADD COLUMN IF NOT EXISTS role_changed_by BIGINT;
ALTER TABLE account ADD COLUMN IF NOT EXISTS blocked_reason TEXT NOT NULL DEFAULT '';

CREATE TABLE account_admin_audit (
    id          BIGSERIAL PRIMARY KEY,
    actor_id    BIGINT NOT NULL,              -- account.id de quem executou
    target_id   BIGINT NOT NULL,              -- account.id afetado (não é FK)
    action      TEXT   NOT NULL,              -- 'set_role' | 'set_blocked' | 'create'
    old_value   TEXT,                         -- papel anterior / 'active'|'blocked'
    new_value   TEXT,                         -- papel novo / 'active'|'blocked'
    reason      TEXT   NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX account_admin_audit_target_idx ON account_admin_audit(target_id, id DESC);
CREATE INDEX account_admin_audit_actor_idx  ON account_admin_audit(actor_id, id DESC);

COMMENT ON TABLE account_admin_audit IS
    'Alterações de papel/bloqueio/criação de conta (issue 32): valor anterior e novo, autor e motivo.';
COMMENT ON COLUMN account.blocked_reason IS
    'Motivo do último bloqueio (vazio quando a conta está ativa).';
