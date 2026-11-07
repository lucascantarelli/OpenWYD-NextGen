package store

import (
	"context"
	"encoding/json"
	"fmt"

	"openwyd-nextgen/internal/domain"
)

// Content catalog persistence (issue 58, fase 1).
//
// Estas tabelas substituem quatro arquivos que o tmServer lia no boot
// (ItemList.csv, SkillData.csv, npc/*, InitItem.csv). O Postgres é a fonte da
// verdade a partir do momento em que o importador roda; os arquivos viram entrada
// do importador e são apagados quando o runtime lê as tabelas (fase 2). Os upserts
// usam como chave a chave natural de cada catálogo, então rodar o importador de
// novo é idempotente.
// UpsertItemCatalog insere ou atualiza todos os itens, devolvendo quantas linhas o
// comando afetou. Roda numa transação só: um catálogo parcial faria
// nomes/preços de item divergirem do arquivo para jogadores em sessão.
func (s *Store) UpsertItemCatalog(ctx context.Context, items []domain.ItemCatalogEntry) (int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("store: begin item catalog: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const q = `
		INSERT INTO item_catalog
			(item_index, name, price, grade, unique_id, extra, pos,
			 req_level, req_str, req_int, req_dex, req_con,
			 volatile, attack_range, effects, fields, sha256, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17, now())
		ON CONFLICT (item_index) DO UPDATE SET
			name = EXCLUDED.name, price = EXCLUDED.price, grade = EXCLUDED.grade,
			unique_id = EXCLUDED.unique_id, extra = EXCLUDED.extra, pos = EXCLUDED.pos,
			req_level = EXCLUDED.req_level, req_str = EXCLUDED.req_str,
			req_int = EXCLUDED.req_int, req_dex = EXCLUDED.req_dex, req_con = EXCLUDED.req_con,
			volatile = EXCLUDED.volatile, attack_range = EXCLUDED.attack_range,
			effects = EXCLUDED.effects, fields = EXCLUDED.fields, sha256 = EXCLUDED.sha256,
			updated_at = now()`

	for _, it := range items {
		effects, err := json.Marshal(it.Effects)
		if err != nil {
			return 0, fmt.Errorf("store: encode item %d effects: %w", it.Index, err)
		}
		fields, err := json.Marshal(it.Fields)
		if err != nil {
			return 0, fmt.Errorf("store: encode item %d fields: %w", it.Index, err)
		}
		if _, err := tx.Exec(ctx, q,
			it.Index, it.Name, it.Price, it.Grade, it.UniqueID, it.Extra, it.Pos,
			it.ReqLevel, it.ReqStr, it.ReqInt, it.ReqDex, it.ReqCon,
			it.Volatile, it.AttackRange, effects, fields, it.SHA256,
		); err != nil {
			return 0, fmt.Errorf("store: upsert item %d: %w", it.Index, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("store: commit item catalog: %w", err)
	}
	return len(items), nil
}

// UpsertSkillCatalog insere ou atualiza todas as skills.
func (s *Store) UpsertSkillCatalog(ctx context.Context, skills []domain.SkillCatalogEntry) (int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("store: begin skill catalog: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const q = `
		INSERT INTO skill_catalog
			(skill_index, name, skill_point, target_type, mana_spent, delay_ticks, range_tiles,
			 instance_type, instance_value, tick_type, tick_value, affect_type, affect_value,
			 affect_time, instance_attribute, tick_attribute, aggressive, max_target, party,
			 affect_resist, passive, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21, now())
		ON CONFLICT (skill_index) DO UPDATE SET
			name = EXCLUDED.name, skill_point = EXCLUDED.skill_point,
			target_type = EXCLUDED.target_type, mana_spent = EXCLUDED.mana_spent,
			delay_ticks = EXCLUDED.delay_ticks, range_tiles = EXCLUDED.range_tiles,
			instance_type = EXCLUDED.instance_type, instance_value = EXCLUDED.instance_value,
			tick_type = EXCLUDED.tick_type, tick_value = EXCLUDED.tick_value,
			affect_type = EXCLUDED.affect_type, affect_value = EXCLUDED.affect_value,
			affect_time = EXCLUDED.affect_time, instance_attribute = EXCLUDED.instance_attribute,
			tick_attribute = EXCLUDED.tick_attribute, aggressive = EXCLUDED.aggressive,
			max_target = EXCLUDED.max_target, party = EXCLUDED.party,
			affect_resist = EXCLUDED.affect_resist, passive = EXCLUDED.passive,
			updated_at = now()`

	for _, sp := range skills {
		if _, err := tx.Exec(ctx, q,
			sp.Index, sp.Name, sp.SkillPoint, sp.TargetType, sp.ManaSpent, sp.Delay, sp.Range,
			sp.InstanceType, sp.InstanceValue, sp.TickType, sp.TickValue, sp.AffectType, sp.AffectValue,
			sp.AffectTime, sp.InstanceAttribute, sp.TickAttribute, sp.Aggressive, sp.MaxTarget, sp.BParty,
			sp.AffectResist, sp.Passive,
		); err != nil {
			return 0, fmt.Errorf("store: upsert skill %d: %w", sp.Index, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("store: commit skill catalog: %w", err)
	}
	return len(skills), nil
}

// UpsertNPCTemplates insere ou atualiza todo blob de template.
func (s *Store) UpsertNPCTemplates(ctx context.Context, tpls []domain.NPCTemplateEntry) (int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("store: begin npc templates: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const q = `
		INSERT INTO npc_template (template_name, file_name, source_size, sha256, blob, updated_at)
		VALUES ($1,$2,$3,$4,$5, now())
		ON CONFLICT (template_name) DO UPDATE SET
			file_name = EXCLUDED.file_name, source_size = EXCLUDED.source_size,
			sha256 = EXCLUDED.sha256, blob = EXCLUDED.blob, updated_at = now()`

	for _, t := range tpls {
		if _, err := tx.Exec(ctx, q, t.Name, t.FileName, t.SourceSize, t.SHA256, t.Blob); err != nil {
			return 0, fmt.Errorf("store: upsert npc template %q: %w", t.Name, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("store: commit npc templates: %w", err)
	}
	return len(tpls), nil
}

// UpsertWorldItemSeeds insere ou atualiza os objetos estáticos do mundo. A chave
// natural (item_index, pos_x, pos_y) é o que impede uma reimportação de plantar o
// mesmo
func (s *Store) UpsertWorldItemSeeds(ctx context.Context, seeds []domain.WorldItemSeedEntry) (int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("store: begin world items: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const q = `
		INSERT INTO world_item_seed (item_index, pos_x, pos_y, rotate, updated_at)
		VALUES ($1,$2,$3,$4, now())
		ON CONFLICT (item_index, pos_x, pos_y) DO UPDATE SET
			rotate = EXCLUDED.rotate, enabled = true, updated_at = now()`

	for _, it := range seeds {
		if _, err := tx.Exec(ctx, q, it.Index, it.X, it.Y, it.Rotate); err != nil {
			return 0, fmt.Errorf("store: upsert world item %d@(%d,%d): %w", it.Index, it.X, it.Y, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("store: commit world items: %w", err)
	}
	return len(seeds), nil
}

// --- leitura (fase 2 do runtime e verificação do importador) ---

// ListItemCatalog devolve o catálogo inteiro ordenado por índice de item.
func (s *Store) ListItemCatalog(ctx context.Context) ([]domain.ItemCatalogEntry, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT item_index, name, price, grade, unique_id, extra, pos,
		       req_level, req_str, req_int, req_dex, req_con,
		       volatile, attack_range, effects, fields, sha256
		FROM item_catalog ORDER BY item_index`)
	if err != nil {
		return nil, fmt.Errorf("store: list item catalog: %w", err)
	}
	defer rows.Close()

	var out []domain.ItemCatalogEntry
	for rows.Next() {
		var it domain.ItemCatalogEntry
		var effects, fields []byte
		if err := rows.Scan(&it.Index, &it.Name, &it.Price, &it.Grade, &it.UniqueID, &it.Extra, &it.Pos,
			&it.ReqLevel, &it.ReqStr, &it.ReqInt, &it.ReqDex, &it.ReqCon,
			&it.Volatile, &it.AttackRange, &effects, &fields, &it.SHA256); err != nil {
			return nil, fmt.Errorf("store: scan item catalog: %w", err)
		}
		if err := json.Unmarshal(effects, &it.Effects); err != nil {
			return nil, fmt.Errorf("store: decode item %d effects: %w", it.Index, err)
		}
		if err := json.Unmarshal(fields, &it.Fields); err != nil {
			return nil, fmt.Errorf("store: decode item %d fields: %w", it.Index, err)
		}
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate item catalog: %w", err)
	}
	return out, nil
}

