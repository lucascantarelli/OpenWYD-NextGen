// Package handler implementa o dispatch por mensagem do tmServer
// (handlers/*.md, game-rules.md). Todo handler roda dentro da goroutine do
// game-loop do mundo, então pode mutar estado do mundo direto; chamadas bloqueantes ao dbServer
// são emitidas fora do loop via World.Go e seus resultados voltam ao loop
// (domain-model.md §5).
//
// Lote 1 (login → seleção de personagem): AccountLogin, CreateCharacter,
// DeleteCharacter, CharacterLogin, CharacterLogout, Restart. São em boa parte
// repasses ao dbServer vigiados pela máquina de estados de sessão (CUser.Mode,
// domain-model.md §3.1).
package handler

import (
	"fmt"
	"log/slog"
	"time"

	"openwyd-nextgen/tmserver/internal/combine"
	"openwyd-nextgen/tmserver/internal/content"
	"openwyd-nextgen/tmserver/internal/level"
	"openwyd-nextgen/tmserver/internal/npccfg"
	"openwyd-nextgen/tmserver/internal/protocol"
	"openwyd-nextgen/tmserver/internal/refine"
	"openwyd-nextgen/tmserver/internal/rng"
	"openwyd-nextgen/tmserver/internal/world"
	"openwyd-nextgen/tmserver/internal/worldcfg"
	"openwyd-nextgen/tmserver/internal/worldevents"
)

