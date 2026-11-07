package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"openwyd-nextgen/internal/domain"
)

// Persistência de stats de template de mob (mob-template-editing-plan.md), o
// equivalente em ferramenta ao EDITAPPMOB legado. O Postgres é dono da
// *sobreposição* de stats para um arquivo STRUCT_MOB cru npc/<template_name>; o
// tmServer só a lê (via dbServer), uma vez no boot — não há
// hot-reload para este recurso (o próprio EDITAPPMOB exigia reiniciar o servidor
// também). Toda mutação roda numa transação que também grava linha de auditoria,
// reusando npc_audit/npc_config_meta (auditAndBump) em vez de uma trilha
// paralela — subir npc_config_meta numa escrita de stats é um no-op inofensivo para
// o poll não relacionado de npc_definition.

const mobTemplateStatColumns = `
	template_name, display_name, clan, merchant, class, coin, exp, spx, spy,
	level, ac, damage, chaos_rate, attack_run, direction, str, intel, dex, con,
	special1, special2, special3, special4, max_hp, hp, max_mp, mp,
	learned_skill, score_bonus, skill_bar1, skill_bar2, skill_bar3, skill_bar4,
	regen_hp, regen_mp, resist1, resist2, resist3, resist4`

// ListMobTemplateStats devolve toda sobreposição de stats de template de mob com seus
// slots de equip, ordenada por template_name. É o snapshot completo que o
// tmServer aplica no boot.
func (s *Store) ListMobTemplateStats(ctx context.Context) ([]domain.MobTemplateStat, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+mobTemplateStatColumns+` FROM mob_template_stat ORDER BY template_name`)
	if err != nil {
		return nil, fmt.Errorf("store: list mob template stats: %w", err)
	}
	defer rows.Close()

	byName := make(map[string]int, 16)
	var out []domain.MobTemplateStat
	for rows.Next() {
		st, err := scanMobTemplateStat(rows)
		if err != nil {
			return nil, err
		}
		byName[st.TemplateName] = len(out)
		out = append(out, st)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate mob template stats: %w", err)
	}

	equipRows, err := s.pool.Query(ctx, `
		SELECT template_name, slot, item_index, eff1, effv1, eff2, effv2, eff3, effv3
		FROM mob_template_equip ORDER BY template_name, slot`)
	if err != nil {
		return nil, fmt.Errorf("store: list mob template equip: %w", err)
	}
	defer equipRows.Close()
	for equipRows.Next() {
		var name string
		var it domain.MobTemplateEquipItem
		if err := equipRows.Scan(&name, &it.Slot, &it.ItemIndex,
			&it.Eff1, &it.EffV1, &it.Eff2, &it.EffV2, &it.Eff3, &it.EffV3); err != nil {
			return nil, fmt.Errorf("store: scan mob template equip: %w", err)
		}
		if idx, ok := byName[name]; ok {
			out[idx].Equip = append(out[idx].Equip, it)
		}
	}
	if err := equipRows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate mob template equip: %w", err)
	}
	return out, nil
}

// GetMobTemplateStat carrega uma sobreposição (com seus slots de equip) pelo nome do
// template. Devolve ErrNotFound quando ausente — quem chama cai para ler
// os valores correntes do próprio arquivo cru do template.
func (s *Store) GetMobTemplateStat(ctx context.Context, templateName string) (domain.MobTemplateStat, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+mobTemplateStatColumns+` FROM mob_template_stat WHERE template_name = $1`, templateName)
	st, err := scanMobTemplateStat(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.MobTemplateStat{}, ErrNotFound
	}
	if err != nil {
		return domain.MobTemplateStat{}, fmt.Errorf("store: get mob template stat %q: %w", templateName, err)
	}
	equip, err := s.loadMobTemplateEquip(ctx, s.pool, templateName)
	if err != nil {
		return domain.MobTemplateStat{}, err
	}
	st.Equip = equip
	return st, nil
}

