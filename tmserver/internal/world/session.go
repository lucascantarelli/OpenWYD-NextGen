package world

import (
	"net"
	"sync"

	"openwyd-nextgen/tmserver/internal/protocol"
)

// outFrame é uma mensagem S→C lógica enfileirada para a goroutine de escrita da sessão,
// que a codifica (CPSock) logo antes de escrever.
type outFrame struct {
	header  protocol.Header
	payload []byte
}

// wireSize é o tamanho do frame no wire (cabeçalho CPSock + corpo), a unidade em que
// a fila de saída é limitada.
func (of outFrame) wireSize() int { return protocol.HeaderSize + len(of.payload) }

// DefaultOutBufferBytes limita os bytes S→C não enviados de uma sessão como o
// buffer de envio por socket legado (CPSock.h:36 SEND_BUFFER_SIZE = 128 KB; AddMessage
// CPSock.cpp:513-520 só falha ao passar dele). Um delta de visão de algumas centenas de
// CreateMob (~45 KB entrando no Coliseu, issue #419) cabe; um cliente que
// para de ler não cabe.
const DefaultOutBufferBytes = 128 * 1024

// outQueue é a fila S→C de uma sessão: o loop empurra, a goroutine de escrita retira.
// Ela é limitada por bytes não enviados (não por contagem de frames), então uma rajada
// grande, porém legítima, é bufferizada e só um cliente que não está drenando cai.
type outQueue struct {
	mu     sync.Mutex
	frames []outFrame
	bytes  int
	ready  chan struct{} // cap 1: acorda o escritor após um push
}

func newOutQueue() *outQueue { return &outQueue{ready: make(chan struct{}, 1)} }

// push acrescenta of a não ser que isso leve a fila além de limit bytes; nunca
// bloqueia. Devolve a profundidade da fila (frames, bytes) após o push.
func (q *outQueue) push(of outFrame, limit int) (frames, bytes int, ok bool) {
	size := of.wireSize()
	q.mu.Lock()
	if q.bytes+size > limit {
		frames, bytes = len(q.frames), q.bytes
		q.mu.Unlock()
		return frames, bytes, false
	}
	q.frames = append(q.frames, of)
	q.bytes += size
	frames, bytes = len(q.frames), q.bytes
	q.mu.Unlock()
	select {
	case q.ready <- struct{}{}:
	default:
	}
	return frames, bytes, true
}

// pop remove o frame mais antigo, se houver.
func (q *outQueue) pop() (outFrame, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.frames) == 0 {
		return outFrame{}, false
	}
	of := q.frames[0]
	q.frames[0] = outFrame{}
	q.frames = q.frames[1:]
	if len(q.frames) == 0 {
		q.frames = q.frames[:0:0] // libera o array de base já drenado
	}
	q.bytes -= of.wireSize()
	return of, true
}

// AccessLevel é o nível de privilégio de GM/moderação de uma sessão, derivado do
// campo account.role no login (issue #122). Ele substitui o frágil backdoor legado
// de "personagem com Level >= 1000" por uma autoridade explícita no servidor.
// Ordenado para que um gate compare com >= (nível maior passa gates de nível menor).
type AccessLevel uint8

// Níveis de acesso. Player é o valor zero: papel desconhecido/ausente nunca é GM.
const (
	AccessPlayer    AccessLevel = iota // sem privilégio de GM
	AccessModerator                    // moderação in-game (os comandos da primeira leva)
	AccessAdmin                        // reservado para operações destrutivas/globais futuras
)

// ParseAccess mapeia a string account.role ('player'/'moderator'/'admin') para seu
// AccessLevel. Qualquer valor não reconhecido é AccessPlayer (falha fechada).
func ParseAccess(role string) AccessLevel {
	switch role {
	case "admin":
		return AccessAdmin
	case "moderator":
		return AccessModerator
	default:
		return AccessPlayer
	}
}

