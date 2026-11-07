package world

import (
	"context"
	"errors"
)

// MobPerAccount é MOB_PER_ACCOUNT (Basedef.h:131): o número de slots de
// personagem por conta.
const MobPerAccount = 4

// LoginResult espelha os resultados de login de conta do dbServer (api/db/v1, derivados
// das mensagens legadas _MSG_DBAccountLoginFail_*, protocol-spec.md §3.3).
type LoginResult int

// Resultados de login de conta.
const (
	LoginOK LoginResult = iota
	LoginNoAccount
	LoginBadPassword
	LoginBlocked
	LoginAlreadyPlaying
)

// PinResult é o resultado de uma verificação de PIN numérico (AccountSecure), espelhando o
// enum PinResult do dbServer. NoAccount/NotSet/BadPin deixam o handler distinguir uma
// configuração inicial de PIN de uma recusa.
type PinResult int

// Resultados de verificação de PIN.
const (
	PinOK PinResult = iota
	PinNoAccount
	PinNotSet
	PinBadPin
)

// CharSummary é a projeção da seleção de personagem (subconjunto de STRUCT_SELCHAR): os
// dados por slot que a tela de seleção pré-visualiza, incluindo o score (nível, ouro,
// HP/MP, atributos) para o slot mostrar o personagem real, não placeholders.
type CharSummary struct {
	Slot    int
	Name    string
	Class   int
	Level   int
	Exp     int64
	GuildID uint16
	Coin    int32
	MaxHp   int32
	Hp      int32
	MaxMp   int32
	Mp      int32
	Str     int16
	Int     int16
	Dex     int16
	Con     int16
	Equip   [MaxEquip]Item
}

// LoginOutcome é o resultado de uma tentativa de login de conta. Em caso de sucesso ele também
// carrega o cargo compartilhado da conta e a caixa de entrada pendente da loja de doação web
// (issue #34), ambos carregados na mesma ida-e-volta ao backend que a lista de personagens
// (têm escopo de conta, então são buscados uma vez por login de conta).
type LoginOutcome struct {
	Result            LoginResult
	AccountID         int64
	Role              string // account.role ('player'/'moderator'/'admin'); autorização de GM (issue #122)
	Characters        []CharSummary
	Cargo             CargoState
	PendingDeliveries []Delivery
}

// CargoState é o armazém compartilhado da conta (o STRUCT_ACCOUNTFILE legado
// Cargo[MAX_CARGO] + CargoMoney). Tem escopo de conta — todos os personagens de uma conta
// depositam e sacam deste único cofre — então o mundo o mantém
// num store por conta, não na Entity por personagem. Os itens são
// posicionais (Index==0 é slot vazio).
type CargoState struct {
	AccountID int64
	Coin      int32
	Items     [MaxCargo]Item
}

// CargoSave é o snapshot que o mundo entrega ao backend para persistir o cargo
// (espelha CharacterSave). Slots vazios são omitidos de Items.
type CargoSave struct {
	AccountID int64
	Coin      int32
	Items     []SavedItem
}

// Delivery é um grant pendente que o loop drena da caixa de entrada delivery_queue
// para o cargo da conta (loja de doação web, issue #34). ID é o id da linha da fila,
// confirmado quando o item é aplicado (ou perdido quando o cargo está cheio).
type Delivery struct {
	ID   int64
	Item Item
}

