package worldstateadmin

import (
	"context"
	"fmt"
	"testing"

	"openwyd-nextgen/internal/domain"
	"openwyd-nextgen/internal/store"
)

type fakeStore struct {
	role      string
	guilds    []domain.Guild
	roster    []store.GuildMemberRow
	relations []domain.GuildRelation
	zones     []domain.GuildZone
	tower     domain.GuildTowerState
	castle    domain.CastleQuestState
	kefra     domain.KefraState

	savedZones      []domain.GuildZone
	kefraApplied    bool
	kefraFailure    bool
	masterApplied   bool
	removeApplied   bool
	relationApplied bool
	zoneApplied     bool
	towerApplied    bool
	writeCalls      []string
}

func (f *fakeStore) AccountRole(context.Context, int64) (string, error) {
	if f.role == "" {
		return "", store.ErrNotFound
	}
	return f.role, nil
}
func (f *fakeStore) ListGuilds(context.Context) ([]domain.Guild, error) { return f.guilds, nil }
func (f *fakeStore) ListGuildRelations(context.Context) ([]domain.GuildRelation, error) {
	return f.relations, nil
}
func (f *fakeStore) GuildRoster(context.Context) ([]store.GuildMemberRow, error) {
	return f.roster, nil
}
func (f *fakeStore) LoadGuildZones(context.Context) ([]domain.GuildZone, error) { return f.zones, nil }
func (f *fakeStore) SaveGuildZone(_ context.Context, z domain.GuildZone) error {
	f.savedZones = append(f.savedZones, z)
	return nil
}
func (f *fakeStore) LoadGuildTowerState(context.Context) (domain.GuildTowerState, error) {
	return f.tower, nil
}
func (f *fakeStore) LoadCastleQuestState(context.Context) (domain.CastleQuestState, error) {
	return f.castle, nil
}
func (f *fakeStore) LoadKefraState(context.Context) (domain.KefraState, error) { return f.kefra, nil }
func (f *fakeStore) SetKefraDefeated(context.Context, bool) (bool, error) {
	return f.kefraApplied, nil
}
func (f *fakeStore) SetGuildMasterByCharacter(_ context.Context, guildID int32, characterID, moderatorID int64, reason string) (bool, error) {
	f.writeCalls = append(f.writeCalls, fmt.Sprintf("master:%d:%d:%s", guildID, characterID, reason))
	return f.masterApplied, nil
}
func (f *fakeStore) RemoveGuildMemberByCharacter(_ context.Context, guildID int32, characterID, moderatorID int64, reason string) (bool, error) {
	f.writeCalls = append(f.writeCalls, fmt.Sprintf("remove:%d:%d:%s", guildID, characterID, reason))
	return f.removeApplied, nil
}
func (f *fakeStore) SetGuildRelationAudited(_ context.Context, guildID, targetGuildID int32, kind int, moderatorID int64, reason string) (bool, error) {
	f.writeCalls = append(f.writeCalls, fmt.Sprintf("relation:%d:%d:%d:%s", guildID, targetGuildID, kind, reason))
	return f.relationApplied, nil
}
func (f *fakeStore) SetZoneOwner(_ context.Context, zone, guildID int32, moderatorID int64, reason string) (bool, error) {
	f.writeCalls = append(f.writeCalls, fmt.Sprintf("zone:%d:%d:%s", zone, guildID, reason))
	return f.zoneApplied, nil
}
func (f *fakeStore) SetTowerOwner(_ context.Context, guildID int32, moderatorID int64, reason string) (bool, error) {
	f.writeCalls = append(f.writeCalls, fmt.Sprintf("tower:%d:%s", guildID, reason))
	return f.towerApplied, nil
}

func TestGetGroupsRosterIntoGuilds(t *testing.T) {
	f := &fakeStore{
		role: "admin",
		guilds: []domain.Guild{
			{ID: 1, Name: "Lua Negra", Clan: 2, Fame: 120, Citizen: 1},
			{ID: 2, Name: "Sem Mestre", Clan: 1},
		},
		roster: []store.GuildMemberRow{
			{GuildID: 1, CharacterID: 10, AccountID: 3, Slot: 0, Name: "Cantarelli", GuildLevel: 9},
			{GuildID: 1, CharacterID: 11, AccountID: 4, Slot: 1, Name: "Aux", GuildLevel: 6},
		},
		zones: []domain.GuildZone{
			{Zone: 0, ChargeGuild: 1, CityTax: 5, TaxVault: 900},
			{Zone: 1, ChallengeGuild: 1, CityTax: 5},
		},
		tower:  domain.GuildTowerState{OwnerGuild: 1, UpdatedAtUnix: 1700000000},
		castle: domain.CastleQuestState{Level: 3, TimeLeft: 60, Clear: true, LeaderName: "Cantarelli"},
		kefra:  domain.KefraState{Defeated: true, NextSpawnUnix: 1800000000, LastSpawnUnix: 1700000000, Revision: 7},
	}

	res, snap, err := New(f).Get(context.Background(), 1)
	if err != nil || res != OK {
		t.Fatalf("Get = %v/%v; quer OK", res, err)
	}
	if len(snap.Guilds) != 2 {
		t.Fatalf("guildas = %d; quer 2", len(snap.Guilds))
	}
	first := snap.Guilds[0]
	if first.MasterName != "Cantarelli" || len(first.Members) != 2 {
		t.Errorf("guilda 1 = mestre %q/%d membros; quer Cantarelli/2", first.MasterName, len(first.Members))
	}
	if !first.TowerOwner || len(first.CityZones) != 1 || first.CityZones[0] != 0 {
		t.Errorf("guilda 1 = torre %v cidades %v; quer dona da torre e da zona 0", first.TowerOwner, first.CityZones)
	}
	if snap.Guilds[1].MasterName != "" || len(snap.Guilds[1].Members) != 0 {
		t.Errorf("guilda sem membros ganhou dados: %+v", snap.Guilds[1])
	}
	if snap.TowerOwnerName != "Lua Negra" {
		t.Errorf("tower name = %q", snap.TowerOwnerName)
	}
	if snap.Zones[0].ChargeGuildName != "Lua Negra" {
		t.Errorf("nome da guilda no poder = %q", snap.Zones[0].ChargeGuildName)
	}
	if !snap.Kefra.Initialized || !snap.Kefra.Defeated || snap.Kefra.Revision != 7 {
		t.Errorf("kefra = %+v; quer inicializado, morto, revisão 7", snap.Kefra)
	}
}

