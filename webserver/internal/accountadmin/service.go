// Package accountadmin holds the account moderation logic behind /contas (issue
// 32): list/search, role changes, block/unblock and account creation, all
// audited.
//
// Authorization is two-layered on purpose. The service re-reads the caller's role
// from Postgres (the BFF's session is only a UI hint) and the store re-reads it
// again inside the write transaction, where the "last active admin" and
// "no self-change" rules are enforced. Nothing here touches live game state:
// `account.role` and `account.is_blocked` are read by the next login, so a change
// takes effect on the next session.
package accountadmin

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"

	"openwyd-nextgen/internal/domain"
	"openwyd-nextgen/internal/secret"
	"openwyd-nextgen/internal/store"
)

// Store is the persistence surface the service needs (satisfied by *store.Store).
type Store interface {
	AccountRole(ctx context.Context, id int64) (string, error)
	ListAccounts(ctx context.Context, f domain.AccountFilter) ([]domain.AccountRow, error)
	CountAccounts(ctx context.Context, f domain.AccountFilter) (int, error)
	AccountAdminRow(ctx context.Context, id int64) (domain.AccountRow, error)
	ListAccountAudit(ctx context.Context, accountID int64, limit int) ([]domain.AccountAudit, error)
	SetAccountRole(ctx context.Context, targetID, actorID int64, role, reason string) (domain.AccountRow, error)
	SetAccountBlocked(ctx context.Context, targetID, actorID int64, blocked bool, reason string) (domain.AccountRow, error)
	CreateAccountAdmin(ctx context.Context, name, passHash, email, role string, actorID int64, reason string) (int64, error)
}

// Result is the business outcome of an operation; infra failures are errors.
type Result int

const (
	// OK means the operation succeeded.
	OK Result = iota
	// Forbidden means the caller lacks the role (or scope) for the operation.
	Forbidden
	// Invalid means the request failed validation or a business rule.
	Invalid
	// NotFound means the target account does not exist.
	NotFound
)

const (
	minNameLen = 4
	maxNameLen = 12
	minPassLen = 4
	// minReasonLen keeps the audit trail explainable; shorter notes say nothing.
	minReasonLen = 10
)

// Service implements the account moderation surface.
type Service struct{ store Store }

// New builds the service over the given store.
func New(s Store) *Service { return &Service{store: s} }

// List returns one page of accounts plus the total matching the filter.
func (s *Service) List(ctx context.Context, actorID int64, f domain.AccountFilter) (Result, []domain.AccountRow, int, error) {
	if r, err := s.requireStaff(ctx, actorID); r != OK || err != nil {
		return r, nil, 0, err
	}
	rows, err := s.store.ListAccounts(ctx, f)
	if err != nil {
		return Invalid, nil, 0, fmt.Errorf("accountadmin: list: %w", err)
	}
	total, err := s.store.CountAccounts(ctx, f)
	if err != nil {
		return Invalid, nil, 0, fmt.Errorf("accountadmin: count: %w", err)
	}
	return OK, rows, total, nil
}

// Get returns one account and its audit trail.
func (s *Service) Get(ctx context.Context, actorID, accountID int64) (Result, domain.AccountRow, []domain.AccountAudit, error) {
	if r, err := s.requireStaff(ctx, actorID); r != OK || err != nil {
		return r, domain.AccountRow{}, nil, err
	}
	if accountID <= 0 {
		return Invalid, domain.AccountRow{}, nil, nil
	}
	row, err := s.store.AccountAdminRow(ctx, accountID)
	if errors.Is(err, store.ErrNotFound) {
		return NotFound, domain.AccountRow{}, nil, nil
	}
	if err != nil {
		return Invalid, domain.AccountRow{}, nil, fmt.Errorf("accountadmin: get %d: %w", accountID, err)
	}
	audit, err := s.store.ListAccountAudit(ctx, accountID, 50)
	if err != nil {
		return Invalid, domain.AccountRow{}, nil, fmt.Errorf("accountadmin: audit %d: %w", accountID, err)
	}
	return OK, row, audit, nil
}

// SetRole promotes/demotes an account. Admin-only; the store owns the
// self-change and last-admin rules.
func (s *Service) SetRole(ctx context.Context, actorID, accountID int64, role, reason string) (Result, domain.AccountRow, string, error) {
	if r, err := s.requireAdmin(ctx, actorID); r != OK || err != nil {
		return r, domain.AccountRow{}, "", err
	}
	role = strings.TrimSpace(role)
	if !store.ValidRole(role) {
		return Invalid, domain.AccountRow{}, "papel inválido", nil
	}
	if accountID <= 0 {
		return Invalid, domain.AccountRow{}, "conta inválida", nil
	}
	if msg := validateReason(reason); msg != "" {
		return Invalid, domain.AccountRow{}, msg, nil
	}

	row, err := s.store.SetAccountRole(ctx, accountID, actorID, role, strings.TrimSpace(reason))
	if err != nil {
		return explain(err, accountID)
	}
	return OK, row, "", nil
}