// Config ajusta o dispatcher. Valores zero recebem padrões sensatos.
type Config struct {
	ClientVersion int32 // versão de cliente exigida (padrão AppVersion 7640)
	MaxNightmare  int   // Admissões contadas por mapa/ciclo; padrão 3.
	MaxFailLogin  int   // limite de bloqueio por senha errada (padrão 3)
	ServerIndex   int   // bits altos de id de guilda legado (server_index * 4096); também o canal a que um cidadão pertence (Citizen = ServerIndex+1)

	// Canal novato ("verde") (canais.md §3), fixado por configuração como em
	// produção: NoviceChannel é o número de canal base 1 que faz o
	// papel de novato (roda NewbieEventServer e recusa dano PvP). 0 desliga o
	// canal novato e deixa o evento com a flag do portal em todo canal.
	NoviceChannel int

	// CrossoverKey assina o token de troca de canal entregue ao cliente por /srv
	// (canais.md §8.1.2). Precisa casar com o valor do dbServer; vazio desativa a
	// troca de canal in-game.
	CrossoverKey string

	// RefinePolicy é a tabela de teto do refino com pó (RefineRate). Valor zero
	// cai no legado 6/9/10/11.
	RefinePolicy content.RefinePolicy
	Log          *slog.Logger     // slog.Default() padrão
	Now          func() time.Time // relógio de parede para operações de guilda condicionadas por calendário

	// KefraEntriesPerScroll é quantas entradas do Salão de Kefra um pergaminho
	// selado concede no Sobrevivente; padrão 100 (o valor legado).
	KefraEntriesPerScroll int

	// AffectDuration ajusta quanto tempo dura um buff/debuff CONJURADO (issue #229). O
	// valor zero é fiel ao legado; os padrões de produção são definidos em
	// cmd/tmserver/main.go. Affects de item/cash/Divino nunca passam por ele.
	AffectDuration world.AffectDuration

	// CombineFamilies sobrescreve a lógica de receita/taxa de combine por Type. Quando nil,
	// placeholders UNVERIFIED são instalados (toda receita informa "sem correspondência").
	CombineFamilies map[protocol.Type]CombineFamily

	// OdinCatalog é a metadata de item que combineItemOdin precisa (o lookup Extra do
	// MatchOdin, o gate de nPos do Plus12). NewCombineCatalog o monta do mesmo modo que
	// DefaultCombineFamilies. Quando zerado, toda receita Odin ainda casa por
	// sIndex, mas a Composição de Sets/armas não consegue resolver um item de resultado.
	OdinCatalog    combine.Catalog
	CombineCatalog combine.Catalog
	CompRate       *content.CompRate
	QuestRates     *content.QuestRates

	// BaseMobs são os templates STRUCT_MOB por classe (índice de classe → 816 bytes crus,
	// content.LoadBaseMobs). Usados para renderizar um personagem ao entrar no mundo com
	// o equipamento/stats iniciais da classe. Quando nil, o snapshot é montado do
	// estado relacional guardado (sem equipamento).
	BaseMobs map[int][]byte

	// SummonMobs são os templates STRUCT_MOB de evocação do BM indexados por id de invocação
	// (content.LoadBaseSummons; mSummon.Mob do GenerateSummon). Slots podem ser nil
	// (templates de montaria opcionais); quando a fatia é nil as conjurações de evocação falham.
	SummonMobs [][]byte

	// VineMob é o template STRUCT_MOB usado pela skill Sephira 98 (Muro de
	// Espinhos). O legado chama CreateMob("Vinha", x, y, "Boss", 3); a árvore de conteúdo
	// Linux entregue pode fornecer isto como npc/Vine. Quando nil a conjuração
	// consome mana mas não consegue criar a parede.
	VineMob []byte

	// ItemPrices mapeia índice de item → Price base (g_pItemList[].Price) para compra/venda em NPC.
	ItemPrices map[int]int32

	// ItemEffects mapeia índice de item → seus efeitos base estáticos (STRUCT_ITEMLIST.stEffect:
	// dano de arma, AC de armadura, bônus de atributo). Usado para dobrar equipamento no
	// CurrentScore (content.ItemList.BaseEffects). Quando nil, equipamento não soma nada.
	ItemEffects map[int][]content.BaseEffect

	// ItemReqs mapeia índice de item → seu requisito de equipar (nível/atributos,
	// content.ItemList.Requirements). Quando nil, nenhum equipamento é condicionado.
	ItemReqs map[int]content.ItemReq

	// ItemVolatiles mapeia índice de item → valor EF_VOLATILE, classificando consumíveis no
	// uso (64-66 Divino, 58 Vigor; content.ItemList.Volatiles). 0/ausente = equipável.
	ItemVolatiles map[int]int

	// ItemPos mapeia índice de item → nPos (classe de slot de equipamento) para os bônus de
	// limiar do refino (+9). ItemUnique mapeia índice → nUnique (condiciona EF_DAMAGEADD a joias).
	// ItemGrades mapeia índice → Grade (peças grade-7 concedem +2 ExpBonus). ItemExtra mapeia
	// índice → Extra (o índice do item de resultado para o combine Anct e o combine
	// de upgrade lendário Adamantita, issue #135).
	ItemPos    map[int]int
	ItemUnique map[int]int
	ItemGrades map[int]int
	ItemExtra  map[int]int

	// SancRate é a tabela de sucesso do refino com pó (Common/Settings/SancRate.txt sobre
	// os padrões de Basedef.cpp; *content.SancRate a satisfaz). Quando nil os
	// padrões compilados são usados, então o refino ainda funciona sem uma árvore de conteúdo
	// montada.
	SancRate refine.RateTable

	// ExpEvents alterna modificadores globais de EXP (MobKilled.cpp:537-549, padrões de Server.cpp).
	ExpEvents level.ExpEvents

	// TrainingFieldExp multiplica a recompensa solo livre de evento de mortes no
	// campo de treino para mortais abaixo do nível guardado 35 (issue #383,
	// level.TrainingFieldExpReward). Zero desativa: o campo então paga o
	// SoloExpReward simples como qualquer outro mapa.
	TrainingFieldExp int64

	// Spells é o catálogo SkillData.csv (g_pSpell). Quando nil, conjurar skill é
	// recusado e aprender skill é recusado (sem ele não há custos conhecíveis).
	Spells *content.SkillData

	// Heights é a grade de caminhabilidade sobre a qual a IA de criaturas traça rotas: HeightMap.dat com
	// AttributeMap.dat já embutido (route.Bake, o BASE_ApplyAttribute de boot).
	// Somente leitura após o boot, então compartilhá-la com o loop é
	// seguro. Quando nil (mapas não montados, ou testes) criaturas caem no passo
	// Chebyshev cego de um tile.
	Heights *content.Grid

	// Attrs é o AttributeMap.dat cru (um byte por tiles 4×4, não embutido). O
	// handler de movimento lê seu bit 0x80 para manter personagens de nível alto fora do
	// campo de treino (newbieZoneGuard). Somente leitura após o boot; nil desativa a
	// guarda (mapas não montados, ou testes).
	Attrs *content.Grid

	// NpcConfig é a fonte de configuração de NPC editada pelo moderador (npc-editing-plan.md).
	// Quando definida, o dispatcher sobrepõe NPCs de mercador definidos no banco no boot e consulta
	// mudanças a cada tick (hot-reload). Quando nil, NPCs vêm só do NPCGener.txt.
	NpcConfig npccfg.Source

	// WorldEvents é a fonte de configuração de eventos globais gerenciada pelo portal (issue #116).
	// Quando definida, o dispatcher aplica configurações de evento de EXP/drop no boot e consulta
	// edições do moderador. Quando nil, ExpEvents fica com as flags de boot.
	WorldEvents worldcfg.Source

	// CastleQuests é a tabela opcional CastleQuest.txt. Vazia mantém o
	// port best-effort de Castle/Zakum desativado.
	CastleQuests []content.CastleQuest

	// EventRNGSeed semeia o fluxo dedicado de eventos de mundo (veja Dispatcher.eventRNG).
	// Zero mantém o worldEventRNGSeed fixo, que é o que os testes querem; main.go
	// passa o relógio de parede para a sequência de clima não ser repetida de forma idêntica
	// após cada restart.
	EventRNGSeed uint32
}