func TestKefraUninitializedWhenNoRevision(t *testing.T) {
	f := &fakeStore{role: "moderator"}
	_, snap, err := New(f).Get(context.Background(), 1)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if snap.Kefra.Initialized {
		t.Error("kefra sem revisão não pode aparecer como inicializado")
	}
}

func TestSetCityTaxValidatesAndSavesWholeRow(t *testing.T) {
	f := &fakeStore{
		role:  "admin",
		zones: []domain.GuildZone{{Zone: 2, ChargeGuild: 1, CityTax: 5, TaxVault: 77, ChallengeMoney: 5}},
	}
	svc := New(f)

	if res, _ := svc.SetCityTax(context.Background(), 1, 2, MaxCityTax+1); res != Invalid {
		t.Errorf("taxa 31 = %v; quer Invalid", res)
	}
	if res, _ := svc.SetCityTax(context.Background(), 1, 9, 5); res != Invalid {
		t.Errorf("zona 9 = %v; quer Invalid", res)
	}
	if res, _ := svc.SetCityTax(context.Background(), 1, 3, 5); res != NotFound {
		t.Errorf("zona sem linha = %v; quer NotFound", res)
	}
	if res, err := svc.SetCityTax(context.Background(), 1, 2, 7); res != OK || err != nil {
		t.Fatalf("taxa 7 = %v/%v; quer OK", res, err)
	}
	if len(f.savedZones) != 1 {
		t.Fatalf("salvou %d zonas; quer 1", len(f.savedZones))
	}
	if got := f.savedZones[0]; got.CityTax != 7 || got.TaxVault != 77 || got.ChallengeMoney != 5 || got.ChargeGuild != 1 {
		t.Errorf("linha salva = %+v; quer só a taxa trocada", got)
	}
}

func TestSetKefraDefeatedReportsNullRow(t *testing.T) {
	f := &fakeStore{role: "admin", kefraApplied: false}
	if res, _ := New(f).SetKefraDefeated(context.Background(), 1, true); res != NotFound {
		t.Errorf("sem linha = %v; quer NotFound", res)
	}
	f.kefraApplied = true
	if res, _ := New(f).SetKefraDefeated(context.Background(), 1, true); res != OK {
		t.Errorf("com linha = %v; quer OK", res)
	}
}

// guildFixture is a guild whose level-9 leader and level-0 member are the only
// rows the write validations look at.
func guildFixture() *fakeStore {
	return &fakeStore{
		role:   "admin",
		guilds: []domain.Guild{{ID: 1, Name: "Lua Negra"}},
		roster: []store.GuildMemberRow{
			{GuildID: 1, CharacterID: 10, Name: "Chefe", GuildLevel: 9},
			{GuildID: 1, CharacterID: 11, Name: "Novato", GuildLevel: 0},
		},
	}
}

func TestSetGuildMasterValidations(t *testing.T) {
	f := guildFixture()
	f.masterApplied = true
	svc := New(f)
	ctx := context.Background()

	if res, _, _ := svc.SetGuildMaster(ctx, 1, 1, 11, "  "); res != Invalid {
		t.Errorf("sem motivo = %v; quer Invalid", res)
	}
	if res, _, _ := svc.SetGuildMaster(ctx, 1, 99, 11, "troca"); res != NotFound {
		t.Errorf("guilda 99 = %v; quer NotFound", res)
	}
	if res, _, _ := svc.SetGuildMaster(ctx, 1, 1, 77, "troca"); res != NotFound {
		t.Errorf("não-membro = %v; quer NotFound", res)
	}
	if res, _, _ := svc.SetGuildMaster(ctx, 1, 1, 10, "troca"); res != Invalid {
		t.Errorf("mestre atual = %v; quer Invalid", res)
	}
	if res, msg, err := svc.SetGuildMaster(ctx, 1, 1, 11, "troca"); res != OK || err != nil {
		t.Fatalf("troca válida = %v/%q/%v; quer OK", res, msg, err)
	}
	if len(f.writeCalls) != 1 || f.writeCalls[0] != "master:1:11:troca" {
		t.Errorf("escritas = %v; quer uma troca de mestre com motivo", f.writeCalls)
	}
}