// ListSkillCatalog devolve toda skill ordenada por índice.
func (s *Store) ListSkillCatalog(ctx context.Context) ([]domain.SkillCatalogEntry, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT skill_index, name, skill_point, target_type, mana_spent, delay_ticks, range_tiles,
		       instance_type, instance_value, tick_type, tick_value, affect_type, affect_value,
		       affect_time, instance_attribute, tick_attribute, aggressive, max_target, party,
		       affect_resist, passive
		FROM skill_catalog ORDER BY skill_index`)
	if err != nil {
		return nil, fmt.Errorf("store: list skill catalog: %w", err)
	}
	defer rows.Close()

	var out []domain.SkillCatalogEntry
	for rows.Next() {
		var sp domain.SkillCatalogEntry
		if err := rows.Scan(&sp.Index, &sp.Name, &sp.SkillPoint, &sp.TargetType, &sp.ManaSpent, &sp.Delay,
			&sp.Range, &sp.InstanceType, &sp.InstanceValue, &sp.TickType, &sp.TickValue,
			&sp.AffectType, &sp.AffectValue, &sp.AffectTime, &sp.InstanceAttribute, &sp.TickAttribute,
			&sp.Aggressive, &sp.MaxTarget, &sp.BParty, &sp.AffectResist, &sp.Passive); err != nil {
			return nil, fmt.Errorf("store: scan skill catalog: %w", err)
		}
		out = append(out, sp)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate skill catalog: %w", err)
	}
	return out, nil
}

// ListNPCTemplates devolve todo template ordenado por nome.
func (s *Store) ListNPCTemplates(ctx context.Context) ([]domain.NPCTemplateEntry, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT template_name, file_name, source_size, sha256, blob
		FROM npc_template ORDER BY template_name`)
	if err != nil {
		return nil, fmt.Errorf("store: list npc templates: %w", err)
	}
	defer rows.Close()

	var out []domain.NPCTemplateEntry
	for rows.Next() {
		var t domain.NPCTemplateEntry
		if err := rows.Scan(&t.Name, &t.FileName, &t.SourceSize, &t.SHA256, &t.Blob); err != nil {
			return nil, fmt.Errorf("store: scan npc template: %w", err)
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate npc templates: %w", err)
	}
	return out, nil
}