// String renderiza o nível para os logs de auditoria.
func (a AccessLevel) String() string {
	switch a {
	case AccessAdmin:
		return "admin"
	case AccessModerator:
		return "moderator"
	default:
		return "player"
	}
}

// Session é o estado de conexão/sessão de um jogador (subconjunto de CUser,
// domain-model.md §2.1). Pertence à goroutine do loop; a fiação conn/out/closeCh
// é compartilhada apenas com as goroutines de leitura e escrita desta sessão.
type Session struct {
	Conn              int    // índice em pUser/pMob; também o HEADER.ID no wire
	TraceID           string // correlação ponta a ponta (OBS-02): gerado no aceite, vai no metadata gRPC e no log
	AccountName       string
	AccountID         int64
	AccessLevel       AccessLevel // nível de account.role; controla os comandos de GM in-game (issue #122)
	Slot              int
	Mode              Mode
	IP                string
	CrackError        int             // contagem de violações anti-cheat (CUser.NumError)
	Whisper           bool            // true bloqueia sussurros recebidos
	LastWhisperFrom   string          // quem sussurrou por último para este jogador; "/r" responde para lá (LastChat legado)
	PartyChat         bool            // true bloqueia o canal de party (partychat)
	GuildChat         bool            // true bloqueia o canal de guilda (guildchat)
	GuildDisable      bool            // esconde a tag de guilda (guildon/guildoff)
	TradeMode         int             // diferente de zero durante o auto-trade (bloqueia ataques)
	Trade             TradeState      // estado da troca direta P2P (lote2-trade-autotrade.md)
	AutoTrade         *AutoTradeState // não nulo enquanto uma loja pessoal está aberta (issue #115); TradeMode==1
	LastAttackTick    uint32          // ClientTick do último ataque aceito (gate de cadência)
	PotionTick        uint32          // CUser.PotionTime: relógio do servidor da última poção aceita
	LastAttack        int             // SkillIndex do último ataque
	LastIllusionTick  uint32          // ClientTick do último movimento de Ilusão da Huntress
	ReqHp             int32           // CUser.ReqHp: alvo de HP do servidor para regen/poções
	ReqMp             int32           // CUser.ReqMp: alvo de MP do servidor para regen/poções
	CriticalProgress  uint16          // CUser.cProgress usado pela BASE_GetDoubleCritical
	ShortSkill        [16]uint8       // layout da hotbar do cliente (CUser.CharShortSkill, _MSG_SetShortSkill)
	LoginSpawnX       int16           // último spawn de login injetado pelo servidor, para diagnóstico de movimento
	LoginSpawnY       int16
	LoginTick         uint32
	LoggedFirstAction bool // o primeiro diagnóstico de _MSG_Action pós-login já foi emitido

	// DuelTarget é CUser.RankingTarget: a conn que convidei mais recentemente para um
	// duelo 1v1 (0 = nenhum). DuelTargetExpiry é o prazo em wall-clock (segundos
	// Unix) desse convite pendente — o _MSG_ReqRanking.cpp legado não tem
	// opcode explícito de recusa, então o timeout é como "recusado" é modelado (mesma
	// forma que Entity.GuiltyUntil).
	DuelTarget       int
	DuelTargetExpiry int64

	seen map[int]struct{} // ids de entidade já enviados como CreateMob a este cliente (conjunto de visão)

	// Diagnóstico de envio S→C (sendstats.go): contagens por tipo, totais, os últimos
	// frames e o pico da fila de saída, despejados na desconexão para que um cliente
	// travado deixe um post-mortem. Pertence ao loop como todo outro campo de Session.
	sentFrames        uint64
	sentBytes         uint64
	sentByType        map[protocol.Type]uint32
	lastSent          [lastSentRing]sentRecord
	lastSentIdx       int
	outHighWater      int // frames
	outHighWaterBytes int
	outWarnStep       int // último passo de aviso de "fila alta" (unidades de outWarnStepBytes)

	conn    net.Conn
	out     *outQueue
	closeCh chan struct{}
	closed  bool
}

