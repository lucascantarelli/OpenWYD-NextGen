// Package worldstateadmin is the web-api side of the world-state admin surface
// (issue 45): guilds with their members and relations, the five kingdom zones
// with the city tax, the GTorre owner, the castle quest and the Kefra cycle.
//
// Two rules are structural, not stylistic:
//
//   - Every field comes from Postgres state that tmServer loads at boot. Nothing
//     here touches live game state, so a write reaches a running world only after
//     a restart — the portal states that on every card.
//   - The city tax is written through LoadGuildZones/SaveGuildZone, the same
//     whole-row round trip tmServer uses, instead of a targeted UPDATE: the
//     guild_zone row is one STRUCT_GUILDZONE, and saving it whole keeps the
//     challenge/vault fields from drifting.
package worldstateadmin

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jeanluca/w2pp-openwyd/internal/domain"
	"github.com/jeanluca/w2pp-openwyd/internal/store"
)

// MaxCityTax is the CHECK on guild_zone.city_tax.
const MaxCityTax = 30

// KingdomZones is the number of rows guild_zone is seeded with (0..4).
const KingdomZones = 5

// Store is the persistence surface the service needs (satisfied by *store.Store).
type Store interface {
	AccountRole(ctx context.Context, id int64) (string, error)
	ListGuilds(ctx context.Context) ([]domain.Guild, error)
	ListGuildRelations(ctx context.Context) ([]domain.GuildRelation, error)
	GuildRoster(ctx context.Context) ([]store.GuildMemberRow, error)
	LoadGuildZones(ctx context.Context) ([]domain.GuildZone, error)
	SaveGuildZone(ctx context.Context, z domain.GuildZone) error
	LoadGuildTowerState(ctx context.Context) (domain.GuildTowerState, error)
	LoadCastleQuestState(ctx context.Context) (domain.CastleQuestState, error)
	LoadKefraState(ctx context.Context) (domain.KefraState, error)
	SetKefraDefeated(ctx context.Context, defeated bool) (bool, error)
	SetGuildMasterByCharacter(ctx context.Context, guildID int32, characterID, moderatorID int64, reason string) (bool, error)
	RemoveGuildMemberByCharacter(ctx context.Context, guildID int32, characterID, moderatorID int64, reason string) (bool, error)
	SetGuildRelationAudited(ctx context.Context, guildID, targetGuildID int32, kind int, moderatorID int64, reason string) (bool, error)
	SetZoneOwner(ctx context.Context, zone, guildID int32, moderatorID int64, reason string) (bool, error)
	SetTowerOwner(ctx context.Context, guildID int32, moderatorID int64, reason string) (bool, error)
}

// Result mirrors web.proto AdminResult. Only infra failures become errors.
type Result int

const (
	// OK means the operation succeeded.
	OK Result = iota
	// Forbidden means the caller is not a moderator/admin.
	Forbidden
	// Invalid means the request failed validation.
	Invalid
	// NotFound means the target row does not exist yet.
	NotFound
)

// Member is one guild membership row.
type Member struct {
	CharacterID int64
	AccountID   int64
	Slot        int
	Name        string
	GuildLevel  int
}

// Guild is one guild plus everything the UI lists inline.
type Guild struct {
	ID         int32
	Name       string
	Clan       int
	Fame       int32
	Citizen    int
	MasterName string
	Members    []Member
	// CityZones are the kingdom zones where this guild is the charge guild.
	CityZones  []int32
	TowerOwner bool
}

// Relation is one directed ally/war row with both names resolved.
type Relation struct {
	GuildID       int32
	GuildName     string
	TargetGuildID int32
	TargetName    string
	Kind          int32
}

// Zone is one kingdom zone row; names are resolved from the guild table.
type Zone struct {
	Zone               int
	ChargeGuild        int32
	ChargeGuildName    string
	ChallengeGuild     int32
	ChallengeGuildName string
	Clan               int
	Victory            int
	CityTax            int
	ChallengeMoney     int64
	TaxVault           int64
}

// CastleQuest is the single castle/Zakum row.
type CastleQuest struct {
	Level      int32
	TimeLeft   int32
	Clear      bool
	LeaderName string
}