func TestRemoveGuildMemberRefusesLeader(t *testing.T) {
	f := guildFixture()
	f.removeApplied = true
	svc := New(f)
	ctx := context.Background()

	if res, msg, _ := svc.RemoveGuildMember(ctx, 1, 1, 10, "saiu"); res != Invalid {
		t.Errorf("remover líder = %v (%q); quer Invalid", res, msg)
	}
	if len(f.writeCalls) != 0 {
		t.Fatalf("remover líder chegou ao banco: %v", f.writeCalls)
	}
	if res, _, err := svc.RemoveGuildMember(ctx, 1, 1, 11, "saiu"); res != OK || err != nil {
		t.Fatalf("remover membro = %v/%v; quer OK", res, err)
	}
}

func TestSetGuildRelationRejectsSelfAndUnknown(t *testing.T) {
	f := guildFixture()
	f.guilds = append(f.guilds, domain.Guild{ID: 2, Name: "Sol Alto"})
	f.relationApplied = true
	svc := New(f)
	ctx := context.Background()

	if res, _, _ := svc.SetGuildRelation(ctx, 1, 1, 1, 1, "aliada"); res != Invalid {
		t.Errorf("relação consigo = %v; quer Invalid", res)
	}
	if res, _, _ := svc.SetGuildRelation(ctx, 1, 1, 3, 1, "aliada"); res != NotFound {
		t.Errorf("guilda inexistente = %v; quer NotFound", res)
	}
	if res, _, _ := svc.SetGuildRelation(ctx, 1, 1, 2, 9, "aliada"); res != Invalid {
		t.Errorf("kind 9 = %v; quer Invalid", res)
	}
	if res, _, err := svc.SetGuildRelation(ctx, 1, 1, 2, 2, "guerra declarada"); res != OK || err != nil {
		t.Fatalf("relação válida = %v/%v; quer OK", res, err)
	}
}

func TestSetZoneOwnerValidations(t *testing.T) {
	f := guildFixture()
	f.zoneApplied = true
	svc := New(f)
	ctx := context.Background()

	if res, _, _ := svc.SetZoneOwner(ctx, 1, 5, 0, "cerco"); res != Invalid {
		t.Errorf("zona 5 = %v; quer Invalid", res)
	}
	if res, _, _ := svc.SetZoneOwner(ctx, 1, 0, 0, "  "); res != Invalid {
		t.Errorf("sem motivo = %v; quer Invalid", res)
	}
	if res, _, _ := svc.SetZoneOwner(ctx, 1, 0, 77, "cerco"); res != NotFound {
		t.Errorf("guilda 77 = %v; quer NotFound", res)
	}
	if res, _, err := svc.SetZoneOwner(ctx, 1, 0, 0, "cerco anulado"); res != OK || err != nil {
		t.Fatalf("limpar dono = %v/%v; quer OK", res, err)
	}
	if res, _, err := svc.SetZoneOwner(ctx, 1, 2, 1, "venceu o cerco"); res != OK || err != nil {
		t.Fatalf("definir dono = %v/%v; quer OK", res, err)
	}
	if len(f.writeCalls) != 2 {
		t.Errorf("escritas = %v; quer duas (limpar e definir)", f.writeCalls)
	}
}

func TestSetTowerOwnerValidations(t *testing.T) {
	f := guildFixture()
	f.towerApplied = true
	svc := New(f)
	ctx := context.Background()

	if res, _, _ := svc.SetTowerOwner(ctx, 1, 1, ""); res != Invalid {
		t.Errorf("sem motivo = %v; quer Invalid", res)
	}
	if res, _, _ := svc.SetTowerOwner(ctx, 1, 42, "torre"); res != NotFound {
		t.Errorf("guilda 42 = %v; quer NotFound", res)
	}
	if res, _, err := svc.SetTowerOwner(ctx, 1, 1, "torre conquistada"); res != OK || err != nil {
		t.Fatalf("dono da torre = %v/%v; quer OK", res, err)
	}
}

func TestPlayerIsForbidden(t *testing.T) {
	f := &fakeStore{role: "player"}
	svc := New(f)
	if res, _, _ := svc.Get(context.Background(), 5); res != Forbidden {
		t.Errorf("Get player = %v; quer Forbidden", res)
	}
	if res, _ := svc.SetCityTax(context.Background(), 5, 0, 5); res != Forbidden {
		t.Errorf("SetCityTax player = %v; quer Forbidden", res)
	}
	if res, _, _ := New(&fakeStore{}).Get(context.Background(), 0); res != Forbidden {
		t.Error("moderator_id 0 precisa ser Forbidden, não leitura")
	}
}