// close sinaliza ao escritor da sessão para esvaziar os frames S→C enfileirados e
// então fechar o socket (o que por sua vez desbloqueia o leitor). O escritor é dono
// do fechamento do socket para que mensagens enfileiradas logo antes de um close
// (ex.: um aviso de erro) ainda sejam entregues. Idempotente; só no loop.
func (s *Session) close() {
	if s.closed {
		return
	}
	s.closed = true
	close(s.closeCh)
}

// TradeState é a troca direta (P2P) de um jogador com outro jogador
// (lote2-trade-autotrade.md). Active é definido quando este lado envia a primeira oferta;
// Items/InvenPos/Money são a última oferta que o servidor repassou ao oponente,
// e Slots lista os slots de carry a que ela se refere. Confirmed é a flag MyCheck,
// zerada nos dois lados sempre que qualquer oferta muda.
type TradeState struct {
	Active     bool
	OpponentID int
	Confirmed  bool
	Money      int32
	Items      [protocol.MaxTrade]Item
	InvenPos   [protocol.MaxTrade]uint8 // 0xFF = entrada vazia (o char -1 legado)
	Slots      []int                    // slots de carry ofertados
}

// AutoTradeState é uma loja pessoal aberta (o pUser[conn].AutoTrade legado, issue
// #115). Vive só na sessão — nunca é persistida, então uma loja nunca sobrevive a relogin —
// e pertence ao loop. Os itens são vendidos do Cargo da conta por CargoPos; o item
// ofertado é copiado para o slot para uma compra poder compará-lo (memcmp) com o slot vivo do Cargo
// (anti troca de item). Enquanto uma loja está aberta o TradeMode da sessão é 1.
type AutoTradeState struct {
	Title string
	Tax   int16
	Slots [MaxAutoTrade]AutoTradeSlot
}

// AutoTradeSlot é uma oferta de loja: o item (cópia do slot de Cargo do vendedor), seu
// slot de Cargo de origem (-1 = vazio) e seu preço. CargoPos < 0 significa slot sem uso.
type AutoTradeSlot struct {
	Item     Item
	CargoPos int
	Price    int32
}