// Kefra is the single Kefra row. Initialized is false until tmServer saves it.
type Kefra struct {
	Initialized   bool
	Defeated      bool
	NextSpawnUnix int64
	LastSpawnUnix int64
	Revision      int64
}

// Snapshot is the whole world state the page renders.
type Snapshot struct {
	Guilds           []Guild
	Relations        []Relation
	Zones            []Zone
	TowerOwnerGuild  int32
	TowerOwnerName   string
	TowerUpdatedUnix int64
	Castle           CastleQuest
	Kefra            Kefra
}

// Service implements the world-state operations.
type Service struct {
	store Store
}

// New builds the service over the given store.
func New(s Store) *Service { return &Service{store: s} }

// Get returns the whole snapshot in one round trip: guilds already carry their
// members, master and cities, so the page never does N+1 queries.
func (s *Service) Get(ctx context.Context, moderatorID int64) (Result, Snapshot, error) {
	if res, err := s.authorize(ctx, moderatorID); res != OK || err != nil {
		return res, Snapshot{}, err
	}

	guilds, err := s.store.ListGuilds(ctx)
	if err != nil {
		return Invalid, Snapshot{}, fmt.Errorf("worldstateadmin: list guilds: %w", err)
	}
	roster, err := s.store.GuildRoster(ctx)
	if err != nil {
		return Invalid, Snapshot{}, fmt.Errorf("worldstateadmin: guild roster: %w", err)
	}
	relations, err := s.store.ListGuildRelations(ctx)
	if err != nil {
		return Invalid, Snapshot{}, fmt.Errorf("worldstateadmin: guild relations: %w", err)
	}
	zoneRows, err := s.store.LoadGuildZones(ctx)
	if err != nil {
		return Invalid, Snapshot{}, fmt.Errorf("worldstateadmin: guild zones: %w", err)
	}

	names := make(map[uint16]string, len(guilds))
	for _, g := range guilds {
		names[g.ID] = g.Name
	}

	tower, err := s.store.LoadGuildTowerState(ctx)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return Invalid, Snapshot{}, fmt.Errorf("worldstateadmin: tower state: %w", err)
	}
	castle, err := s.store.LoadCastleQuestState(ctx)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return Invalid, Snapshot{}, fmt.Errorf("worldstateadmin: castle quest: %w", err)
	}
	kefra, err := s.store.LoadKefraState(ctx)
	if err != nil {
		return Invalid, Snapshot{}, fmt.Errorf("worldstateadmin: kefra state: %w", err)
	}

	snap := Snapshot{
		Relations:        make([]Relation, 0, len(relations)),
		Zones:            make([]Zone, 0, len(zoneRows)),
		TowerOwnerGuild:  int32(tower.OwnerGuild),
		TowerOwnerName:   names[tower.OwnerGuild],
		TowerUpdatedUnix: tower.UpdatedAtUnix,
		Castle: CastleQuest{
			Level: castle.Level, TimeLeft: castle.TimeLeft,
			Clear: castle.Clear, LeaderName: castle.LeaderName,
		},
		Kefra: Kefra{
			Initialized:   kefra.Revision > 0,
			Defeated:      kefra.Defeated,
			NextSpawnUnix: kefra.NextSpawnUnix,
			LastSpawnUnix: kefra.LastSpawnUnix,
			Revision:      kefra.Revision,
		},
	}

	for _, z := range zoneRows {
		snap.Zones = append(snap.Zones, Zone{
			Zone:               z.Zone,
			ChargeGuild:        int32(z.ChargeGuild),
			ChargeGuildName:    names[z.ChargeGuild],
			ChallengeGuild:     int32(z.ChallengeGuild),
			ChallengeGuildName: names[z.ChallengeGuild],
			Clan:               int(z.Clan),
			Victory:            int(z.Victory),
			CityTax:            int(z.CityTax),
			ChallengeMoney:     z.ChallengeMoney,
			TaxVault:           z.TaxVault,
		})
	}

	// Group the roster by guild in one pass, keeping the level-9 member as the
	// master. Members whose guild row disappeared are dropped: the cascade on
	// guild_member.guild_id is what should have removed them.
	members := make(map[int32][]Member, len(guilds))
	master := make(map[int32]string, len(guilds))
	for _, m := range roster {
		members[m.GuildID] = append(members[m.GuildID], Member{
			CharacterID: m.CharacterID, AccountID: m.AccountID,
			Slot: int(m.Slot), Name: m.Name, GuildLevel: int(m.GuildLevel),
		})
		if m.GuildLevel == 9 {
			master[m.GuildID] = m.Name
		}
	}

	snap.Guilds = make([]Guild, 0, len(guilds))
	for _, g := range guilds {
		guild := Guild{
			ID:         int32(g.ID),
			Name:       g.Name,
			Clan:       int(g.Clan),
			Fame:       g.Fame,
			Citizen:    int(g.Citizen),
			MasterName: master[int32(g.ID)],
			Members:    members[int32(g.ID)],
			TowerOwner: g.ID == tower.OwnerGuild,
		}
		for _, z := range zoneRows {
			if z.ChargeGuild == g.ID {
				guild.CityZones = append(guild.CityZones, int32(z.Zone))
			}
		}
		snap.Guilds = append(snap.Guilds, guild)
	}

	for _, r := range relations {
		snap.Relations = append(snap.Relations, Relation{
			GuildID:       int32(r.GuildID),
			GuildName:     names[r.GuildID],
			TargetGuildID: int32(r.TargetGuildID),
			TargetName:    names[r.TargetGuildID],
			Kind:          int32(r.Kind),
		})
	}

	return OK, snap, nil
}

