// Criação de uma conta password-only para o bring-up local; separada do `main`
// pelo mesmo motivo do `convert` (REFACTOR-01).
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"openwyd-nextgen/internal/domain"
	"openwyd-nextgen/internal/secret"
	"openwyd-nextgen/internal/store"
)

// runSeedAccount creates one password-only account for local testing. The name
// is canonicalized to lowercase (matching the converter, convert.Account); the
// password is stored only as an argon2id hash (never plaintext). It is
// idempotent: an existing account is left untouched and reported, not an error,
// so the local bring-up script can run repeatedly.
func runSeedAccount(args []string, logger *slog.Logger) error {
	fs := flag.NewFlagSet("seed-account", flag.ContinueOnError)
	name := fs.String("name", "", "account login (canonicalized to lowercase)")
	pass := fs.String("pass", "", "account password (hashed with argon2id)")
	dsn := fs.String("dsn", envOr("W2PP_DB_DSN", ""), "PostgreSQL DSN (or W2PP_DB_DSN)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *name == "" || *pass == "" {
		return fmt.Errorf("-name and -pass are required")
	}
	if *dsn == "" {
		return fmt.Errorf("-dsn (or W2PP_DB_DSN) is required")
	}
	canonical := strings.ToLower(*name)

	passHash, err := secret.HashSecret(*pass)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, *dsn)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer pool.Close()
	if err := store.MigrateWithEnv(ctx, pool); err != nil {
		return err
	}
	s := store.New(pool)

	if _, err := s.AccountByName(ctx, canonical); err == nil {
		logger.Info("account already exists — leaving untouched", "name", canonical)
		return nil
	} else if !errors.Is(err, store.ErrNotFound) {
		return fmt.Errorf("lookup: %w", err)
	}

	id, err := s.SaveAccount(ctx, domain.Account{Name: canonical, PassHash: passHash})
	if err != nil {
		return fmt.Errorf("save account: %w", err)
	}
	logger.Info("account seeded", "name", canonical, "id", id, "characters", 0)
	return nil
}
