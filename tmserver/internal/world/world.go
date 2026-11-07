// Package world é o estado de jogo autorotativo em memória do tmServer e
// do seu laço de jogo de dono único (domain-model.md §1/§5, migration-plan.md §3.5).
//
// Modelo de concorrência — a única regra que preserva a paridade e mata a duplicação de item:
// TODO o estado do mundo pertence a exatamente uma goroutine (Run); ele nunca é mutado
// em outro lugar. O I/O de rede roda em goroutines por conexão que só trocam
// mensagens com o loop por canais (eventos entram, saída por sessão). Não há
// locks no estado do mundo, espelhando o servidor WinSock single-threaded
// original (domain-model.md §5, guidelines §9).
package world

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"openwyd-nextgen/internal/metrics"
	"openwyd-nextgen/tmserver/internal/protocol"
	"openwyd-nextgen/tmserver/internal/rng"
)

// Limites de capacidade (Basedef.h via domain-model.md §6). O espaço de índices é compartilhado:
// pMob[0..MaxUser) são jogadores, pMob[MaxUser..MaxMob) são mobs/NPCs.
const (
	MaxUser        = 1000
	MaxMob         = 25000
	MaxItem        = 5000 // itens no chão (pItem[])
	MaxCarry       = 64   // slots de inventário por entidade (MAX_CARRY)
	MaxEquip       = 16   // slots de equipamento (MAX_EQUIP)
	MaxCargo       = 128  // slots do armazém compartilhado da conta (MAX_CARGO)
	MaxAutoTrade   = 12   // slots de itens da loja pessoal (MAX_AUTOTRADE, issue #115)
	MaxParty       = 12   // membros de party (MAX_PARTY)
	DefaultGridDim = 4096

	// KefraBossGenIndex é KEFRA_BOSS (Basedef.h:475), o bloco NPCGener com
	// regras especiais de combate de alcance/posição fixos no CMob.cpp.
	KefraBossGenIndex = 396

	// GroundItemIDOffset é somado ao índice de um item no chão no wire
	// (_MSG_GetItem decodifica ItemID-10000; handlers/_MSG_GetItem.md).
	GroundItemIDOffset = 10000
)

// Mode é a máquina de estados da sessão CUser.Mode (domain-model.md §3.1).
type Mode uint8

// Modos de sessão (CUser.h:26-37).
const (
	UserEmpty       Mode = 0
	UserAccept      Mode = 1
	UserLogin       Mode = 2
	UserSelChar     Mode = 11
	UserCharWait    Mode = 12
	UserWaitDB      Mode = 13
	UserPlay        Mode = 22
	UserSaving4Quit Mode = 24
)

// EntityMode é a máquina de estados de entidade do mundo CMob.Mode (domain-model.md §3.2).
type EntityMode uint8

// Modos de entidade (CMob.h:26-35).
const (
	MobEmpty    EntityMode = 0
	MobUserDock EntityMode = 1
	MobUser     EntityMode = 2
	MobIdle     EntityMode = 3
	MobPeace    EntityMode = 4
	MobCombat   EntityMode = 5
	MobReturn   EntityMode = 6
	MobFlee     EntityMode = 7
	MobRoam     EntityMode = 8
	MobWaitDB   EntityMode = 9
)

// Handler processa um frame de cliente já decodificado dentro da goroutine do loop, então
// pode mutar o estado do mundo à vontade. A Fase 4 substitui o default pelo
// despacho real por mensagem (handlers/*.md).
type Handler func(w *World, s *Session, h protocol.Header, payload []byte)

