package handler

import (
	"strings"

	"openwyd-nextgen/tmserver/internal/level"
	"openwyd-nextgen/tmserver/internal/protocol"
	"openwyd-nextgen/tmserver/internal/route"
	"openwyd-nextgen/tmserver/internal/world"
)

// Evocações do BM (spell InstanceType 11 → GenerateSummon, Server.cpp:2956).
// Uma invocação é uma criatura em slot de NPC presa ao dono: Clan 4 (o clã "mascote",
// amistoso a jogadores), Summoner = o conn do dono, Leader = o líder de grupo
// do dono, RouteType 5. Ela segue o dono (summonTick), entra nas lutas
// dele (commandSummons), e morre quando seu affect de vida Type-24 acaba
// ou o dono sai.

// summonClan marca um mascote invocado (pMob.MOB.Clan = 4, Server.cpp:3168).
const summonClan = 4

// affectSummonLife é o affect de vida da invocação (Type 24 num MOB — num
// jogador o mesmo tipo é Samaritano, affectSamaritano em combat.go; o legado
// sobrecarrega e desambigua por idx >= MAX_USER, Server.cpp:5843).
const affectSummonLife = 24

// summonLifeTicks é a vida de invocação não-item: 20 affect ticks de 8s
// (Server.cpp:3193-3196). As montarias "permanentes" 28..37 estão fora de escopo.
const summonLifeTicks = 20

// Montarias bebês (crias, item 2330..2359) são invocações sustentadas por equipamento: enquanto o
// item está vivo em Equip[14], o MountProcess cria a entrada BaseSummon
// correspondente em sIndex-2320 e a mantém presa ao dono sem affect
// de vida (Server.cpp:4633-4676, ramo sItem do GenerateSummon).
const (
	babyMountLo        = 2330
	babyMountHi        = 2360
	babyMountSummonAdd = -2320
	babyMountFaceLo    = 315
	babyMountFaceHi    = 345
)

// summonFollowMax / summonTeleport / summonFollowMin moldam a IA de seguir
// (CMob.cpp:84-152): acima de 12 o mascote teleporta para o dono, entre 5 e 12 ele
// anda, a 4 ou menos ele fica parado. summonLeash larga a luta do mascote quando o
// dono se afasta (CMob.cpp:268-274).
const (
	summonTeleport  = 13
	summonFollowMin = 4
	summonLeash     = 20
)

// summonBonus é pSummonBonus[0..8] (Basedef.cpp:745-756), os percentuais
// escalados pelo dono das 8 evocações do BM mais o template valor-9 zerado usado pela
// Invocação Final: cada stat ganha Int*int/100 + Evo*evo/100 em cima do
// template BaseSummon (Server.cpp:3073-3086; Evo = Evocação = Special[2]).
var summonBonus = [9]struct {
	damInt, damEvo int32
	acInt, acEvo   int32
	hpInt, hpEvo   int32
}{
	{80, 300, 50, 75, 100, 400},   // 0 Condor
	{80, 250, 50, 150, 125, 400},  // 1 Javali
	{80, 400, 50, 125, 125, 400},  // 2 Lobo
	{80, 350, 50, 200, 150, 400},  // 3 Urso
	{80, 500, 50, 175, 150, 400},  // 4 Tigre
	{80, 450, 50, 250, 175, 400},  // 5 Gorila
	{100, 500, 50, 250, 174, 400}, // 6 Dragão Negro
	{130, 250, 60, 200, 180, 250}, // 7 Succubus
	{0, 0, 0, 0, 0, 0},            // 8 Porco/Invocação Final: sem escala pelo dono
}

// summonCount é a regra de quantidade da evocação (_MSG_Attack.cpp:817-828):
// Evocação (Special[2]) compra mais das criaturas baratas; a Succubus (iv 8)
// é sempre exatamente uma.
func summonCount(instanceValue, evocacao int) int {
	switch instanceValue {
	case 1, 2:
		return evocacao / 30
	case 3, 4, 5:
		return evocacao / 40
	case 6, 7:
		return evocacao / 80
	case 8, 9:
		return 1
	}
	return 0
}