type handlerFunc func(w *world.World, s *world.Session, h protocol.Header, payload []byte)

// Dispatcher roteia frames decodificados do cliente para handlers. É criado uma vez e
// instalado como o Handler do mundo. Seu estado mutável (os contadores de
// senha errada por conta) só é tocado pela goroutine do loop, então não precisa
// de locks.
type Dispatcher struct {
	waterRooms      [3][9]waterRoom // do loop; aliases centrais compartilham o slot 8
	cfg             Config
	log             *slog.Logger
	routes          map[protocol.Type]handlerFunc
	fails           map[string]int // contagem de senha errada por conta (CheckFailAccount)
	combineFamilies map[protocol.Type]CombineFamily
	odinCatalog     combine.Catalog
	combineCatalog  combine.Catalog
	compRate        *content.CompRate
	questRates      *content.QuestRates
	baseMobs        map[int][]byte               // templates STRUCT_MOB por classe
	summonMobs      [][]byte                     // templates de evocação do BM (id de invocação → STRUCT_MOB)
	vineMob         []byte                       // template do Muro de Espinhos da Sephira
	itemPrices      map[int]int32                // índice de item → preço base (loja de NPC)
	itemEffects     map[int][]content.BaseEffect // índice de item → efeitos base estáticos (score de equipamento)
	itemReqs        map[int]content.ItemReq      // índice de item → requisito de equipar (nível/atributos)
	itemVolatiles   map[int]int                  // índice de item → EF_VOLATILE (classe de consumível)
	itemPos         map[int]int                  // índice de item → nPos (limiar de refino)
	itemUnique      map[int]int                  // índice de item → nUnique (gate de EF_DAMAGEADD)
	itemGrades      map[int]int                  // índice de item → Grade (ExpBonus)
	itemExtra       map[int]int                  // índice de item → Extra (resultado de combine Anct/Adamantita)
	sancRate        refine.RateTable             // tabela de sucesso do refino com pó (g_pSancRate)
	refinePolicy    content.RefinePolicy         // tetos do refino com pó (RefineRate + overlay)
	expEvents       level.ExpEvents              // flags globais de evento de EXP
	fieldExp        int64                        // multiplicador de EXP do campo de treino (0 = desligado)
	kefra           kefraRuntime                 // rastreamento de I/O de evento semanal do loop
	spells          *content.SkillData           // catálogo de skills (g_pSpell)
	heights         *content.Grid                // grade de caminhabilidade embutida (pathfinding de criaturas)
	attrs           *content.Grid                // AttributeMap cru (bit do campo de treino)
	now             func() time.Time             // relógio de parede para operações de guilda condicionadas por calendário
	tickCount       int                          // contador de tick só do loop (fase da varredura de affect)
	affectDur       world.AffectDuration         // ajuste de duração de affect conjurado (issue #229)
	serverIndex     int                          // bits altos de id de guilda legado
	crossoverKey    string                       // chave HMAC do token de troca de canal /srv
	guildZones      [5]world.GuildZone           // cache de zona de cidade/guilda do loop
	taxChangedAt    [5]time.Time                 // dia em que o guildtax de cada zona mudou por último (uma mudança/dia, lote2-chat.md)
	guildWars       map[uint16]uint16            // guilda dirigida -> alvo de guerra atual
	guildAllies     map[uint16]uint16            // guilda dirigida -> alvo aliado atual
	towerState      world.GuildTowerState        // cache de posse de GTorre do loop
	castleState     world.CastleQuestState       // cache de estado Castle/Zakum do loop
	castleQuests    []content.CastleQuest
	castleParty     [world.MaxParty + 1]int
	guildStateLoad  bool // snapshot de boot da persistência de guilda foi aplicado
	guildStateBusy  bool // uma carga de estado de guilda em andamento

	// Overlay de config de NPC (npc-editing-plan.md). Tudo só do loop. baseItemPrices é o
	// catálogo de conteúdo imutável; itemPrices é o mapa efetivo (base + overrides
	// globais), reconstruído no reload. managedNPCs mapeia slug de definição → seu id de
	// entidade vivo. npcVersion é a última versão de config aplicada; npcPolling guarda um
	// único poll de versão em andamento; npcPollTick faseia a cadência do poll.
	npcSource      npccfg.Source
	baseItemPrices map[int]int32
	managedNPCs    map[string]managedNPC
	npcVersion     int64
	npcPolling     bool
	npcPollTick    int

	// Eventos de mundo gerenciados pelo portal (issue #116). worldEventSource é a ponte de
	// leitura/progresso do dbServer; o estado real do evento vive em world.World.
	worldEventSource   worldcfg.Source
	worldEventVersion  int64
	worldEventPolling  bool
	worldEventPollTick int

	// Estado do canal novato ("verde") (newbie_channel.go). newbieCfg é o valor
	// NewbieEventServer cru das flags/portal; noviceChannel é o canal novato configurado
	// base 1 (0 = nenhum). Tudo só do loop.
	noviceChannel    int
	newbieCfg        bool
	noviceRoleLogged bool

	// playersX/Y são snapshots de rascunho por tick das posições de jogadores em jogo
	// (gate de dormência da IA de criaturas, mobai.go). Só do loop, reutilizados para evitar alocação.
	playersX, playersY []int16

	// duel é o duelo 1v1 ativo do servidor (issue #118, duel.go). Os legados
	// RankingProgress/Ranking1/Ranking2/RankingTime são globais de processo, não
	// por par — só um duelo ocupa a arena por vez.
	duel duelArena

	// events é o estado de evento de mundo dirigido por timer (issue #116): clima, a
	// janela de guerra da torre, o atraso de limpeza de área do reino. Como todo
	// equivalente legado em Server.cpp esses são globais de processo, então uma instância
	// por servidor. Só do loop.
	events worldEventState

	// eventRNG é um fluxo MSVC DEDICADO para sorteios de evento de mundo. O legado os
	// tira do único rand() global, mas esse fluxo é o que nossos goldens de
	// drop/refino/crítico fixam a ordem de chamadas (refine_test.go,
	// loot_test.go, combat_test.go): um sorteio de clima por minuto nele
	// invalidaria todos eles. Desvio de paridade documentado, issue #116.
	//
	// Tipado como a interface (não *rng.MSVC) para testes poderem substituir por um gerador
	// roteirizado ou contador; produção sempre guarda um MSVC.
	eventRNG worldevents.Rand
}

