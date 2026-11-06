package store

import (
	"context"
	"encoding/json"
	"fmt"

	"openwyd-nextgen/internal/domain"
)

// Persistência da configuração de conteúdo da issue 59 (Guard.txt,
// LevelItem.txt, MobMerc.txt, gameconfig.txt, Regions.txt) e da proveniência de
// import (issue 50, passo 1).
//
// Todas as escritas são upsert por chave natural, então reimportar é
// idempotente: o arquivo é a semente, o banco guarda o último import e a coluna
// `raw` preserva a linha original para auditoria de paridade.

// UpsertGuardSpawns grava as guardas de cidade.
func (s *Store) UpsertGuardSpawns(ctx context.Context, guards []domain.GuardSpawnEntry) (int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("store: begin guard_spawn: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const q = `
		INSERT INTO guard_spawn (guard_index, generator_index, rules, raw)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (guard_index) DO UPDATE SET
			generator_index = EXCLUDED.generator_index,
			rules = EXCLUDED.rules, raw = EXCLUDED.raw`

	for _, g := range guards {
		rules, err := json.Marshal(g.Rules)
		if err != nil {
			return 0, fmt.Errorf("store: encode guard %d: %w", g.Index, err)
		}
		if _, err := tx.Exec(ctx, q, g.Index, g.GeneratorIndex, rules, g.Raw); err != nil {
			return 0, fmt.Errorf("store: upsert guard %d: %w", g.Index, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("store: commit guard_spawn: %w", err)
	}
	return len(guards), nil
}

// UpsertLevelItems grava a tabela nível × classe → item.
func (s *Store) UpsertLevelItems(ctx context.Context, items []domain.LevelItemEntry) (int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("store: begin level_item: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const q = `
		INSERT INTO level_item (ordinal, class_id, seq, level, type_id, item_index, extra, raw)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (ordinal) DO UPDATE SET
			class_id = EXCLUDED.class_id, seq = EXCLUDED.seq, level = EXCLUDED.level,
			type_id = EXCLUDED.type_id, item_index = EXCLUDED.item_index,
			extra = EXCLUDED.extra, raw = EXCLUDED.raw`

	for _, it := range items {
		extra, err := json.Marshal(it.Extra)
		if err != nil {
			return 0, fmt.Errorf("store: encode level item #%d: %w", it.Ordinal, err)
		}
		if _, err := tx.Exec(ctx, q, it.Ordinal, it.Class, it.Seq, it.Level, it.Type, it.ItemIndex, extra, it.Raw); err != nil {
			return 0, fmt.Errorf("store: upsert level item #%d: %w", it.Ordinal, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("store: commit level_item: %w", err)
	}
	return len(items), nil
}

// UpsertMobMercs grava o restock dos mercadores.
func (s *Store) UpsertMobMercs(ctx context.Context, mercs []domain.MobMercEntry) (int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("store: begin mob_merc: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const q = `
		INSERT INTO mob_merc (generator_index, template_name, renew_seconds, reborn_seconds, items, raw)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (generator_index) DO UPDATE SET
			template_name = EXCLUDED.template_name, renew_seconds = EXCLUDED.renew_seconds,
			reborn_seconds = EXCLUDED.reborn_seconds, items = EXCLUDED.items, raw = EXCLUDED.raw`

	for _, m := range mercs {
		items, err := json.Marshal(m.Items)
		if err != nil {
			return 0, fmt.Errorf("store: encode mob merc %d: %w", m.GeneratorIndex, err)
		}
		if _, err := tx.Exec(ctx, q, m.GeneratorIndex, m.TemplateName, m.RenewSeconds, m.RebornSeconds, items, m.Raw); err != nil {
			return 0, fmt.Errorf("store: upsert mob merc %d: %w", m.GeneratorIndex, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("store: commit mob_merc: %w", err)
	}
	return len(mercs), nil
}

// UpsertGameConfig grava os pares do gameconfig e, na mesma transação, a matriz
// de bônus de drop: meio gameconfig no banco faria o loot ler uma matriz velha
// com pares novos.
func (s *Store) UpsertGameConfig(ctx context.Context, settings []domain.GameSetting, matrix []domain.GameDropBonus) (int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("store: begin game_setting: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const q = `
		INSERT INTO game_setting (key, section, kind, value_int, value_text)
		VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (key) DO UPDATE SET
			section = EXCLUDED.section, kind = EXCLUDED.kind,
			value_int = EXCLUDED.value_int, value_text = EXCLUDED.value_text`

	for _, s := range settings {
		if _, err := tx.Exec(ctx, q, s.Key, s.Section, s.Kind, s.ValueInt, s.ValueText); err != nil {
			return 0, fmt.Errorf("store: upsert setting %q: %w", s.Key, err)
		}
	}

	const delMatrix = `DELETE FROM game_drop_bonus`
	if _, err := tx.Exec(ctx, delMatrix); err != nil {
		return 0, fmt.Errorf("store: limpar matriz de drop: %w", err)
	}
	const qm = `INSERT INTO game_drop_bonus (row_index, col_index, pct) VALUES ($1,$2,$3)`
	for _, c := range matrix {
		if _, err := tx.Exec(ctx, qm, c.Row, c.Col, c.Pct); err != nil {
			return 0, fmt.Errorf("store: upsert matriz %d/%d: %w", c.Row, c.Col, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("store: commit game_setting: %w", err)
	}
	return len(settings) + len(matrix), nil
}

// UpsertRegions grava os retângulos de Regions.txt.
func (s *Store) UpsertRegions(ctx context.Context, regions []domain.RegionEntry) (int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("store: begin region: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const q = `
		INSERT INTO region (name, x1, y1, x2, y2, source)
		VALUES ($1,$2,$3,$4,$5,'Regions.txt')
		ON CONFLICT (name) DO UPDATE SET
			x1 = EXCLUDED.x1, y1 = EXCLUDED.y1, x2 = EXCLUDED.x2, y2 = EXCLUDED.y2`

	for _, r := range regions {
		if _, err := tx.Exec(ctx, q, r.Name, r.X1, r.Y1, r.X2, r.Y2); err != nil {
			return 0, fmt.Errorf("store: upsert region %q: %w", r.Name, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("store: commit region: %w", err)
	}
	return len(regions), nil
}

// RecordContentSources grava a proveniência (hash, tamanho e linhas) de cada
// arquivo importado. É o que o -check usa para detectar edição de conteúdo sem
// reimport.
func (s *Store) RecordContentSources(ctx context.Context, files []domain.ContentSourceFile) (int, error) {
	if len(files) == 0 {
		return 0, nil
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("store: begin content_source_file: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const q = `
		INSERT INTO content_source_file (path, sha256, size_bytes, rows, imported_at)
		VALUES ($1,$2,$3,$4, now())
		ON CONFLICT (path) DO UPDATE SET
			sha256 = EXCLUDED.sha256, size_bytes = EXCLUDED.size_bytes,
			rows = EXCLUDED.rows, imported_at = now()`

	for _, f := range files {
		if _, err := tx.Exec(ctx, q, f.Path, f.SHA256, f.SizeBytes, f.Rows); err != nil {
			return 0, fmt.Errorf("store: registrar origem %q: %w", f.Path, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("store: commit content_source_file: %w", err)
	}
	return len(files), nil
}

// ContentSettingsCounts é a contagem por tabela usada pelo -check.
type ContentSettingsCounts struct {
	Guards     int
	LevelItems int
	MobMercs   int
	Settings   int
	Matrix     int
	Regions    int
}

// CountContentSettings conta as tabelas da issue 59.
func (s *Store) CountContentSettings(ctx context.Context) (ContentSettingsCounts, error) {
	var c ContentSettingsCounts
	rows := []struct {
		q string
		d *int
	}{
		{`SELECT count(*) FROM guard_spawn`, &c.Guards},
		{`SELECT count(*) FROM level_item`, &c.LevelItems},
		{`SELECT count(*) FROM mob_merc`, &c.MobMercs},
		{`SELECT count(*) FROM game_setting`, &c.Settings},
		{`SELECT count(*) FROM game_drop_bonus`, &c.Matrix},
		{`SELECT count(*) FROM region`, &c.Regions},
	}
	for _, r := range rows {
		if err := s.pool.QueryRow(ctx, r.q).Scan(r.d); err != nil {
			return ContentSettingsCounts{}, fmt.Errorf("store: contar (%s): %w", r.q, err)
		}
	}
	return c, nil
}

// ContentSourceHashes devolve path → sha256 registrado no último import.
func (s *Store) ContentSourceHashes(ctx context.Context) (map[string]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT path, sha256 FROM content_source_file`)
	if err != nil {
		return nil, fmt.Errorf("store: ler proveniência: %w", err)
	}
	defer rows.Close()

	out := make(map[string]string)
	for rows.Next() {
		var path, sum string
		if err := rows.Scan(&path, &sum); err != nil {
			return nil, fmt.Errorf("store: scan proveniência: %w", err)
		}
		out[path] = sum
	}
	return out, rows.Err()
}