// SetCityTax rewrites one kingdom zone with a new tax. The whole row is saved so
// the challenge/vault fields keep their current values.
func (s *Service) SetCityTax(ctx context.Context, moderatorID int64, zone, tax int) (Result, error) {
	if res, err := s.authorize(ctx, moderatorID); res != OK || err != nil {
		return res, err
	}
	if zone < 0 || zone >= KingdomZones {
		return Invalid, nil
	}
	if tax < 0 || tax > MaxCityTax {
		return Invalid, nil
	}

	zones, err := s.store.LoadGuildZones(ctx)
	if err != nil {
		return Invalid, fmt.Errorf("worldstateadmin: load zones: %w", err)
	}
	for _, z := range zones {
		if z.Zone != zone {
			continue
		}
		z.CityTax = uint8(tax)
		if err := s.store.SaveGuildZone(ctx, z); err != nil {
			return Invalid, fmt.Errorf("worldstateadmin: save zone %d: %w", zone, err)
		}
		return OK, nil
	}
	return NotFound, nil
}

// SetKefraDefeated flips the Kefra flag. Refused with NotFound before tmServer
// ever saved a spawn: the row's CHECKs need both timestamps, and inventing them
// here would fight the world's schedule.
func (s *Service) SetKefraDefeated(ctx context.Context, moderatorID int64, defeated bool) (Result, error) {
	if res, err := s.authorize(ctx, moderatorID); res != OK || err != nil {
		return res, err
	}
	applied, err := s.store.SetKefraDefeated(ctx, defeated)
	if err != nil {
		return Invalid, fmt.Errorf("worldstateadmin: set kefra: %w", err)
	}
	if !applied {
		return NotFound, nil
	}
	return OK, nil
}

// SetGuildMaster promotes one member to level 9. The validations live here rather
// than in the store so the refusal can say *why* (não é membro, já é mestre, falta
// motivo) — the store keeps only the transactional half.
func (s *Service) SetGuildMaster(ctx context.Context, moderatorID int64, guildID int32, characterID int64, reason string) (Result, string, error) {
	if res, err := s.authorize(ctx, moderatorID); res != OK || err != nil {
		return res, "", err
	}
	if strings.TrimSpace(reason) == "" {
		return Invalid, "informe o motivo: troca de mestre fica na auditoria", nil
	}
	guild, members, err := s.guildAndMembers(ctx, guildID)
	if err != nil {
		return Invalid, "", err
	}
	if guild == nil {
		return NotFound, "guilda não encontrada", nil
	}
	target := findMember(members, characterID)
	if target == nil {
		return NotFound, "personagem não é membro desta guilda", nil
	}
	if target.GuildLevel == 9 {
		return Invalid, "este personagem já é o mestre", nil
	}

	applied, err := s.store.SetGuildMasterByCharacter(ctx, guildID, characterID, moderatorID, reason)
	if err != nil {
		return Invalid, "", fmt.Errorf("worldstateadmin: set guild master: %w", err)
	}
	if !applied {
		return NotFound, "o vínculo mudou durante a operação; recarregue a página", nil
	}
	return OK, "", nil
}