// worldEventRNGSeed é a semente de fallback do eventRNG, usada quando Config deixa
// EventRNGSeed zerado (testes). NÃO é um valor legado — o original não tem
// srand() nenhum (doc do pacote rng) — existe só para manter os sorteios de evento de mundo
// fora do fluxo de paridade do mundo.
//
// Produção o sobrescreve com o relógio de parede: os sorteios de clima legados compartilham
// o rand() global, cujo estado cada drop e crítico de jogador avança, então
// eles nunca se repetem de fato entre restarts. Nosso fluxo dedicado não tem esse
// movimento, e uma semente fixa repetiria a mesma sequência de clima nos mesmos
// ticks em todo boot.
const worldEventRNGSeed uint32 = 116

// worldEventState é o estado de runtime do loop dos eventos de mundo
// dirigidos por timer. Cada campo nomeia o global legado que ele porta.
type worldEventState struct {
	weather      int32 // CurrentWeather, 0/1/2 (Server.cpp:688)
	forceWeather int32 // ForceWeather; -1 = sorteios automáticos (Server.cpp:637)
	kingdom1     uint8 // Kingdom1Clear, estado de limpeza atrasada da sala do trono
	kingdom2     uint8 // Kingdom2Clear, estado de limpeza atrasada da sala do trono
	tower        worldevents.Tower
	towerOwner   uint16
	castle       worldevents.Castle
}

