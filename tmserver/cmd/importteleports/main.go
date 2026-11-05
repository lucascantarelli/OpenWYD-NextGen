// Command importteleports materializes the compiled teleport table into
// teleport_route (issue 42), so the moderator UI can list and edit all 23 routes
// instead of only the ones someone created by hand.
//
// It lives under tmserver/cmd (not dbserver/cmd) because it reads
// internal/teleportdata — Go forbids importing that from another tree only when
// it is under tmserver/internal, so keeping the tool next to its data keeps the
// picture honest. The command is idempotent: it upserts by origin tile.
//
// Usage:
//
//	go run ./tmserver/cmd/importteleports -dsn "postgres://postgres:dev@db:5432/postgres"
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"

	"openwyd-nextgen/internal/domain"
	"openwyd-nextgen/internal/store"
	"openwyd-nextgen/internal/teleportdata"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("importteleports failed", "err", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	dsn := flag.String("dsn", os.Getenv("W2PP_DB_DSN"), "PostgreSQL DSN (or W2PP_DB_DSN)")
	moderator := flag.Int64("moderator", 0, "account id recorded as updated_by (0 = seed)")
	flag.Parse()

	if *dsn == "" {
		return fmt.Errorf("-dsn (or W2PP_DB_DSN) is required")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, *dsn)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer pool.Close()
	if err := store.Migrate(ctx, pool); err != nil {
		return err
	}

	st := store.New(pool)
	routes := teleportdata.Routes()
	for _, r := range routes {
		if err := st.UpsertTeleportRoute(ctx, domain.TeleportRoute{
			SrcX: int(r.SrcX), SrcY: int(r.SrcY),
			DstX: int(r.DstX), DstY: int(r.DstY),
			Cost:        int(r.Cost),
			Requirement: r.Requirement,
			Enabled:     true,
			Note:        "import-teleports (tabela compilada)",
		}, *moderator); err != nil {
			return err
		}
	}

	seeded, err := st.ListTeleportRoutes(ctx)
	if err != nil {
		return err
	}
	logger.Info("teleport overlay seeded", "compiled", len(routes), "rows", len(seeded))
	return nil
}