// RemoveGuildMember drops one membership. The leader is refused on purpose: a
// guild whose level-9 row disappeared can no longer be handed over by the game.
func (s *Service) RemoveGuildMember(ctx context.Context, moderatorID int64, guildID int32, characterID int64, reason string) (Result, string, error) {
	if res, err := s.authorize(ctx, moderatorID); res != OK || err != nil {
		return res, "", err
	}
	if strings.TrimSpace(reason) == "" {
		return Invalid, "informe o motivo: remoção de membro fica na auditoria", nil
	}
	guild, members, err := s.guildAndMembers(ctx, guildID)
	if err != nil {
		return Invalid, "", err
	}
	if guild == nil {
		return NotFound, "guilda não encontrada", nil
	}
	target := findMember(members, characterID)
	if target == nil {
		return NotFound, "personagem não é membro desta guilda", nil
	}
	if target.GuildLevel == 9 {
		return Invalid, "transfira o mestrado antes de remover o líder", nil
	}

	applied, err := s.store.RemoveGuildMemberByCharacter(ctx, guildID, characterID, moderatorID, reason)
	if err != nil {
		return Invalid, "", fmt.Errorf("worldstateadmin: remove guild member: %w", err)
	}
	if !applied {
		return NotFound, "o vínculo mudou durante a operação; recarregue a página", nil
	}
	return OK, "", nil
}

// SetGuildRelation upserts (1 ally, 2 war) or removes (0) one directed relation.
func (s *Service) SetGuildRelation(ctx context.Context, moderatorID int64, guildID, targetGuildID int32, kind int, reason string) (Result, string, error) {
	if res, err := s.authorize(ctx, moderatorID); res != OK || err != nil {
		return res, "", err
	}
	if kind < 0 || kind > 2 {
		return Invalid, "tipo de relação inválido (0 remove, 1 aliada, 2 inimiga)", nil
	}
	if guildID <= 0 || targetGuildID <= 0 || guildID == targetGuildID {
		return Invalid, "escolha outra guilda: uma relação não pode apontar para si mesma", nil
	}
	if strings.TrimSpace(reason) == "" {
		return Invalid, "informe o motivo: relações ficam na auditoria", nil
	}

	guilds, err := s.store.ListGuilds(ctx)
	if err != nil {
		return Invalid, "", fmt.Errorf("worldstateadmin: list guilds for relation: %w", err)
	}
	known := make(map[uint16]bool, len(guilds))
	for _, g := range guilds {
		known[g.ID] = true
	}
	if !known[uint16(guildID)] {
		return NotFound, "guilda de origem não encontrada", nil
	}
	if !known[uint16(targetGuildID)] {
		return NotFound, "guilda de destino não encontrada", nil
	}

	applied, err := s.store.SetGuildRelationAudited(ctx, guildID, targetGuildID, kind, moderatorID, reason)
	if err != nil {
		return Invalid, "", fmt.Errorf("worldstateadmin: set guild relation: %w", err)
	}
	if !applied {
		return NotFound, "não havia relação entre essas guildas para remover", nil
	}
	return OK, "", nil
}