// New constrói um Dispatcher com as rotas do lote 1 registradas.
func New(cfg Config) *Dispatcher {
	if cfg.ClientVersion == 0 {
		cfg.ClientVersion = protocol.AppVersion
	}
	if cfg.MaxFailLogin <= 0 {
		cfg.MaxFailLogin = 3
	}
	if cfg.KefraEntriesPerScroll <= 0 {
		cfg.KefraEntriesPerScroll = defaultKefraEntriesPerScroll
	}
	if cfg.Log == nil {
		cfg.Log = slog.Default()
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.EventRNGSeed == 0 {
		cfg.EventRNGSeed = worldEventRNGSeed
	}
	d := &Dispatcher{
		cfg:              cfg,
		log:              cfg.Log,
		routes:           make(map[protocol.Type]handlerFunc),
		fails:            make(map[string]int),
		combineFamilies:  cfg.CombineFamilies,
		odinCatalog:      cfg.OdinCatalog,
		combineCatalog:   cfg.CombineCatalog,
		compRate:         cfg.CompRate,
		questRates:       cfg.QuestRates,
		baseMobs:         cfg.BaseMobs,
		summonMobs:       cfg.SummonMobs,
		vineMob:          cfg.VineMob,
		itemPrices:       cfg.ItemPrices,
		itemEffects:      cfg.ItemEffects,
		itemReqs:         cfg.ItemReqs,
		itemVolatiles:    cfg.ItemVolatiles,
		itemPos:          cfg.ItemPos,
		itemUnique:       cfg.ItemUnique,
		itemGrades:       cfg.ItemGrades,
		itemExtra:        cfg.ItemExtra,
		sancRate:         cfg.SancRate,
		refinePolicy:     cfg.RefinePolicy,
		expEvents:        cfg.ExpEvents,
		fieldExp:         cfg.TrainingFieldExp,
		spells:           cfg.Spells,
		heights:          cfg.Heights,
		attrs:            cfg.Attrs,
		now:              cfg.Now,
		affectDur:        cfg.AffectDuration,
		serverIndex:      cfg.ServerIndex,
		crossoverKey:     cfg.CrossoverKey,
		noviceChannel:    cfg.NoviceChannel,
		guildWars:        make(map[uint16]uint16),
		guildAllies:      make(map[uint16]uint16),
		npcSource:        cfg.NpcConfig,
		managedNPCs:      make(map[string]managedNPC),
		worldEventSource: cfg.WorldEvents,
		castleQuests:     cfg.CastleQuests,
		eventRNG:         rng.NewSeeded(cfg.EventRNGSeed),
		events:           worldEventState{forceWeather: weatherAuto},
	}
	d.events.tower = worldevents.NewTower(20)
	for i := range d.guildZones {
		d.guildZones[i].Zone = i
	}
	// O mapa de preço efetivo começa como o catálogo de conteúdo e depois é sobreposto
	// com overrides do moderador; manter a base imutável para um override poder ser
	// limpo. Quando não há catálogo os dois ficam nil (compra/venda simplesmente não acha preço).
	d.baseItemPrices = cfg.ItemPrices
	d.itemPrices = cloneInt32Map(cfg.ItemPrices)
	if d.sancRate == nil {
		// Sem árvore de conteúdo: cair no g_pSancRate compilado, como o servidor
		// original antes de ler SancRate.txt por cima dele.
		d.sancRate = content.DefaultSancRate()
	}
	if d.questRates == nil {
		d.questRates = content.DefaultQuestRates()
	}
	if d.combineFamilies == nil {
		d.combineFamilies = make(map[protocol.Type]CombineFamily)
	}
	for _, ty := range combineItemTypes {
		if _, ok := d.combineFamilies[ty]; !ok {
			d.combineFamilies[ty] = defaultCombineFamily(fmt.Sprintf("combine-%#x", uint16(ty)))
		}
	}
	d.routes[protocol.MsgAccountLogin] = d.accountLogin
	d.routes[protocol.MsgCreateCharacter] = d.createCharacter
	d.routes[protocol.MsgDeleteCharacter] = d.deleteCharacter
	d.routes[protocol.MsgCharacterLogin] = d.characterLogin
	d.routes[protocol.MsgCharacterLogout] = d.characterLogout
	d.routes[protocol.MsgRestart] = d.restart
	// Lote 2 — movimento e visão.
	d.routes[protocol.MsgAction] = d.action
	d.routes[protocol.MsgAction2] = d.action
	d.routes[protocol.MsgAction3] = d.action
	d.routes[protocol.MsgMotion] = d.motion
	d.routes[protocol.MsgChangeCity] = d.changeCity
	d.routes[protocol.MsgReqTeleport] = d.reqTeleport
	d.routes[protocol.MsgNoViewMob] = d.noViewMob
	d.routes[protocol.MsgPKMode] = d.pkMode
	// Lote 3 — combate.
	d.routes[protocol.MsgAttack] = d.attack
	d.routes[protocol.MsgAttackOne] = d.attack
	d.routes[protocol.MsgAttackTwo] = d.attack
	// Lote 4 — itens.
	d.routes[protocol.MsgDropItem] = d.dropItem
	d.routes[protocol.MsgGetItem] = d.getItem
	d.routes[protocol.MsgUseItem] = d.useItem
	d.routes[protocol.MsgDeleteItem] = d.deleteItem
	d.routes[protocol.MsgSplitItem] = d.splitItem
	d.routes[protocol.MsgUpdateItem] = d.updateItem
	// Loja de NPC.
	d.routes[protocol.MsgREQShopList] = d.reqShopList
	d.routes[protocol.MsgBuy] = d.buy
	d.routes[protocol.MsgSell] = d.sell
	// Carga (armazém da conta) — depósito/saque de ouro; movimentos de item vão via useItem.
	d.routes[protocol.MsgDeposit] = d.deposit
	d.routes[protocol.MsgWithdraw] = d.withdraw
	// Lote 5 — troca P2P.
	d.routes[protocol.MsgTradingItem] = d.tradingItem
	d.routes[protocol.MsgTrade] = d.trade
	d.routes[protocol.MsgQuitTrade] = d.quitTrade

	// Loja pessoal / trade automático (issue #115). Fechar uma loja é RemoveTrade
	// (movimento / QuitTrade / operações de item todas passam por removeTrade), então não há
	// rota de fechamento dedicada — _MSG_Deprivate é operação de guilda, deixada sem rota.
	d.routes[protocol.MsgSendAutoTrade] = d.sendAutoTrade
	d.routes[protocol.MsgReqTradeList] = d.reqTradeList
	d.routes[protocol.MsgReqBuy] = d.reqBuy
	// Lote 6 — combine/refino (um motor, todas as variantes baseadas em Item[]).
	for _, ty := range combineItemTypes {
		d.routes[ty] = d.combineItem
	}
	d.routes[protocol.MsgCombineItemExtracao] = d.combineExtracao
	d.routes[protocol.MsgCombineItemOdin] = d.combineItemOdin
	d.routes[protocol.MsgCombineItemAilyn] = d.combineItemAilyn
	d.routes[protocol.MsgCombineItemTiny] = d.combineItemTiny
	d.routes[protocol.MsgCombineItemShany] = d.combineItemShany
	d.routes[protocol.MsgCombineItemAgatha] = d.combineItemAgatha
	d.routes[protocol.MsgCombineItemLindy] = d.combineItemLindy
	d.routes[protocol.MsgCombineItemAlquimia] = d.combineItemAlquimia
	d.routes[protocol.MsgCombineItemEhre] = d.combineItemEhre
	// Lote 7 — grupo e guilda.
	d.routes[protocol.MsgSendReqParty] = d.sendReqParty
	d.routes[protocol.MsgAcceptParty] = d.acceptParty
	d.routes[protocol.MsgRemoveParty] = d.removeParty
	d.routes[protocol.MsgInviteGuild] = d.inviteGuild
	d.routes[protocol.MsgGuildAlly] = d.guildAlly
	d.routes[protocol.MsgWar] = d.war
	d.routes[protocol.MsgChallange] = d.challange
	d.routes[protocol.MsgChallangeConfirm] = d.challangeConfirm
	// Lote 8 — chat, bônus, quest/cash (stubs).
	d.routes[protocol.MsgMessageChat] = d.messageChat
	d.routes[protocol.MsgMessageWhisper] = d.messageWhisper
	d.routes[protocol.MsgApplyBonus] = d.applyBonus
	d.routes[protocol.MsgSetShortSkill] = d.setShortSkill
	d.routes[protocol.MsgAccountSecure] = d.accountSecure
	d.routes[protocol.MsgQuest] = d.quest
	d.routes[protocol.MsgMasterGriff] = d.masterGriff
	d.routes[protocol.MsgReqRanking] = d.reqRanking
	d.routes[protocol.MsgCapsuleInfo] = d.capsuleInfo
	d.routes[protocol.MsgPutoutSeal] = d.putoutSeal
	return d
}

// Handle é o world.Handler. Roda na goroutine do loop.
func (d *Dispatcher) Handle(w *world.World, s *world.Session, h protocol.Header, payload []byte) {
	d.ensureGuildStateLoaded(w)
	fn, ok := d.routes[h.Type]
	// DIAGNÓSTICO: logar todo Type recebido (hex) para pacotes do cliente poderem ser mapeados.
	d.log.Info("recv packet", "conn", s.Conn, "type", formatType(h.Type), "len", len(payload), "routed", ok)
	if !ok {
		return
	}
	fn(w, s, h, payload)
}

// refineLimits devolve os tetos efetivos do pó. Um dispatcher construído sem
// policy (testes, ferramentas pequenas) mantém o legado 6/9/10/11 em vez de zerar todo
// teto, o que recusaria todo refino.
func (d *Dispatcher) refineLimits() content.RefinePolicy {
	if d.refinePolicy.HardCap <= 0 {
		return content.DefaultRefinePolicy()
	}
	return d.refinePolicy
}

func formatType(t protocol.Type) string {
	const hexdigits = "0123456789abcdef"
	v := uint16(t)
	return "0x" + string([]byte{hexdigits[v>>12&0xf], hexdigits[v>>8&0xf], hexdigits[v>>4&0xf], hexdigits[v&0xf]})
}
