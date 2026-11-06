// Package domain guarda o modelo relacional de destino das contas migradas. É a
// representação independente do compilador que o conversor produz a partir das
// structs de save cruas (savefmt) e que a camada de persistência grava no
// PostgreSQL. Arrays de tamanho fixo das structs C são normalizados em slices;
// slots de item vazios (sIndex==0) são descartados.
package domain

import "time"

// Account é uma conta migrada com seus personagens e o cargo compartilhado.
//
// Segredos existem SÓ como hash: PassHash, PinHash e BlockPassHash são argon2id
// (o texto plano original é descartado na importação). Name é o login canônico,
// em minúsculas.
type Account struct {
	Name          string
	PassHash      string
	PinHash       string
	BlockPassHash string
	RealName      string
	Email         string
	Telephone     string
	Address       string
	SSN1          int32
	SSN2          int32
	DonateBalance int32
	CargoCoin     int32
	IsBlocked     bool
	Year          int32 // controles legados de "uma vez por dia", mantidos crus
	YearDay       int32
	Characters    []Character
	Cargo         []Item // owner_kind = account_cargo
}

// Character é um dos até quatro personagens de uma conta.
type Character struct {
	Slot         int
	Name         string
	Class        uint8
	Clan         uint8
	GuildID      uint16
	GuildLevel   uint8
	Level        int32
	Exp          int64
	Coin         int32
	Str          int16
	Int          int16
	Dex          int16
	Con          int16
	ScoreBonus   uint16
	SpecialBonus uint16
	SkillBonus   uint16
	Special      [4]int16 // BaseScore.Special[4]: pontos de maestria alocados
	// Base é o BaseScore sem equipamento (atributos e máximos de HP/MP). Str..Con
	// e MaxHp/MaxMp acima são o CurrentScore; a base era reconstruída no login
	// subtraindo o bônus de equipamento daquele instante, o que derivava de vez
	// quando as regras de bônus mudavam entre um save e um login. HasBase é falso
	// para linha gravada antes da migration 0027.
	HasBase            bool
	BaseStr            int16
	BaseInt            int16
	BaseDex            int16
	BaseCon            int16
	BaseMaxHp          int32
	BaseMaxMp          int32
	MaxHp              int32
	MaxMp              int32
	Hp                 int32
	Mp                 int32
	Critical           uint8
	RegenHP            uint16
	RegenMP            uint16
	ResistFire         int8
	ResistIce          int8
	ResistThunder      int8
	ResistMagic        int8
	LearnedSkill       int32
	SecLearnedSkill    int32
	Magic              uint32
	SaveX              int16
	SaveY              int16
	LastCity           int16  // última cidade (0..3); o spawn no login é a área padrão dela
	Citizen            uint8  // MobExtra.Citizen
	ClassMaster        uint8  // MobExtra.ClassMaster
	CelLv40            uint8  // MobExtra.QuestInfo.Celestial.Lv40 (gate de nível 40 celestial)
	CelLv90            uint8  // MobExtra.QuestInfo.Celestial.Lv90 (gate de nível 90 celestial)
	CelCircle          uint8  // MobExtra.QuestInfo.Circle (quest de Cythera Arcana concluída)
	CelReset           uint8  // MobExtra.QuestInfo.Celestial.Reset (resets da Pedra da Furia 0..3)
	CelAdd120          uint8  // MobExtra.QuestInfo.Celestial.Add120 (adicional de 120 do diadema Sub-Celestial)
	CelAdd150          uint8  // MobExtra.QuestInfo.Celestial.Add150 (adicional de 150 do diadema Sub-Celestial)
	CelAdd180          uint8  // MobExtra.QuestInfo.Celestial.Add180 (adicional de 180 do diadema Sub-Celestial)
	CelAdd200          uint8  // MobExtra.QuestInfo.Celestial.Add200 (adicional de 200 do diadema, gravado 199)
	CelestialLevel     int16  // MobExtra.QuestInfo.Celestial.CelestialLevel (nível do Celestial estacionado)
	SubCelestialLevel  int16  // MobExtra.QuestInfo.Celestial.SubCelestialLevel (nível do Sub-Celestial estacionado)
	SaveCelestial      []byte // MobExtra.SaveCelestial[2], opaco para o dbServer (world.MarshalCelestialSheets)
	TerraMistica       uint8  // MobExtra.QuestInfo.Mortal.TerraMistica (AMU_MISTICO)
	NewbieQuest        uint8  // MobExtra.QuestInfo.Mortal.Newbie (passo do Campo de Treino 0..4)
	MolarGargula       uint8  // MobExtra.QuestInfo.Mortal.MolarGargula (Molar de Gargula usado)
	PilulaOrc          uint8  // MobExtra.QuestInfo.Mortal.PilulaOrc (Pilula do Poder Magico usada)
	ArchLv355          uint8  // MobExtra.QuestInfo.Arch.Level355
	ArchLv370          uint8  // MobExtra.QuestInfo.Arch.Level370
	MortalLevel        uint16 // MobExtra.QuestInfo.Arch.MortalLevel
	CelestialArchLevel uint8  // MobExtra.QuestInfo.Celestial.ArchLevel
	ArchCrystalStage   uint8  // QuestInfo.Arch.Cristal (0..4)
	Soul               uint8  // MobExtra.Soul
	Fame               int32  // MobExtra.Fame
	KefraTicket        int32  // Entradas restantes no Salão de Kefra, por personagem.
	NightmareEntries   int32  // Entradas restantes no Pesadelo Arcano.
	LastNightmareUse   int64  // Instante Unix do último uso do deed de Pesadelo.
	PKPoint            uint8  // Slot KILL_MARK: contador de caos/karma, 75 = neutro
	Guilty             uint8  // Slot KILL_MARK: contador de decaimento do "red nick" de PvP
	CurKill            uint8  // sequência atual de kills em PvP (MobName[13])
	TotKill            uint16 // kills de PvP na vida (MobName[14..15])
	SkillBar           [4]uint8
	ShortSkill         [16]uint8
	Equip              []Item // owner_kind = char_equip
	Carry              []Item // owner_kind = char_carry
	Affects            []Affect
}

