package world

import (
	"openwyd-nextgen/tmserver/internal/protocol"
	"openwyd-nextgen/tmserver/internal/rng"
)

// Este arquivo é a API que o pacote handler usa para agir no mundo.
// Todo método aqui DEVE ser chamado da goroutine do loop (ou seja, de dentro de um
// Handler ou de um callback de Go) — nunca de uma goroutine de conexão — para o estado
// do mundo ficar de dono único.

// ViewRange é o raio de broadcast (em células do grid) usado no multicast de visão:
// o GridMulticast varre a janela [Target−HALFGRIDX, Target+HALFGRIDX] por eixo
// (VIEWGRIDX=33, HALFGRIDX=16 — Basedef.h:155-158, SendFunc.cpp GridMulticast),
// ou seja, um raio Chebyshev de 16.
const ViewRange = 16

// NoViewRange é o raio mais folgado que o GetInView usa (±VIEWGRIDX, GetFunc.cpp:764)
// quando o cliente pede para ressincronizar uma entidade via _MSG_NoViewMob.
const NoViewRange = 33

// efGrade0 é o tipo de efeito de item EF_GRADE0. No Equip[0] de um NPC de quest ele carrega
// o subtipo de NPC usado para despachar o _MSG_Quest (efeitos de item do Basedef.h).
const efGrade0 = 100

// efRange é o tipo de efeito de item EF_RANGE (ItemEffect.h:83): o alcance de ataque que um
// item equipado concede a quem o usa.
const efRange = 27

// Send enfileira uma mensagem S→C "sobre você": o HEADER.ID é o da própria
// conn da sessão e o ClientTick é preenchido.
func (w *World) Send(s *Session, t protocol.Type, payload []byte) {
	w.enqueue(s, protocol.Header{Type: t, ID: uint16(s.Conn)}, payload)
}

// SendTo enfileira uma mensagem S→C com cabeçalho controlado por quem chama (notadamente
// HEADER.ID = a entidade de ORIGEM nos broadcasts, ex.: quem se moveu). O ClientTick é
// preenchido.
func (w *World) SendTo(s *Session, h protocol.Header, payload []byte) {
	w.enqueue(s, h, payload)
}

// BroadcastInView envia (t, payload) a todo outro jogador em jogo dentro do
// ViewRange da entidade de origem. O HEADER.ID é o id de origem, para os destinatários saberem
// de qual entidade a mensagem trata. A própria origem é excluída.
//
// A leva 2 varre as sessões em jogo (≤ MaxUser) e filtra por distância Chebyshev;
// uma varredura por células do grid é otimização futura.
func (w *World) BroadcastInView(srcID int, t protocol.Type, payload []byte) {
	src := w.Entity(srcID)
	if src == nil {
		return
	}
	for _, s := range w.sessions {
		if s == nil || s.Conn == srcID || s.Mode != UserPlay {
			continue
		}
		e := w.entities[s.Conn]
		if e == nil || e.Mode != MobUser {
			continue
		}
		if chebyshev(src.X, src.Y, e.X, e.Y) <= ViewRange {
			w.enqueue(s, protocol.Header{Type: t, ID: uint16(srcID)}, payload)
		}
	}
}

// MobSpawn descreve uma instância de mob a criar: o template cru, onde ele
// aparece, e sua rota de patrulha (waypoints da instância, já randomizados por quem
// chama — o GenerateMob randomiza SegmentList[i]±SegmentRange[i] por mob,
// Server.cpp:3536-3546). Rota zerada = guarda parado em (X,Y).
type MobSpawn struct {
	Template   []byte
	X, Y       int16
	RouteType  uint8
	SegX, SegY [5]int16
	SegWait    [5]int16
	GenIndex   int16 // índice do bloco NPCGener (-1 = nenhum; contabilidade de respawn, M5)
}

// SpawnMob cria um NPC/monstro parado a partir de um template STRUCT_MOB cru em
// (x,y) — o atalho sem rota para SpawnMobAt.
func (w *World) SpawnMob(template []byte, x, y int16) int {
	return w.SpawnMobAt(MobSpawn{Template: template, X: x, Y: y, GenIndex: -1})
}

