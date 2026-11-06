package main

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"openwyd-nextgen/internal/domain"
	"openwyd-nextgen/tmserver/internal/content"
)

// TestRegionColorDistinct: 52 named regions need 52 distinguishable tints. The
// golden-angle walk must not repeat a colour over the whole catalog, which a
// fixed palette could not promise.
func TestRegionColorDistinct(t *testing.T) {
	seen := make(map[string]int, 64)
	for i := 0; i < 64; i++ {
		c := regionColor(i)
		if len(c) != 7 || c[0] != '#' {
			t.Fatalf("regionColor(%d) = %q, want #rrggbb", i, c)
		}
		if other, dup := seen[c]; dup {
			t.Fatalf("regionColor(%d) = %q já usado por %d", i, c, other)
		}
		seen[c] = i
	}
}

// TestRegionColorStable pins the first tints: the mask PNG and the table swatch
// are drawn by different code paths, so a change here would silently
// desynchronise them.
func TestRegionColorStable(t *testing.T) {
	if got := regionColor(0); got != "#d36868" {
		t.Errorf("regionColor(0) = %q, want #d36868", got)
	}
	if got := regionColor(1); got != "#68d387" {
		t.Errorf("regionColor(1) = %q, want #68d387", got)
	}
}

// regionsFile is the tree the importer reads. The atlas no longer touches it at
// runtime; tests use it as the source of truth for parity.
func regionsFile() string {
	return filepath.Join("..", "..", "content", "TMsrv", "run", "Regions.txt")
}

// TestRegionsFromEntriesMapsRows proves the pure mapping: the JSON the portal
// consumes carries the same rectangle the row holds, tagged with the table it
// came from.
func TestRegionsFromEntriesMapsRows(t *testing.T) {
	got := regionsFromEntries([]domain.RegionEntry{
		{Name: "Big_Cubo", X1: 1272, Y1: 1427, X2: 1403, Y2: 1532},
		{Name: "Kefra_City", X1: 3199, Y1: 1667, X2: 3321, Y2: 1785},
	})
	if len(got) != 2 {
		t.Fatalf("regiões = %d, want 2", len(got))
	}
	want := region{Name: "Big_Cubo", X1: 1272, Y1: 1427, X2: 1403, Y2: 1532, Source: "region"}
	if got[0] != want {
		t.Errorf("região[0] = %+v, want %+v", got[0], want)
	}
	if got[1].Name != "Kefra_City" || got[1].Source != "region" {
		t.Errorf("região[1] = %+v", got[1])
	}
}

// TestRegionsFileMatchesImporterParity is the parity that can be proven without
// a database: the atlas' own Regions.txt parser and the importer's
// (content.LoadRegions) must agree on every field and on the file order, which
// is what used to set the palette.
func TestRegionsFileMatchesImporterParity(t *testing.T) {
	path := regionsFile()
	if _, err := os.Stat(path); err != nil {
		t.Skip("motivo: árvore tmserver/content ausente neste checkout")
	}

	atlas, err := loadRegionsFile(path)
	if err != nil {
		t.Fatalf("loadRegionsFile: %v", err)
	}
	importer, err := content.LoadRegions(path)
	if err != nil {
		t.Fatalf("content.LoadRegions: %v", err)
	}
	if len(atlas) != len(importer) {
		t.Fatalf("atlas = %d regiões, importador = %d", len(atlas), len(importer))
	}
	if len(atlas) != 52 {
		t.Errorf("regiões = %d, want 52", len(atlas))
	}
	for i := range atlas {
		a, e := atlas[i], importer[i]
		if a.Name != e.Name || a.X1 != int(e.X1) || a.Y1 != int(e.Y1) || a.X2 != int(e.X2) || a.Y2 != int(e.Y2) {
			t.Fatalf("região[%d] diverge: atlas=%+v importador=%+v", i, a, e)
		}
	}
}

// TestRegionsDBParityWithFile is the end-to-end parity: what the atlas reads from
// the `region` table must equal what the importer writes from Regions.txt. Needs
// a seeded database (importcontent), so it is skipped — with the reason declared
// — when the host has no DSN.
func TestRegionsDBParityWithFile(t *testing.T) {
	dsn := os.Getenv("W2PP_TEST_DSN")
	if dsn == "" {
		dsn = os.Getenv("W2PP_DB_DSN")
	}
	if dsn == "" {
		t.Skip("motivo: sem banco neste host (defina W2PP_TEST_DSN para rodar a paridade region × Regions.txt)")
	}
	path := regionsFile()
	if _, err := os.Stat(path); err != nil {
		t.Skip("motivo: árvore tmserver/content ausente neste checkout")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("conectar: %v", err)
	}
	defer pool.Close()

	fromDB, err := loadRegionsFromDB(ctx, pool)
	if err != nil {
		t.Fatalf("loadRegionsFromDB: %v", err)
	}
	fromFile, err := loadRegionsFile(path)
	if err != nil {
		t.Fatalf("loadRegionsFile: %v", err)
	}

	// A tabela é lida ordenada por nome; o arquivo, na ordem em que está. A
	// paridade é de conteúdo, não de ordem (a ordem do arquivo só definia cor).
	sort.Slice(fromFile, func(i, j int) bool { return fromFile[i].Name < fromFile[j].Name })
	if len(fromDB) != len(fromFile) {
		t.Fatalf("banco = %d regiões, arquivo = %d — rode o importcontent", len(fromDB), len(fromFile))
	}
	for i := range fromDB {
		db, file := fromDB[i], fromFile[i]
		if db.Name != file.Name || db.X1 != file.X1 || db.Y1 != file.Y1 || db.X2 != file.X2 || db.Y2 != file.Y2 {
			t.Fatalf("região[%d] diverge: banco=%+v arquivo=%+v", i, db, file)
		}
		if db.Source != "region" {
			t.Errorf("região %q com source %q, want \"region\"", db.Name, db.Source)
		}
	}
}
