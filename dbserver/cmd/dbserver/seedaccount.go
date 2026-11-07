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

// runSeedAccount cria uma conta só com senha para teste local. O nome é
// canonicalizado em minúsculas (igual ao conversor, convert.Account); a senha é
// guardada só como hash argon2id (nunca texto plano). É idempotente: conta
// existente fica intocada e é apenas reportada, não é erro, para o script de
// bring-up local poder rodar várias vezes.
func runSeedAccount(args []string, logger *slog.Logger) error {
	fs := flag.NewFlagSet("seed-account", flag.ContinueOnError)
	name := fs.String("name", "", "login da conta (canonicalizado em minúsculas)")
	pass := fs.String("pass", "", "senha da conta (com hash argon2id)")
	dsn := fs.String("dsn", envOr("W2PP_DB_DSN", ""), "DSN do PostgreSQL (ou W2PP_DB_DSN)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *name == "" || *pass == "" {
		return fmt.Errorf("-name e -pass são obrigatórios")
	}
	if *dsn == "" {
		return fmt.Errorf("-dsn (ou W2PP_DB_DSN) é obrigatório")
	}
	canonical := strings.ToLower(*name)

	passHash, err := secret.HashSecret(*pass)
	if err != nil {
		return fmt.Errorf("gerar hash da senha: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, *dsn)
	if err != nil {
		return fmt.Errorf("conectar: %w", err)
	}
	defer pool.Close()
	if err := store.MigrateWithEnv(ctx, pool); err != nil {
		return err
	}
	s := store.New(pool)

	if _, err := s.AccountByName(ctx, canonical); err == nil {
		logger.Info("conta já existe — deixando intocada", "name", canonical)
		return nil
	} else if !errors.Is(err, store.ErrNotFound) {
		return fmt.Errorf("buscar conta: %w", err)
	}

	id, err := s.SaveAccount(ctx, domain.Account{Name: canonical, PassHash: passHash})
	if err != nil {
		return fmt.Errorf("salvar conta: %w", err)
	}
	logger.Info("conta semeada", "name", canonical, "id", id, "characters", 0)
	return nil
}