// SpawnMobAt cria uma entidade NPC/monstro conforme o MobSpawn e a insere no
// grid. Devolve o novo id de mob (>= MaxUser) ou -1 quando o mundo está cheio.
// Ele só toca o estado do mundo, então é seguro no loop: é chamado tanto durante
// a inicialização (antes do Serve) quanto em runtime pelo SpawnDueRespawns (dentro do loop,
// via o tick).
func (w *World) SpawnMobAt(sp MobSpawn) int {
	template, x, y := sp.Template, sp.X, sp.Y
	id := -1
	for i := MaxUser; i < MaxMob; i++ {
		if w.entities[i] == nil {
			id = i
			break
		}
	}
	if id < 0 {
		return -1
	}
	b := protocol.ParseMobBasics(template)
	e := &Entity{
		ID: id, Mode: MobIdle, Name: b.Name, Clan: b.Clan, Class: b.Class, Merchant: b.Merchant,
		AttackRun: b.AttackRun,
		X:         x, Y: y, SpawnX: x, SpawnY: y, Level: b.Level, AC: b.Ac, Damage: b.Damage, Exp: b.Exp,
		// O ouro do template alimenta o drop de ouro do abate (loot.GoldDrop, MOB.Coin).
		Coin:  b.Coin,
		MaxHP: b.MaxHp, HP: b.Hp, Str: b.Str, Int: b.Int, Dex: b.Dex, Con: b.Con,
		// O BaseScore de um mob também precisa ser semeado, não só o CurrentScore acima:
		// handler.refreshScore roda também em mobs (qualquer skill que aplique um affect
		// num deles recalcula o score) e reconstrói o CurrentScore como BaseScore +
		// equipamento — com BaseScore zero isso apaga o AC, o dano, os atributos e o
		// MaxHP do template, e o clamp de MaxHP então derruba o HP do mob para 0.
		// O score do template É a base sem equipamento: o EDITAPPMOB grava
		// CurrentScore = BaseScore ao salvar (veja internal/domain.MobTemplateStat), e a
		// BASE_GetCurrentScore soma o equipamento por cima em runtime (CMob.cpp:709).
		BaseAC: b.Ac, BaseDamage: b.Damage, BaseMaxHP: b.MaxHp,
		BaseStr: b.Str, BaseInt: b.Int, BaseDex: b.Dex, BaseCon: b.Con,
		Template:  template, // mantido para o respawn em runtime (world/respawn.go)
		RouteType: sp.RouteType, SegListX: sp.SegX, SegListY: sp.SegY, SegWait: sp.SegWait,
		GenIndex: sp.GenIndex,
		// O waypoint atual serve também de âncora de aggro/leash (CMob.cpp:292);
		// ele começa no waypoint 0 = o ponto de spawn (GenerateMob Server.cpp:3649).
		// A pausa inicial no waypoint já vem armada (WaitSec = SegmentWait[0],
		// Server.cpp:3665), então uma patrulha descansa em casa antes do primeiro trecho.
		SegmentX: x, SegmentY: y,
		WaitTicks: sp.SegWait[0],
	}
	e.NonCombatNPC = nonCombatNPC(e.Merchant, e.Clan, e.X, e.Y)
	for i, r := range b.Resist {
		e.Resist[i] = int16(r)
	}
	eq := protocol.MobEquip(template)
	for i := range eq {
		e.EquipVisual[i], e.EquipAnct[i] = protocol.VisualEquip(eq[i], i)
	}
	// Alcance de ataque = BASE_GetMobAbility(EF_RANGE) (Basedef.cpp:2415): o MÁXIMO sobre
	// os equips do EF_RANGE de cada item — efeito base do catálogo mais os efeitos
	// da instância no STRUCT_ITEM do template (BASE_GetItemAbility soma os dois;
	// EF_RANGE é isento do multiplicador de refino, Basedef.cpp:1854). O caso
	// especial do HT (Equip[0]/10==3 + bit 20 de LearnedSkill → mín 2) é só de jogador
	// e fica de fora aqui.
	for i := range eq {
		if eq[i].Index == 0 {
			continue
		}
		v := w.cfg.ItemRanges[int(eq[i].Index)]
		for _, ef := range eq[i].Eff {
			if ef[0] == efRange {
				v += int16(ef[1])
			}
		}
		if v > e.Range {
			e.Range = v
		}
	}
	// O subtipo de NPC de quest (Merchant==100) é o EF_GRADE0 (efeito 100) do
	// Equip[0] do NPC — ex.: graus 7/8/9 do Perzen (handlers/_MSG_Quest-npcs.md).
	for _, ef := range eq[0].Eff {
		if ef[0] == efGrade0 {
			e.Grade = ef[1]
		}
	}
	// Num NPC mercador, Carry[] é o estoque da loja (enviado no MSG_ShopList).
	carry := protocol.MobCarry(template)
	for i := range carry {
		e.Carry[i] = Item{
			Index: int16(carry[i].Index),
			Effects: [3]Effect{
				{Effect: carry[i].Eff[0][0], Value: carry[i].Eff[0][1]},
				{Effect: carry[i].Eff[1][0], Value: carry[i].Eff[1][1]},
				{Effect: carry[i].Eff[2][0], Value: carry[i].Eff[2][1]},
			},
		}
	}
	// Handicap de newbie no spawn, aplicado por último para enxergar o HP final — o
	// legado faz igual, logo após CurrentScore.Hp = MaxHp (Server.cpp:3326).
	w.applyNewbieHandicap(e)
	w.entities[id] = e
	w.grid.SetMob(int(x), int(y), uint16(id))
	w.mobCount++
	// Contabilidade de população: todo mob de um bloco NPCGener conta para o seu
	// CurrentNumMob (o GenerateMob incrementa por líder E por seguidor,
	// Server.cpp:3660/3810) — centralizado aqui para os respawns da fila contarem também.
	if g := w.GeneratorAt(int(sp.GenIndex)); g != nil {
		g.CurrentNumMob++
	}
	return id
}

