-- Reverte a administração de contas (issue 32).
DROP TABLE IF EXISTS account_admin_audit;
ALTER TABLE account DROP COLUMN IF EXISTS blocked_reason;
ALTER TABLE account DROP COLUMN IF EXISTS role_changed_by;
ALTER TABLE account DROP COLUMN IF EXISTS role_changed_at;