// CharacterState é o mínimo necessário para injetar um jogador no mundo no
// login de personagem. O snapshot completo de STRUCT_MOB para o
// _MSG_CNFCharacterLogin byte a byte está UNVERIFIED (o layout do SELCHAR/snapshot não está
// documentado por completo) e será completado quando capturado.
type CharacterState struct {
	Slot               int
	Name               string
	Class              int
	Level              int
	Exp                int64
	X                  int16
	Y                  int16
	LastCity           int16 // última cidade (0..3); spawn de login = a área padrão dessa cidade
	SaveX              int16 // ponto de save do warp da Gema Estelar (STRUCT_MOB.SPX), 0 = nunca definido
	SaveY              int16 // ponto de save do warp da Gema Estelar (STRUCT_MOB.SPY), 0 = nunca definido
	HP                 int32
	MaxHP              int32
	MP                 int32
	MaxMP              int32
	Damage             int32 // CurrentScore.Damage
	AC                 int32 // CurrentScore.Ac
	Master             int   // maestria de arma
	Critical           uint8
	Coin               int32
	Clan               uint8
	GuildID            uint16
	GuildLevel         uint8
	Citizen            uint8 // MobExtra.Citizen; lealdade de cidade e metadados de criação de guilda
	ClassMaster        uint8
	CelLv40            uint8 // gate QuestInfo.Celestial.Lv40
	CelLv90            uint8 // gate QuestInfo.Celestial.Lv90
	CelCircle          uint8 // QuestInfo.Circle (quest Arcana concluída)
	CelReset           uint8 // QuestInfo.Celestial.Reset: resets de Pedra da Furia feitos 0..3 (issue #408)
	CelAdd120          uint8 // QuestInfo.Celestial.Add120: add do diadema Sub-Celestial concedido (issue #409)
	CelAdd150          uint8 // QuestInfo.Celestial.Add150: add do diadema Sub-Celestial concedido (issue #409)
	CelAdd180          uint8 // QuestInfo.Celestial.Add180: add do diadema Sub-Celestial concedido (issue #409)
	CelAdd200          uint8 // QuestInfo.Celestial.Add200: add do diadema Sub-Celestial concedido (issue #409)
	CelestialLevel     int16 // QuestInfo.Celestial.CelestialLevel (issue #410)
	SubCelestialLevel  int16 // QuestInfo.Celestial.SubCelestialLevel (issue #410)
	TerraMistica       uint8 // gate QuestInfo.Mortal.TerraMistica (AMU_MISTICO, issue #139)
	NewbieQuest        uint8 // QuestInfo.Mortal.Newbie: passo do Campo de Treino (issue #395)
	MolarGargula       uint8 // QuestInfo.Mortal.MolarGargula: Molar de Gargula usado (issue #398)
	PilulaOrc          uint8 // QuestInfo.Mortal.PilulaOrc: Pilula do Poder Magico usada, +9 SkillBonus (issue #404)
	ArchLv355          uint8
	ArchLv370          uint8
	MortalLevel        uint16
	CelestialArchLevel uint8
	ArchCrystalStage   uint8
	Soul               uint8
	Fame               int32 // MobExtra.Fame
	KefraTicket        int32 // Entradas restantes no Salão de Kefra.
	NightmareEntries   int32 // Entradas restantes no Pesadelo Arcano.
	LastNightmareUse   int64 // Horário Unix do último uso do título do Pesadelo.
	// Estado de PK/karma (slot de carry KILL_MARK do GetFunc.cpp, issue #210). PKPoint == 0
	// significa "nunca persistido" (o SetPKPoint nunca grava 0 legitimamente) — o caminho
	// de login trata isso como neutro (75), a mesma convenção de ClassMaster == 0.
	PKPoint    uint8
	Guilty     uint8
	CurKill    uint8
	TotKill    uint16
	Str        int16
	Int        int16
	Dex        int16
	Con        int16
	ScoreBonus uint16
	DivineEnd  int64 // prazo em segundos Unix do buff Divino (0 = nenhum)

	// BaseScore sem equipamento como foi salvo por último (migration 0027). Str..Con e
	// MaxHP/MaxMP são do CurrentScore. HasBase é falso para um personagem salvo
	// antes da base ser persistida: o login então a deriva por subtração,
	// uma vez, e o save seguinte a guarda.
	HasBase   bool
	BaseStr   int16
	BaseInt   int16
	BaseDex   int16
	BaseCon   int16
	BaseMaxHP int32
	BaseMaxMP int32

	// Estado de skills (frente de skills). SkillBonus não é carregado do banco — o
	// caminho de login o rederiva de Level e LearnedSkill, como a
	// BASE_GetBonusSkillPoint faz ao carregar o personagem.
	SpecialBonus    uint16
	LearnedSkill    int32
	SecLearnedSkill int32
	Magic           int16
	BaseSpecial     [4]int16 // pontos de maestria alocados (BaseScore.Special)
	SkillBar        [4]uint8
	ShortSkill      [16]uint8

	// Affects são os slots de buff persistidos (menos Divine, que viaja como
	// DivineEnd — o Time dele é um prazo em wall-clock, não ticks).
	Affects []Affect

	// SaveCelestial é MobExtra.SaveCelestial: as fichas estacionadas de
	// Celestial/Sub-Celestial (issue #410).
	SaveCelestial [2]CelestialSheet

	Equip [MaxEquip]Item // equipamento equipado
	Carry [MaxCarry]Item // inventário
}

