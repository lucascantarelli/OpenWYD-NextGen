package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"openwyd-nextgen/internal/domain"
)

// ErrNotFound é devolvido quando a conta ou o personagem consultado não existe.
var ErrNotFound = errors.New("store: not found")

// ErrContentOwned impede apagar conteúdo de mundo versionado; o moderador o esconde.
var ErrContentOwned = errors.New("store: content-owned definition")

// ErrNoFreeSlot é devolvido quando a conta já tem todos os slots de personagem em uso.
var ErrNoFreeSlot = errors.New("store: no free character slot")

// ErrConflict é devolvido quando uma escrita tem forma válida mas viola o estado
// atual de guild/conta, como criar uma guild já estando em uma.
var ErrConflict = errors.New("store: conflict")

// AccountAuth é o mínimo de dados de conta para autenticar um login: o
// id, o hash argon2id da senha guardado e a flag de bloqueio. Quem chama
// verifica a senha (o store nunca vê texto plano além do hash).
type AccountAuth struct {
	ID        int64
	PassHash  string
	IsBlocked bool
	Role      string // account.role ('player'/'moderator'/'admin'); gate de UI só da web
}

// AccountByName busca a linha de autenticação de um nome canônico (minúsculo).
// Devolve ErrNotFound quando essa conta não existe.
func (s *Store) AccountByName(ctx context.Context, name string) (AccountAuth, error) {
	var a AccountAuth
	err := s.pool.QueryRow(ctx,
		`SELECT id, pass_hash, is_blocked, role FROM account WHERE name = $1`, name).
		Scan(&a.ID, &a.PassHash, &a.IsBlocked, &a.Role)
	if errors.Is(err, pgx.ErrNoRows) {
		return AccountAuth{}, ErrNotFound
	}
	if err != nil {
		return AccountAuth{}, fmt.Errorf("store: account by name %q: %w", name, err)
	}
	return a, nil
}

// AccountAuthByID busca a linha de autenticação por id de conta (usado para confirmar
// a senha antes de uma operação destrutiva). Devolve ErrNotFound se ausente.
func (s *Store) AccountAuthByID(ctx context.Context, id int64) (AccountAuth, error) {
	a := AccountAuth{ID: id}
	err := s.pool.QueryRow(ctx,
		`SELECT pass_hash, is_blocked FROM account WHERE id = $1`, id).
		Scan(&a.PassHash, &a.IsBlocked)
	if errors.Is(err, pgx.ErrNoRows) {
		return AccountAuth{}, ErrNotFound
	}
	if err != nil {
		return AccountAuth{}, fmt.Errorf("store: account by id %d: %w", id, err)
	}
	return a, nil
}

// PinHashByID devolve o hash argon2id do PIN guardado da conta. String vazia
// significa que ainda não há PIN (o padrão da coluna). Devolve ErrNotFound quando a
// conta não existe.
func (s *Store) PinHashByID(ctx context.Context, id int64) (string, error) {
	var hash string
	err := s.pool.QueryRow(ctx, `SELECT pin_hash FROM account WHERE id = $1`, id).Scan(&hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("store: pin hash by id %d: %w", id, err)
	}
	return hash, nil
}