// Config ajusta um World. GridDim usa DefaultGridDim (4096); os testes usam um
// valor pequeno para não alocar os grids espaciais densos completos.
type Config struct {
	GridDim        int
	OutBufferBytes int           // bytes S→C não enviados por sessão (default DefaultOutBufferBytes)
	EventQueue     int           // profundidade da fila de eventos de entrada
	Now            func() uint32 // relógio do servidor (ClientTick); injetável para testes

	// Endurecimento (Fase 7, migration-plan.md §5), tudo opcional:
	// RejectChecksum derruba a conexão em caso de checksum CPSock divergente. A pilha
	// legada não rejeita e o ClientPatch anula as checagens do cliente, então isto fica
	// desligado por default; ligue quando uma captura confirmar que o cliente envia
	// checksums corretos (protocol-spec.md §1.5).
	RejectChecksum bool
	// MaxMsgPerSec limita a taxa de mensagens de entrada por conexão (0 = desligado);
	// MsgBurst é a profundidade do balde (usa MaxMsgPerSec quando <=0). Um flood
	// desconecta a conexão infratora, protegendo o reator (NF1).
	MaxMsgPerSec float64
	MsgBurst     int

	// ItemRanges mapeia índice de item → seu valor EF_RANGE do catálogo (content
	// ItemList.Ranges). SpawnMob usa isso para derivar o alcance de ataque de um mob
	// dos equips do template (BASE_GetMobAbility, Basedef.cpp:2415). Imutável após
	// o boot; nil significa sem catálogo (todo mob luta em alcance corpo a corpo).
	ItemRanges map[int]int16

	// LogSends registra em INFO cada frame S→C enfileirado (conn/tipo/id/len) — o
	// espelho de saída do log "recv packet" do dispatcher. Volume alto: ligue
	// só enquanto reproduz um incidente (investigação de freeze,
	// docs/migration/investigacao-freeze-cliente.md).
	LogSends bool

	// StatusFile é o caminho da página de status do canal (serv00.htm) que o cliente
	// busca por HTTP antes de abrir a conexão CPSock do jogo. Quando definido, o
	// edge responde a uma sondagem "GET" com o conteúdo deste arquivo; vazio serve um
	// default embutido. A checagem HTTP de status do client-edge não está documentada
	// no protocol-spec.md (só CPSock) — descoberta numa captura de cliente ao vivo.
	StatusFile string

	// Metrics recebe os contadores do loop (tick, conexões) e é exportado pelo
	// endpoint /metrics do processo. nil desliga a instrumentação (testes).
	Metrics *metrics.Registry
}

// World guarda todo o estado de jogo mutável. Cada campo é tocado só pela
// goroutine de Run (e por auxiliares que ela chama). Não acesse de outras goroutines.
type World struct {
	cfg     Config
	log     *slog.Logger
	metrics *metrics.Registry
	persist Persistence
	billing Billing
	handler Handler

	sessions []*Session    // index = conn ∈ [0, MaxUser)
	entities []*Entity     // espaço de índices compartilhado com os jogadores (domain-model.md §1)
	ground   []*GroundItem // pItem[]: itens no chão, índice ∈ [1, MaxItem)
	grid     *Grid
	rng      *rng.MSVC // LCG do MSVC do loop (paridade; como o rand() global original)

	// cargo é o armazém compartilhado da conta, com chave por id de conta. Ele é carregado no
	// login da conta e vive por toda a sessão da conta (atravessa a seleção de
	// personagem ↔ jogo), então a chave é a conta, não a sessão/conn. Do loop.
	cargo map[int64]*CargoState

	// savingAccounts conta, por id de conta, as sessões fechadas cujo save final de
	// personagem/cargo ainda está em voo (saveAccountOnClose). Um novo login
	// dessa conta espera chegar a zero, então nunca carrega linhas mais velhas do que
	// a sessão anterior tinha em memória. Criado no primeiro uso. Do loop.
	savingAccounts map[int64]int

	// guilds é o registro mínimo de guildas (guild.go): nome/fama com chave por id
	// de guilda. Só em memória — ainda não há fluxo de criação de guilda a
	// persistir. Do loop.
	guilds map[uint16]GuildInfo

	// worldEvent é o estado do evento global de drop gerenciado pelo portal. Do loop; o
	// dispatcher aplica snapshots vindos do dbServer e avança o CurrentIndex em
	// drops bem-sucedidos do evento.
	worldEvent EventConfig

	// O ciclo carregado também fornece o snapshot final de shutdown, para que um save
	// assíncrono mais antigo pendente não perca uma derrota num restart normal.
	kefraState  KefraState
	nightmare   NightmareState
	runeQuest   RuneQuestState
	kefraLoaded bool

	// newbieEvent espelha a flag legada NewbieEventServer (Server.cpp:617).
	// O próprio mundo só precisa dela para o handicap de HP no spawn; o lado do EXP
	// vive nos ExpEvents do dispatcher. Do loop.
	newbieEvent bool

	events    chan event
	callbacks chan event // resultados assíncronos de handler (World.Go / World.GoDetached); separado
	// dos eventos para um tick longo da IA de mob não travar login/db na fila principal.
	done   chan struct{}  // fechado quando o loop para; desbloqueia as goroutines de conexão
	saveWG sync.WaitGroup // acompanha os saves assíncronos de personagem em voo (logout/desconexão)

	// onTick é o gancho periódico de simulação (IA de mob), roda dentro do loop; veja
	// tick.go. nil desliga o ticker (ex.: em testes de protocol/transporte).
	onTick       func(*World)
	tickInterval time.Duration

	// onSessionEnd é o gancho de desmonte (desvincula a party), roda dentro do loop
	// logo antes de o slot da sessão ser liberado; veja event.go. nil o desliga.
	onSessionEnd func(*World, *Session)

	// respawnQueue guarda criaturas mortas aguardando respawn, drenada por SpawnDueRespawns
	// a partir do tick (world/respawn.go). Do loop.
	respawnQueue []respawnEntry

	// generators é a tabela de blocos NPCGener (world/generator.go): receitas de spawn
	// mais a contabilidade viva de CurrentNumMob. mobCount acompanha o total vivo de mobs/NPCs
	// (o gate generateWorldCap). Do loop.
	generators []*Generator
	mobCount   int
}

