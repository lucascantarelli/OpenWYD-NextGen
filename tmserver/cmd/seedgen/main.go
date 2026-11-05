// Command seedgen writes the reference-data seeds used by the baseline (issue
// 58, fase 3). It reads the allowlisted tables from a populated database and
// emits one SQL file per table, deterministic by construction: rows are ordered
// by their own canonical JSON, so two runs on the same data produce byte-equal
// files.
//
// The generated statement is schema-driven:
//
//	INSERT INTO t SELECT * FROM json_populate_record(NULL::t, $json${...}$json$::json)
//	ON CONFLICT DO NOTHING;
//
// That keeps the dump lossless (every column, including bytea blobs and jsonb)
// without a second copy of each table's column list that would silently rot when
// a migration adds a column.
//
// Usage:
//
//	seedgen -dsn <postgres-url> [-out internal/migrations] [-tables a,b]
//
// The default output is the migration package root, not a subdirectory: the
// runner embeds `*.sql` at the top level only, so a seed in `seeds/` would never
// be applied. Since the squash (fase 4) the generated files ARE the baseline
// data — regenerating them is how content updates reach a fresh database.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jeanluca/w2pp-openwyd/internal/store"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	fs := flag.NewFlagSet("seedgen", flag.ExitOnError)
	dsn := fs.String("dsn", os.Getenv("W2PP_DB_DSN"), "DSN do PostgreSQL (obrigatório)")
	out := fs.String("out", filepath.Join("internal", "migrations"), "diretório de saída")
	only := fs.String("tables", "", "lista separada por vírgula (vazio = todas as tabelas de referência)")
	prefix := fs.String("prefix", "", "prefixo do nome dos arquivos (ex.: 0003_)")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	if *dsn == "" {
		logger.Error("-dsn é obrigatório (ou W2PP_DB_DSN)")
		os.Exit(2)
	}

	tables, err := pickTables(*only)
	if err != nil {
		logger.Error("tabelas inválidas", "err", err)
		os.Exit(2)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	pool, err := pgxpool.New(ctx, *dsn)
	if err != nil {
		logger.Error("conectar", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	s := store.New(pool)

	if err := os.MkdirAll(*out, 0o755); err != nil {
		logger.Error("criar diretório", "err", err)
		os.Exit(1)
	}

	total := 0
	for _, table := range tables {
		rows, err := s.DumpTableJSON(ctx, table)
		if err != nil {
			logger.Error("dump", "tabela", table, "err", err)
			os.Exit(1)
		}
		// A lista de colunas vem do catálogo, sem identidade/geradas: a tabela
		// npc_definition tem id GENERATED ALWAYS, que recusa valor explícito.
		columns, err := s.SeedColumns(ctx, table)
		if err != nil {
			logger.Error("colunas", "tabela", table, "err", err)
			os.Exit(1)
		}
		columns = dropPlayerRefs(table, columns)
		path := filepath.Join(*out, *prefix+"seed_"+table+".up.sql")
		if err := os.WriteFile(path, renderSeed(table, columns, rows), 0o644); err != nil {
			logger.Error("escrever seed", "arquivo", path, "err", err)
			os.Exit(1)
		}
		logger.Info("seed gerado", "tabela", table, "linhas", len(rows), "arquivo", path)
		total += len(rows)
	}
	logger.Info("seeds gerados", "tabelas", len(tables), "linhas", total, "dir", *out)
}

// pickTables validates -tables against the store allowlist, so a typo fails
// loudly instead of silently producing an empty seed.
func pickTables(list string) ([]string, error) {
	allowed := store.SeedTables()
	if strings.TrimSpace(list) == "" {
		return allowed, nil
	}
	var out []string
	for _, name := range strings.Split(list, ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		found := false
		for _, a := range allowed {
			if a == name {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("tabela %q não é de referência (válidas: %s)", name, strings.Join(allowed, ", "))
		}
		out = append(out, name)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("nenhuma tabela selecionada")
	}
	return out, nil
}

// playerRefColumns lists reference columns that point at PLAYER data. A seed
// must not carry them: npc_definition.updated_by references account(id), and a
// fresh database has no accounts yet — the FK aborts the whole seed. NULL is
// also the truthful value: the row came from content, not from a moderator.
var playerRefColumns = map[string]map[string]bool{
	"npc_definition": {"updated_by": true},
}

// dropPlayerRefs removes player-referencing columns from the seed entirely.
// They are omitted rather than written as NULL: the column list of an INSERT
// cannot carry an expression (`INSERT INTO t (..., NULL AS x)` is a syntax
// error), and omitting is semantically identical — the column default is NULL.
func dropPlayerRefs(table string, columns []string) []string {
	out := make([]string, 0, len(columns))
	for _, col := range columns {
		if playerRefColumns[table][col] {
			continue
		}
		out = append(out, col)
	}
	return out
}

// renderSeed builds the SQL file. No timestamp in the header: the artifact must
// be byte-identical across runs so a re-dump that changes nothing shows no diff.
//
// The column list is explicit (identity columns excluded) and repeated per row:
// the alternative, `SELECT *`, dies on any table whose id is GENERATED ALWAYS.
func renderSeed(table string, columns []string, rows [][]byte) []byte {
	cols := strings.Join(columns, ", ")
	var b strings.Builder
	fmt.Fprintf(&b, "-- Gerado por `tmserver/cmd/seedgen` (issue 58) — NÃO EDITAR À MÃO.\n")
	fmt.Fprintf(&b, "-- Tabela: %s (%d linhas). Regerar: `seedgen -dsn <dsn>`.\n", table, len(rows))
	fmt.Fprintf(&b, "-- Dados de referência: nenhum dado de jogador entra aqui (allowlist em internal/store/seed.go).\n\n")
	for _, row := range rows {
		fmt.Fprintf(&b, "INSERT INTO %s (%s)\n", table, cols)
		fmt.Fprintf(&b, "SELECT %s FROM json_populate_record(NULL::%s, $json$%s$json$::json)\n", cols, table, row)
		b.WriteString("ON CONFLICT DO NOTHING;\n")
	}
	return []byte(b.String())
}
