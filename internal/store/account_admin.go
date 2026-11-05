package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"openwyd-nextgen/internal/domain"
)

// Account administration (issue 32): list/get plus the three audited writes.
//
// The BUSINESS RULES live here, inside the transaction, not only in the gRPC
// layer: a second caller (a script, a future RPC) cannot bypass "admin can't
// demote themselves" or "the last active admin is protected". Each write locks
// the target row (and, when an admin is leaving, the admin set) and writes
// account_admin_audit in the same transaction.
var (
	// ErrInvalidRole means the role is not player/moderator/admin.
	ErrInvalidRole = errors.New("store: invalid account role")
	// ErrNotAdminActor means the caller is not an admin.
	ErrNotAdminActor = errors.New("store: actor is not an admin")
	// ErrSelfChange means the caller tried to change their own role/block.
	ErrSelfChange = errors.New("store: actor cannot change own account")
	// ErrLastAdmin means the change would leave no active admin.
	ErrLastAdmin = errors.New("store: last active admin")
	// ErrModeratorScope means a moderator targeted a non-player account.
	ErrModeratorScope = errors.New("store: moderator can only manage players")
)

// ValidRole reports whether role is one of the three stored values.
func ValidRole(role string) bool {
	return role == "player" || role == "moderator" || role == "admin"
}

const accountRowCols = `
	a.id, a.name, a.email, a.role, a.is_blocked, a.blocked_reason, a.donate_balance,
	a.created_at, a.role_changed_at, a.role_changed_by,
	(SELECT count(*) FROM character c WHERE c.account_id = a.id)`

const accountRowFrom = ` FROM account a`

// ListAccounts returns one keyset page ordered by id. The filter is built from
// fixed SQL fragments with bound parameters — never from string interpolation of
// caller data.
func (s *Store) ListAccounts(ctx context.Context, f domain.AccountFilter) ([]domain.AccountRow, error) {
	where, args := accountFilterSQL(f)
	limit := f.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	args = append(args, limit)
	sql := `SELECT ` + accountRowCols + accountRowFrom + where +
		fmt.Sprintf(` ORDER BY a.id LIMIT $%d`, len(args))

	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("store: list accounts: %w", err)
	}
	defer rows.Close()

	var out []domain.AccountRow
	for rows.Next() {
		row, err := scanAccountRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// CountAccounts counts the accounts matching the filter (the page footer).
func (s *Store) CountAccounts(ctx context.Context, f domain.AccountFilter) (int, error) {
	where, args := accountFilterSQL(f)
	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*)`+accountRowFrom+where, args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("store: count accounts: %w", err)
	}
	return total, nil
}

// AccountAdminRow loads one account for the detail page.
func (s *Store) AccountAdminRow(ctx context.Context, id int64) (domain.AccountRow, error) {
	row, err := scanAccountRow(s.pool.QueryRow(ctx, `SELECT `+accountRowCols+accountRowFrom+` WHERE a.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.AccountRow{}, ErrNotFound
	}
	if err != nil {
		return domain.AccountRow{}, fmt.Errorf("store: account admin row %d: %w", id, err)
	}
	return row, nil
}