// New cria um World com as dependências dadas. Um handler nil instala um
// default sem efeito (Fase 3: só a fiação de transporte).
func New(cfg Config, log *slog.Logger, persist Persistence, handler Handler) *World {
	if cfg.GridDim <= 0 {
		cfg.GridDim = DefaultGridDim
	}
	if cfg.OutBufferBytes <= 0 {
		cfg.OutBufferBytes = DefaultOutBufferBytes
	}
	if cfg.EventQueue <= 0 {
		cfg.EventQueue = 1024
	}
	if cfg.Now == nil {
		cfg.Now = func() uint32 { return uint32(time.Now().UnixMilli()) }
	}
	if log == nil {
		log = slog.Default()
	}
	if handler == nil {
		handler = func(*World, *Session, protocol.Header, []byte) {}
	}
	return &World{
		cfg:       cfg,
		log:       log,
		metrics:   cfg.Metrics,
		persist:   persist,
		billing:   AllowAllBilling{},
		handler:   handler,
		sessions:  make([]*Session, MaxUser),
		entities:  make([]*Entity, MaxMob),
		ground:    make([]*GroundItem, MaxItem),
		cargo:     make(map[int64]*CargoState),
		guilds:    make(map[uint16]GuildInfo),
		grid:      newGrid(cfg.GridDim),
		rng:       rng.New(),
		events:    make(chan event, cfg.EventQueue),
		callbacks: make(chan event, 256),
		done:      make(chan struct{}),
	}
}

// Run é o dono único do estado do mundo. Processa eventos de entrada até ctx
// ser cancelado, depois drena/salva as sessões ativas e devolve ctx.Err().
func (w *World) Run(ctx context.Context) error {
	w.log.Info("world loop started", "grid", w.cfg.GridDim)
	if w.onTick != nil && w.tickInterval > 0 {
		go w.runTicker(ctx)
	}
	for {
		select {
		case <-ctx.Done():
			w.shutdown()
			return ctx.Err()
		case cb := <-w.callbacks:
			w.applyTimed(cb)
			continue
		default:
		}
		select {
		case <-ctx.Done():
			w.shutdown()
			return ctx.Err()
		case cb := <-w.callbacks:
			w.applyTimed(cb)
		case ev := <-w.events:
			w.applyTimed(ev)
		}
	}
}

// slowEventThreshold é quanto um evento de loop pode durar antes de ser logado:
// qualquer coisa acima disso trava todas as sessões de uma vez (o loop é de dono único), então
// um WARN de "slow world event" é a prova de lag global do lado do servidor.
const slowEventThreshold = 100 * time.Millisecond

