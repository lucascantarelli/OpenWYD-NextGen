// Command importcontent seeds the content catalogs into PostgreSQL (issue 58,
// fase 1): item_catalog (ItemList.csv), skill_catalog (SkillData.csv),
// npc_template (Release/TMsrv/run/npc/*) and world_item_seed (InitItem.csv).
//
// It lives under tmserver/cmd — not dbserver — because the CSV/STRUCT_MOB
// parsers are in tmserver/internal/content, and Go's internal visibility rule
// forbids a sibling (dbserver) from importing them. The command is a one-shot
// tool: it opens its own pool, applies migrations, upserts the catalogs and
// exits. The tmServer never calls it.
//
// Usage:
//
//	importcontent -content <Release/> [-dsn <postgres-url>] [-sets item,skill,npc-template,world-item] [-check]
//
// Without -dsn it is a dry run (parses and reports counts). -check compares the
// database counts against the files and exits 1 on any divergence, so CI can
// fail when someone edits content without reimporting.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"openwyd-nextgen/internal/domain"
	"openwyd-nextgen/internal/npctemplate"
	"openwyd-nextgen/internal/savefmt"
	"openwyd-nextgen/internal/store"
	"openwyd-nextgen/internal/textenc"
	"openwyd-nextgen/tmserver/internal/content"
)

// defaultSets is what a bare run imports.
const defaultSets = "item,skill,npc-template,world-item"

