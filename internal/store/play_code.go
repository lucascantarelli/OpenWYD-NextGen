package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// maxActivePlayCodes limita os códigos não expirados por conta: cada clique em
// "jogar no navegador" emite um, e só os poucos mais novos ainda podem estar em voo.
const maxActivePlayCodes = 5

// PlayCodeIssue é o resultado de IssuePlayCode.
type PlayCodeIssue struct {
	Name    string // nome canônico da conta, o campo de login com que o código se pareia
	Blocked bool   // true: nenhum código foi gravado
}

// IssuePlayCode guarda o hash de um código de login de uso único para uma conta
// existente e desbloqueada (web client ADR 017). Descarta os códigos expirados da
// conta e mantém no máximo maxActivePlayCodes, os mais antigos primeiro fora.
// Devolve ErrNotFound quando nenhuma conta tem esse id.
func (s *Store) IssuePlayCode(ctx context.Context, accountID int64, codeHash []byte, expiresAt time.Time) (PlayCodeIssue, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return PlayCodeIssue{}, fmt.Errorf("store: begin issue play code: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// FOR UPDATE serializa emissões concorrentes para uma conta, então o limite se mantém.
	var out PlayCodeIssue
	err = tx.QueryRow(ctx, `SELECT name, is_blocked FROM account WHERE id = $1 FOR UPDATE`, accountID).
		Scan(&out.Name, &out.Blocked)
	if errors.Is(err, pgx.ErrNoRows) {
		return PlayCodeIssue{}, ErrNotFound
	}
	if err != nil {
		return PlayCodeIssue{}, fmt.Errorf("store: lock account %d for play code: %w", accountID, err)
	}
	if out.Blocked {
		return out, nil
	}
	if _, err := tx.Exec(ctx,
		`DELETE FROM account_play_code WHERE account_id = $1 AND expires_at <= now()`, accountID); err != nil {
		return PlayCodeIssue{}, fmt.Errorf("store: prune expired play codes: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM account_play_code WHERE code_hash IN (
			SELECT code_hash FROM account_play_code WHERE account_id = $1
			 ORDER BY created_at DESC OFFSET $2)`, accountID, maxActivePlayCodes-1); err != nil {
		return PlayCodeIssue{}, fmt.Errorf("store: cap play codes: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO account_play_code (code_hash, account_id, expires_at) VALUES ($1, $2, $3)`,
		codeHash, accountID, expiresAt); err != nil {
		return PlayCodeIssue{}, fmt.Errorf("store: insert play code: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return PlayCodeIssue{}, fmt.Errorf("store: commit play code: %w", err)
	}
	return out, nil
}

// ConsumePlayCode informa se codeHash é um código não expirado de accountID
// e o apaga no mesmo comando, então um código loga no máximo uma vez mesmo
// sob logins concorrentes.
func (s *Store) ConsumePlayCode(ctx context.Context, accountID int64, codeHash []byte) (bool, error) {
	var one int
	err := s.pool.QueryRow(ctx, `
		DELETE FROM account_play_code
		WHERE account_id = $1 AND code_hash = $2 AND expires_at > now()
		RETURNING 1`, accountID, codeHash).Scan(&one)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("store: consume play code for account %d: %w", accountID, err)
	}
	return true, nil
}
