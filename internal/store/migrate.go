// Package store é a camada de persistência PostgreSQL do dbServer (pgx,
// guidelines §22). Ele aplica as migrations embutidas e grava dados convertidos
// accounts transactionally.
package store

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"openwyd-nextgen/internal/migrations"
)

// migrationLockKey serializa Migrate entre dbserver, webserver e containers
// avulsos de seed/convert que todos chamam store.Migrate em boots paralelos do
// docker compose; sem isso dois processos podem correr no mesmo *.up.sql (por
// exemplo ADD COLUMN já existe enquanto schema_migrations ainda está vazio para
// aquela versão).
const migrationLockKey int64 = 0x77327070

// Migrate aplica toda migration *.up.sql ainda não aplicada em ordem lexical,
// cada uma em sua própria transação, registrando as versões aplicadas em
// schema_migrations. É idempotente e seguro de rodar a cada boot.
// MigrateWithEnv aplica as migrations com o DSN privilegiado quando o deploy
// separa DDL do papel da aplicação (DB-01): W2PP_DB_MIGRATE_DSN pertence ao
// openwyd_migrator, dono dos objetos, enquanto W2PP_DB_DSN é a role de DML.
// Sem a variável, o comportamento é exatamente o de Migrate — um host local sem
// roles separadas continua funcionando.
func MigrateWithEnv(ctx context.Context, pool *pgxpool.Pool) error {
	dsn := os.Getenv("W2PP_DB_MIGRATE_DSN")
	if dsn == "" {
		return Migrate(ctx, pool)
	}
	mpool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return fmt.Errorf("store: connect migrator: %w", err)
	}
	defer mpool.Close()
	return Migrate(ctx, mpool)
}

func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("store: acquire migration conn: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock($1)`, migrationLockKey); err != nil {
		return fmt.Errorf("store: migration lock: %w", err)
	}
	defer func() {
		_, _ = conn.Exec(context.WithoutCancel(ctx), `SELECT pg_advisory_unlock($1)`, migrationLockKey)
	}()

	if _, err := conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`); err != nil {
		return fmt.Errorf("store: ensure schema_migrations: %w", err)
	}

	names, err := fs.Glob(migrations.FS, "*.up.sql")
	if err != nil {
		return fmt.Errorf("store: list migrations: %w", err)
	}
	sort.Strings(names)

	for _, name := range names {
		version := strings.TrimSuffix(name, ".up.sql")

		var applied bool
		if err := conn.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`, version,
		).Scan(&applied); err != nil {
			return fmt.Errorf("store: check migration %s: %w", version, err)
		}
		if applied {
			continue
		}

		sqlText, err := migrations.FS.ReadFile(name)
		if err != nil {
			return fmt.Errorf("store: read migration %s: %w", name, err)
		}
		if err := applyOne(ctx, conn, version, string(sqlText)); err != nil {
			return err
		}
	}
	return nil
}

func applyOne(ctx context.Context, conn *pgxpool.Conn, version, sqlText string) error {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("store: begin migration %s: %w", version, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, sqlText); err != nil {
		return fmt.Errorf("store: apply migration %s: %w", version, err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations(version) VALUES ($1)`, version); err != nil {
		return fmt.Errorf("store: record migration %s: %w", version, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("store: commit migration %s: %w", version, err)
	}
	return nil
}