// applyTimed aplica um evento de loop e avisa quando ele excede slowEventThreshold,
// identificando o evento (tipo de frame/conn, tick, callback) para que um travamento possa
// ser rastreado até seu handler (instrumentação da investigação de freeze).
func (w *World) applyTimed(ev event) {
	start := time.Now()
	ev.apply(w)
	d := time.Since(start)
	kind := "other"
	switch ev.(type) {
	case frameEvent:
		kind = "frame"
	case tickEvent:
		kind = "tick"
	case callbackEvent:
		kind = "callback"
	}
	w.metricObserve("world_event_duration_seconds", "Duração de um evento do loop.", d.Seconds(), "kind", kind)
	w.metricInc("world_events_total", "Eventos processados pelo loop.", "kind", kind)
	if d < slowEventThreshold {
		return
	}
	switch e := ev.(type) {
	case frameEvent:
		w.log.Warn("slow world event", "dur_ms", d.Milliseconds(), "kind", "frame", "conn", e.s.Conn, "type", formatSendType(e.header.Type))
	case tickEvent:
		w.log.Warn("slow world event", "dur_ms", d.Milliseconds(), "kind", "tick")
	case callbackEvent:
		w.log.Warn("slow world event", "dur_ms", d.Milliseconds(), "kind", "callback", "conn", e.conn)
	default:
		w.log.Warn("slow world event", "dur_ms", d.Milliseconds(), "kind", fmt.Sprintf("%T", ev))
	}
}

// shutdown drena as sessões ativas: persiste os jogadores no mundo, depois para o I/O deles.
func (w *World) shutdown() {
	close(w.done) // sinaliza às goroutines de conexão para pararem de enviar eventos
	if w.kefraLoaded && w.persist != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := w.persist.SaveKefraState(ctx, w.kefraState); err != nil {
			w.log.Error("save kefra state on shutdown failed", "err", err)
		}
		cancel()
	}
	saved := 0
	for _, s := range w.sessions {
		if s == nil {
			continue
		}
		if s.Mode == UserPlay && s.AccountID != 0 {
			if err := w.persist.SaveOnShutdown(context.Background(), w.characterSave(s)); err != nil {
				w.log.Warn("save on shutdown failed", "conn", s.Conn, "err", err)
			} else {
				saved++
			}
		}
		s.close()
	}
	// Persiste os armazéns de conta ainda carregados (escopo da conta, logo salvos uma vez
	// por conta, independente dos saves de personagem por sessão acima).
	for accountID := range w.cargo {
		if err := w.persist.SaveCargo(context.Background(), w.cargoSave(accountID)); err != nil {
			w.log.Warn("save cargo on shutdown failed", "account", accountID, "err", err)
		}
	}
	// Espera os saves de desconexão/logout em voo, para um shutdown nunca perder um.
	w.saveWG.Wait()
	w.log.Info("world loop stopped", "sessions_saved", saved)
}

// SaveCharacterAsync persiste o estado vivo de um personagem em jogo (Carry/Coin/stats)
// sem bloquear o loop: captura o CharacterSave no loop (uma cópia
// por valor) e roda o save gRPC numa goroutine. Chamado no logout/desconexão para
// compras, ouro e progresso sobreviverem à sessão. Só no loop (captura estado).
func (w *World) SaveCharacterAsync(s *Session) {
	if s == nil || s.Mode != UserPlay || s.AccountID == 0 {
		return
	}
	cs := w.characterSave(s)
	w.saveWG.Add(1)
	go func() {
		defer w.saveWG.Done()
		if err := w.persist.SaveOnShutdown(context.Background(), cs); err != nil {
			w.log.Warn("save character failed", "account", cs.AccountID, "slot", cs.Slot, "err", err)
		}
	}()
}

