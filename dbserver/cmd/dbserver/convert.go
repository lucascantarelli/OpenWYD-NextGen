// Conversão one-shot do formato de conta legado (AccountFile ×.txt) para o
// Postgres. Vive fora do `main` para que o ponto de entrada caiba em flags →
// comando → erro (REFACTOR-01).
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"openwyd-nextgen/dbserver/internal/convert"
	"openwyd-nextgen/internal/store"
)

// runConvert lê uma árvore legada de contas e, com -dsn, persiste o resultado.
// Sem -dsn é dry run: converte, relata e não toca no banco.
func runConvert(args []string, logger *slog.Logger) error {
	fs := flag.NewFlagSet("convert", flag.ContinueOnError)
	accounts := fs.String("accounts", "", "path to the legacy account/ directory")
	dsn := fs.String("dsn", "", "PostgreSQL DSN; if set, persist (else dry run)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *accounts == "" {
		return fmt.Errorf("-accounts is required")
	}

	rep, err := convert.WalkAccounts(*accounts)
	if err != nil {
		return err
	}
	logger.Info("conversion scan complete",
		"total", rep.Total, "converted", rep.Converted, "skipped", rep.Skipped, "failed", rep.Failed)
	for _, r := range rep.Results {
		switch {
		case r.Err != nil:
			logger.Warn("failed", "path", r.Path, "err", r.Err)
		case r.Skipped != "":
			logger.Warn("skipped", "path", r.Path, "reason", r.Skipped)
		}
	}

	if *dsn == "" {
		logger.Info("dry run (no -dsn) — not persisting")
		if rep.Failed > 0 {
			return fmt.Errorf("%d file(s) failed to convert", rep.Failed)
		}
		return nil
	}
	return persist(rep, *dsn, logger)
}

// persist aplica as migrations e grava cada conta convertida; a primeira falha
// interrompe a importação.
func persist(rep *convert.Report, dsn string, logger *slog.Logger) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer pool.Close()

	if err := store.MigrateWithEnv(ctx, pool); err != nil {
		return err
	}
	s := store.New(pool)

	imported := 0
	for _, r := range rep.Results {
		if r.Account == nil {
			continue
		}
		if _, err := s.SaveAccount(ctx, *r.Account); err != nil {
			return fmt.Errorf("persist %q: %w", r.Account.Name, err)
		}
		imported++
	}
	logger.Info("import complete", "imported", imported)
	return nil
}