// DespawnMob remove um mob/NPC do mundo depois que ele morre (ou sai de outro modo):
// avisa os jogadores na visão para descartar a entidade (MSG_RemoveMob; removeType 1 = morte,
// 0 = fora de visão), limpa sua célula do grid e libera o slot da entidade. Só no loop.
//
// O broadcast roda ANTES de o slot ser liberado, para a varredura de visão ainda ler
// a posição do mob. É sem efeito para ids de jogador e slots vazios. O despawn de jogador
// (logout) tem caminho próprio em removeSession; este vale para a faixa
// [MaxUser,MaxMob). Um monstro abatido (removeType 1) é agendado para respawn antes de seu slot
// ser liberado; como o slot liberado pode ser reusado por um spawn posterior, o id também é
// limpo do conjunto `seen` de toda sessão para o id reusado disparar um
// CreateMob novo.
func (w *World) DespawnMob(id int, removeType int32) {
	e := w.Entity(id)
	if e == nil || IsPlayer(id) {
		return
	}
	body := protocol.EncodeRemoveMobBody(removeType)
	w.ForEachInView(id, func(vs *Session, _ *Entity) {
		w.enqueue(vs, protocol.Header{Type: protocol.MsgRemoveMob, ID: uint16(id)}, body)
	})
	// Contabilidade de população do gerador na morte (o DeleteMob decrementa o
	// CurrentNumMob, Server.cpp:7825-7831, limitado a 0).
	gen := w.GeneratorAt(int(e.GenIndex))
	if removeType == 1 && e.Merchant == 0 && !e.NonCombatNPC && gen != nil {
		if gen.CurrentNumMob--; gen.CurrentNumMob < 0 {
			gen.CurrentNumMob = 0
		}
	}
	// Vínculos de grupo: um seguidor que morre sai da PartyList do líder; um líder
	// que morre libera os seguidores (eles ficam autossuficientes — Leader=0
	// religa o aggro próprio). Evita que um slot reusado herde um
	// grupo velho.
	if e.Leader != 0 {
		if le := w.Entity(e.Leader); le != nil {
			for i, m := range le.PartyList {
				if m == id {
					le.PartyList[i] = 0
				}
			}
		}
	}
	for _, m := range e.PartyList {
		if m >= MaxUser {
			if fe := w.Entity(m); fe != nil && fe.Leader == id {
				fe.Leader = 0
			}
		}
	}
	// Um monstro abatido respawna no seu ponto de spawn após um atraso, mantendo sua
	// rota da instância (waypoints/RouteType) para a patrulha voltar a patrulhar —
	// A NÃO SER que seu gerador se refaça no timer de minutos (MinuteGenerate>0):
	// ali o timer repõe grupos inteiros e a fila duplicaria o spawn.
	// A fila de 15s para blocos com MinuteGenerate<=0 é nossa divergência deliberada
	// (o original nunca regenera esses fora de eventos). NPCs (lojas/dadores de
	// quest) não morrem, então só agendamos criaturas (Merchant==0) abatidas em
	// combate (removeType 1). Pets invocados nunca respawnam — eles também carregam um
	// Template, mas o ciclo de vida deles pertence ao invocador.
	if removeType == 1 && e.Merchant == 0 && !e.NonCombatNPC && e.Template != nil && e.Summoner == 0 &&
		!IsKefraGenerator(int(e.GenIndex)) && !IsWaterGenerator(int(e.GenIndex)) && NightmareGenerator(int(e.GenIndex)) < 0 && !IsRuneQuestGenerator(int(e.GenIndex)) &&
		(gen == nil || gen.MinuteGenerate <= 0) {
		w.respawnQueue = append(w.respawnQueue, respawnEntry{
			spawn: MobSpawn{
				Template: e.Template, X: e.SpawnX, Y: e.SpawnY,
				RouteType: e.RouteType, SegX: e.SegListX, SegY: e.SegListY,
				SegWait: e.SegWait, GenIndex: e.GenIndex,
			},
			due: w.Now() + DefaultRespawnDelay,
		})
	}
	if cur, ok := w.grid.MobAt(int(e.X), int(e.Y)); ok && int(cur) == id {
		w.grid.ClearMob(int(e.X), int(e.Y))
	}
	w.entities[id] = nil
	if w.mobCount--; w.mobCount < 0 {
		w.mobCount = 0
	}
	w.clearSeenAll(id)
}