// SaveCharacterThen persiste o personagem e roda then (de volta no loop) só
// depois do save commitar. Use onde o cliente pode reler o personagem do banco
// imediatamente (logout para a seleção de personagem): adiar a
// confirmação até o save chegar evita a releitura correr contra a escrita. then
// sempre roda, mesmo quando não há nada a salvar.
func (w *World) SaveCharacterThen(s *Session, then func(*World, *Session)) {
	if s == nil || s.Mode != UserPlay || s.AccountID == 0 {
		then(w, s)
		return
	}
	cs := w.characterSave(s)
	p := w.persist
	w.Go(s, func() func(*World, *Session) {
		err := p.SaveOnShutdown(context.Background(), cs)
		return func(w *World, s *Session) {
			if err != nil {
				w.log.Warn("save character failed", "account", cs.AccountID, "slot", cs.Slot, "err", err)
			}
			then(w, s)
		}
	})
}

// characterSave faz snapshot da entidade em jogo de uma sessão para um CharacterSave. Só
// campos autoritativos do mundo são capturados (veja CharacterSave). Só no loop.
func (w *World) characterSave(s *Session) CharacterSave {
	e := w.entities[s.Conn]
	return w.CharacterSaveFor(s, e)
}

// CharacterSaveFor faz snapshot de uma entidade preparada sem publicá-la no mundo.
// Os handlers usam isso para operações que persistem primeiro, como a compra da capa de reino.
func (w *World) CharacterSaveFor(s *Session, e *Entity) CharacterSave {
	cs := CharacterSave{AccountID: s.AccountID, Slot: s.Slot}
	if e == nil {
		return cs
	}
	cs.Clan, cs.GuildID, cs.GuildLevel, cs.Soul, cs.Fame = e.Clan, e.Guild, e.GuildLevel, e.Soul, e.Fame
	cs.KefraTicket = e.KefraTicket
	cs.NightmareEntries = e.NightmareEntries
	cs.LastNightmareUse = e.LastNightmareUse
	cs.Class, cs.ClassMaster = e.Class, e.ClassMaster
	cs.CelLv40, cs.CelLv90, cs.CelCircle = e.CelLv40, e.CelLv90, e.CelCircle
	cs.ArchLv355, cs.ArchLv370 = e.ArchLv355, e.ArchLv370
	cs.MortalLevel, cs.CelestialArchLevel = e.MortalLevel, e.CelestialArchLevel
	cs.ArchCrystalStage = e.ArchCrystalStage
	cs.TerraMistica = e.TerraMistica
	cs.NewbieQuest = e.NewbieQuest
	cs.MolarGargula = e.MolarGargula
	cs.PilulaOrc = e.PilulaOrc
	cs.CelReset = e.CelReset
	cs.CelAdd120, cs.CelAdd150, cs.CelAdd180, cs.CelAdd200 = e.CelAdd120, e.CelAdd150, e.CelAdd180, e.CelAdd200
	cs.CelestialLevel, cs.SubCelestialLevel, cs.SaveCelestial = e.CelestialLevel, e.SubCelestialLevel, e.SaveCelestial
	cs.LastCity = e.LastCity
	cs.SaveX, cs.SaveY = e.SaveX, e.SaveY
	cs.Level, cs.Exp, cs.Coin = e.Level, e.Exp, e.Coin
	cs.Str, cs.Int, cs.Dex, cs.Con = e.Str, e.Int, e.Dex, e.Con
	// Mantém a representação plana histórica de save: saves antigos não têm os recursos
	// CON/INT do equipamento. O login deriva a base deste valor e então soma a
	// contribuição do equipamento atual uma vez, inclusive em saves anteriores à issue 321.
	cs.HP, cs.MaxHP = e.HP, e.MaxHP-e.EquipmentAttributeHP
	cs.MP, cs.MaxMP = e.MP, e.MaxMP-e.EquipmentAttributeMP
	// A base sem equipamento viaja por conta própria (migration 0027): reconstruí-la
	// no login a partir dos valores acima depende das regras de bônus daquele momento.
	cs.BaseStr, cs.BaseInt, cs.BaseDex, cs.BaseCon = e.BaseStr, e.BaseInt, e.BaseDex, e.BaseCon
	cs.BaseMaxHP, cs.BaseMaxMP = e.BaseMaxHP, e.BaseMaxMP
	cs.DivineEnd = e.DivineEnd // 0 quando o buff expirou (zerado pela varredura do tick)
	cs.ScoreBonus, cs.SpecialBonus = e.ScoreBonus, e.SpecialBonus
	cs.LearnedSkill, cs.SecLearnedSkill, cs.BaseSpecial = e.LearnedSkill, e.SecLearnedSkill, e.BaseSpecial
	cs.SkillBar, cs.ShortSkill = e.SkillBar, s.ShortSkill
	cs.PKPoint, cs.Guilty, cs.CurKill, cs.TotKill = e.PKPoint, e.Guilty, e.CurKill, e.TotKill
	for _, a := range e.Affect {
		// Divine persiste à parte como o DivineEnd em wall-clock; slots vazios caem.
		if a.Type == 0 || a.Type == AffectDivine {
			continue
		}
		cs.Affects = append(cs.Affects, a)
	}
	cs.Carry = savedItems(e.Carry[:])
	cs.Equip = savedItems(e.Equip[:])
	return cs
}