// freeCellNear varre os anéis em volta de (x,y) por uma célula desocupada e caminhável,
// casando com a ordem de expansão quadrada 1..4 do GetEmptyMobGrid. Deliberadamente
// não consome RNG para a paridade de chamadas de rand do fluxo de ataque ser preservada.
func (d *Dispatcher) freeCellNear(w *world.World, x, y int16) (int16, int16, bool) {
	for ring := 1; ring <= 4; ring++ {
		for dy := -ring; dy <= ring; dy++ {
			for dx := -ring; dx <= ring; dx++ {
				if abs16(int16(dx)) != ring && abs16(int16(dy)) != ring {
					continue // célula interior: já varrida num anel menor
				}
				nx, ny := x+int16(dx), y+int16(dy)
				if d.cellAvailable(w, nx, ny) {
					return nx, ny, true
				}
			}
		}
	}
	return 0, 0, false
}

func (d *Dispatcher) freeCellAtOrNear(w *world.World, x, y int16) (int16, int16, bool) {
	if d.cellAvailable(w, x, y) {
		return x, y, true
	}
	return d.freeCellNear(w, x, y)
}

func (d *Dispatcher) cellAvailable(w *world.World, x, y int16) bool {
	if x < 0 || y < 0 || int(x) >= w.GridDim() || int(y) >= w.GridDim() {
		return false
	}
	if _, occupied := w.EntityAt(x, y); occupied {
		return false
	}
	// A grade embutida marca células intransponíveis com route.Blocked (127); checagens
	// finas de inclinação ficam com o caminhante (route.Next).
	return d.heights == nil || d.heights.At(int(x), int(y)) != route.Blocked
}

// generateSummon porta o GenerateSummon (Server.cpp:2956) para as evocações
// (sem o caminho sItem/montaria): gera summonID cópias do template em volta do
// conjurador, presas a ele. Devolve false quando NADA foi gerado (a conjuração então
// devolve sua mana, _MSG_Attack.cpp:830-834). Sucesso parcial é true.
//
// O caminho de recall de montaria sustentada por item usa refreshBabyMountSummon abaixo; esta
// função fica no ramo de spell/sItem==nil.
func (d *Dispatcher) generateSummon(w *world.World, s *world.Session, e *world.Entity, summonID, count int) bool {
	if count <= 0 || summonID < 0 || summonID >= len(d.summonMobs) || summonID >= len(summonBonus) || d.summonMobs[summonID] == nil {
		return false
	}
	leaderID := e.Leader
	if leaderID == 0 {
		leaderID = s.Conn
	}
	le := w.Entity(leaderID)
	if le == nil {
		return false
	}
	face := summonTemplateFace(d.summonMobs[summonID])
	existing := 0
	for _, memberID := range le.PartyList {
		if memberID < world.MaxUser {
			continue
		}
		pet := w.Entity(memberID)
		if pet == nil || pet.Clan != summonClan {
			continue
		}
		if pet.EquipVisual[0] != face {
			return false
		}
		existing++
	}
	if existing >= count {
		return false
	}
	bonus := summonBonus[summonID]
	spawned := 0
	for i := existing; i < count; i++ {
		slot := -1
		for j := range le.PartyList {
			if le.PartyList[j] == 0 {
				slot = j
				break
			}
		}
		if slot < 0 {
			break // grupo cheio (_NN_Party_Full_Cant_Summon; mensagem UNVERIFIED — ignorada)
		}
		x, y, ok := d.freeCellNear(w, e.X, e.Y)
		if !ok {
			break
		}
		id := w.SpawnMobAt(world.MobSpawn{Template: d.summonMobs[summonID], X: x, Y: y, RouteType: 5, GenIndex: -1})
		if id < 0 {
			break // mundo cheio (_NN_Cant_Create_More_Summons — ignorado)
		}
		mob := w.Entity(id)
		// O sufixo '^' marca o nome como mascote no cliente; sublinhados nos
		// nomes dos arquivos de template aparecem como espaços (Server.cpp:3063-3070).
		mob.Name = strings.ReplaceAll(mob.Name, "_", " ") + "^"
		mob.Level = e.Level
		if mob.Level > level.MaxLevel {
			mob.Level = level.MaxLevel
		}
		ownerInt := int32(e.Int)
		evo := int32(effectiveSpecial(e, 2))
		mob.Damage += ownerInt*bonus.damInt/100 + evo*bonus.damEvo/100
		mob.AC += ownerInt*bonus.acInt/100 + evo*bonus.acEvo/100
		mob.MaxHP += ownerInt*bonus.hpInt/100 + evo*bonus.hpEvo/100
		mob.HP = mob.MaxHP
		mob.Clan = summonClan
		mob.Leader = leaderID
		mob.Summoner = s.Conn
		mob.Affect[0] = world.Affect{Type: affectSummonLife, Time: summonLifeTicks}
		le.PartyList[slot] = id
		if spawned == 0 {
			d.sendAddParty(w, leaderID, leaderID, 0)
		}
		d.sendSummonPartySlot(w, leaderID, id, slot+1)

		// Revelar com CreateType|=3 (o efeito de aparecimento da invocação, Server.cpp:3210-3218).
		body := protocol.EncodeCreateMobBody(d.createMobFrom(mob, 3))
		w.ForEachInView(id, func(vs *world.Session, _ *world.Entity) {
			if w.MarkSeen(vs, id) {
				w.SendTo(vs, protocol.Header{Type: protocol.MsgCreateMob, ID: protocol.IDScene}, body)
			}
		})
		spawned++
	}
	return spawned > 0
}