// UpsertMobTemplateStat insere uma sobreposição nova ou substitui a existente
// para o mesmo template_name, incluindo seus slots Equip[] (st.Equip substitui
// por inteiro o que estava guardado — mesma semântica autoritativa de
// SetMobTemplateEquip, que continua disponível para edições só de equip da
// grade de equipamento dedicada). Sobe a versão da config e grava uma linha
// de auditoria
func (s *Store) UpsertMobTemplateStat(ctx context.Context, st domain.MobTemplateStat, moderatorID int64) error {
	return s.inTx(ctx, func(tx pgx.Tx) error {
		before, _ := fetchMobTemplateStatJSON(ctx, tx, st.TemplateName)
		_, err := tx.Exec(ctx, `
			INSERT INTO mob_template_stat (`+mobTemplateStatColumns+`, updated_by, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,
			        $20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,$35,$36,$37,$38,$39, $40, now())
			ON CONFLICT (template_name) DO UPDATE SET
				display_name  = EXCLUDED.display_name,
				clan          = EXCLUDED.clan,
				merchant      = EXCLUDED.merchant,
				class         = EXCLUDED.class,
				coin          = EXCLUDED.coin,
				exp           = EXCLUDED.exp,
				spx           = EXCLUDED.spx,
				spy           = EXCLUDED.spy,
				level         = EXCLUDED.level,
				ac            = EXCLUDED.ac,
				damage        = EXCLUDED.damage,
				chaos_rate    = EXCLUDED.chaos_rate,
				attack_run    = EXCLUDED.attack_run,
				direction     = EXCLUDED.direction,
				str           = EXCLUDED.str,
				intel         = EXCLUDED.intel,
				dex           = EXCLUDED.dex,
				con           = EXCLUDED.con,
				special1      = EXCLUDED.special1,
				special2      = EXCLUDED.special2,
				special3      = EXCLUDED.special3,
				special4      = EXCLUDED.special4,
				max_hp        = EXCLUDED.max_hp,
				hp            = EXCLUDED.hp,
				max_mp        = EXCLUDED.max_mp,
				mp            = EXCLUDED.mp,
				learned_skill = EXCLUDED.learned_skill,
				score_bonus   = EXCLUDED.score_bonus,
				skill_bar1    = EXCLUDED.skill_bar1,
				skill_bar2    = EXCLUDED.skill_bar2,
				skill_bar3    = EXCLUDED.skill_bar3,
				skill_bar4    = EXCLUDED.skill_bar4,
				regen_hp      = EXCLUDED.regen_hp,
				regen_mp      = EXCLUDED.regen_mp,
				resist1       = EXCLUDED.resist1,
				resist2       = EXCLUDED.resist2,
				resist3       = EXCLUDED.resist3,
				resist4       = EXCLUDED.resist4,
				updated_by    = EXCLUDED.updated_by,
				updated_at    = now()`,
			st.TemplateName, st.DisplayName, st.Clan, st.Merchant, st.Class, st.Coin, st.Exp, st.SPX, st.SPY,
			st.Level, st.AC, st.Damage, st.ChaosRate, st.AttackRun, st.Direction, st.Str, st.Int, st.Dex, st.Con,
			st.Special[0], st.Special[1], st.Special[2], st.Special[3], st.MaxHp, st.Hp, st.MaxMp, st.Mp,
			st.LearnedSkill, st.ScoreBonus, st.SkillBar[0], st.SkillBar[1], st.SkillBar[2], st.SkillBar[3],
			st.RegenHP, st.RegenMP, st.Resist[0], st.Resist[1], st.Resist[2], st.Resist[3],
			nullableID(moderatorID),
		)
		if err != nil {
			return fmt.Errorf("store: upsert mob template stat %q: %w", st.TemplateName, err)
		}
		if err := replaceMobTemplateEquip(ctx, tx, st.TemplateName, st.Equip); err != nil {
			return err
		}
		action := "update_template_stat"
		if before == nil {
			action = "create_template_stat"
		}
		after, _ := fetchMobTemplateStatJSON(ctx, tx, st.TemplateName)
		return auditAndBump(ctx, tx, nil, moderatorID, action, before, after)
	})
}

// SetMobTemplateEquip substitui os slots Equip[] de um template, sem tocar nos stats. Devolve
// ErrNotFound se ainda não existe linha mob_template_stat para template_name (a
// linha de stats precisa ser inserida antes, mesma dependência de npc_shop_item em
// npc_definition).
func (s *Store) SetMobTemplateEquip(ctx context.Context, templateName string, items []domain.MobTemplateEquipItem, moderatorID int64) error {
	return s.inTx(ctx, func(tx pgx.Tx) error {
		if err := ensureMobTemplateStatExists(ctx, tx, templateName); err != nil {
			return err
		}
		before, _ := fetchMobTemplateEquipJSON(ctx, tx, templateName)
		if err := replaceMobTemplateEquip(ctx, tx, templateName, items); err != nil {
			return err
		}
		after, _ := fetchMobTemplateEquipJSON(ctx, tx, templateName)
		return auditAndBump(ctx, tx, nil, moderatorID, "set_template_equip", before, after)
	})
}