// savedItems achata um vetor posicional de itens nos slots não vazios de SavedItem.
func savedItems(items []Item) []SavedItem {
	var out []SavedItem
	for i, it := range items {
		if it.Empty() {
			continue
		}
		out = append(out, SavedItem{
			Slot:  i,
			Index: it.Index,
			Eff1:  it.Effects[0].Effect, EffV1: it.Effects[0].Value,
			Eff2: it.Effects[1].Effect, EffV2: it.Effects[1].Value,
			Eff3: it.Effects[2].Effect, EffV3: it.Effects[2].Value,
			ExpiresAt: it.ExpiresAt,
		})
	}
	return out
}

// Cargo devolve o armazém carregado da conta, ou nil se nenhum estiver carregado (ex.:
// a conta não está logada, ou o LoadCargo falhou). Só no loop.
func (w *World) Cargo(accountID int64) *CargoState {
	if accountID == 0 {
		return nil
	}
	return w.cargo[accountID]
}

// SetCargo instala no store o armazém carregado de uma conta. Chamado do
// loop logo após um login de conta bem-sucedido. Só no loop.
func (w *World) SetCargo(accountID int64, st *CargoState) {
	if accountID == 0 || st == nil {
		return
	}
	st.AccountID = accountID
	w.cargo[accountID] = st
}

// ApplyDeliveries coloca no cargo da conta os grants pendentes já buscados da
// delivery_queue (loja de doação web, issue #34). pending é carregado na
// mesma ida-e-volta fora do loop do próprio login da conta (dbclient.AccountLogin),
// então drenar a caixa de entrada não custa ida-e-volta extra ao backend. Chamado do loop
// logo após o cargo ser instalado no login: coloca cada item no próximo slot
// livre do cargo (perdido quando o cargo está cheio) e persiste o cargo + confirma as
// linhas da fila numa única transação de backend, fora do loop. Só no loop.
func (w *World) ApplyDeliveries(s *Session, pending []Delivery) {
	if s == nil || s.AccountID == 0 || len(pending) == 0 {
		return
	}
	accountID := s.AccountID
	cargo := w.cargo[accountID]
	if cargo == nil {
		return
	}
	var deliveredIDs, lostIDs []int64
	for _, d := range pending {
		if w.AddToCargo(cargo, d.Item) >= 0 {
			deliveredIDs = append(deliveredIDs, d.ID)
		} else {
			lostIDs = append(lostIDs, d.ID)
		}
	}
	if len(lostIDs) > 0 {
		w.log.Warn("donate deliveries lost: cargo full", "account", accountID, "count", len(lostIDs))
	}
	w.log.Info("drained donate deliveries", "account", accountID, "delivered", len(deliveredIDs), "lost", len(lostIDs))

	cs := w.cargoSave(accountID)
	p := w.persist
	w.Go(s, func() func(*World, *Session) {
		e := p.SaveCargoWithDeliveries(context.Background(), cs, deliveredIDs, lostIDs)
		return func(w *World, _ *Session) {
			if e != nil {
				w.log.Warn("save cargo with deliveries failed", "account", accountID, "err", e)
			}
		}
	})
}