// catalog is everything parsed from Release/ in one pass.
type catalog struct {
	Items      []domain.ItemCatalogEntry
	Skills     []domain.SkillCatalogEntry
	Templates  []domain.NPCTemplateEntry
	WorldItems []domain.WorldItemSeedEntry
	// Skipped lists files that could not be decoded (wrong size). They are
	// reported but never fatal: one stray file in npc/ must not block the
	// catalog.
	Skipped []string
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	fs := flag.NewFlagSet("importcontent", flag.ExitOnError)
	contentDir := fs.String("content", "", "caminho da árvore Release/")
	dsn := fs.String("dsn", os.Getenv("W2PP_DB_DSN"), "DSN do PostgreSQL; vazio = dry-run")
	sets := fs.String("sets", defaultSets, "conjuntos: item,skill,npc-template,world-item")
	check := fs.Bool("check", false, "não grava: compara banco com os arquivos e sai 1 se divergir")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	if *contentDir == "" {
		logger.Error("-content é obrigatório")
		os.Exit(2)
	}
	want, err := parseSets(*sets)
	if err != nil {
		logger.Error("conjuntos inválidos", "err", err)
		os.Exit(2)
	}

	cat, err := buildCatalog(*contentDir, want, logger)
	if err != nil {
		logger.Error("ler conteúdo", "err", err)
		os.Exit(1)
	}
	logger.Info("conteúdo lido",
		"items", len(cat.Items), "skills", len(cat.Skills),
		"templates", len(cat.Templates), "world_items", len(cat.WorldItems),
		"templates_ignorados", len(cat.Skipped))

	if *dsn == "" {
		logger.Info("dry-run (sem -dsn) — nada gravado")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	pool, err := pgxpool.New(ctx, *dsn)
	if err != nil {
		logger.Error("conectar", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := store.Migrate(ctx, pool); err != nil {
		logger.Error("migrar", "err", err)
		os.Exit(1)
	}
	s := store.New(pool)

	if *check {
		os.Exit(runCheck(ctx, s, cat, want, logger))
	}

	if want["item"] {
		n, err := s.UpsertItemCatalog(ctx, cat.Items)
		if err != nil {
			logger.Error("importar itens", "err", err)
			os.Exit(1)
		}
		logger.Info("itens importados", "rows", n)
	}
	if want["skill"] {
		n, err := s.UpsertSkillCatalog(ctx, cat.Skills)
		if err != nil {
			logger.Error("importar skills", "err", err)
			os.Exit(1)
		}
		logger.Info("skills importadas", "rows", n)
	}
	if want["npc-template"] {
		n, err := s.UpsertNPCTemplates(ctx, cat.Templates)
		if err != nil {
			logger.Error("importar templates", "err", err)
			os.Exit(1)
		}
		logger.Info("templates importados", "rows", n)
	}
	if want["world-item"] {
		n, err := s.UpsertWorldItemSeeds(ctx, cat.WorldItems)
		if err != nil {
			logger.Error("importar objetos do mundo", "err", err)
			os.Exit(1)
		}
		logger.Info("objetos do mundo importados", "rows", n)
	}
}

// parseSets validates and expands the -sets flag.
func parseSets(list string) (map[string]bool, error) {
	valid := map[string]bool{"item": true, "skill": true, "npc-template": true, "world-item": true}
	want := make(map[string]bool, len(valid))
	for _, name := range strings.Split(list, ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if !valid[name] {
			return nil, fmt.Errorf("conjunto desconhecido %q (use %s)", name, defaultSets)
		}
		want[name] = true
	}
	if len(want) == 0 {
		return nil, fmt.Errorf("nenhum conjunto selecionado (use %s)", defaultSets)
	}
	return want, nil
}

// buildCatalog parses the selected content sets from the Release tree.
func buildCatalog(contentDir string, want map[string]bool, logger *slog.Logger) (catalog, error) {
	var cat catalog

	if want["item"] {
		l, err := content.LoadItemList(filepath.Join(contentDir, "Common", "ItemList.csv"))
		if err != nil {
			return catalog{}, err
		}
		cat.Items = buildItems(l)
	}
	if want["skill"] {
		s, err := content.LoadSkillData(filepath.Join(contentDir, "Common", "SkillData.csv"))
		if err != nil {
			return catalog{}, err
		}
		cat.Skills = buildSkills(s)
	}
	if want["npc-template"] {
		tpls, skipped, err := buildTemplates(contentDir, logger)
		if err != nil {
			return catalog{}, err
		}
		cat.Templates, cat.Skipped = tpls, skipped
	}
	if want["world-item"] {
		items, err := content.LoadInitItems(filepath.Join(contentDir, "TMsrv", "run", "InitItem.csv"))
		if err != nil {
			return catalog{}, err
		}
		cat.WorldItems = buildWorldItems(items)
	}
	return cat, nil
}

// buildItems joins content.ItemList's derived accessors into one row per item.
// Every accessor is computed from the same raw Fields, so the row is exactly
// what the tmServer would derive from the CSV at boot.
func buildItems(l *content.ItemList) []domain.ItemCatalogEntry {
	prices := l.Prices()
	grades := l.Grades()
	uniques := l.Uniques()
	extras := l.Extras()
	positions := l.Positions()
	reqs := l.Requirements()
	effects := l.BaseEffects()
	volatiles := l.Volatiles()
	ranges := l.Ranges()

	entries := l.Entries()
	out := make([]domain.ItemCatalogEntry, 0, len(entries))
	for _, e := range entries {
		// The CSVs are Windows-1252; the database only accepts UTF-8.
		fields := textenc.Slice(e.Fields)
		it := domain.ItemCatalogEntry{
			Index:  e.Index,
			Name:   textenc.ToUTF8(e.Name),
			Fields: fields,
			// Slice vazio, nunca nil: json.Marshal de nil vira `null`, que o
			// json_populate_record relê como SQL NULL e viola o NOT NULL do
			// seed (bug real: 1.5k itens sem efeito ficaram com jsonb 'null').
			Effects:  []domain.ItemCatalogEffect{},
			Price:    prices[e.Index],
			Grade:    int16(grades[e.Index]),
			UniqueID: int16(uniques[e.Index]),
			Extra:    int32(extras[e.Index]),
			Pos:      int16(positions[e.Index]),
			SHA256:   rowHash(fields),
		}
		if req, ok := reqs[e.Index]; ok {
			it.ReqLevel, it.ReqStr, it.ReqInt, it.ReqDex, it.ReqCon =
				req.Lvl, req.Str, req.Int, req.Dex, req.Con
		}
		if v, ok := volatiles[e.Index]; ok {
			vv := int16(v)
			it.Volatile = &vv
		}
		if r, ok := ranges[e.Index]; ok {
			rr := r
			it.AttackRange = &rr
		}
		for _, eff := range effects[e.Index] {
			it.Effects = append(it.Effects, domain.ItemCatalogEffect{Eff: eff.Eff, Val: eff.Val})
		}
		out = append(out, it)
	}
	return out
}

// buildSkills converts every skill index in the spell table. Indexes are
// sparse, so it walks the whole index space and keeps the hits.
func buildSkills(s *content.SkillData) []domain.SkillCatalogEntry {
	out := make([]domain.SkillCatalogEntry, 0, s.Len())
	for i := 0; i < content.MaxSkillIndex; i++ {
		sp, ok := s.Get(i)
		if !ok {
			continue
		}
		out = append(out, domain.SkillCatalogEntry{
			Index:             sp.Index,
			Name:              textenc.ToUTF8(sp.Name),
			SkillPoint:        int16(sp.SkillPoint),
			TargetType:        int16(sp.TargetType),
			ManaSpent:         int16(sp.ManaSpent),
			Delay:             int16(sp.Delay),
			Range:             int16(sp.Range),
			InstanceType:      int16(sp.InstanceType),
			InstanceValue:     int16(sp.InstanceValue),
			TickType:          int16(sp.TickType),
			TickValue:         int16(sp.TickValue),
			AffectType:        int16(sp.AffectType),
			AffectValue:       int16(sp.AffectValue),
			AffectTime:        int16(sp.AffectTime),
			InstanceAttribute: int16(sp.InstanceAttribute),
			TickAttribute:     int16(sp.TickAttribute),
			Aggressive:        int16(sp.Aggressive),
			MaxTarget:         int16(sp.MaxTarget),
			BParty:            int16(sp.BParty),
			AffectResist:      int16(sp.AffectResist),
			Passive:           int16(sp.Passive),
		})
	}
	return out
}

// buildTemplates reads every file in npc/ and stores the canonical 816-byte
// form. Files that do not normalize (wrong size, unreadable) are skipped and
// reported — one stray file must not block the catalog.
func buildTemplates(contentDir string, logger *slog.Logger) ([]domain.NPCTemplateEntry, []string, error) {
	dir := npctemplate.Dir(contentDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("ler %s: %w", dir, err)
	}
	out := make([]domain.NPCTemplateEntry, 0, len(entries))
	var skipped []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			skipped = append(skipped, name+": "+err.Error())
			continue
		}
		blob, _, err := savefmt.NormalizeMob(raw)
		if err != nil {
			skipped = append(skipped, fmt.Sprintf("%s (%d bytes): %v", name, len(raw), err))
			continue
		}
		sum := sha256.Sum256(blob)
		out = append(out, domain.NPCTemplateEntry{
			Name:       textenc.ToUTF8(name),
			FileName:   textenc.ToUTF8(name),
			SourceSize: int16(len(raw)),
			SHA256:     hex.EncodeToString(sum[:]),
			Blob:       blob,
		})
	}
	for _, s := range skipped {
		logger.Warn("template ignorado", "motivo", s)
	}
	return out, skipped, nil
}

// buildWorldItems converts the InitItem rows; the order is the file's, which is
// the order the world used to seed them in.
func buildWorldItems(items []content.InitItem) []domain.WorldItemSeedEntry {
	out := make([]domain.WorldItemSeedEntry, 0, len(items))
	for _, it := range items {
		out = append(out, domain.WorldItemSeedEntry{Index: it.Index, X: it.PosX, Y: it.PosY, Rotate: it.Rotate})
	}
	return out
}

// rowHash is the drift marker for one CSV row: the columns joined exactly as
// they appear, so whitespace changes count as changes.
func rowHash(fields []string) string {
	sum := sha256.Sum256([]byte(strings.Join(fields, ",")))
	return hex.EncodeToString(sum[:])
}

// runCheck compares the database row counts with the files and returns the
// process exit code (0 ok, 1 divergent).
func runCheck(ctx context.Context, s *store.Store, cat catalog, want map[string]bool, logger *slog.Logger) int {
	counts, err := s.CountContentCatalog(ctx)
	if err != nil {
		logger.Error("contar catálogos", "err", err)
		return 1
	}
	rows := []struct {
		set       string
		name      string
		file, db  int
	}{
		{"item", "item_catalog", len(cat.Items), counts.Items},
		{"skill", "skill_catalog", len(cat.Skills), counts.Skills},
		{"npc-template", "npc_template", len(cat.Templates), counts.NPCTemplates},
		{"world-item", "world_item_seed", len(cat.WorldItems), counts.WorldItems},
	}
	bad := false
	for _, r := range rows {
		if !want[r.set] {
			continue
		}
		status := "ok"
		if r.file != r.db {
			status = "DIVERGE"
			bad = true
		}
		logger.Info("check", "tabela", r.name, "arquivo", r.file, "banco", r.db, "status", status)
	}
	if bad {
		logger.Error("catálogos divergem do conteúdo — rode o importador")
		return 1
	}
	logger.Info("catálogos em sincronia com o conteúdo")
	return 0
}