// KingdomCapeQuote é o retrato versionado e persistido do preço em safira.
type KingdomCapeQuote struct {
	Revision      int64
	HekalotiaCost int32
	AkeloniaCost  int32
}

// RankingEntry é a projeção de ranking de personagem para a web. Rank é
// atribuído pelo chamador após a paginação; o store devolve na ordem do ranking.
type RankingEntry struct {
	Rank        int32
	Name        string
	Class       uint8
	Clan        uint8
	GuildID     uint16
	Level       int32
	Exp         int64
	ClassMaster uint8
}

// DuelRankingEntry é a projeção de placar de vitórias/derrotas PvP para a web
// (character_pvp_stats). Rank é atribuído pelo chamador após a paginação, como
// em RankingEntry.
type DuelRankingEntry struct {
	Rank    int32
	Name    string
	Class   uint8
	Clan    uint8
	GuildID uint16
	Wins    int32
	Losses  int32
}

// Guild é o registro durável do cadastro de guildas. ID é o ushort legado
// gravado em STRUCT_MOB.Guild e exibido pelo cliente 7662.
type Guild struct {
	ID      uint16
	Name    string
	Clan    uint8
	Fame    int32
	Citizen uint8
}

// GuildRelationKind identifica uma linha direcionada de relação entre guildas.
type GuildRelationKind uint8

// Tipos de relação entre guildas.
const (
	GuildRelationNone GuildRelationKind = iota
	GuildRelationAlly
	GuildRelationWar
)

// GuildRelation é uma relação direcionada de aliança/guerra entre guildas.
type GuildRelation struct {
	GuildID       uint16
	TargetGuildID uint16
	Kind          GuildRelationKind
}

// GuildZone é o subconjunto persistido de STRUCT_GUILDZONE usado por domínio de
// cidade, lances de desafio, imposto urbano e posse de castelo.
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

// CastleQuestState guarda o único estado ativo da quest de Castelo/Zakum.
type CastleQuestState struct {
	Level      int32
	TimeLeft   int32
	Clear      bool
	LeaderName string
}