// SavedItem é um slot posicional de inventário/equipamento num CharacterSave. Slot é
// o índice do vetor (significado posicional preservado); slots vazios são omitidos.
type SavedItem struct {
	Slot      int
	Index     int16
	Eff1      uint8
	EffV1     uint8
	Eff2      uint8
	EffV2     uint8
	Eff3      uint8
	EffV3     uint8
	ExpiresAt int64 // expiração em segundos Unix para itens temporizados (0 = permanente)
}

// CharacterSave é o snapshot que o mundo entrega ao backend de persistência no
// shutdown. Ele carrega SÓ os campos que a Entity em jogo acompanha de forma
// autoritativa nesta fase (domain-model.md §2.2): a posição de entrada no mundo (X/Y)
// ainda não é persistida, e os scores derivados de classe/combate faltam porque o
// mundo ainda não os simula como campos autoritativos (PROGRESSO Fase 4 —
// STRUCT_MOB completo está UNVERIFIED). Exp É persistido agora (ganho em abates), e
// também nível de guilda, fama do personagem e o ponto de save do warp da Gema Estelar
// (SaveX/SaveY) — distinto da posição de entrada no mundo não persistida. O mundo
// o monta (é dono da Entity); o adaptador só o transporta.
type CharacterSave struct {
	AccountID  int64
	Slot       int
	LastCity   int16
	SaveX      int16 // ponto de save do warp da Gema Estelar (STRUCT_MOB.SPX)
	SaveY      int16 // ponto de save do warp da Gema Estelar (STRUCT_MOB.SPY)
	Class      uint8 // a Pedra Misteriosa (4148) a troca; o SaveUser legado envia o STRUCT_MOB inteiro (Server.cpp:7845)
	Clan       uint8
	GuildID    uint16
	GuildLevel uint8
	Level      int32
	Exp        int64
	Coin       int32
	Str        int16
	Int        int16
	Dex        int16
	Con        int16
	HP         int32
	MaxHP      int32
	MP         int32
	MaxMP      int32
	DivineEnd  int64 // prazo em segundos Unix do buff Divino (0 = nenhum/expirado)

	// BaseScore sem equipamento (BaseStr..BaseCon, BaseMaxHP/MP), salvo para o
	// login nunca precisar reconstruí-lo a partir do CurrentScore (migration 0027).
	BaseStr   int16
	BaseInt   int16
	BaseDex   int16
	BaseCon   int16
	BaseMaxHP int32
	BaseMaxMP int32

	ScoreBonus       uint16
	SpecialBonus     uint16
	LearnedSkill     int32
	SecLearnedSkill  int32
	Soul             uint8
	Fame             int32
	KefraTicket      int32 // Entradas restantes no Salão de Kefra.
	NightmareEntries int32 // Entradas restantes no Pesadelo Arcano.
	LastNightmareUse int64 // Horário Unix do último uso do título do Pesadelo.
	// Estado de tier persistido pelo save in-game (do mundo): ClassMaster carrega
	// transformações de tier; CelLv40/CelLv90/CelCircle são os gates de quest celestial.
	ClassMaster        uint8
	CelLv40            uint8
	CelLv90            uint8
	CelCircle          uint8
	CelReset           uint8
	CelAdd120          uint8
	CelAdd150          uint8
	CelAdd180          uint8
	CelAdd200          uint8
	CelestialLevel     int16
	SubCelestialLevel  int16
	SaveCelestial      [2]CelestialSheet
	TerraMistica       uint8
	NewbieQuest        uint8
	MolarGargula       uint8
	PilulaOrc          uint8
	ArchLv355          uint8
	ArchLv370          uint8
	MortalLevel        uint16
	CelestialArchLevel uint8
	ArchCrystalStage   uint8
	// Estado de PK/karma (issue #210) — veja CharacterState para o significado dos campos.
	PKPoint     uint8
	Guilty      uint8
	CurKill     uint8
	TotKill     uint16
	BaseSpecial [4]int16
	SkillBar    [4]uint8
	ShortSkill  [16]uint8
	Affects     []Affect // slots de buff ativos (menos Divine — veja DivineEnd)

	Carry []SavedItem
	Equip []SavedItem
}