// ListWorldItemSeeds devolve todo objeto estático do mundo, ordenado para um
// deterministic boot.
func (s *Store) ListWorldItemSeeds(ctx context.Context) ([]domain.WorldItemSeedEntry, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT item_index, pos_x, pos_y, rotate
		FROM world_item_seed WHERE enabled ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("store: list world items: %w", err)
	}
	defer rows.Close()

	var out []domain.WorldItemSeedEntry
	for rows.Next() {
		var it domain.WorldItemSeedEntry
		if err := rows.Scan(&it.Index, &it.X, &it.Y, &it.Rotate); err != nil {
			return nil, fmt.Errorf("store: scan world item: %w", err)
		}
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate world items: %w", err)
	}
	return out, nil
}

// ContentCatalogCounts informa quantas linhas cada catálogo tem — o
// --check do importador as compara com o que os arquivos de conteúdo declaram.
type ContentCatalogCounts struct {
	Items        int
	Skills       int
	NPCTemplates int
	WorldItems   int
}

// CountContentCatalog devolve a contagem de linhas de cada catálogo de conteúdo.
func (s *Store) CountContentCatalog(ctx context.Context) (ContentCatalogCounts, error) {
	var c ContentCatalogCounts
	for _, q := range []struct {
		table string
		dst   *int
	}{
		{"item_catalog", &c.Items},
		{"skill_catalog", &c.Skills},
		{"npc_template", &c.NPCTemplates},
		{"world_item_seed", &c.WorldItems},
	} {
		if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM `+q.table).Scan(q.dst); err != nil {
			return ContentCatalogCounts{}, fmt.Errorf("store: count %s: %w", q.table, err)
		}
	}
	return c, nil
}