// cargoSave faz snapshot do armazém de uma conta num CargoSave. Só no loop.
func (w *World) cargoSave(accountID int64) CargoSave {
	cs := CargoSave{AccountID: accountID}
	if c := w.cargo[accountID]; c != nil {
		cs.Coin = c.Coin
		cs.Items = savedItems(c.Items[:])
	}
	return cs
}

// ReleaseCargo persiste o armazém de uma conta e o remove do store. É
// chamado quando a sessão da conta termina (desconexão/logout) para o cofre em
// memória não vazar e os últimos depósitos sobreviverem. O save roda fora do loop
// (acompanhado por saveWG, como SaveCharacterAsync) para um shutdown nunca perdê-lo.
// Só no loop (captura o estado antes de ir para async).
func (w *World) ReleaseCargo(accountID int64) {
	if accountID == 0 || w.cargo[accountID] == nil {
		return
	}
	cs := w.cargoSave(accountID)
	delete(w.cargo, accountID)
	w.saveWG.Add(1)
	go func() {
		defer w.saveWG.Done()
		if err := w.persist.SaveCargo(context.Background(), cs); err != nil {
			w.log.Warn("save cargo failed", "account", cs.AccountID, "err", err)
		}
	}()
}

// saveAccountOnClose é o desmonte da conta de uma sessão que fecha: faz snapshot
// do personagem em jogo e, quando não resta outra sessão da conta, do
// cargo da conta (despejando-o), depois salva ambos fora do loop sob saveWG. A
// conta conta como salvando até os dois saves retornarem; AccountSaving deixa um novo
// login da mesma conta esperar por isso, para o chute de login duplicado
// (handler/login.go) nunca reler linhas mais velhas que o estado desta sessão.
// Um save que falha é logado e ainda encerra o estado de salvamento, como os saves
// dispara-e-esquece que ele substitui faziam. Só no loop (captura antes de ir para async).
func (w *World) saveAccountOnClose(s *Session) {
	if s == nil || s.AccountID == 0 {
		return
	}
	accountID := s.AccountID
	var char *CharacterSave
	if s.Mode == UserPlay {
		cs := w.characterSave(s)
		char = &cs
	}
	var cargo *CargoSave
	if w.cargo[accountID] != nil && w.SessionByAccount(accountID, s) == nil {
		cs := w.cargoSave(accountID)
		cargo = &cs
		delete(w.cargo, accountID)
	}
	if char == nil && cargo == nil {
		return
	}
	if w.savingAccounts == nil {
		w.savingAccounts = make(map[int64]int)
	}
	w.savingAccounts[accountID]++
	w.saveWG.Add(1)
	go func() {
		defer w.saveWG.Done()
		if char != nil {
			if err := w.persist.SaveOnShutdown(context.Background(), *char); err != nil {
				w.log.Warn("save character failed", "account", char.AccountID, "slot", char.Slot, "err", err)
			}
		}
		if cargo != nil {
			if err := w.persist.SaveCargo(context.Background(), *cargo); err != nil {
				w.log.Warn("save cargo failed", "account", cargo.AccountID, "err", err)
			}
		}
		done := worldCallbackEvent{cb: func(w *World) {
			if w.savingAccounts[accountID]--; w.savingAccounts[accountID] <= 0 {
				delete(w.savingAccounts, accountID)
			}
		}}
		select {
		case w.callbacks <- done:
		case <-w.done:
		}
	}()
}

// AccountSaving informa se uma sessão fechada da conta ainda tem seu
// save final em voo. Só no loop.
func (w *World) AccountSaving(accountID int64) bool { return w.savingAccounts[accountID] > 0 }

// SessionByAccount devolve uma sessão da conta, diferente de except, que
// concluiu o login da conta (seleção de personagem ou adiante), ou nil. As sessões
// são indexadas só por conn; uma varredura linear sobre MaxUser é suficiente para uma
// checagem no login. Só no loop.
func (w *World) SessionByAccount(accountID int64, except *Session) *Session {
	if accountID == 0 {
		return nil
	}
	for _, o := range w.sessions {
		if o != nil && o != except && !o.closed && o.AccountID == accountID && o.Mode >= UserSelChar {
			return o
		}
	}
	return nil
}