// SetPinHash guarda o hash argon2id do PIN da conta (o PIN numérico vai para hash
// por quem chama; o store nunca vê texto plano). Devolve ErrNotFound se não houver conta.
func (s *Store) SetPinHash(ctx context.Context, id int64, hash string) error {
	tag, err := s.pool.Exec(ctx, `UPDATE account SET pin_hash = $2 WHERE id = $1`, id, hash)
	if err != nil {
		return fmt.Errorf("store: set pin hash for id %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SetBlockedByName inverte account.is_blocked para um nome de conta canônico
// (minúsculo) — o lado de escrita do comando in-game de ban/unban de GM (issue #122). O login
// bloqueia por esta flag (AccountByName → LOGIN_RESULT_BLOCKED), então um ban nega
// o relog imediatamente. Devolve ErrNotFound quando nenhuma conta casou o nome.
func (s *Store) SetBlockedByName(ctx context.Context, name string, blocked bool) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE account SET is_blocked = $2 WHERE name = $1`, name, blocked)
	if err != nil {
		return fmt.Errorf("store: set blocked %q: %w", name, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ListCharacters devolve a projeção da seleção de personagens (slot, nome, classe,
// prévia de score e equipamento) de uma conta, ordenada por slot.
func (s *Store) ListCharacters(ctx context.Context, accountID int64) ([]domain.Character, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, slot, name, class, guild_id, level, exp, coin,
		        max_hp, hp, max_mp, mp, str, int, dex, con
		   FROM characters WHERE account_id = $1 ORDER BY slot`, accountID)
	if err != nil {
		return nil, fmt.Errorf("store: list characters: %w", err)
	}
	defer rows.Close()

	var charIDs []int64
	var out []domain.Character
	for rows.Next() {
		var charID int64
		var ch domain.Character
		if err := rows.Scan(&charID, &ch.Slot, &ch.Name, &ch.Class, &ch.GuildID, &ch.Level, &ch.Exp, &ch.Coin,
			&ch.MaxHp, &ch.Hp, &ch.MaxMp, &ch.Mp, &ch.Str, &ch.Int, &ch.Dex, &ch.Con); err != nil {
			return nil, fmt.Errorf("store: scan character summary: %w", err)
		}
		charIDs = append(charIDs, charID)
		out = append(out, ch)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: list characters: %w", err)
	}
	rows.Close()

	for i, charID := range charIDs {
		equip, err := s.loadItems(ctx, charID, "char_equip")
		if err != nil {
			return nil, fmt.Errorf("store: list character equipment: %w", err)
		}
		out[i].Equip = equip
	}
	return out, nil
}

// LoadCharacter carrega o estado completo de um personagem (stats + equip + carry +
// affects) por conta e slot. Devolve ErrNotFound quando o slot está vazio.
func (s *Store) LoadCharacter(ctx context.Context, accountID int64, slot int) (domain.Character, error) {
	var ch domain.Character
	var charID int64
	var skillBar, shortSkill, special []int16
	var baseStr, baseInt, baseDex, baseCon *int16
	var baseMaxHp, baseMaxMp *int32
	err := s.pool.QueryRow(ctx, `
		SELECT id, slot, name, class, clan, guild_id, guild_level, level, exp, coin,
		       str, int, dex, con, score_bonus, special_bonus, skill_bonus,
		       max_hp, max_mp, hp, mp, critical, regen_hp, regen_mp,
		       resist_fire, resist_ice, resist_thunder, resist_magic,
		       learned_skill, sec_learned_skill, magic, save_x, save_y, last_city, citizen, class_master, soul, fame,
		       celestial_lv40, celestial_lv90, celestial_circle, terra_mistica, arch_lv355, arch_lv370,
		       skill_bar, short_skill, special, pk_point, guilty, cur_kill, tot_kill, mortal_level, celestial_arch_level, arch_crystal_stage, kefra_ticket, nightmare_entries, last_nightmare_use,
		       base_str, base_int, base_dex, base_con, base_max_hp, base_max_mp, newbie_quest, molar_gargula, pilula_orc, celestial_reset,
		       celestial_add120, celestial_add150, celestial_add180, celestial_add200,
		       celestial_level, sub_celestial_level, save_celestial
		  FROM characters WHERE account_id = $1 AND slot = $2`, accountID, slot).
		Scan(&charID, &ch.Slot, &ch.Name, &ch.Class, &ch.Clan, &ch.GuildID, &ch.GuildLevel,
			&ch.Level, &ch.Exp, &ch.Coin, &ch.Str, &ch.Int, &ch.Dex, &ch.Con,
			&ch.ScoreBonus, &ch.SpecialBonus, &ch.SkillBonus, &ch.MaxHp, &ch.MaxMp, &ch.Hp, &ch.Mp,
			&ch.Critical, &ch.RegenHP, &ch.RegenMP, &ch.ResistFire, &ch.ResistIce, &ch.ResistThunder,
			&ch.ResistMagic, &ch.LearnedSkill, &ch.SecLearnedSkill, &ch.Magic, &ch.SaveX, &ch.SaveY, &ch.LastCity, &ch.Citizen,
			&ch.ClassMaster, &ch.Soul, &ch.Fame, &ch.CelLv40, &ch.CelLv90, &ch.CelCircle, &ch.TerraMistica, &ch.ArchLv355, &ch.ArchLv370, &skillBar, &shortSkill, &special,
			&ch.PKPoint, &ch.Guilty, &ch.CurKill, &ch.TotKill, &ch.MortalLevel, &ch.CelestialArchLevel, &ch.ArchCrystalStage, &ch.KefraTicket, &ch.NightmareEntries, &ch.LastNightmareUse,
			&baseStr, &baseInt, &baseDex, &baseCon, &baseMaxHp, &baseMaxMp, &ch.NewbieQuest, &ch.MolarGargula, &ch.PilulaOrc, &ch.CelReset,
			&ch.CelAdd120, &ch.CelAdd150, &ch.CelAdd180, &ch.CelAdd200,
			&ch.CelestialLevel, &ch.SubCelestialLevel, &ch.SaveCelestial)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Character{}, ErrNotFound
	}
	if err != nil {
		return domain.Character{}, fmt.Errorf("store: load character a=%d slot=%d: %w", accountID, slot, err)
	}
	int16ArrToByteArr(skillBar, ch.SkillBar[:])
	int16ArrToByteArr(shortSkill, ch.ShortSkill[:])
	for i := 0; i < len(special) && i < len(ch.Special); i++ {
		ch.Special[i] = special[i]
	}
	// As seis colunas de base são escritas juntas; qualquer NULL indica linha salva
	// antes da migration 0027, cuja base o tmServer ainda precisa derivar uma vez.
	if baseStr != nil && baseInt != nil && baseDex != nil && baseCon != nil && baseMaxHp != nil && baseMaxMp != nil {
		ch.HasBase = true
		ch.BaseStr, ch.BaseInt, ch.BaseDex, ch.BaseCon = *baseStr, *baseInt, *baseDex, *baseCon
		ch.BaseMaxHp, ch.BaseMaxMp = *baseMaxHp, *baseMaxMp
	}

	if ch.Equip, err = s.loadItems(ctx, charID, "char_equip"); err != nil {
		return domain.Character{}, err
	}
	if ch.Carry, err = s.loadItems(ctx, charID, "char_carry"); err != nil {
		return domain.Character{}, err
	}
	if ch.Affects, err = s.loadAffects(ctx, charID); err != nil {
		return domain.Character{}, err
	}
	return ch, nil
}

