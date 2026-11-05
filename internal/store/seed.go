package store

import (
	"context"
	"fmt"
	"slices"
)

// Reference tables that can be dumped as a deterministic SQL seed (issue 58,
// fase 3). The list is an ALLOWLIST on purpose: a dump that walks pg_tables
// would copy account/character/item/affect and the audit trails into the
// repository, and those are player data, not content.
//
// server_instance_flags is deliberately absent: it is boot-time runtime state,
// not content, and a seeded copy would pin flags the operator may have changed.
var seedTables = []string{
	// world defaults (small, edited only by migration)
	"guild_zone", "castle_quest_state", "kefra_state",
	"world_event_config", "world_event_meta", "npc_config_meta",
	// NPC spawn catalogue (was NPCGener.txt)
	"npc_definition", "npc_shop_item",
	// content catalogs (were CSV/STRUCT_MOB files)
	"item_catalog", "skill_catalog", "npc_template", "world_item_seed",
}

// ContentNPCDefinitionCount reports how many npc_definition rows the content
// owns (origin='content') — exactly the number NPCGener.txt used to provide.
// The dbServer boots on this instead of on the file (issue 58, fase 3): zero
// means neither the file nor the seed produced a catalogue, which must fail
// loudly rather than boot a world that never spawns a monster.
func (s *Store) ContentNPCDefinitionCount(ctx context.Context) (int, error) {
	var n int
	if err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM npc_definition WHERE origin = 'content'`).Scan(&n); err != nil {
		return 0, fmt.Errorf("store: count content npc definitions: %w", err)
	}
	return n, nil
}

// SeedColumns returns the insertable columns of a seed table: every column
// except identity columns (GENERATED ALWAYS rejects an explicit value) and
// generated columns. The seed writes the column list explicitly for exactly
// that reason — `INSERT INTO t SELECT * FROM json_populate_record(...)` fails on
// npc_definition.id with "cannot insert a non-DEFAULT value into column id".
func (s *Store) SeedColumns(ctx context.Context, table string) ([]string, error) {
	if !slices.Contains(seedTables, table) {
		return nil, fmt.Errorf("store: %q is not an allowlisted seed table", table)
	}
	rows, err := s.pool.Query(ctx, `
		SELECT a.attname
		FROM pg_attribute a
		WHERE a.attrelid = $1::regclass
		  AND a.attnum > 0 AND NOT a.attisdropped
		  AND a.attidentity = '' AND a.attgenerated = ''
		ORDER BY a.attnum`, table)
	if err != nil {
		return nil, fmt.Errorf("store: seed columns of %s: %w", table, err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("store: scan column of %s: %w", table, err)
		}
		out = append(out, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate columns of %s: %w", table, err)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("store: table %s has no insertable columns", table)
	}
	return out, nil
}

// SeedTables returns the allowlisted reference tables in dump order.
func SeedTables() []string { return slices.Clone(seedTables) }

// DumpTableJSON returns one JSON object per row, ordered by the row's own JSON
// text. Sorting by the canonical JSON instead of a hand-written column list
// keeps the dump deterministic for every table without duplicating its schema
// here — and two rows can never be textually equal, so the order is total.
func (s *Store) DumpTableJSON(ctx context.Context, table string) ([][]byte, error) {
	if !slices.Contains(seedTables, table) {
		return nil, fmt.Errorf("store: %q is not an allowlisted seed table", table)
	}
	// The identifier comes from the allowlist above, never from a caller.
	rows, err := s.pool.Query(ctx, `SELECT to_jsonb(t)::text FROM `+table+` t ORDER BY to_jsonb(t)::text`)
	if err != nil {
		return nil, fmt.Errorf("store: dump %s: %w", table, err)
	}
	defer rows.Close()

	var out [][]byte
	for rows.Next() {
		var js string
		if err := rows.Scan(&js); err != nil {
			return nil, fmt.Errorf("store: scan %s row: %w", table, err)
		}
		out = append(out, []byte(js))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate %s: %w", table, err)
	}
	return out, nil
}