// ClearSeen zera o conjunto de visão de uma sessão (ex.: ao entrar no mundo), para todas as
// entidades na visão receberem um CreateMob novo.
func (w *World) ClearSeen(s *Session) { s.seen = nil }

// MarkSeen registra que o cliente da sessão s agora conhece a entidade id; devolve true
// só na primeira vez (para um CreateMob ser enviado uma vez por entidade ao entrar na visão).
func (w *World) MarkSeen(s *Session, id int) bool {
	if s.seen == nil {
		s.seen = make(map[int]struct{})
	}
	if _, ok := s.seen[id]; ok {
		return false
	}
	s.seen[id] = struct{}{}
	return true
}

// UnmarkSeen esquece que o cliente da sessão s conhece a entidade id, para um
// MarkSeen posterior enviar um CreateMob novo. Precisa acompanhar todo RemoveMob, senão uma
// entidade que volta a entrar na visão nunca é recriada.
func (w *World) UnmarkSeen(s *Session, id int) {
	delete(s.seen, id)
}

// ForEachMobInView chama fn para cada entidade de mob (id >= MaxUser) numa célula do grid
// dentro do ViewRange do jogador playerID. Usado para spawnar NPCs próximos no cliente.
func (w *World) ForEachMobInView(playerID int, fn func(e *Entity)) {
	src := w.Entity(playerID)
	if src == nil {
		return
	}
	w.ForEachMobInViewAt(src.X, src.Y, fn)
}