// KingdomCapeQuote é uma revisão durável de preço compartilhada pelos dois reinos.
type KingdomCapeQuote struct {
	Revision      int64
	HekalotiaCost int
	AkeloniaCost  int
}

// GuildRecord é a linha do registro de guildas voltada ao tmServer. ID é o
// ushort legado guardado em Entity.Guild.
type GuildRecord struct {
	ID      uint16
	Name    string
	Clan    uint8
	Fame    int32
	Citizen uint8
}

// GuildRelationKind identifica uma relação de guilda direcionada.
type GuildRelationKind uint8

// Tipos de relação de guilda.
const (
	GuildRelationNone GuildRelationKind = iota
	GuildRelationAlly
	GuildRelationWar
)

// GuildRelation é uma relação direcionada de aliança/guerra.
type GuildRelation struct {
	GuildID       uint16
	TargetGuildID uint16
	Kind          GuildRelationKind
}

// GuildZone é o subconjunto persistido de posse de zona/cidade de guilda.
type GuildZone struct {
	Zone           int
	ChargeGuild    uint16
	ChallengeGuild uint16
	Clan           uint8
	Victory        uint8
	CityTax        uint8
	ChallengeMoney int64
	TaxVault       int64
}

// GuildTowerState guarda o dono atual da GTorre.
type GuildTowerState struct {
	OwnerGuild    uint16
	UpdatedAtUnix int64
}

// CastleQuestState guarda o estado da única quest Castle/Zakum ativa.
type CastleQuestState struct {
	Level      int32
	TimeLeft   int32
	Clear      bool
	LeaderName string
}

// Persistence é a porta que o loop/handlers usam para falar com o dbServer. A
// implementação real é um adaptador de cliente gRPC sobre api/db/v1; o mundo depende
// só desta interface (migration-plan.md §3.5). AccountLogin/CreateCharacter/
// DeleteCharacter/LoadCharacter são chamados FORA do loop via World.Go (I/O
// bloqueante); SaveOnShutdown é chamado inline durante a drenagem de shutdown.
type Persistence interface {
	LoadKefraState(ctx context.Context) (KefraState, error)
	SaveKefraState(ctx context.Context, state KefraState) error
	SaveOnShutdown(ctx context.Context, save CharacterSave) error
	QuoteKingdomCape(ctx context.Context) (KingdomCapeQuote, error)
	PurchaseKingdomCape(ctx context.Context, expectedRevision int64, kingdom uint8, save CharacterSave) (KingdomCapeQuote, bool, error)
	AccountLogin(ctx context.Context, name, password string) (LoginOutcome, error)
	ListCharacters(ctx context.Context, accountID int64) ([]CharSummary, error)
	CreateCharacter(ctx context.Context, accountID int64, slot int, name string, class int) (bool, error)
	CreateArchCharacter(ctx context.Context, accountID int64, name string, class, mortalFace, mortalSlot, mortalLevel int) (int, bool, error)
	DeleteCharacter(ctx context.Context, accountID int64, slot int, name, password string) (bool, error)
	// SetPin define/troca o PIN numérico da conta (com hash argon2id no
	// dbServer). VerifyPin verifica um PIN. Ambos rodam fora do loop via World.Go.
	SetPin(ctx context.Context, accountID int64, pin string) (bool, error)
	VerifyPin(ctx context.Context, accountID int64, pin string) (PinResult, error)
	LoadCharacter(ctx context.Context, accountID int64, slot int) (CharacterState, error)
	LoadCargo(ctx context.Context, accountID int64) (CargoState, error)
	SaveCargo(ctx context.Context, save CargoSave) error
	// ListPendingDeliveries devolve os grants de item pendentes da conta na
	// caixa de entrada delivery_queue (issue #34). Chamado fora do loop no login.
	ListPendingDeliveries(ctx context.Context, accountID int64) ([]Delivery, error)
	// SaveCargoWithDeliveries persiste o cargo (substitui tudo) e marca as
	// linhas drenadas da caixa como entregues/perdidas numa única transação de backend — a fronteira
	// anti-duplicação da drenagem.
	SaveCargoWithDeliveries(ctx context.Context, save CargoSave, deliveredIDs, lostIDs []int64) error
	// SetAccountBlocked alterna account.is_blocked por nome — o lado de escrita do
	// comando in-game de ban/unban de GM (issue #122). Chamado fora do loop via World.Go.
	SetAccountBlocked(ctx context.Context, name string, blocked bool) error
	// RecordDuelResult persiste o resultado de um duelo 1v1 (issue #118): as vitórias de
	// winnerName e as derrotas de loserName são incrementadas em um cada. Chamado fora do
	// loop via World.GoDetached (não ligado à sessão de nenhum dos duelistas).
	RecordDuelResult(ctx context.Context, winnerName, loserName string) error

	// Ciclo de vida/estado de guilda (issue #114). Estas chamadas bloqueiam no dbServer e precisam
	// ser feitas via World.Go/GoDetached pelos handlers do loop.
	CreateGuild(ctx context.Context, accountID int64, slot int, characterName, guildName string, clan, citizen uint8, serverIndex int, cost int32) (GuildRecord, bool, error)
	SetGuildMember(ctx context.Context, accountID int64, slot int, characterName string, guildID uint16, guildLevel uint8) error
	LeaveGuild(ctx context.Context, accountID int64, slot int) error
	PromoteGuildMember(ctx context.Context, guildID uint16, leaderAccountID int64, leaderSlot int, accountID int64, slot int, cost int32) (uint8, bool, error)
	TransferGuildLeader(ctx context.Context, guildID uint16, oldAccountID int64, oldSlot int, newAccountID int64, newSlot int) error
	SetGuildRelation(ctx context.Context, guildID, targetGuildID uint16, kind GuildRelationKind) error
	ListGuilds(ctx context.Context) ([]GuildRecord, error)
	ListGuildRelations(ctx context.Context) ([]GuildRelation, error)
	LoadGuildZones(ctx context.Context) ([]GuildZone, error)
	SaveGuildZone(ctx context.Context, zone GuildZone) error
	LoadGuildTowerState(ctx context.Context) (GuildTowerState, error)
	SaveGuildTowerState(ctx context.Context, state GuildTowerState) error
	LoadCastleQuestState(ctx context.Context) (CastleQuestState, error)
	SaveCastleQuestState(ctx context.Context, state CastleQuestState) error
}