// SetZoneOwner sets (or clears with guildID 0) the guild in charge of one kingdom
// zone. The tax is a separate call on purpose: a siege result that also moved the
// shop tax would be a silent economy change.
func (s *Service) SetZoneOwner(ctx context.Context, moderatorID int64, zone, guildID int32, reason string) (Result, string, error) {
	if res, err := s.authorize(ctx, moderatorID); res != OK || err != nil {
		return res, "", err
	}
	if zone < 0 || zone >= KingdomZones {
		return Invalid, "zona inválida: o reino tem 5 zonas (0..4)", nil
	}
	if strings.TrimSpace(reason) == "" {
		return Invalid, "informe o motivo: troca de dono de cidade fica na auditoria", nil
	}
	if res, msg, err := s.validateGuild(ctx, guildID); res != OK || err != nil {
		return res, msg, err
	}

	applied, err := s.store.SetZoneOwner(ctx, zone, guildID, moderatorID, reason)
	if err != nil {
		return Invalid, "", fmt.Errorf("worldstateadmin: set zone owner: %w", err)
	}
	if !applied {
		return NotFound, "nada a fazer: a zona não existe ou já tem esse dono", nil
	}
	return OK, "", nil
}

// SetTowerOwner sets (or clears with guildID 0) the single GTorre owner.
func (s *Service) SetTowerOwner(ctx context.Context, moderatorID int64, guildID int32, reason string) (Result, string, error) {
	if res, err := s.authorize(ctx, moderatorID); res != OK || err != nil {
		return res, "", err
	}
	if strings.TrimSpace(reason) == "" {
		return Invalid, "informe o motivo: troca de dono da torre fica na auditoria", nil
	}
	if res, msg, err := s.validateGuild(ctx, guildID); res != OK || err != nil {
		return res, msg, err
	}

	applied, err := s.store.SetTowerOwner(ctx, guildID, moderatorID, reason)
	if err != nil {
		return Invalid, "", fmt.Errorf("worldstateadmin: set tower owner: %w", err)
	}
	if !applied {
		return NotFound, "nada a fazer: a torre já tem esse dono ou a linha não existe", nil
	}
	return OK, "", nil
}

// validateGuild accepts guildID 0 ("sem dono") or the id of a registered guild.
func (s *Service) validateGuild(ctx context.Context, guildID int32) (Result, string, error) {
	if guildID == 0 {
		return OK, "", nil
	}
	if guildID < 0 {
		return Invalid, "guilda inválida", nil
	}
	guilds, err := s.store.ListGuilds(ctx)
	if err != nil {
		return Invalid, "", fmt.Errorf("worldstateadmin: list guilds for owner: %w", err)
	}
	for _, g := range guilds {
		if int32(g.ID) == guildID {
			return OK, "", nil
		}
	}
	return NotFound, "guilda não encontrada", nil
}

// guildAndMembers resolves one guild and its members from the same two reads the
// snapshot uses.
func (s *Service) guildAndMembers(ctx context.Context, guildID int32) (*Guild, []Member, error) {
	guilds, err := s.store.ListGuilds(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("worldstateadmin: list guilds: %w", err)
	}
	roster, err := s.store.GuildRoster(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("worldstateadmin: guild roster: %w", err)
	}

	var found *Guild
	for _, g := range guilds {
		if int32(g.ID) == guildID {
			guild := Guild{ID: int32(g.ID), Name: g.Name}
			found = &guild
			break
		}
	}
	var members []Member
	for _, m := range roster {
		if m.GuildID != guildID {
			continue
		}
		members = append(members, Member{
			CharacterID: m.CharacterID, AccountID: m.AccountID,
			Slot: int(m.Slot), Name: m.Name, GuildLevel: int(m.GuildLevel),
		})
	}
	return found, members, nil
}

func findMember(members []Member, characterID int64) *Member {
	for i := range members {
		if members[i].CharacterID == characterID {
			return &members[i]
		}
	}
	return nil
}

func (s *Service) authorize(ctx context.Context, moderatorID int64) (Result, error) {
	if moderatorID <= 0 {
		return Forbidden, nil
	}
	role, err := s.store.AccountRole(ctx, moderatorID)
	if errors.Is(err, store.ErrNotFound) {
		return Forbidden, nil
	}
	if err != nil {
		return Invalid, fmt.Errorf("worldstateadmin: role lookup %d: %w", moderatorID, err)
	}
	if role != "moderator" && role != "admin" {
		return Forbidden, nil
	}
	return OK, nil
}