// ForEachMobInViewAt é o ForEachMobInView centrado num ponto qualquer — usado
// para a janela de visão ANTIGA quando um jogador se move (varredura da janela antiga do GridMulticast).
func (w *World) ForEachMobInViewAt(x, y int16, fn func(e *Entity)) {
	for dy := -ViewRange; dy <= ViewRange; dy++ {
		for dx := -ViewRange; dx <= ViewRange; dx++ {
			id, ok := w.grid.MobAt(int(x)+dx, int(y)+dy)
			if !ok || int(id) < MaxUser {
				continue
			}
			if e := w.entities[id]; e != nil && e.Mode != MobEmpty {
				fn(e)
			}
		}
	}
}

// ForEachInView chama fn para todo outro jogador em jogo cuja entidade está dentro
// do ViewRange de srcID (usado para ligar a visibilidade de criação/remoção de entidade). Só no loop.
func (w *World) ForEachInView(srcID int, fn func(s *Session, e *Entity)) {
	src := w.Entity(srcID)
	if src == nil {
		return
	}
	w.ForEachInViewAt(src.X, src.Y, srcID, fn)
}

// ForEachInViewAt é o ForEachInView centrado num ponto qualquer — necessário para
// a janela de visão ANTIGA de uma entidade que já se moveu (o GridMulticast varre
// as janelas da posição antiga e da nova). exclude é ignorado. Só no loop.
func (w *World) ForEachInViewAt(x, y int16, exclude int, fn func(s *Session, e *Entity)) {
	w.ForEachPlaying(exclude, func(s *Session, e *Entity) {
		if chebyshev(x, y, e.X, e.Y) <= ViewRange {
			fn(s, e)
		}
	})
}

// ForEachPlaying chama fn para toda sessão de jogador em jogo exceto exclude —
// sem filtro de distância. O multicast de movimento precisa dele para classificar cada observador
// contra AMBAS as janelas de visão, antiga e nova, numa passada. Só no loop.
func (w *World) ForEachPlaying(exclude int, fn func(s *Session, e *Entity)) {
	for _, s := range w.sessions {
		if s == nil || s.Conn == exclude || s.Mode != UserPlay {
			continue
		}
		e := w.entities[s.Conn]
		if e == nil || e.Mode != MobUser {
			continue
		}
		fn(s, e)
	}
}

func chebyshev(x1, y1, x2, y2 int16) int {
	dx := int(x1) - int(x2)
	if dx < 0 {
		dx = -dx
	}
	dy := int(y1) - int(y2)
	if dy < 0 {
		dy = -dy
	}
	if dx > dy {
		return dx
	}
	return dy
}

// SetEntityPos move a entidade id para (x,y) e mantém o grid espacial em sincronia
// (limpa a célula antiga se ela ainda apontava para esta entidade, define a nova).
func (w *World) SetEntityPos(id int, x, y int16) {
	e := w.Entity(id)
	if e == nil {
		return
	}
	if cur, ok := w.grid.MobAt(int(e.X), int(e.Y)); ok && int(cur) == id {
		w.grid.ClearMob(int(e.X), int(e.Y))
	}
	e.X, e.Y = x, y
	w.grid.SetMob(int(x), int(y), uint16(id))
}

// EmptyCellNear devolve uma célula livre do grid em ou perto de (x,y), usando a mesma
// varredura em anéis do substituto legado GetEmptyMobGrid usado nos spawns de mob. Só no loop.
func (w *World) EmptyCellNear(x, y int16) (int16, int16, bool) {
	return w.emptyCellNear(x, y)
}

// GridDim devolve o lado do grid do mundo (limite válido de coordenada).
func (w *World) GridDim() int { return w.grid.dim }

// AddCrackError registra uma violação anti-cheat contra uma sessão (CUser.NumError
// / AddCrackError). Passado um limiar, a sessão é derrubada.
//
// UNVERIFIED: o limiar exato e a semântica por grupo não estão documentados;
// CrackErrorLimit é um valor provisório.
func (w *World) AddCrackError(s *Session, group, code int) {
	s.CrackError++
	w.log.Warn("crack error", "conn", s.Conn, "group", group, "code", code, "total", s.CrackError)
	if s.CrackError >= CrackErrorLimit {
		w.removeSession(s)
	}
}