// SaveCargoThen persiste o cargo da conta SEM despejá-lo (a sessão da conta
// continua, ex.: voltando à seleção de personagem) e roda then de volta no
// loop depois do save commitar. Esta é a fronteira anti-duplicação na troca de
// personagem: depósitos/saques movem itens entre o carry do personagem e o
// cargo compartilhado, então o cargo precisa ser persistido junto do save do personagem —
// senão um item sacado para o carry fica salvo na linha do personagem enquanto
// a linha velha de account_cargo ainda o guarda, duplicando-o na próxima carga.
// then sempre roda, mesmo quando não há cargo a salvar. Só no loop.
func (w *World) SaveCargoThen(s *Session, then func(*World, *Session)) {
	if s == nil || s.AccountID == 0 || w.cargo[s.AccountID] == nil {
		then(w, s)
		return
	}
	cs := w.cargoSave(s.AccountID)
	p := w.persist
	w.Go(s, func() func(*World, *Session) {
		err := p.SaveCargo(context.Background(), cs)
		return func(w *World, s *Session) {
			if err != nil {
				w.log.Warn("save cargo failed", "account", cs.AccountID, "err", err)
			}
			then(w, s)
		}
	})
}

// outWarnStepBytes espaça os avisos de "fila alta": um por novo degrau de 8 KB de
// pico acima da metade do limite, então uma rajada deixa um rastro limitado antes de qualquer derrubada.
const outWarnStepBytes = 8 * 1024

// send enfileira uma mensagem de saída para a goroutine de escrita da sessão. Nunca
// bloqueia o loop: se os bytes não enviados da sessão passarem do limite (um
// cliente lento/travado), a sessão é derrubada em vez de travar o mundo
// inteiro (segurança contra bloqueio de cabeça de fila). Uma sessão já fechada (ex.: derrubada antes
// no mesmo tick) é ignorada, então um overflow a derruba uma vez.
func (w *World) enqueue(s *Session, h protocol.Header, payload []byte) {
	if s.closed {
		return
	}
	if s.out == nil { // sessões de teste sem conexão; o connectEvent sempre define
		s.out = newOutQueue()
	}
	h.ClientTick = w.cfg.Now()
	s.noteSend(h, len(payload))
	if w.cfg.LogSends {
		w.log.Info("send packet", "conn", s.Conn, "type", formatSendType(h.Type), "id", h.ID, "len", len(payload))
	}
	frames, bytes, ok := s.out.push(outFrame{header: h, payload: payload}, w.cfg.OutBufferBytes)
	if !ok {
		w.log.Warn("session out queue full; dropping connection",
			"conn", s.Conn, "frames", frames, "bytes", bytes, "cap_bytes", w.cfg.OutBufferBytes)
		w.dropSession(s)
		return
	}
	// Acompanha a contrapressão do escritor: uma fila crescendo significa que o socket (ou o
	// cliente) não está drenando o que o loop produz. Avisa por novo degrau de
	// pico acima de metade da capacidade — limitado, e precede a derrubada dura acima.
	if frames > s.outHighWater {
		s.outHighWater = frames
	}
	if bytes > s.outHighWaterBytes {
		s.outHighWaterBytes = bytes
		if step := bytes / outWarnStepBytes; bytes >= w.cfg.OutBufferBytes/2 && step > s.outWarnStep {
			s.outWarnStep = step
			w.log.Warn("session out queue high", "conn", s.Conn, "frames", frames, "bytes", bytes, "cap_bytes", w.cfg.OutBufferBytes)
		}
	}
}

// SessionMode devolve o modo de uma sessão (auxiliar de teste/observabilidade; só no loop).
func (w *World) SessionMode(conn int) (Mode, bool) {
	if conn < 0 || conn >= MaxUser || w.sessions[conn] == nil {
		return UserEmpty, false
	}
	return w.sessions[conn].Mode, true
}

// ActiveSessions conta as sessões não nulas (auxiliar do loop).
func (w *World) ActiveSessions() int {
	n := 0
	for _, s := range w.sessions {
		if s != nil {
			n++
		}
	}
	return n
}
