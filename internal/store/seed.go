package store

import (
	"context"
	"fmt"
	"slices"
)

// Tabelas de referência que podem ser despejadas como um seed SQL determinístico
// (issue 58, fase 3). A lista é uma ALLOWLIST de propósito: um dump que percorre
// pg_tables copiaria conta/personagem/item/affect e as trilhas de auditoria para o
// repositório, e isso é dado de jogador, não conteúdo.
//
// server_instance_flags está deliberadamente ausente: é estado de runtime de boot,
// não conteúdo, e uma cópia semeada fixaria flags que o operador pode ter mudado.
var seedTables = []string{
	// padrões de mundo (pequeno, editado só por migration)
	"guild_zone", "castle_quest_state", "kefra_state",
	"world_event_config", "world_event_meta", "npc_config_meta",
	// catálogo de spawn de NPC (era NPCGener.txt)
	"npc_definition", "npc_shop_item",
	// catálogos de conteúdo (eram arquivos CSV/STRUCT_MOB)
	"item_catalog", "skill_catalog", "npc_template", "world_item_seed",
}

// ContentNPCDefinitionCount informa quantas linhas npc_definition o conteúdo
// possui (origin='content') — exatamente o número que NPCGener.txt costumava fornecer.
// O dbServer sobe com base nisto em vez de no arquivo (issue 58, fase 3): zero
// significa que nem o arquivo nem o seed produziram catálogo, o que precisa falhar
// alto em vez de subir um mundo que nunca gera monstro.
func (s *Store) ContentNPCDefinitionCount(ctx context.Context) (int, error) {
	var n int
	if err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM npc_definition WHERE origin = 'content'`).Scan(&n); err != nil {
		return 0, fmt.Errorf("store: count content npc definitions: %w", err)
	}
	return n, nil
}

// SeedColumns devolve as colunas inseríveis de uma tabela de seed: toda coluna
// exceto colunas de identidade (GENERATED ALWAYS rejeita valor explícito) e
// colunas geradas. O seed escreve a lista de colunas explicitamente exatamente
// por isso — `INSERT INTO t SELECT * FROM json_populate_record(...)` falha em
// npc_definition.id, reclamando que a coluna id não aceita valor não-DEFAULT.
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

// SeedTables devolve as tabelas de referência da allowlist na ordem de dump.
func SeedTables() []string { return slices.Clone(seedTables) }

// DumpTableJSON devolve um objeto JSON por linha, ordenado pelo próprio texto
// JSON da linha. Ordenar pelo JSON canônico em vez de uma lista de colunas escrita à mão
// mantém o dump determinístico para toda tabela sem duplicar seu esquema
// aqui — e duas linhas nunca são textualmente iguais, então a ordem é total.
func (s *Store) DumpTableJSON(ctx context.Context, table string) ([][]byte, error) {
	if !slices.Contains(seedTables, table) {
		return nil, fmt.Errorf("store: %q is not an allowlisted seed table", table)
	}
	// O identificador vem da allowlist acima, nunca de quem chama.
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