// errNoPersistence é devolvido pelo NopPersistence para operações que precisam de banco.
var errNoPersistence = errors.New("world: no persistence backend configured")

// NopPersistence é um backend sem efeito para rodar o tmServer sem dbServer
// (bring-up inicial). Operações de login/personagem falham; saves de shutdown são descartados.
type NopPersistence struct{}

// SaveOnShutdown não faz nada.
func (NopPersistence) SaveOnShutdown(context.Context, CharacterSave) error { return nil }

// QuoteKingdomCape devolve o preço de desenvolvimento equilibrado sem persistência.
func (NopPersistence) QuoteKingdomCape(context.Context) (KingdomCapeQuote, error) {
	return KingdomCapeQuote{Revision: 1, HekalotiaCost: 8, AkeloniaCost: 8}, nil
}

// PurchaseKingdomCape não consegue oferecer compra atômica sem backend.
func (NopPersistence) PurchaseKingdomCape(context.Context, int64, uint8, CharacterSave) (KingdomCapeQuote, bool, error) {
	return KingdomCapeQuote{}, false, errNoPersistence
}

// AccountLogin sempre reporta conta inexistente.
func (NopPersistence) AccountLogin(context.Context, string, string) (LoginOutcome, error) {
	return LoginOutcome{Result: LoginNoAccount}, nil
}

// ListCharacters devolve uma lista vazia.
func (NopPersistence) ListCharacters(context.Context, int64) ([]CharSummary, error) {
	return nil, nil
}

// CreateCharacter não é suportado sem backend.
func (NopPersistence) CreateCharacter(context.Context, int64, int, string, int) (bool, error) {
	return false, errNoPersistence
}

// CreateArchCharacter não é suportado sem backend.
func (NopPersistence) CreateArchCharacter(context.Context, int64, string, int, int, int, int) (int, bool, error) {
	return 0, false, errNoPersistence
}

// DeleteCharacter não é suportado sem backend.
func (NopPersistence) DeleteCharacter(context.Context, int64, int, string, string) (bool, error) {
	return false, errNoPersistence
}