func accountFilterSQL(f domain.AccountFilter) (string, []any) {
	var (
		conds []string
		args  []any
	)
	add := func(cond string, value any) {
		args = append(args, value)
		conds = append(conds, fmt.Sprintf(cond, len(args)))
	}

	if q := strings.ToLower(strings.TrimSpace(f.Query)); q != "" {
		add(`a.name LIKE $%d`, q+"%")
	}
	if f.Role != "" && ValidRole(f.Role) {
		add(`a.role = $%d`, f.Role)
	}
	switch f.Blocked {
	case domain.BlockedNo:
		conds = append(conds, `a.is_blocked = FALSE`)
	case domain.BlockedYes:
		conds = append(conds, `a.is_blocked = TRUE`)
	}
	if f.AfterID > 0 {
		add(`a.id > $%d`, f.AfterID)
	}

	if len(conds) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

func scanAccountRow(row scanRow) (domain.AccountRow, error) {
	var a domain.AccountRow
	err := row.Scan(&a.ID, &a.Name, &a.Email, &a.Role, &a.IsBlocked, &a.BlockedReason,
		&a.DonateBalance, &a.CreatedAt, &a.RoleChangedAt, &a.RoleChangedBy, &a.Characters)
	return a, err
}

// SetAccountRole changes an account's role, enforcing the admin rules inside the
// transaction:
//
//   - the actor must be an admin (re-read here, not trusted from the caller);
//   - nobody changes their own role;
//   - demoting an admin is refused when it would leave no active admin.
//
// Returns the refreshed row. A no-op (same role) writes no audit.
func (s *Store) SetAccountRole(ctx context.Context, targetID, actorID int64, role, reason string) (domain.AccountRow, error) {
	if !ValidRole(role) {
		return domain.AccountRow{}, ErrInvalidRole
	}

	err := s.inTx(ctx, func(tx pgx.Tx) error {
		if err := requireAdmin(ctx, tx, actorID); err != nil {
			return err
		}
		if actorID == targetID {
			return ErrSelfChange
		}

		var oldRole string
		var blocked bool
		err := tx.QueryRow(ctx, `SELECT role, is_blocked FROM account WHERE id = $1 FOR UPDATE`, targetID).
			Scan(&oldRole, &blocked)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("store: set role: load %d: %w", targetID, err)
		}
		if oldRole == role {
			return nil
		}
		if oldRole == "admin" && role != "admin" {
			active, err := lockActiveAdmins(ctx, tx)
			if err != nil {
				return err
			}
			if active <= 1 {
				return ErrLastAdmin
			}
		}

		if _, err := tx.Exec(ctx,
			`UPDATE account SET role = $2, role_changed_at = now(), role_changed_by = $3 WHERE id = $1`,
			targetID, role, actorID); err != nil {
			return fmt.Errorf("store: set role %d: %w", targetID, err)
		}
		return accountAudit(ctx, tx, actorID, targetID, "set_role", oldRole, role, reason)
	})
	if err != nil {
		return domain.AccountRow{}, err
	}
	return s.AccountAdminRow(ctx, targetID)
}

// SetAccountBlocked blocks or unblocks an account. A moderator may only target
// players; nobody blocks themselves; blocking an admin requires another active
// admin to remain. Audited in the same transaction.
func (s *Store) SetAccountBlocked(ctx context.Context, targetID, actorID int64, blocked bool, reason string) (domain.AccountRow, error) {
	err := s.inTx(ctx, func(tx pgx.Tx) error {
		actorRole, err := actorRoleForUpdate(ctx, tx, actorID)
		if err != nil {
			return err
		}
		if actorRole != "moderator" && actorRole != "admin" {
			return ErrNotAdminActor
		}
		if actorID == targetID {
			return ErrSelfChange
		}

		var targetRole string
		var current bool
		err = tx.QueryRow(ctx, `SELECT role, is_blocked FROM account WHERE id = $1 FOR UPDATE`, targetID).
			Scan(&targetRole, &current)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("store: set blocked: load %d: %w", targetID, err)
		}
		if actorRole == "moderator" && targetRole != "player" {
			return ErrModeratorScope
		}
		if current == blocked {
			return nil
		}
		if blocked && targetRole == "admin" {
			active, err := lockActiveAdmins(ctx, tx)
			if err != nil {
				return err
			}
			if active <= 1 {
				return ErrLastAdmin
			}
		}

		// blocked_reason keeps the last reason while blocked and is cleared on
		// unblock, so the login error can explain the ban without the audit table.
		if _, err := tx.Exec(ctx,
			`UPDATE account SET is_blocked = $2, blocked_reason = CASE WHEN $2 THEN $3 ELSE '' END WHERE id = $1`,
			targetID, blocked, reason); err != nil {
			return fmt.Errorf("store: set blocked %d: %w", targetID, err)
		}
		oldValue, newValue := "active", "active"
		if current {
			oldValue = "blocked"
		}
		if blocked {
			newValue = "blocked"
		}
		return accountAudit(ctx, tx, actorID, targetID, "set_blocked", oldValue, newValue, reason)
	})
	if err != nil {
		return domain.AccountRow{}, err
	}
	return s.AccountAdminRow(ctx, targetID)
}