// Entity é uma entidade do mundo (subconjunto de CMob, domain-model.md §2.2). Jogadores
// (ID < MaxUser) e mobs (ID >= MaxUser) compartilham este tipo e o mesmo espaço
// de índices. A Fase 3 carrega só o mínimo; o estado completo de STRUCT_MOB chega com
// os handlers (Fase 4).
type Entity struct {
	ID   int
	Mode EntityMode
	Name string
	X    int16
	Y    int16
	// SaveX/SaveY são o ponto de save do warp da Gema Estelar (STRUCT_MOB.SPX/SPY,
	// _MSG_UseItem.cpp Vol 12/13) — distintos de X/Y, a posição viva do jogador.
	// 0/0 significa que nenhum ponto foi salvo ainda.
	SaveX    int16
	SaveY    int16
	HP       int32
	MaxHP    int32
	MP       int32 // mana atual (exibição de status)
	MaxMP    int32
	Damage   int32 // CurrentScore.Damage (saída do atacante, combat §4.3)
	AC       int32 // CurrentScore.Ac (mitigação do defensor)
	Master   int   // maestria de arma (nível de combate)
	Critical uint8 // MOB.Critical: limiar rand()%255 para crítico parcial
	Parry    int   // adendo CMob.Parry para GetParryRate
	Level    int32 // CurrentScore.Level (curvas de drop/exp)
	Exp      int64 // STRUCT_MOB.Exp: jogadores acumulam; num mob é a recompensa por abate
	Coin     int32 // ouro carregado

	// IA de mob (mobai.go; só faz sentido para criaturas). EnemyList é o
	// CMob.EnemyList[MAX_ENEMY=13] legado; Target é a entrada selecionada no momento
	// (0 = nenhuma). AtkTick é o horário do servidor do último ataque do mob (cadência);
	// SpawnX/SpawnY é a posição em que o mob (re)spawnou. Range é o alcance
	// de ataque do mob — o maior EF_RANGE entre os equips do seu template
	// (BASE_GetMobAbility, Basedef.cpp:2415), cacheado no spawn; 0 significa sem
	// equipamento de alcance (a IA cai para adjacência corpo a corpo).
	Target         int
	EnemyList      [protocol.MaxTarget]int
	AtkTick        uint32
	SpawnX, SpawnY int16
	Range          int16

	// Perambulação do mob (CMob.h:47-69, StandingByProcessor/SetSegment). SegX/SegY são
	// os waypoints desta INSTÂNCIA (já randomizados ±SegmentRange no spawn,
	// GenerateMob Server.cpp:3536-3546; 0 = slot sem uso, ignorado pelo caminhante).
	// SegmentX/SegmentY é o waypoint atual — também a âncora de aggro/leash
	// (o BattleProcessor prende em SegmentX±HALFGRIDX, CMob.cpp:292) — sempre definido,
	// mesmo para mobs sem rota (= ponto de spawn). WaitTicks conta regressivamente a
	// pausa num waypoint (WaitSec legado; nosso tick≈1s então ticks≈segundos,
	// cadência UNVERIFIED). GenIndex é o bloco NPCGener que spawnou este mob
	// (-1 = nenhum), reservado para a contagem de respawn por gerador (M5).
	RouteType          uint8
	SegListX, SegListY [5]int16
	SegWait            [5]int16
	SegProgress        int8
	SegDir             uint8 // 0 = para frente, 1 = para trás (ping-pong do RouteType 2/3)
	WaitTicks          int16
	SegmentX, SegmentY int16
	GenIndex           int16
	// Template são os bytes crus de STRUCT_MOB de onde este mob spawnou (template de boot,
	// compartilhado por referência — sem cópia). Mantido para o mob poder respawnar em seu
	// SpawnX/SpawnY depois de morrer (world/respawn.go). nil para jogadores.
	Template     []byte
	Merchant     uint8 // bit-packed: cidade de spawn nos bits 6-7 (lote2-movimento.md ChangeCity)
	NonCombatNPC bool  // true para NPCs de cidade/serviço protegidos do dano de jogador
	Grade        uint8 // subtipo de NPC para NPCs de quest com Merchant==100 (EF_GRADE0 de Equip[0])

	Class       uint8    // classe do personagem (0=TK 1=FM 2=BM 3=HT); define o modelo visual
	AttackRun   uint8    // byte de velocidade do CurrentScore.AttackRun — mobs: valor do template (definido no spawn); jogadores: derivado ao vivo (handler attackRunOf)
	Route       [24]byte // última rota de caminhada do _MSG_Action (pMob.Route, MAX_ROUTE=24)
	LastCity    int16    // última cidade visitada (0..3); spawn de login = sua área padrão
	Clan        uint8    // clã/raça
	Guild       uint16   // id da guilda (0 = nenhuma)
	GuildLevel  uint8    // 0 = membro … 9 = líder
	Citizen     uint8    // MobExtra.Citizen; lealdade de cidade e metadados de criação de guilda
	ClassMaster uint8    // tier de party (MobExtra.ClassMaster)
	// Flags de gate da quest celestial (MobExtra.QuestInfo.Celestial, Basedef.h:659-678).
	// CelLv40/CelLv90 liberam os tetos de nível 40/90 Celestial (gate do CheckGetLevel,
	// CMob.cpp:1107); CelCircle marca a quest Cythera Arcana como feita (/arcana). Persistido.
	CelLv40, CelLv90, CelCircle uint8
	// TerraMistica é MobExtra.QuestInfo.Mortal.TerraMistica (_MSG_Quest.cpp
	// AMU_MISTICO, issue #139): definido quando a quest de party é concluída, para o
	// NPC não entregá-la duas vezes. Persistido.
	TerraMistica         uint8
	NewbieQuest          uint8 // QuestInfo.Mortal.Newbie: passo 0..4 do Campo de Treino (issue #395). Persistido.
	MolarGargula         uint8 // QuestInfo.Mortal.MolarGargula: Molar de Gargula usado (issue #398). Persistido.
	PilulaOrc            uint8 // QuestInfo.Mortal.PilulaOrc: Pilula do Poder Magico usada, +9 SkillBonus (issue #404). Persistido.
	CelReset             uint8 // QuestInfo.Celestial.Reset: resets de Pedra da Furia feitos 0..3, +200 pontos de atributo cada (issue #408). Persistido.
	CelAdd120            uint8 // QuestInfo.Celestial.Add120: add do diadema Sub-Celestial já concedido por SetCircletSubGod (issue #409). Persistido.
	CelAdd150            uint8 // QuestInfo.Celestial.Add150: add do diadema Sub-Celestial já concedido por SetCircletSubGod (issue #409). Persistido.
	CelAdd180            uint8 // QuestInfo.Celestial.Add180: add do diadema Sub-Celestial já concedido por SetCircletSubGod (issue #409). Persistido.
	CelAdd200            uint8 // QuestInfo.Celestial.Add200: add do diadema Sub-Celestial já concedido por SetCircletSubGod (issue #409). Persistido.
	CelestialLevel       int16 // QuestInfo.Celestial.CelestialLevel: nível do Celestial quando foi estacionado para o Sub (issue #410). Persistido.
	SubCelestialLevel    int16 // QuestInfo.Celestial.SubCelestialLevel: nível do Sub quando foi estacionado para o Celestial (issue #410). Persistido.
	ArchLv355, ArchLv370 uint8
	MortalLevel          uint16
	CelestialArchLevel   uint8
	ArchCrystalStage     uint8
	Soul                 uint8 // MobExtra.Soul; 0 significa que não há alma modelada
	Fame                 int32 // MobExtra.Fame; carregado do banco, atualizado pelo Selo do Guerreiro e exibido por /nick
	KefraTicket          int32 // Entradas restantes no Salão de Kefra, por personagem.
	NightmareEntries     int32 // Entradas restantes no Pesadelo Arcano.
	LastNightmareUse     int64 // Horário Unix do último uso do título do Pesadelo.
	QuestFlag            uint8 // passe volátil de área de quest (CMob.QuestFlag; ex.: quest 256)

	// SaveCelestial é MobExtra.SaveCelestial[2]: [0] o Celestial estacionado,
	// [1] o Sub-Celestial estacionado (issue #410). Persistido.
	SaveCelestial [2]CelestialSheet

	// PKMode é a flag de consentimento de Player-Killer que o jogador alterna (tecla K, _MSG_PKMode;
	// pUser[conn].PKMode legado). Ela controla se o jogador pode acertar golpes de PvP,
	// mas NÃO faz o nick piscar por si só. Só na sessão, não persistida.
	PKMode bool

	// PKPoint é o byte PKPoint legado (GetFunc.cpp GetPKPoint/SetPKPoint, o slot
	// oculto de carry KILL_MARK): o contador de caos/karma, limitado a [1,150] na
	// escrita, 75 = neutro. O valor no wire/exibição é PKPoint-75 (faixa [-74,+75],
	// o comando /cp). Define a cor do nick via handler.pkPoint (MobName[12],
	// GetFunc.cpp GetCreateMob): caos = PKPoint a menos que Guilty > 0, caso em que
	// caos = 0 (vermelho/piscando). Persistido.
	PKPoint uint8
	// Guilty é o byte Guilty legado (mesmo slot KILL_MARK, limitado a [0,50] na
	// escrita): vira 8 ao acertar (ou receber) um golpe PvP fora de duelo
	// (_MSG_Attack.cpp "PK - War - Miss"), decrementado em 1 a cada ~8s enquanto
	// conectado (RegenMob, Server.cpp). Guilty > 0 força o nick vermelho independente
	// de PKPoint. Persistido (quem desloga no meio do decaimento deve retomar, não zerar).
	Guilty uint8
	// CurKill/TotKill são os contadores legados de sequência de abates (GetCurKill/GetTotKill,
	// mesmo slot KILL_MARK): CurKill é a sequência atual ininterrupta de abates PvP
	// (zerada ao morrer, MobName[13]); TotKill é a contagem de abates PvP de toda a vida
	// (MobName[14..15] LE). Cosmético (empacotado em todo MSG_CreateMob para outros
	// clientes renderizarem) — nenhum efeito de gameplay modelado. Persistido.
	CurKill uint8
	TotKill uint16

	Str        int16 // atributos do CurrentScore (base + equipamento, mantidos vivos por refreshScore)
	Int        int16
	Dex        int16
	Con        int16
	Special    [4]int16 // CurrentScore.Special[4] = BaseSpecial + equipamento/affects
	ScoreBonus uint16   // pontos de atributo livres

	// Estado de skills (frente de skills). SkillBonus é derivado (level*3 − Σ custos
	// aprendidos, BASE_GetBonusSkillPoint) no login e ao subir de nível; SpecialBonus é
	// incremental (+2/nível) e persistido. Magic escala o dano de skill do conjurador;
	// SaveMana desconta custos de mana (origem de ambos nos jogadores UNVERIFIED —
	// zero até ser capturado). Resist[4] alimenta SkillResistScale (mobs: template).
	LearnedSkill    int32
	SecLearnedSkill int32
	SkillBonus      uint16
	SpecialBonus    uint16
	BaseSpecial     [4]int16 // pontos de maestria alocados (BaseScore.Special)
	SkillBar        [4]uint8 // MOB.SkillBar (persistido com o personagem)
	Magic           int16
	SaveMana        int16
	Resist          [4]int16

	// BaseScore: o score sem equipamento (atributos alocados + AC/Damage/MaxHP/MaxMP
	// derivados de nível/classe). CurrentScore (os campos vivos acima + AC/Damage/MaxHP/
	// MaxMP) = BaseScore + equipamento, recalculado por handler.refreshScore sempre que o
	// equipamento ou os atributos mudam. Derivado uma vez no login (atual − equipamento) e não
	// persistido (é rederivado do CurrentScore persistido a cada login).
	BaseStr, BaseInt, BaseDex, BaseCon int16
	BaseAC, BaseDamage                 int32
	BaseMaxHP, BaseMaxMP               int32
	// EquipmentAttributeHP/MP são as contribuições de recurso CON/INT do equipamento
	// incluídas nos máximos vivos mas excluídas dos saves por compatibilidade com
	// personagens salvos antes da issue #321. Substituídas a cada refresh do score.
	EquipmentAttributeHP, EquipmentAttributeMP int32
	// Magic, Parry (esquiva) e Resist NÃO têm termo de base: o legado também não tem.
	// BASE_GetCurrentScore semeia um `magic` local novo a partir do equipamento (Basedef.cpp:3194),
	// acumula os termos derivados e o guarda em :4729 — não existe BaseScore.Magic
	// em lugar nenhum do original, e o MOB.Magic guardado nunca volta a entrar. refreshScore
	// deriva os três inteiramente do equipamento atual a cada chamada (issue #211 bug 3,
	// issue #231).

	// HpAddPct/MpAddPct: bônus percentual de EF_HPADD/EF_MPADD do equipamento (ex.: +10 =
	// +10%). Cacheado por refreshScore e aplicado no momento da LEITURA (máximo efetivo de HP/MP),
	// nunca embutido em MaxHP/MaxMP — então o score persistido fica plano e a
	// derivação por subtração se sustenta (captura-wyd-affect-divina.md §E).
	HpAddPct, MpAddPct int32

	// RunSpeedBonus é a soma de EF_RUNSPEED do equipamento equipado (botas), cacheada por
	// refreshScore e aplicada na leitura por handler.attackRunOf ao nibble de velocidade
	// de movimento (baixo) do AttackRun.
	RunSpeedBonus int32

	// Affect guarda os buffs/debuffs ativos (STRUCT_AFFECT[32]). DivineEnd é o prazo
	// em wall-clock (segundos Unix) do buff Divino — a fonte da verdade da
	// sua expiração; o Affect.Time do slot é só o timer do ícone no cliente.
	Affect    [MaxAffect]Affect
	DivineEnd int64

	// Rsv é o byte de flags de estado MOB.Rsv (RSV_HASTE/BLOCK/…), recalculado a partir
	// dos affects ativos por refreshScore. As contribuições de score dos affects (Aff*)
	// são cacheadas do mesmo jeito e aplicadas no momento da LEITURA (getters efetivos), para
	// o score plano persistido nunca embutir um buff (sem contagem dupla no
	// relogin — mesma política de HpAddPct/Divine).
	Rsv               uint8
	AffDamage         int32
	AffAC             int32
	AffMaxHP          int32
	AffMaxMP          int32
	AffStr            int16
	AffInt            int16
	AffDex            int16
	AffCon            int16
	AffRunSpeed       int32
	AffAttackSpeed    int32
	AffCritical       int16
	AffSpecial        [4]int16
	AffResist         [4]int16
	AffForceDamage    int32 // ForceDamage, ex.: Ligacao Espectral
	AffForceMobDamage int32 // ForceMobDamage, ex.: Toxina da Serpente
	AffExpBonus       int32 // do affect tipo 39 (Baú de XP)
	// AffHpAbs é o percentual de roubo de vida ao acertar da Jóia da Absorção (affect
	// tipo 8, bit 3): o atacante recupera AffHpAbs% do dano causado, com 50% de
	// chance, limitado a 350 por golpe (_MSG_Attack.cpp:1651). AffMagic é o buff de +20%
	// Magic da Jóia do Poder (affect 8, bit 5), lido no Magic efetivo.
	AffHpAbs int32
	AffMagic int32
	// AffDamageMultiPct é o percentual legado DAMAGEMULTI (100 = neutro): um
	// multiplicador de dano no momento da LEITURA aplicado sobre Damage+AffDamage mas antes de
	// WeaponDamage, exatamente onde Basedef.cpp:4654 multiplica CurrentScore.Damage.
	// Deltas aditivos não expressam isso — attributeDamageBonus cai no Damage
	// plano depois da passagem de affects, e o multiplicador precisa cobrir isso também.
	AffDamageMultiPct int32
	EquipExpBonus     int32 // do slot de fada + equipamento de grade/gema (CMob.cpp:711-870)

	EquipVisual [16]uint16 // códigos visuais dos itens para MSG_CreateMob/UpdateEquip
	EquipAnct   [16]uint8  // bytes do brilho de refino/ancient emparelhados com EquipVisual

	// Estado de party (lote2-party-guilda-guerra.md). Membros apontam Leader para a
	// conn do líder; líderes mantêm Leader=0 e são donos da PartyList. LastReqParty é
	// quem convidou esta entidade por último (gate anti-forja).
	Leader       int
	LastReqParty int

	// Summoner é a conn do jogador que evocou este mob (o pMob.Summoner do
	// GenerateSummon, Server.cpp:3244); 0 = não é invocação. O Leader de uma invocação é
	// o líder de party do dono (ou o próprio dono), como o vínculo legado.
	Summoner  int
	PartyList [MaxParty]int

	Equip [MaxEquip]Item // itens equipados
	Carry [MaxCarry]Item // inventário; para mobs também é a tabela de loot (§2.2)
}

// IsPlayer informa se um índice de entidade pertence a um jogador (domain-model.md
// §1: id < MaxUser ⇒ jogador).
func IsPlayer(id int) bool { return id >= 0 && id < MaxUser }