func (d *Dispatcher) refreshBabyMountSummon(w *world.World, s *world.Session, e *world.Entity) {
	d.removeBabyMountSummons(w, s.Conn, e)

	mount := e.Equip[mountEquipSlot]
	if !isBabyMount(mount) || mountHP(mount) <= 0 {
		return
	}
	summonID := int(mount.Index) + babyMountSummonAdd
	d.generateBabyMountSummon(w, s, e, summonID, mount)
}

func isBabyMount(it world.Item) bool {
	return it.Index >= babyMountLo && it.Index < babyMountHi
}

func mountHP(it world.Item) int16 {
	return int16(uint16(it.Effects[0].Effect) | uint16(it.Effects[0].Value)<<8)
}

func (d *Dispatcher) removeBabyMountSummons(w *world.World, ownerID int, owner *world.Entity) {
	leaderID := owner.Leader
	if leaderID == 0 {
		leaderID = ownerID
	}
	d.despawnSummons(w, leaderID, func(pet *world.Entity) bool {
		return pet.Summoner == ownerID &&
			pet.EquipVisual[0] >= babyMountFaceLo && pet.EquipVisual[0] < babyMountFaceHi
	})
}

// despawnSummonsOf remove toda invocação parada no PartyList de leaderID que
// pertence a ownerID; ownerID == 0 significa todas (o grupo se desfez).
// É a varredura DeleteMob dentro do RemoveParty (Server.cpp:8185-8190 quando um
// membro sai, Server.cpp:8237-8243 quando o líder sai) — sem ela os
// mascotes sobrevivem ao vínculo de grupo que os prendia (issue #234).
func (d *Dispatcher) despawnSummonsOf(w *world.World, leaderID, ownerID int) {
	d.despawnSummons(w, leaderID, func(pet *world.Entity) bool {
		// Summoner 0 é uma criatura avulsa parada no slot de grupo de um jogador; o legado
		// recolhe essas em qualquer saída também (Server.cpp:8188-8190).
		return ownerID == 0 || pet.Summoner == ownerID || pet.Summoner == 0
	})
}

// despawnSummons é a caminhada compartilhada dos slots de mascote de um líder: match seleciona quais
// mascotes vão. Os ids e os destinatários do pacote são fotografados primeiro porque
// o DespawnMob libera o slot conforme avança (world/api.go:215-223).
func (d *Dispatcher) despawnSummons(w *world.World, leaderID int, match func(*world.Entity) bool) {
	leader := w.Entity(leaderID)
	if leader == nil {
		return
	}
	var doomed []int
	recipients := []int{leaderID}
	for _, memberID := range leader.PartyList {
		if memberID <= 0 {
			continue
		}
		if world.IsPlayer(memberID) {
			recipients = append(recipients, memberID)
			continue
		}
		if pet := w.Entity(memberID); pet != nil && match(pet) {
			doomed = append(doomed, memberID)
		}
	}
	for _, id := range doomed {
		// O DespawnMob só envia MsgRemoveMob, então a linha de grupo precisa ser descartada
		// explicitamente ou o cliente mantém o slot do mascote (o legado obtém isto do
		// DeleteMob → RemoveParty, Server.cpp:7840).
		for _, recipientID := range recipients {
			d.sendRemoveParty(w, recipientID, id)
		}
		w.DespawnMob(id, 3)
	}
}