// replaceMobTemplateEquip apaga e reinsere o slot Equip[] de um template
// sobreposições de uma vez — compartilhado por UpsertMobTemplateStat (save do objeto inteiro,
// AdminMobTemplateStat.equip incluído) e SetMobTemplateEquip (o
// editor de grade só de equip), para os dois caminhos de escrita persistirem do
// mesmo jeito em vez de um deles descartar o campo em silêncio.
func replaceMobTemplateEquip(ctx context.Context, tx pgx.Tx, templateName string, items []domain.MobTemplateEquipItem) error {
	if _, err := tx.Exec(ctx, `DELETE FROM mob_template_equip WHERE template_name = $1`, templateName); err != nil {
		return fmt.Errorf("store: clear mob template equip %q: %w", templateName, err)
	}
	for _, it := range items {
		if _, err := tx.Exec(ctx, `
			INSERT INTO mob_template_equip (template_name, slot, item_index, eff1, effv1, eff2, effv2, eff3, effv3)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			templateName, it.Slot, it.ItemIndex, it.Eff1, it.EffV1, it.Eff2, it.EffV2, it.Eff3, it.EffV3,
		); err != nil {
			return fmt.Errorf("store: insert mob template equip (%q slot %d): %w", templateName, it.Slot, err)
		}
	}
	return nil
}

// DeleteMobTemplateStat remove uma sobreposição (seus slots de equip cascateiam),
// revertendo o template para os padrões do arquivo cru. Devolve ErrNotFound se
// absent.
func (s *Store) DeleteMobTemplateStat(ctx context.Context, templateName string, moderatorID int64) error {
	return s.inTx(ctx, func(tx pgx.Tx) error {
		before, err := fetchMobTemplateStatJSON(ctx, tx, templateName)
		if err != nil {
			return err
		}
		if before == nil {
			return ErrNotFound
		}
		if _, err := tx.Exec(ctx, `DELETE FROM mob_template_stat WHERE template_name = $1`, templateName); err != nil {
			return fmt.Errorf("store: delete mob template stat %q: %w", templateName, err)
		}
		return auditAndBump(ctx, tx, nil, moderatorID, "delete_template_stat", before, nil)
	})
}

// --- helpers ---

// mobTemplateStatRow é a superfície de leitura compartilhada por *pgxpool.Pool.QueryRow e
// pgx.Rows (both expose Scan).
type mobTemplateStatRow interface {
	Scan(dest ...any) error
}

func scanMobTemplateStat(row mobTemplateStatRow) (domain.MobTemplateStat, error) {
	var st domain.MobTemplateStat
	err := row.Scan(
		&st.TemplateName, &st.DisplayName, &st.Clan, &st.Merchant, &st.Class, &st.Coin, &st.Exp, &st.SPX, &st.SPY,
		&st.Level, &st.AC, &st.Damage, &st.ChaosRate, &st.AttackRun, &st.Direction, &st.Str, &st.Int, &st.Dex, &st.Con,
		&st.Special[0], &st.Special[1], &st.Special[2], &st.Special[3], &st.MaxHp, &st.Hp, &st.MaxMp, &st.Mp,
		&st.LearnedSkill, &st.ScoreBonus, &st.SkillBar[0], &st.SkillBar[1], &st.SkillBar[2], &st.SkillBar[3],
		&st.RegenHP, &st.RegenMP, &st.Resist[0], &st.Resist[1], &st.Resist[2], &st.Resist[3],
	)
	if err != nil {
		return domain.MobTemplateStat{}, err
	}
	return st, nil
}

func (s *Store) loadMobTemplateEquip(ctx context.Context, q pgxQuerier, templateName string) ([]domain.MobTemplateEquipItem, error) {
	rows, err := q.Query(ctx, `
		SELECT slot, item_index, eff1, effv1, eff2, effv2, eff3, effv3
		FROM mob_template_equip WHERE template_name = $1 ORDER BY slot`, templateName)
	if err != nil {
		return nil, fmt.Errorf("store: load mob template equip %q: %w", templateName, err)
	}
	defer rows.Close()
	var out []domain.MobTemplateEquipItem
	for rows.Next() {
		var it domain.MobTemplateEquipItem
		if err := rows.Scan(&it.Slot, &it.ItemIndex, &it.Eff1, &it.EffV1, &it.Eff2, &it.EffV2, &it.Eff3, &it.EffV3); err != nil {
			return nil, fmt.Errorf("store: scan mob template equip: %w", err)
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func ensureMobTemplateStatExists(ctx context.Context, tx pgx.Tx, templateName string) error {
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM mob_template_stat WHERE template_name = $1)`, templateName).Scan(&exists); err != nil {
		return fmt.Errorf("store: check mob template stat %q: %w", templateName, err)
	}
	if !exists {
		return ErrNotFound
	}
	return nil
}

func fetchMobTemplateStatJSON(ctx context.Context, tx pgx.Tx, templateName string) ([]byte, error) {
	var js []byte
	err := tx.QueryRow(ctx, `SELECT to_jsonb(s) FROM mob_template_stat s WHERE template_name = $1`, templateName).Scan(&js)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return js, err
}

func fetchMobTemplateEquipJSON(ctx context.Context, tx pgx.Tx, templateName string) ([]byte, error) {
	var js []byte
	err := tx.QueryRow(ctx,
		`SELECT coalesce(jsonb_agg(to_jsonb(e) ORDER BY e.slot), '[]'::jsonb)
		 FROM mob_template_equip e WHERE template_name = $1`, templateName).Scan(&js)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return js, err
}