// Item é uma entrada normalizada de inventário/equipamento/cargo. Slot preserva
// o índice do array (o significado é posicional); slot vazio não é representado.
type Item struct {
	Slot  int
	Index int16
	Eff1  uint8
	EffV1 uint8
	Eff2  uint8
	EffV2 uint8
	Eff3  uint8
	EffV3 uint8
	// ExpiresAt é o vencimento em segundos Unix para item temporizado (0 = permanente).
	ExpiresAt int64
}

// Affect é um buff/debuff persistido (affect[char][32]).
type Affect struct {
	Type  uint8
	Value uint8
	Level uint16
	Time  uint32
}

// NPCDefinition é um bloco de NPC/spawn editável pelo moderador. É configuração
// fria, dona no Postgres, materializada em entidade viva pelo loop de dono único
// do tmServer — nunca o contrário. Slug é o id estável; TemplateName aponta para
// o STRUCT_MOB de 816 bytes em tmserver/content/TMsrv/run/npc/.
type NPCDefinition struct {
	ID                            int64
	Slug                          string
	TemplateName                  string
	DisplayName                   string
	Enabled                       bool
	MapID                         int32
	PosX                          int32
	PosY                          int32
	RouteType                     int16
	Merchant                      int16
	Origin                        string
	GeneratorIndex                int32
	FollowerTemplate              string
	MinuteGenerate                int32
	MinGroup, MaxGroup, MaxNumMob int32
	Formation                     int16
	SegX, SegY, SegRange, SegWait [5]int32
	FightAction, DieAction        [4]string
	Shop                          []NPCShopItem // estoque do mercador; sobrepõe o Carry[] do template
}

// NPCShopItem é um slot de loja de um NPC mercador. Preço NÃO mora aqui: o
// moderador edita o preço global do catálogo (ItemPriceOverride).
type NPCShopItem struct {
	Slot      int16
	ItemIndex int32
	Quantity  int16 // quantidade empilhada; 1 é um item único
	Eff1      uint8
	EffV1     uint8
	Eff2      uint8
	EffV2     uint8
	Eff3      uint8
	EffV3     uint8
}

// ItemPriceOverride é o preço global por item definido pelo moderador. Ele
// sobrepõe o preço do catálogo de conteúdo para todo NPC que vende o item.
type ItemPriceOverride struct {
	ItemIndex int32
	Price     int64
}

// WorldEventConfig é o estado global de evento administrado pelo moderador. É
// configuração fria, dona no Postgres, materializada no loop do tmServer pelo
// polling do dbServer. CurrentIndex é progresso vivo do evento: o tmServer o
// avança após um drop indexado/global bem-sucedido e o persiste de volta, sem
// tratar essa escrita de progresso como mudança de config do moderador.
type WorldEventConfig struct {
	Enabled            bool
	ItemIndex          int32
	Rate               int32
	StartIndex         int32
	CurrentIndex       int32
	EndIndex           int32
	Indexed            bool
	NoticeEnabled      bool
	DoubleExpEnabled   bool
	NewbieEventEnabled bool
}

// MobTemplateStat é o override de atributos, editável pelo moderador, de um
// arquivo cru de template STRUCT_MOB — a ferramenta que substitui o EDITAPPMOB
// legado. Cobre todo campo que o EDITAPPMOB editava, exceto Carry[] (já em
// npc_shop_item) e a posição de spawn administrada pelo banco (já em
// NPCDefinition). BaseScore e CurrentScore não são modelados à parte: o tmServer
// espelha os mesmos valores nos dois, como o próprio `CurrentScore = BaseScore`
// do EDITAPPMOB ao salvar. A ausência de linha significa bytes crus intactos.
type MobTemplateStat struct {
	TemplateName string
	DisplayName  string // "" mantém o nome do próprio arquivo de template
	Clan         uint8
	Merchant     uint8 // Merchant do topo do STRUCT_MOB — distinto de NPCDefinition.Merchant
	Class        uint8
	Coin         int32
	Exp          int64
	SPX, SPY     int32
	Level        int32
	AC           int32
	Damage       int32
	ChaosRate    uint8
	AttackRun    uint8
	Direction    uint8
	Str          int16
	Int          int16
	Dex          int16
	Con          int16
	Special      [4]int16
	MaxHp        int32
	Hp           int32
	MaxMp        int32
	Mp           int32
	LearnedSkill int32
	ScoreBonus   uint16
	SkillBar     [4]uint8
	RegenHP      uint16
	RegenMP      uint16
	Resist       [4]int8
	Equip        []MobTemplateEquipItem
}

