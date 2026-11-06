//go:build integration

package store

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"openwyd-nextgen/internal/migrations"
)

// TestBaselineDown prova o `down` do baseline num banco descartável: aplica o up,
// confere que o schema tem os objetos, roda o down e confere pelo catálogo que
// nada sobrou — tabelas, índices, sequências, tipos compostos e to_regclass.
//
// É o mesmo caminho dos arquivos embedded (`internal/migrations`), então roda no
// CI com Postgres limpo e localmente com W2PP_TEST_DSN. Não usa `docker exec`.
func TestBaselineDown(t *testing.T) {
	dsn := os.Getenv("W2PP_TEST_DSN")
	if dsn == "" {
		t.Skip("motivo: ambiente — W2PP_TEST_DSN não definido; roda no job de banco (CI-02)")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	admin, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect admin: %v", err)
	}

	name := fmt.Sprintf("downcheck_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize()); err != nil {
		t.Fatalf("create scratch database: %v", err)
	}
	t.Cleanup(func() {
		dropCtx, dropCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer dropCancel()
		if _, err := admin.Exec(dropCtx, "DROP DATABASE IF EXISTS "+pgx.Identifier{name}.Sanitize()+" WITH (FORCE)"); err != nil {
			t.Logf("aviso: não derrubou %s: %v", name, err)
		}
		if err := admin.Close(dropCtx); err != nil {
			t.Logf("aviso: fechar conexão admin: %v", err)
		}
	})

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("parse dsn: %v", err)
	}
	cfg.ConnConfig.Database = name
	conn, err := pgx.ConnectConfig(ctx, cfg.ConnConfig)
	if err != nil {
		t.Fatalf("connect scratch: %v", err)
	}
	defer conn.Close(ctx)

	up, err := migrations.FS.ReadFile("0001_schema.up.sql")
	if err != nil {
		t.Fatalf("read up: %v", err)
	}
	down, err := migrations.FS.ReadFile("0001_schema.down.sql")
	if err != nil {
		t.Fatalf("read down: %v", err)
	}

	if _, err := conn.Exec(ctx, string(up)); err != nil {
		t.Fatalf("apply up: %v", err)
	}

	count := func(query string) int {
		t.Helper()
		var n int
		if err := conn.QueryRow(ctx, query).Scan(&n); err != nil {
			t.Fatalf("count (%s): %v", query, err)
		}
		return n
	}

	qTables := `SELECT count(*) FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
	            WHERE n.nspname = 'public' AND c.relkind = 'r'`
	qIndexes := `SELECT count(*) FROM pg_indexes WHERE schemaname = 'public'`
	qSeqs := `SELECT count(*) FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
	          WHERE n.nspname = 'public' AND c.relkind = 'S'`
	qTypes := `SELECT count(*) FROM pg_type t JOIN pg_namespace n ON n.oid = t.typnamespace
	           WHERE n.nspname = 'public' AND t.typtype = 'c'`

	upTables, upIndexes := count(qTables), count(qIndexes)
	upSeqs, upTypes := count(qSeqs), count(qTypes)
	t.Logf("up: tables=%d indexes=%d sequences=%d composite_types=%d", upTables, upIndexes, upSeqs, upTypes)
	if upTables < 40 || upIndexes < 40 {
		t.Fatalf("up não criou o baseline esperado: tables=%d indexes=%d", upTables, upIndexes)
	}

	if _, err := conn.Exec(ctx, string(down)); err != nil {
		t.Fatalf("apply down: %v", err)
	}

	downTables, downIndexes := count(qTables), count(qIndexes)
	downSeqs, downTypes := count(qSeqs), count(qTypes)
	var rel *string
	if err := conn.QueryRow(ctx, `SELECT to_regclass('public.characters')::text`).Scan(&rel); err != nil {
		t.Fatalf("to_regclass: %v", err)
	}
	relText := "NULL"
	if rel != nil {
		relText = *rel
	}
	t.Logf("down: tables=%d indexes=%d sequences=%d composite_types=%d characters=%s",
		downTables, downIndexes, downSeqs, downTypes, relText)

	if downTables != 0 || downIndexes != 0 || downSeqs != 0 || downTypes != 0 || rel != nil {
		t.Fatalf("down deixou objeto órfão: tables=%d indexes=%d sequences=%d types=%d characters=%s",
			downTables, downIndexes, downSeqs, downTypes, relText)
	}
	t.Log("ok: up+down do baseline verificado (schema vazio, sem objeto órfão)")
}