func (d *Dispatcher) generateBabyMountSummon(w *world.World, s *world.Session, e *world.Entity, summonID int, mount world.Item) bool {
	if summonID < 0 || summonID >= len(d.summonMobs) || d.summonMobs[summonID] == nil {
		return false
	}
	leaderID := e.Leader
	if leaderID == 0 {
		leaderID = s.Conn
	}
	leader := w.Entity(leaderID)
	if leader == nil {
		return false
	}
	slot := -1
	for i := range leader.PartyList {
		if leader.PartyList[i] == 0 {
			slot = i
			break
		}
	}
	if slot < 0 {
		return false
	}
	x, y, ok := d.freeCellNear(w, e.X, e.Y)
	if !ok {
		return false
	}
	id := w.SpawnMobAt(world.MobSpawn{Template: d.summonMobs[summonID], X: x, Y: y, RouteType: 5, GenIndex: -1})
	if id < 0 {
		return false
	}
	mob := w.Entity(id)
	mob.Name = strings.ReplaceAll(mob.Name, "_", " ") + "^"
	mob.Level = e.Level
	if mob.Level > level.MaxLevel {
		mob.Level = level.MaxLevel
	}
	mob.Clan = summonClan
	mob.Leader = leaderID
	mob.Summoner = s.Conn
	mountSanc := int32(mount.Effects[1].Effect)
	if mountSanc > 100 {
		mountSanc = 100
	}
	mob.Damage += 6 * mountSanc
	hp := int32(mountHP(mount))
	if mob.MaxHP > 0 && hp > mob.MaxHP {
		hp = mob.MaxHP
	}
	mob.HP = hp
	leader.PartyList[slot] = id
	d.sendAddParty(w, leaderID, leaderID, 0)
	d.sendSummonPartySlot(w, leaderID, id, slot+1)

	body := protocol.EncodeCreateMobBody(d.createMobFrom(mob, 3))
	w.ForEachInView(id, func(vs *world.Session, _ *world.Entity) {
		if w.MarkSeen(vs, id) {
			w.SendTo(vs, protocol.Header{Type: protocol.MsgCreateMob, ID: protocol.IDScene}, body)
		}
	})
	return true
}

func summonTemplateFace(template []byte) uint16 {
	eq := protocol.MobEquip(template)
	return eq[0].Index
}

func (d *Dispatcher) sendSummonPartySlot(w *world.World, leaderID, summonID, slot int) {
	sent := map[int]bool{}
	send := func(recipientID int) {
		if sent[recipientID] {
			return
		}
		sent[recipientID] = true
		d.sendAddParty(w, recipientID, summonID, slot)
	}
	send(leaderID)
	if leader := w.Entity(leaderID); leader != nil {
		for _, memberID := range leader.PartyList {
			if memberID > 0 && world.IsPlayer(memberID) {
				send(memberID)
			}
		}
	}
}

// petIsListed informa se o mascote ainda tem slot no PartyList do líder —
// o único registro que invocações têm (generateSummon, commandSummons).
func petIsListed(w *world.World, id int, e *world.Entity) bool {
	leader := w.Entity(e.Leader)
	if leader == nil {
		return false
	}
	return hasMember(leader, id)
}