// MobTemplateEquipItem é o override de um slot de Equip[] de template (0..15),
// com a forma de NPCShopItem sem Quantity (slot de equipamento não empilha).
type MobTemplateEquipItem struct {
	Slot      int16
	ItemIndex int32
	Eff1      uint8
	EffV1     uint8
	Eff2      uint8
	EffV2     uint8
	Eff3      uint8
	EffV3     uint8
}

// MobCarryOverride é a tabela Carry[] de um template STRUCT_MOB editada pelo
// moderador. Tem a mesma forma slot/item/efeito de Equip, porque as duas são
// arrays de STRUCT_ITEM dentro do registro de 816 bytes; aqui o slot é a chave de
// raridade do drop, não uma posição de equipamento.

type MobCarryOverride struct {
	TemplateName string
	Slots        []MobTemplateEquipItem
}

// DonateShopItem é uma oferta da loja de doações administrada pelo moderador: um
// item (índice + até três pares efeito/valor) vendido por Price unidades do saldo
// de doações da conta. É config fria, dona no Postgres — o tmServer nunca a lê;
// só o web-api serve a vitrine e processa compras. ExpiresDays > 0 torna o item
// entregue temporizado.
type DonateShopItem struct {
	ID          int64
	ItemIndex   int32
	Eff1        uint8
	EffV1       uint8
	Eff2        uint8
	EffV2       uint8
	Eff3        uint8
	EffV3       uint8
	Price       int32
	Title       string
	Description string
	Enabled     bool
	ExpiresDays int32
}

// Delivery é uma concessão de item pendente que o tmServer drena da
// delivery_queue para o cargo da conta. ExpiresAt no Item é segundos Unix
// absolutos (0 = permanente).
type Delivery struct {
	ID   int64
	Item Item
}

// DailyRewardItem é uma oferta do catálogo de recompensa diária administrada pelo
// moderador: um item (índice + até três pares efeito/valor) resgatável uma vez por
// conta por dia de calendário UTC, de graça. Config fria, dona no Postgres — o
// tmServer nunca a lê; só o web-api serve a vitrine e processa resgates.
// ExpiresDays > 0 torna o item entregue temporizado.
type DailyRewardItem struct {
	ID          int64
	ItemIndex   int32
	Eff1        uint8
	EffV1       uint8
	Eff2        uint8
	EffV2       uint8
	Eff3        uint8
	EffV3       uint8
	Title       string
	Description string
	Enabled     bool
	ExpiresDays int32
}

// TopupOrder é uma recarga de doações agnóstica de meio de pagamento (web-api).
// O portal não tem banco, então o pedido e a idempotência do crédito vivem no
// Postgres. ExternalReference (o UUID do portal) é a âncora de idempotência;
// PaymentMethod/Status são os inteiros dos enums de wire (1=PIX/2=CREDIT_CARD;
// 1=PENDING/2=PAID). AmountCents é dinheiro em centavos inteiros — nunca float.
type TopupOrder struct {
	ID                int64
	ExternalReference string
	AccountID         int64
	Credits           int32
	AmountCents       int64
	PaymentMethod     int16
	Status            int16
}

// --- read models do painel de faturamento (web.v1.DonateRevenueAdminService) ---
//
// São só projeções de LEITURA. TopupOrder acima continua sendo a entrada do
// caminho de escrita de CreateTopupOrder: created_at/confirmed_at são postos pelo
// Postgres e provider não é escrito por ninguém, então carregá-los lá seria três
// campos que mentem.
//
// Duas unidades aparecem abaixo e NUNCA se misturam. *Cents é dinheiro real em
// centavos inteiros de BRL (nunca float). Credits/*Credits é a unidade da
// carteira de doações do jogo, cuja taxa de troca é propriedade de cada pedido.
//
// A receita é reconhecida em ConfirmedAt, não em CreatedAt: o pedido só é
// dinheiro quando o gateway liquidou e ConfirmTopupOrder o virou PAID.