// CrackErrorLimit é a contagem de crack-error em que uma sessão é derrubada.
const CrackErrorLimit = 10

// Close desmonta uma sessão (ex.: depois de uma falha fatal de validação).
func (w *World) Close(s *Session) { w.removeSession(s) }

// Session devolve a sessão em conn ∈ [0, MaxUser), ou nil. Só no loop.
func (w *World) Session(conn int) *Session {
	if conn < 0 || conn >= MaxUser {
		return nil
	}
	return w.sessions[conn]
}

// SessionByName acha um jogador em jogo pelo nome (exato) de personagem, devolvendo a
// sessão e a entidade, ou nils. Só no loop; usado para mirar sussurro/duelo.
func (w *World) SessionByName(name string) (*Session, *Entity) {
	for _, s := range w.sessions {
		if s == nil || s.Mode != UserPlay {
			continue
		}
		if e := w.entities[s.Conn]; e != nil && e.Name == name {
			return s, e
		}
	}
	return nil, nil
}

// Entity devolve a entidade do mundo no índice id, ou nil. Jogadores são id < MaxUser.
func (w *World) Entity(id int) *Entity {
	if id < 0 || id >= MaxMob {
		return nil
	}
	return w.entities[id]
}

// Grid expõe o índice espacial (só no loop).
func (w *World) Grid() *Grid { return w.grid }

// Persistence devolve o backend configurado para os handlers poderem fazer chamadas de banco
// (sempre via Go, nunca inline no loop).
func (w *World) Persistence() Persistence { return w.persist }

// Billing devolve o gate de billing configurado (o adaptador do binServer, ou o
// default AllowAllBilling). Usado pelo gate de login de personagem via Go.
func (w *World) Billing() Billing { return w.billing }

// SetBilling instala o gate de billing. Precisa ser chamado na fiação, antes de
// Serve/Run começar, já que o World passa a ser dono do estado single-threaded dali em diante. Um
// gate nil é ignorado (o default AllowAllBilling permanece).
func (w *World) SetBilling(b Billing) {
	if b != nil {
		w.billing = b
	}
}

// Now devolve o tick atual do servidor (relógio injetável).
func (w *World) Now() uint32 { return w.cfg.Now() }

// Rand devolve o RNG do MSVC do loop. Só é seguro usar da goroutine do
// loop (handlers, callbacks de Go); reproduz o fluxo rand() original
// para paridade (parity-tests.md §4.0).
func (w *World) Rand() *rng.MSVC { return w.rng }

// Go roda trabalho bloqueante (tipicamente uma chamada gRPC ao dbServer) fora do loop, em sua própria
// goroutine, e então reentra no loop para aplicar o callback devolvido. O
// callback é ignorado se o slot da sessão foi liberado ou reusado nesse meio-tempo, então
// os handlers podem assumir que s ainda é válido. Um callback nil não faz nada.
func (w *World) Go(s *Session, work func() func(*World, *Session)) {
	go func() {
		cb := work()
		if cb == nil {
			return
		}
		ce := callbackEvent{conn: s.Conn, sess: s, cb: cb}
		select {
		case w.callbacks <- ce:
		case <-w.done:
		}
	}()
}

// GoDetached roda trabalho bloqueante fora do loop (como Go) mas reentra com um
// callback de escopo do mundo — para manutenção de fundo não ligada a nenhuma sessão,
// como consultar periodicamente a config de NPC. O callback roda na goroutine do loop e
// pode mutar o estado do mundo direto. Um callback nil não faz nada. Ele pega carona
// na fila de callbacks para um tick lento da IA de mob nunca atrasá-lo.
func (w *World) GoDetached(work func() func(*World)) {
	go func() {
		cb := work()
		if cb == nil {
			return
		}
		select {
		case w.callbacks <- worldCallbackEvent{cb: cb}:
		case <-w.done:
		}
	}()
}