// SetBlocked blocks or unblocks an account. Moderators may only manage players;
// the store enforces that and the last-admin rule.
func (s *Service) SetBlocked(ctx context.Context, actorID, accountID int64, blocked bool, reason string) (Result, domain.AccountRow, string, error) {
	if r, err := s.requireStaff(ctx, actorID); r != OK || err != nil {
		return r, domain.AccountRow{}, "", err
	}
	if accountID <= 0 {
		return Invalid, domain.AccountRow{}, "conta inválida", nil
	}
	if msg := validateReason(reason); msg != "" {
		return Invalid, domain.AccountRow{}, msg, nil
	}

	row, err := s.store.SetAccountBlocked(ctx, accountID, actorID, blocked, strings.TrimSpace(reason))
	if err != nil {
		return explain(err, accountID)
	}
	return OK, row, "", nil
}

// Create registers a new account with an explicit role (admin only) and audits
// it. Name/password/email follow the same rules as the public sign-up.
func (s *Service) Create(ctx context.Context, actorID int64, name, password, email, role, reason string) (Result, int64, string, error) {
	if r, err := s.requireAdmin(ctx, actorID); r != OK || err != nil {
		return r, 0, "", err
	}
	canonical := strings.ToLower(strings.TrimSpace(name))
	email = strings.TrimSpace(email)
	role = strings.TrimSpace(role)
	if !validName(canonical) {
		return Invalid, 0, "nome deve ter de 4 a 12 letras/números", nil
	}
	if len(password) < minPassLen {
		return Invalid, 0, "senha curta demais (mínimo 4)", nil
	}
	if !validEmail(email) {
		return Invalid, 0, "e-mail inválido", nil
	}
	if !store.ValidRole(role) {
		return Invalid, 0, "papel inválido", nil
	}
	if msg := validateReason(reason); msg != "" {
		return Invalid, 0, msg, nil
	}

	passHash, err := secret.HashSecret(password)
	if err != nil {
		return Invalid, 0, "", fmt.Errorf("accountadmin: hash password: %w", err)
	}
	id, err := s.store.CreateAccountAdmin(ctx, canonical, passHash, email, role, actorID, strings.TrimSpace(reason))
	if err != nil {
		if isUniqueViolation(err) {
			return Invalid, 0, "já existe uma conta com esse nome", nil
		}
		if r, _, msg, mapped := explain(err, 0); r != OK {
			return r, 0, msg, mapped
		}
		return Invalid, 0, "", fmt.Errorf("accountadmin: create %q: %w", canonical, err)
	}
	return OK, id, "", nil
}

// --- helpers ---

func (s *Service) requireStaff(ctx context.Context, actorID int64) (Result, error) {
	if actorID <= 0 {
		return Forbidden, nil
	}
	role, err := s.store.AccountRole(ctx, actorID)
	if errors.Is(err, store.ErrNotFound) {
		return Forbidden, nil
	}
	if err != nil {
		return Invalid, fmt.Errorf("accountadmin: role lookup %d: %w", actorID, err)
	}
	if role != "moderator" && role != "admin" {
		return Forbidden, nil
	}
	return OK, nil
}

func (s *Service) requireAdmin(ctx context.Context, actorID int64) (Result, error) {
	if actorID <= 0 {
		return Forbidden, nil
	}
	role, err := s.store.AccountRole(ctx, actorID)
	if errors.Is(err, store.ErrNotFound) {
		return Forbidden, nil
	}
	if err != nil {
		return Invalid, fmt.Errorf("accountadmin: role lookup %d: %w", actorID, err)
	}
	if role != "admin" {
		return Forbidden, nil
	}
	return OK, nil
}

// explain maps store sentinel errors to a Result plus the message the operator
// sees. Unknown errors are returned as Invalid with the message empty, so the
// caller wraps them as infra failures.
func explain(err error, accountID int64) (Result, domain.AccountRow, string, error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		return NotFound, domain.AccountRow{}, "conta não encontrada", nil
	case errors.Is(err, store.ErrSelfChange):
		return Invalid, domain.AccountRow{}, "não é possível alterar o próprio papel ou bloqueio", nil
	case errors.Is(err, store.ErrLastAdmin):
		return Invalid, domain.AccountRow{}, "esta é a última conta admin ativa — promova outra antes", nil
	case errors.Is(err, store.ErrModeratorScope):
		return Forbidden, domain.AccountRow{}, "moderador só pode bloquear contas de jogador", nil
	case errors.Is(err, store.ErrNotAdminActor):
		return Forbidden, domain.AccountRow{}, "só um admin pode alterar papéis", nil
	case errors.Is(err, store.ErrInvalidRole):
		return Invalid, domain.AccountRow{}, "papel inválido", nil
	default:
		_ = accountID
		return Invalid, domain.AccountRow{}, "", fmt.Errorf("accountadmin: %w", err)
	}
}

func validateReason(reason string) string {
	if len(strings.TrimSpace(reason)) < minReasonLen {
		return fmt.Sprintf("informe o motivo (mínimo %d caracteres) — ele vai para a auditoria", minReasonLen)
	}
	return ""
}

func validName(name string) bool {
	if len(name) < minNameLen || len(name) > maxNameLen {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') {
			return false
		}
	}
	return true
}

func validEmail(email string) bool {
	if email == "" {
		return true
	}
	_, err := mail.ParseAddress(email)
	return err == nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