// CreateAccountAdmin inserts an account with an explicit role and audits the
// creation in the same transaction. passHash must already be argon2id. A unique
// violation on name is returned wrapped (the service maps it to NAME_TAKEN).
func (s *Store) CreateAccountAdmin(ctx context.Context, name, passHash, email, role string, actorID int64, reason string) (int64, error) {
	if !ValidRole(role) {
		return 0, ErrInvalidRole
	}
	var id int64
	err := s.inTx(ctx, func(tx pgx.Tx) error {
		if err := requireAdmin(ctx, tx, actorID); err != nil {
			return err
		}
		if err := tx.QueryRow(ctx,
			`INSERT INTO account (name, pass_hash, email, role) VALUES ($1,$2,$3,$4) RETURNING id`,
			name, passHash, email, role).Scan(&id); err != nil {
			return fmt.Errorf("store: create account %q: %w", name, err)
		}
		return accountAudit(ctx, tx, actorID, id, "create", "", role, reason)
	})
	if err != nil {
		return 0, err
	}
	return id, nil
}

// ListAccountAudit returns the newest audit rows for one account.
func (s *Store) ListAccountAudit(ctx context.Context, accountID int64, limit int) ([]domain.AccountAudit, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, actor_id, target_id, action, COALESCE(old_value, ''), COALESCE(new_value, ''), reason, created_at
		FROM account_admin_audit WHERE target_id = $1 ORDER BY id DESC LIMIT $2`, accountID, limit)
	if err != nil {
		return nil, fmt.Errorf("store: list account audit %d: %w", accountID, err)
	}
	defer rows.Close()
	var out []domain.AccountAudit
	for rows.Next() {
		var a domain.AccountAudit
		if err := rows.Scan(&a.ID, &a.ActorID, &a.TargetID, &a.Action, &a.OldValue, &a.NewValue, &a.Reason, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("store: scan account audit: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// --- helpers ---

// requireAdmin re-reads the actor's role inside the transaction (TOCTOU-safe).
func requireAdmin(ctx context.Context, tx pgx.Tx, actorID int64) error {
	role, err := actorRoleForUpdate(ctx, tx, actorID)
	if err != nil {
		return err
	}
	if role != "admin" {
		return ErrNotAdminActor
	}
	return nil
}

func actorRoleForUpdate(ctx context.Context, tx pgx.Tx, actorID int64) (string, error) {
	var role string
	err := tx.QueryRow(ctx, `SELECT role FROM account WHERE id = $1`, actorID).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("store: actor role %d: %w", actorID, err)
	}
	return role, nil
}

// lockActiveAdmins locks every active admin row and returns how many there are.
// Locking the set (not just counting) makes two concurrent demotions serialize:
// the second transaction sees the first one's result.
func lockActiveAdmins(ctx context.Context, tx pgx.Tx) (int, error) {
	rows, err := tx.Query(ctx, `SELECT id FROM account WHERE role = 'admin' AND NOT is_blocked ORDER BY id FOR UPDATE`)
	if err != nil {
		return 0, fmt.Errorf("store: lock admins: %w", err)
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return 0, fmt.Errorf("store: scan admin: %w", err)
		}
		count++
	}
	return count, rows.Err()
}

func accountAudit(ctx context.Context, tx pgx.Tx, actorID, targetID int64, action, oldValue, newValue, reason string) error {
	if _, err := tx.Exec(ctx, `
		INSERT INTO account_admin_audit (actor_id, target_id, action, old_value, new_value, reason)
		VALUES ($1,$2,$3,$4,$5,$6)`,
		actorID, targetID, action, oldValue, newValue, reason); err != nil {
		return fmt.Errorf("store: write account audit: %w", err)
	}
	return nil
}