func (s *Store) loadItems(ctx context.Context, charID int64, kind string) ([]domain.Item, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT slot, item_index, eff1, effv1, eff2, effv2, eff3, effv3, expires_at
		  FROM item WHERE character_id = $1 AND owner_kind = $2 ORDER BY slot`, charID, kind)
	if err != nil {
		return nil, fmt.Errorf("store: load %s: %w", kind, err)
	}
	defer rows.Close()
	var out []domain.Item
	for rows.Next() {
		var it domain.Item
		var exp *time.Time
		if err := rows.Scan(&it.Slot, &it.Index, &it.Eff1, &it.EffV1, &it.Eff2, &it.EffV2, &it.Eff3, &it.EffV3, &exp); err != nil {
			return nil, fmt.Errorf("store: scan %s item: %w", kind, err)
		}
		it.ExpiresAt = expirySeconds(exp)
		out = append(out, it)
	}
	return out, rows.Err()
}

func (s *Store) loadAffects(ctx context.Context, charID int64) ([]domain.Affect, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT type, value, level, time FROM affect WHERE character_id = $1`, charID)
	if err != nil {
		return nil, fmt.Errorf("store: load affects: %w", err)
	}
	defer rows.Close()
	var out []domain.Affect
	for rows.Next() {
		var a domain.Affect
		if err := rows.Scan(&a.Type, &a.Value, &a.Level, &a.Time); err != nil {
			return nil, fmt.Errorf("store: scan affect: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// CreateCharacter insere um personagem novo num slot livre e devolve seu id.
// Quem chama é responsável por validar que o slot está livre (as restrições
// UNIQUE de (account_id, slot) e de nome também o impõem).
func (s *Store) CreateCharacter(ctx context.Context, accountID int64, ch domain.Character) (int64, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("store: begin create character: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	id, err := insertCharacter(ctx, tx, accountID, ch)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("store: commit create character: %w", err)
	}
	return id, nil
}

// CreateArchCharacter insere um personagem ARCH no primeiro slot livre da conta.
// O _MSG_DBCreateArchCharacter legado não é guiado pelo slot do cliente: o DBSrv varre
// MOB_PER_ACCOUNT e escolhe o primeiro slot vazio, preservando o nome do
// Mortal para o novo gêmeo Arch.
func (s *Store) CreateArchCharacter(ctx context.Context, accountID int64, ch domain.Character) (int64, int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("store: begin create arch character: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var lockedID int64
	err = tx.QueryRow(ctx, `SELECT id FROM account WHERE id = $1 FOR UPDATE`, accountID).Scan(&lockedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, 0, ErrNotFound
	}
	if err != nil {
		return 0, 0, fmt.Errorf("store: lock account %d for arch create: %w", accountID, err)
	}

	rows, err := tx.Query(ctx, `SELECT slot FROM characters WHERE account_id = $1 ORDER BY slot`, accountID)
	if err != nil {
		return 0, 0, fmt.Errorf("store: list slots for arch create: %w", err)
	}
	used := [4]bool{}
	for rows.Next() {
		var slot int
		if err := rows.Scan(&slot); err != nil {
			rows.Close()
			return 0, 0, fmt.Errorf("store: scan slot for arch create: %w", err)
		}
		if slot >= 0 && slot < len(used) {
			used[slot] = true
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, 0, fmt.Errorf("store: list slots for arch create: %w", err)
	}
	rows.Close()

	slot := -1
	for i, ok := range used {
		if !ok {
			slot = i
			break
		}
	}
	if slot < 0 {
		return 0, 0, ErrNoFreeSlot
	}

	ch.Slot = slot
	id, err := insertCharacter(ctx, tx, accountID, ch)
	if err != nil {
		return 0, 0, err
	}
	for _, it := range ch.Equip {
		if err := insertItem(ctx, tx, "char_equip", nil, &id, it); err != nil {
			return 0, 0, err
		}
	}
	for _, it := range ch.Carry {
		if err := insertItem(ctx, tx, "char_carry", nil, &id, it); err != nil {
			return 0, 0, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, 0, fmt.Errorf("store: commit create arch character: %w", err)
	}
	return id, slot, nil
}

// DeleteCharacter remove um personagem (e seus itens/affects via ON DELETE
// CASCADE) por conta e slot. Devolve ErrNotFound quando o slot está vazio.
func (s *Store) DeleteCharacter(ctx context.Context, accountID int64, slot int) error {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM characters WHERE account_id = $1 AND slot = $2`, accountID, slot)
	if err != nil {
		return fmt.Errorf("store: delete character a=%d slot=%d: %w", accountID, slot, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SaveCharacter persiste o estado vivo de um personagem nos saves do jogo
// (_MSG_DBSaveMob / SavingQuit). O personagem precisa já existir (casado por
// account_id + slot); equip, carry e affects são substituídos atomicamente.
//
// É uma atualização PARCIAL de propósito: toca só as colunas que a Entity
// do mundo rastreia com autoridade (world.CharacterSave) — clan, guild_id, level,
// guild_level, exp, coin, str/int/dex/con, hp/max_hp, mp/max_mp, last_city, o
// ponto de save do warp da Gema Estelar (save_x/y), o estado de skill que o mundo
// agora simula (score_bonus, special_bonus, learned_skill, sec_learned_skill, soul,
// fame, special, skill_bar, short_skill) e o estado de tier (class_master + os
// gates de quest celestial celestial_lv40/90/circle + o gate terra_mistica), e
// o estado de PK/karma (pk_point, guilty, cur_kill, tot_kill — issue #210),
// o passo da quest de newbie (newbie_quest, migration 0028 — issue #395),
// o gate do Molar de Gargula (molar_gargula, migration 0029 — issue #398),
// o gate da Pilula do Poder Magico (pilula_orc, migration 0030 — issue #404),
// o contador de reset Celestial (celestial_reset, migration 0031 — issue #408),
// as flags de add do diadema Sub-Celestial (celestial_add120..200, migration 0032 — issue #409),
// as fichas e níveis guardados de Celestial/Sub-Celestial (save_celestial,
// celestial_level, sub_celestial_level, migration 0033 — issue #410),
// e a base sem equipamento (base_str..base_max_mp, migration 0027). Quem chama
// que não manda a base (HasBase falso, por exemplo um tmServer anterior a 0027)
// a limpa, então o próximo login a deriva do CurrentScore recém-salvo
// em vez de confiar numa base que já não casa com ele.
// class também é salva: a Pedra Misteriosa (4148) troca a classe da ficha ativa,
// e o SaveUser legado escreve o STRUCT_MOB inteiro (Server.cpp:7845).
// Todo o resto (regen/resist, magia, cidadania) fica INTOCADO para um save in-game nunca apagar
// dados importados que o mundo não simula.
// skill_bonus também fica intocado: o tmServer o rederiva no login
// (BASE_GetBonusSkillPoint) em vez de confiar no valor guardado.
func (s *Store) SaveCharacter(ctx context.Context, accountID int64, ch domain.Character) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("store: begin save character: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var charID int64
	err = tx.QueryRow(ctx, `
		UPDATE characters SET
			clan=$3, guild_id=$4, guild_level=$5, level=$6, coin=$7,
			str=$8, int=$9, dex=$10, con=$11, hp=$12, max_hp=$13, last_city=$14, exp=$15,
			mp=$16, max_mp=$17,
			score_bonus=$18, special_bonus=$19, learned_skill=$20, sec_learned_skill=$21, soul=$22, fame=$23,
			special=$24, skill_bar=$25, short_skill=$26, save_x=$27, save_y=$28,
			class_master=$29, celestial_lv40=$30, celestial_lv90=$31, celestial_circle=$32, terra_mistica=$33,
			arch_lv355=$34, arch_lv370=$35, pk_point=$36, guilty=$37, cur_kill=$38, tot_kill=$39,
			mortal_level=$40, celestial_arch_level=$41, arch_crystal_stage=$42, kefra_ticket=$43, nightmare_entries=$44, last_nightmare_use=$45,
			base_str=CASE WHEN $46 THEN $47::smallint END,
			base_int=CASE WHEN $46 THEN $48::smallint END,
			base_dex=CASE WHEN $46 THEN $49::smallint END,
			base_con=CASE WHEN $46 THEN $50::smallint END,
			base_max_hp=CASE WHEN $46 THEN $51::integer END,
			base_max_mp=CASE WHEN $46 THEN $52::integer END,
			newbie_quest=$53,
			molar_gargula=$54,
			pilula_orc=$55,
			celestial_reset=$56,
			celestial_add120=$57, celestial_add150=$58, celestial_add180=$59, celestial_add200=$60,
			celestial_level=$61, sub_celestial_level=$62, save_celestial=COALESCE($63::bytea, ''::bytea),
			class=$64
		WHERE account_id=$1 AND slot=$2
		RETURNING id`,
		accountID, ch.Slot, ch.Clan, ch.GuildID, ch.GuildLevel, ch.Level, ch.Coin,
		ch.Str, ch.Int, ch.Dex, ch.Con, ch.Hp, ch.MaxHp, ch.LastCity, ch.Exp,
		ch.Mp, ch.MaxMp,
		ch.ScoreBonus, ch.SpecialBonus, ch.LearnedSkill, ch.SecLearnedSkill, ch.Soul,
		ch.Fame,
		ch.Special[:], byteArrToInt16Arr(ch.SkillBar[:]), byteArrToInt16Arr(ch.ShortSkill[:]),
		ch.SaveX, ch.SaveY,
		ch.ClassMaster, ch.CelLv40, ch.CelLv90, ch.CelCircle, ch.TerraMistica, ch.ArchLv355, ch.ArchLv370,
		ch.PKPoint, ch.Guilty, ch.CurKill, ch.TotKill, ch.MortalLevel, ch.CelestialArchLevel, ch.ArchCrystalStage, ch.KefraTicket, ch.NightmareEntries, ch.LastNightmareUse,
		ch.HasBase, ch.BaseStr, ch.BaseInt, ch.BaseDex, ch.BaseCon, ch.BaseMaxHp, ch.BaseMaxMp,
		ch.NewbieQuest,
		ch.MolarGargula,
		ch.PilulaOrc,
		ch.CelReset,
		ch.CelAdd120, ch.CelAdd150, ch.CelAdd180, ch.CelAdd200,
		ch.CelestialLevel, ch.SubCelestialLevel, ch.SaveCelestial,
		ch.Class,
	).Scan(&charID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("store: update character %q: %w", ch.Name, err)
	}

	if _, err := tx.Exec(ctx,
		`DELETE FROM item WHERE character_id=$1 AND owner_kind IN ('char_equip','char_carry')`, charID); err != nil {
		return fmt.Errorf("store: clear items: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM affect WHERE character_id=$1`, charID); err != nil {
		return fmt.Errorf("store: clear affects: %w", err)
	}
	for _, it := range ch.Equip {
		if err := insertItem(ctx, tx, "char_equip", nil, &charID, it); err != nil {
			return err
		}
	}
	for _, it := range ch.Carry {
		if err := insertItem(ctx, tx, "char_carry", nil, &charID, it); err != nil {
			return err
		}
	}
	for _, a := range ch.Affects {
		if _, err := tx.Exec(ctx,
			`INSERT INTO affect (character_id, type, value, level, time) VALUES ($1,$2,$3,$4,$5)`,
			charID, a.Type, a.Value, a.Level, a.Time); err != nil {
			return fmt.Errorf("store: insert affect: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("store: commit save character %q: %w", ch.Name, err)
	}
	return nil
}

// LoadCargo carrega o cargo compartilhado da conta: o ouro (account.cargo_coin) e os
// itens guardados (linhas de item com owner_kind='account_cargo'). O cargo tem chave
// por conta — todo personagem da conta compartilha o mesmo cofre — então é lido
// uma vez por sessão, independente do slot do personagem. Conta inexistente devolve
// ErrNotFound; conta sem itens guardados devolve (coin, nil).
func (s *Store) LoadCargo(ctx context.Context, accountID int64) (int32, []domain.Item, error) {
	var coin int32
	err := s.pool.QueryRow(ctx,
		`SELECT cargo_coin FROM account WHERE id = $1`, accountID).Scan(&coin)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil, ErrNotFound
	}
	if err != nil {
		return 0, nil, fmt.Errorf("store: load cargo coin a=%d: %w", accountID, err)
	}
	items, err := s.loadAccountItems(ctx, accountID, "account_cargo")
	if err != nil {
		return 0, nil, err
	}
	return coin, items, nil
}

// loadAccountItems espelha loadItems mas usa account_id como chave (o cargo é escopo
// de conta, não de personagem).
func (s *Store) loadAccountItems(ctx context.Context, accountID int64, kind string) ([]domain.Item, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT slot, item_index, eff1, effv1, eff2, effv2, eff3, effv3, expires_at
		  FROM item WHERE account_id = $1 AND owner_kind = $2 ORDER BY slot`, accountID, kind)
	if err != nil {
		return nil, fmt.Errorf("store: load %s: %w", kind, err)
	}
	defer rows.Close()
	var out []domain.Item
	for rows.Next() {
		var it domain.Item
		var exp *time.Time
		if err := rows.Scan(&it.Slot, &it.Index, &it.Eff1, &it.EffV1, &it.Eff2, &it.EffV2, &it.Eff3, &it.EffV3, &exp); err != nil {
			return nil, fmt.Errorf("store: scan %s item: %w", kind, err)
		}
		it.ExpiresAt = expirySeconds(exp)
		out = append(out, it)
	}
	return out, rows.Err()
}

// SaveCargo persiste ouro + itens do cargo compartilhado como substituição total,
// espelhando a troca atômica de itens do SaveCharacter (anti-duplicação: o conjunto
// antigo é apagado e o novo reinserido numa transação). Conta inexistente devolve
// ErrNotFound.
func (s *Store) SaveCargo(ctx context.Context, accountID int64, coin int32, items []domain.Item) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("store: begin save cargo: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `UPDATE account SET cargo_coin = $2 WHERE id = $1`, accountID, coin)
	if err != nil {
		return fmt.Errorf("store: update cargo coin a=%d: %w", accountID, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	if _, err := tx.Exec(ctx,
		`DELETE FROM item WHERE account_id = $1 AND owner_kind = 'account_cargo'`, accountID); err != nil {
		return fmt.Errorf("store: clear cargo items a=%d: %w", accountID, err)
	}
	for _, it := range items {
		if err := insertItem(ctx, tx, "account_cargo", &accountID, nil, it); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("store: commit save cargo a=%d: %w", accountID, err)
	}
	return nil
}

// int16ArrToByteArr estreita uma coluna smallint[] de volta para um vetor fixo de bytes,
// o inverso de byteArrToInt16. Elementos extras são ignorados; os ausentes ficam
// zero.
func int16ArrToByteArr(src []int16, dst []uint8) {
	for i := range dst {
		if i < len(src) {
			dst[i] = uint8(src[i])
		}
	}
}

// byteArrToInt16Arr alarga um vetor fixo de bytes para o formato da coluna smallint[].
func byteArrToInt16Arr(src []uint8) []int16 {
	out := make([]int16, len(src))
	for i, v := range src {
		out[i] = int16(v)
	}
	return out
}