// TopupOrderRow é um donate_topup_order com a conta do comprador e o perfil do
// pagador — a linha da tabela de moderação. PayerCPF guarda os 11 dígitos como
// estão; o serviço mascara antes de chegar ao wire.
type TopupOrderRow struct {
	ID                int64
	ExternalReference string
	AccountID         int64
	AccountName       string
	AccountEmail      string
	PayerName         string
	PayerCPF          string
	Credits           int32
	AmountCents       int64
	PaymentMethod     int16
	Provider          string // sempre "" hoje: CreateTopupOrder nunca grava
	Status            int16
	CreatedAt         time.Time
	ConfirmedAt       *time.Time // nil enquanto PENDING
}

// RevenueTotals é o cabeçalho de KPIs de dinheiro. Os Paid* são reconhecidos em
// confirmed_at; Created/Pending* usam created_at, porque medem volume de funil e
// não receita. PendingCents NÃO é receita e nunca pode ser somado a GrossCents.
type RevenueTotals struct {
	PaidOrders     int64
	GrossCents     int64
	CreditsSold    int64
	DistinctBuyers int64
	CreatedOrders  int64
	PendingOrders  int64
	PendingCents   int64
}

// RevenueByMethod é a receita reconhecida por meio de pagamento.
type RevenueByMethod struct {
	PaymentMethod int16
	PaidOrders    int64
	GrossCents    int64
}

// RevenuePoint é um balde da série temporal de receita. BucketStart é o instante
// em que o balde abre; baldes vazios existem com contadores zerados, então o
// chamador nunca faz conta de data para preencher lacuna.
type RevenuePoint struct {
	BucketStart    time.Time
	PaidOrders     int64
	GrossCents     int64
	CreditsSold    int64
	DistinctBuyers int64
}

// TopBuyer ranqueia uma conta por receita numa janela ao lado do agregado de
// toda a vida. Os campos Lifetime* ignoram a janela de propósito — é isso que os
// torna um LTV.
type TopBuyer struct {
	AccountID          int64
	AccountName        string
	AccountEmail       string
	WindowPaidOrders   int64
	WindowGrossCents   int64
	LifetimePaidOrders int64
	LifetimeGrossCents int64
	LifetimeCredits    int64
	FirstPaidAt        time.Time
	LastPaidAt         time.Time
	DonateBalance      int32 // carteira atual: valor pontual, não da janela
}

// DonateLedgerEntry é uma movimentação da carteira de doações lida de
// donate_shop_audit. Subject é de quem o saldo mudou; Actor é quem causou. Só
// diferem em 'credit_balance', onde o actor é o moderador que creditou e o
// subject está no JSON da auditoria. CreditsDelta é COM SINAL — negativo na
// compra, positivo no crédito —, então a coluna soma num líquido com sentido.
type DonateLedgerEntry struct {
	ID                 int64
	Action             string // "purchase" | "credit_balance"
	CreatedAt          time.Time
	SubjectAccountID   int64
	SubjectAccountName string
	ActorAccountID     int64
	ActorAccountName   string
	CreditsDelta       int64
	BalanceAfter       int64
	ShopItemID         int64  // 0 em credit_balance
	ShopItemTitle      string // "" quando a oferta foi apagada depois da compra
	Reason             string // só em credit_balance: a nota do moderador
}

// DonateLedgerTotals agrega as movimentações da carteira numa janela, em créditos.
type DonateLedgerTotals struct {
	ShopPurchases  int64
	CreditsSpent   int64
	ManualCredits  int64
	CreditsGranted int64
}

// AccountSummary é a projeção de identidade que o seletor de conta do painel usa
// para virar um login digitado em filtro por account_id. Não expõe credencial.
type AccountSummary struct {
	ID            int64
	Name          string
	Email         string
	DonateBalance int32
	Role          string
	IsBlocked     bool
}