// SetPin não é suportado sem backend.
func (NopPersistence) SetPin(context.Context, int64, string) (bool, error) {
	return false, errNoPersistence
}

// VerifyPin reporta conta inexistente sem backend.
func (NopPersistence) VerifyPin(context.Context, int64, string) (PinResult, error) {
	return PinNoAccount, errNoPersistence
}

// LoadCharacter não é suportado sem backend.
func (NopPersistence) LoadCharacter(context.Context, int64, int) (CharacterState, error) {
	return CharacterState{}, errNoPersistence
}

// LoadCargo devolve um cofre vazio: sem backend o cargo é só em memória
// (depósito/saque ainda funcionam na sessão, mas nada persiste).
func (NopPersistence) LoadCargo(context.Context, int64) (CargoState, error) {
	return CargoState{}, nil
}

// SaveCargo descarta o snapshot (não há backend para persistir).
func (NopPersistence) SaveCargo(context.Context, CargoSave) error { return nil }

// ListPendingDeliveries não devolve grants: sem backend não há caixa de entrada.
func (NopPersistence) ListPendingDeliveries(context.Context, int64) ([]Delivery, error) {
	return nil, nil
}

// SaveCargoWithDeliveries descarta o snapshot (não há backend para persistir).
func (NopPersistence) SaveCargoWithDeliveries(context.Context, CargoSave, []int64, []int64) error {
	return nil
}

// SetAccountBlocked não é suportado sem backend (o ban precisa do banco de contas).
func (NopPersistence) SetAccountBlocked(context.Context, string, bool) error {
	return errNoPersistence
}

// RecordDuelResult não faz nada.
func (NopPersistence) RecordDuelResult(context.Context, string, string) error {
	return nil
}

// CreateGuild não é suportado sem backend.
func (NopPersistence) CreateGuild(context.Context, int64, int, string, string, uint8, uint8, int, int32) (GuildRecord, bool, error) {
	return GuildRecord{}, false, errNoPersistence
}

// SetGuildMember não é suportado sem backend.
func (NopPersistence) SetGuildMember(context.Context, int64, int, string, uint16, uint8) error {
	return errNoPersistence
}

// LeaveGuild não é suportado sem backend.
func (NopPersistence) LeaveGuild(context.Context, int64, int) error { return errNoPersistence }

// PromoteGuildMember não é suportado sem backend.
func (NopPersistence) PromoteGuildMember(context.Context, uint16, int64, int, int64, int, int32) (uint8, bool, error) {
	return 0, false, errNoPersistence
}

// TransferGuildLeader não é suportado sem backend.
func (NopPersistence) TransferGuildLeader(context.Context, uint16, int64, int, int64, int) error {
	return errNoPersistence
}

// SetGuildRelation não é suportado sem backend.
func (NopPersistence) SetGuildRelation(context.Context, uint16, uint16, GuildRelationKind) error {
	return errNoPersistence
}

// ListGuilds não devolve guildas sem backend.
func (NopPersistence) ListGuilds(context.Context) ([]GuildRecord, error) { return nil, nil }

// ListGuildRelations não devolve relações sem backend.
func (NopPersistence) ListGuildRelations(context.Context) ([]GuildRelation, error) {
	return nil, nil
}

// LoadGuildZones não devolve zonas persistidas sem backend.
func (NopPersistence) LoadGuildZones(context.Context) ([]GuildZone, error) { return nil, nil }

// SaveGuildZone não é suportado sem backend.
func (NopPersistence) SaveGuildZone(context.Context, GuildZone) error { return errNoPersistence }

// LoadGuildTowerState devolve estado zero sem backend.
func (NopPersistence) LoadGuildTowerState(context.Context) (GuildTowerState, error) {
	return GuildTowerState{}, nil
}

// SaveGuildTowerState não é suportado sem backend.
func (NopPersistence) SaveGuildTowerState(context.Context, GuildTowerState) error {
	return errNoPersistence
}

// LoadCastleQuestState devolve estado zero sem backend.
func (NopPersistence) LoadCastleQuestState(context.Context) (CastleQuestState, error) {
	return CastleQuestState{}, nil
}

// SaveCastleQuestState não é suportado sem backend.
func (NopPersistence) SaveCastleQuestState(context.Context, CastleQuestState) error {
	return errNoPersistence
}