// summonTick substitui a IA normal de criatura para um mascote invocado (o ramo
// RouteType-5 do StandingByProcessor, CMob.cpp:84-152): despawnar quando o dono
// sumiu, contar a vida, lutar a luta do dono dentro da contenção, senão
// seguir.
func (d *Dispatcher) summonTick(w *world.World, id int, e *world.Entity) {
	owner := w.Entity(e.Summoner)
	ownerGone := owner == nil || owner.HP <= 0
	if !ownerGone {
		if m, ok := w.SessionMode(e.Summoner); !ok || m != world.UserPlay {
			ownerGone = true // logout / volta para a seleção de personagem / desconexão
		}
	}
	if !ownerGone && !petIsListed(w, id, e) {
		// Redundância de segurança: um mascote cujo slot de grupo sumiu não tem mais vínculo de dono,
		// e nada mais iria recolhê-lo (sua vida conta aqui mesmo). O
		// legado sai do mesmo jeito quando o líder de uma invocação é limpo
		// (CMob.cpp:118-124). Impede que qualquer reset futuro de PartyList vaze mascotes.
		ownerGone = true
	}
	if ownerGone {
		// O legado limpa invocações pelos próprios timers na morte/logout
		// (ProcessSecMinTimer.cpp:2381,2498); esta checagem preguiçosa por tick cobre
		// todo caminho de saída num lugar só. O despawn por morte do dono é UNVERIFIED
		// quanto ao timing (o original pode manter o mascote até o timer).
		w.DespawnMob(id, 3)
		return
	}
	// Vida: o affect Type-24 conta regressivamente na mesma fase de 8s que a varredura
	// de jogador usa; em zero o mascote some (DeleteMob(idx,3), Server.cpp:5843).
	// Varrido aqui em vez de em sweepAffects para a varredura de 10k criaturas ficar livre de
	// trabalho de affect para tudo que não é invocação.
	if d.tickCount%affectTickPeriod == id%affectTickPeriod {
		if af := &e.Affect[0]; af.Type == affectSummonLife && af.Time < affectInfiniteTime {
			if af.Time > 0 {
				af.Time--
			}
			if af.Time == 0 {
				w.DespawnMob(id, 3)
				return
			}
		}
	}
	if e.Target != 0 {
		// Lutar, mas nunca além da contenção do dono: passando dela o
		// mascote larga a luta e volta (CMob.cpp:268-274).
		if mobDistance(e.X, e.Y, owner.X, owner.Y) >= summonLeash {
			e.Target = 0
			clearEnemyList(e)
			e.Mode = world.MobIdle
		} else {
			d.mobBattle(w, id, e)
			return
		}
	}
	dis := mobDistance(e.X, e.Y, owner.X, owner.Y)
	switch {
	case dis >= summonTeleport:
		// Teleportar para o lado do dono (o salto de recall: MSG_Action Effect=8 Speed=6,
		// Server.cpp:3008-3020) e revelar de novo para jogadores que nunca o viram.
		x, y, ok := d.freeCellNear(w, owner.X, owner.Y)
		if !ok {
			return
		}
		oldX, oldY := e.X, e.Y
		w.SetEntityPos(id, x, y)
		body := protocol.MsgActionBody{PosX: oldX, PosY: oldY, Speed: 6, TargetX: x, TargetY: y, Effect: 8}
		d.moveMulticast(w, id, oldX, oldY, protocol.MsgAction, body.Encode())
	case dis > summonFollowMin:
		d.stepToward(w, id, e, owner.X, owner.Y)
	}
}

// commandSummons aponta os mascotes ociosos do dono para target — o substituto da
// propagação de EnemyList legada que fazia mascotes ajudarem (atacar o que o dono
// ataca, defender quando o dono é atingido). Mesma caixa de engajamento de ±23 que o
// arrasto de grupo de criaturas (setGroupBattle).
func (d *Dispatcher) commandSummons(w *world.World, ownerID int, target *world.Entity) {
	owner := w.Entity(ownerID)
	if owner == nil || target == nil {
		return
	}
	leaderID := owner.Leader
	if leaderID == 0 {
		leaderID = ownerID
	}
	le := w.Entity(leaderID)
	if le == nil {
		return
	}
	for _, m := range le.PartyList {
		if m < world.MaxUser {
			continue // slot vazio ou membro jogador
		}
		pet := w.Entity(m)
		if pet == nil || pet.Summoner != ownerID || pet.HP <= 0 || pet.Target != 0 {
			continue
		}
		if abs16(pet.X-target.X) <= battleDragBox && abs16(pet.Y-target.Y) <= battleDragBox {
			setBattle(w, m, pet, target)
		}
	}
}
