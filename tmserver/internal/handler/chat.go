package handler

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"openwyd-nextgen/tmserver/internal/protocol"
	"openwyd-nextgen/tmserver/internal/world"
)

// messageChat trata _MSG_MessageChat (0x0333): chat público mais alguns comandos
// de barra (lote2-chat.md). Uma linha que não é comando é transmitida aos jogadores em visão.
//
// UNVERIFIED: a lista completa de comandos (partychat/kingdomchat/guildchat/chatting
// roteamento) não é reproduzida — só os toggles e o guildtax abaixo; todo
// o resto é tratado como fala pública. Migração recomendada: separar um command-bus
// do transporte de chat.
func (d *Dispatcher) messageChat(w *world.World, s *world.Session, _ protocol.Header, payload []byte) {
	if s.Mode != world.UserPlay {
		return
	}
	text := cstr(payload)
	switch firstToken(text) {
	case "whisper":
		s.Whisper = !s.Whisper
	case "partychat":
		// _MSG_MessageChat.cpp:117-126: alterna se este jogador recebe o
		// canal de grupo; o legado confirma com uma mensagem ao cliente.
		s.PartyChat = !s.PartyChat
		if s.PartyChat {
			d.sendChatText(w, s, "Party Chatting : Off")
		} else {
			d.sendChatText(w, s, "Party Chatting : On")
		}
	case "guildchat":
		// _MSG_MessageChat.cpp:140-150: alterna se este jogador recebe o
		// canal de guilda e confirma com SendClientMessage (um painel de mensagem). O
		// cliente casa esses textos exatos para virar seu botão de guilda
		// (TMScene.cpp:1660-1669).
		s.GuildChat = !s.GuildChat
		if s.GuildChat {
			d.sendClientMessage(w, s, "Guild Chatting : Off")
		} else {
			d.sendClientMessage(w, s, "Guild Chatting : On")
		}
	case "guildon":
		s.GuildDisable = false
	case "guildoff":
		s.GuildDisable = true
	case "guildtax":
		d.guildTax(w, s, text)
	default:
		// Fala pública → todos em visão (HEADER.ID = falante).
		w.BroadcastInView(s.Conn, protocol.MsgMessageChat, payload)
	}
}

// messageWhisper trata _MSG_MessageWhisper (0x0334): uma mensagem privada para um
// jogador online nomeado (lote2-chat.md).
//
// UNVERIFIED: as 55 palavras-chave de comando (o grosso do handler original de 1710
// linhas, incl. comandos de GM/backdoor) NÃO são tratadas aqui — MobName é tratado
// puramente como alvo de sussurro. Migrar comandos para um command-bus autorizado é
// uma tarefa deliberada e separada (e remove os backdoors documentados).
func (d *Dispatcher) messageWhisper(w *world.World, s *world.Session, _ protocol.Header, payload []byte) {
	if s.Mode != world.UserPlay {
		return
	}
	var body protocol.MsgWhisperBody
	if err := body.Decode(payload); err != nil {
		return
	}
	name := cstr(body.MobName[:])
	if name == "" && len(body.String) > 0 && body.String[0] == '-' {
		d.guildChat(w, s, body)
		return
	}
	if name == "" && len(body.String) > 0 && body.String[0] == '=' {
		d.partyChat(w, s, body)
		return
	}
	if d.runCommand(w, s, name, body.String) {
		// Investigação de freeze: comandos eram invisíveis nos logs (a linha do pacote
		// recebido só mostra 0x0334), então linhas do tempo de incidente não conseguiam distinguir um
		// comando de teleporte de um sussurro comum. Só a palavra-chave — sem o texto do sussurro.
		d.log.Info("chat command", "conn", s.Conn, "cmd", strings.TrimPrefix(name, "/"))
		return // um comando de barra (o cliente envia "/x" como sussurro para "x")
	}
	// "/r" responde a quem sussurrou por último (a região "PM, /r" de
	// _MSG_MessageWhisper.cpp, pUser[conn].LastChat). Ninguém ainda: não conectado.
	if name == whisperReply {
		name = s.LastWhisperFrom
	}
	target, _ := w.SessionByName(name)
	if target == nil {
		d.notify(w, s, NoticeNotConnected)
		return
	}
	if target.Whisper {
		d.notify(w, s, NoticeDenyWhisper)
		return
	}
	e := w.Entity(s.Conn)
	if e == nil {
		return
	}
	// O legado escreve o nome do falante em MobName antes de repassar, e
	// o cliente receptor mostra MobName como remetente. Repassar o payload
	// como enviado deixava ali o nome do próprio alvo.
	body.MobName = [16]byte{}
	copy(body.MobName[:len(body.MobName)-1], e.Name)
	body.String = privateWhisperText(body.String)
	target.LastWhisperFrom = e.Name
	w.SendTo(target, protocol.Header{Type: protocol.MsgMessageWhisper, ID: uint16(s.Conn)}, body.Encode())
}

// whisperReply é o nome de alvo que o cliente envia para "/r".
const whisperReply = "r"

// whisperTextMax é o tamanho de MSG_MessageWhisper.String do 7662 (offsets de frame
// 28..155); o receptor força String[127] = 0.
const whisperTextMax = 128

// privateWhisperText reescreve o String de um sussurro privado para o cliente 7662,
// mantendo o comprimento recebido (158 do dialeto web, 160 do WYD.exe):
// - o memo do receptor imprime &String[1] (TMFieldScene.cpp:17945), também
// com o servidor legado, que repassa o texto como digitado. Um espaço à frente
// mantém a mensagem inteira. Também impede um jogador de começar a linha
// com um marcador de canal ('-' guilda, '=' grupo, '@' reino) para ela
// aparecer como aquele canal;
// - Color (offset whisperColorOffset) é zerado: 7 moveria a linha para o
// chat cinza (TMFieldScene.cpp:17901).
//
// O texto é cortado para caber em String com seu NUL (126 bytes após o espaço).
func privateWhisperText(in []byte) []byte {
	text := in
	if len(text) > whisperTextMax {
		text = text[:whisperTextMax]
	}
	if i := bytes.IndexByte(text, 0); i >= 0 {
		text = text[:i]
	}
	if len(text) > whisperTextMax-2 {
		text = text[:whisperTextMax-2]
	}
	out := make([]byte, max(len(in), len(text)+2))
	out[0] = ' '
	copy(out[1:], text)
	return out
}

// partyChat roteia uma linha "=texto" (MobName vazio) para o grupo do falante, como a
// região "Chat Party" de _MSG_MessageWhisper.cpp:1470-1510: MobName vira o
// nome do falante, o líder recebe a menos que seja o falante, e também os
// membros do PartyList do líder exceto o falante e os que têm partychat desligado.
// Um jogador fora de qualquer grupo é seu próprio líder com lista vazia, então ninguém
// recebe. Invocações compartilham o PartyList e são puladas (sem sessão).
//
// Não reproduzido: o gate MuteChat, que ainda não é modelado (veja magicTrumpet),
// e a linha de ChatLog chat_party — o texto nunca é logado.
func (d *Dispatcher) partyChat(w *world.World, s *world.Session, body protocol.MsgWhisperBody) {
	e := w.Entity(s.Conn)
	if e == nil {
		return
	}
	body.MobName = [16]byte{}
	copy(body.MobName[:len(body.MobName)-1], e.Name)
	out := body.Encode()
	leaderConn := e.Leader
	if leaderConn <= 0 {
		leaderConn = s.Conn
	}
	le := w.Entity(leaderConn)
	ls := w.Session(leaderConn)
	if le == nil || ls == nil || ls.Mode != world.UserPlay {
		return
	}
	h := protocol.Header{Type: protocol.MsgMessageWhisper, ID: uint16(s.Conn)}
	if leaderConn != s.Conn {
		w.SendTo(ls, h, out)
	}
	for _, m := range le.PartyList {
		if m <= 0 || m == s.Conn {
			continue
		}
		ms := w.Session(m)
		if ms == nil || ms.Mode != world.UserPlay || ms.PartyChat {
			continue
		}
		w.SendTo(ms, h, out)
	}
}

// guildChatColor é a Color que o legado carimba numa linha de guilda
// (m->String[MESSAGE_LENGTH] = 3); o cliente só mostra uma linha "-" como chat de guilda
// quando Color == 3 (TMFieldScene.cpp:17885).
const guildChatColor = 3

// whisperColorOffset é onde o cliente 7662 guarda MSG_MessageWhisper.Color
// dentro do String do corpo: MobName[16] + String[128] + short Color (offset de frame
// 156). O TMSrv legado escreve String[MESSAGE_LENGTH] = String[96] porque sua
// própria struct de sussurro tem um String de 100 bytes; o cliente Windows que este servidor
// mira lê o offset 128.
const whisperColorOffset = 128

// onlyGuildMemberCan é Language.txt _NN_Only_Guild_Member_Can (104), mantido nos
// bytes Windows-1252 da tabela como o resto da tabela de strings legada.
const onlyGuildMemberCan = "S\xf3 poder\xe1 usar o Chat Guildas se pertence a alguma."

// guildChat roteia uma linha "-texto" (MobName vazio) para a guilda do falante, como a
// região "Chat Guild" de _MSG_MessageWhisper.cpp:1426-1468: MobName vira o
// nome do falante e Color vira 3. Todo outro membro em jogo da guilda
// recebe, a menos que tenha desligado o guildchat. Uma linha "--texto" também alcança os
// membros da guilda aliada à guilda do falante (g_pGuildAlly[guild]).
// Um jogador fora de qualquer guilda recebe _NN_Only_Guild_Member_Can.
//
// Não reproduzido: a linha de log chat_guild — o texto nunca é logado.
func (d *Dispatcher) guildChat(w *world.World, s *world.Session, body protocol.MsgWhisperBody) {
	e := w.Entity(s.Conn)
	if e == nil {
		return
	}
	guild := e.Guild
	if guild == 0 {
		d.sendClientMessage(w, s, onlyGuildMemberCan)
		return
	}
	body.MobName = [16]byte{}
	copy(body.MobName[:len(body.MobName)-1], e.Name)
	// Uma linha curta é preenchida até Color, que é um short little-endian.
	if len(body.String) < whisperColorOffset+2 {
		body.String = append(body.String, make([]byte, whisperColorOffset+2-len(body.String))...)
	}
	body.String[whisperColorOffset] = guildChatColor
	body.String[whisperColorOffset+1] = 0
	out := body.Encode()
	ally := uint16(0)
	if len(body.String) > 1 && body.String[1] == '-' {
		ally = d.guildAllies[guild]
	}
	h := protocol.Header{Type: protocol.MsgMessageWhisper, ID: uint16(s.Conn)}
	w.ForEachPlaying(s.Conn, func(ms *world.Session, me *world.Entity) {
		if me.Guild != guild && (ally == 0 || me.Guild != ally) {
			return
		}
		if ms.GuildChat {
			return
		}
		w.SendTo(ms, h, out)
	})
}

// teleportCmds mapeia um comando de barra para seu tile de destino. O cliente envia
// "/armia" como um sussurro cujo nome de alvo é "armia", então a palavra-chave do comando É o
// nome do sussurro (_MSG_MessageWhisper.cpp). As coordenadas são as de tempo de paz; as
// variantes de estado de guerra RvR/Torre/Castelo não são modeladas.
var teleportCmds = map[string][2]int16{
	"armia": {2100, 2100}, "azran": {2500, 1716}, "erion": {2461, 2003},
	"gelo": {3650, 3130}, "torre": {2506, 1878},
	"arch":    {1706, 1723},
	"selados": {1843, 3652}, "amagos": {3910, 2878}, "agua": {1966, 1770},
	"noatun": {1052, 1726},
}

// destinos e textos de /kefra (_MSG_MessageWhisper.cpp:836-850). Com o
// Kefra vivo o comando para no gate do chefe; depois de derrotado ele abre
// o "Continente desconhecido", onde o Urnammu fica (issue #410).
var (
	kefraDestAlive    = [2]int16{2365, 3884}
	kefraDestDefeated = [2]int16{3250, 1695}
)

const (
	msgKefraStillAlive = "O kefra ainda esta vivo, precisa ser derrotado"
	msgTeleportKefra   = "Teleportado para Kefra." // _NN_TP_KEFRA (548)
)

// teleportKefra é /kefra. A guarda de estado de guerra (RvR/Torre/Castelo) não é
// modelada, como os outros comandos de teleporte.
func (d *Dispatcher) teleportKefra(w *world.World, s *world.Session) {
	if w.Entity(s.Conn) == nil {
		return
	}
	dest, msg := kefraDestAlive, msgKefraStillAlive
	if d.expEvents.KefraLive { // nome legado: true significa derrotado
		dest, msg = kefraDestDefeated, msgTeleportKefra
	}
	d.doTeleport(w, s, dest[0]+int16(w.Rand().Intn(3)), dest[1]+int16(w.Rand().Intn(3)))
	d.sendClientMessage(w, s, msg)
}

// reinoCapeSlot é o slot de equipamento de capa (Equip[15]), confirmado por _MSG_Quest.cpp:702
// (STRUCT_ITEM *Capa = &pMob[conn].MOB.Equip[15]) e o nPos=-32768 de toda capa
// (bit 15) em tmserver/content/Common/ItemList.csv.
const reinoCapeSlot = 15

// capaBrancaDoMonstroIndex é ItemList.csv #550 "Capa_Branca_do_Monstro" — a capa
// branca neutra. Como toda outra capa que não é de reino ela roteia /reino para o destino
// neutro (issue #208); é nomeada aqui porque é a capa que os jogadores chamam de
// "capa branca".
const capaBrancaDoMonstroIndex = 550

// Os destinos dos comandos de reino continuam fixos como os handlers legados /king e
// /kingdom. Destinos de Rei e de comércio são distintos de propósito (issue #318).
var (
	kingDestHekalotia = [2]int16{1748, 1574}
	kingDestAkelonia  = [2]int16{1748, 1880}

	kingdomDestNeutral   = [2]int16{1702, 1726}
	kingdomDestHekalotia = [2]int16{1690, 1618}
	kingdomDestAkelonia  = [2]int16{1690, 1842}
)

// runCommand executa um comando de barra entregue como sussurro cujo nome de alvo é
// o comando. Devolve true quando name era um comando (tratado); false para cair na
// entrega normal de sussurro. Espelha o dispatch em _MSG_MessageWhisper.cpp.
//
// UNVERIFIED / adiado: os comandos de destrave/quest (destravar40/90, arcana,
// crias) dependem dos sistemas Arch/Celestial e de quest que ainda não são modelados,
// então não são tratados aqui.
func (d *Dispatcher) runCommand(w *world.World, s *world.Session, name string, args []byte) bool {
	cmd := strings.TrimPrefix(name, "/")
	if cmd == "srv" {
		d.changeServer(w, s, args)
		return true
	}
	if cmd == "nt" {
		if e := w.Entity(s.Conn); e != nil {
			d.sendKefraBalance(w, s, e.NightmareEntries)
		}
		return true
	}
	if cmd == "nig" {
		// SendClientMessage (ID 0): w.Send carimba o conn e o 7662 descarta o painel.
		d.sendClientMessage(w, s, d.now().In(time.Local).Format("!!150405"))
		return true
	}
	// "day": o 7662 pergunta em toda atualização de affect enquanto não tem ano/dia
	// (TMHuman.cpp:4755, m_nYear/m_nDays ainda 0). O legado responde um fixo
	// "!#11  2" (_MSG_MessageWhisper.cpp:603), que o TMScene interpreta como ano/dia
	// do campo. Sem resposta, cada pedido caía no caminho de sussurro
	// como um sussurro para "day" e mostrava o aviso de não conectado.
	if cmd == "day" {
		d.sendClientMessage(w, s, "!#11  2")
		return true
	}
	if cmd == "kefra" {
		d.teleportKefra(w, s)
		return true
	}
	if dest, ok := teleportCmds[cmd]; ok {
		if e := w.Entity(s.Conn); e != nil {
			d.doTeleport(w, s, dest[0]+int16(w.Rand().Intn(3)), dest[1]+int16(w.Rand().Intn(3)))
		}
		return true
	}
	if cmd == "rei" || cmd == "king" {
		d.teleportKing(w, s)
		return true
	}
	if cmd == "reino" || cmd == "kingdom" {
		d.teleportKingdom(w, s)
		return true
	}
	if cmd == "buffs" {
		d.clearBuffs(w, s)
		return true
	}
	if cmd == "create" {
		d.createGuild(w, s, args)
		return true
	}
	if cmd == "subcreate" {
		d.subcreate(w, s, args)
		return true
	}
	if cmd == "handover" {
		d.handoverGuild(w, s, args)
		return true
	}
	if cmd == "summonguild" {
		d.summonGuild(w, s)
		return true
	}
	if cmd == "expulsar" {
		d.kickGuild(w, s, args)
		return true
	}
	if cmd == "sair" || cmd == "abandonar" {
		d.leaveGuild(w, s)
		return true
	}
	if cmd == "destravar40" {
		d.destravarCelestial(w, s, false)
		return true
	}
	if cmd == "destravar90" {
		d.destravarCelestial(w, s, true)
		return true
	}
	if cmd == "arcana" {
		d.arcana(w, s)
		return true
	}
	if cmd == "cp" {
		d.showChaosPoints(w, s)
		return true
	}
	if cmd == "gritar" || cmd == "spk" {
		d.magicTrumpet(w, s, args)
		return true
	}
	if cmd == "gm" {
		// Command bus de GM/moderação (issue #122). args é o String do sussurro — o
		// resto da linha digitada (ex.: "/gm kick foo" → args = "kick foo").
		d.runGMCommand(w, s, args)
		return true
	}
	if cmd == "nick" {
		d.showNick(w, s, cstr(args))
		return true
	}
	return false
}

// teleportKing roteia /rei e /king pela capa equipada. Estados de capa neutros
// são no-ops tratados porque o comando legado /king não tem destino neutro.
func (d *Dispatcher) teleportKing(w *world.World, s *world.Session) {
	e := w.Entity(s.Conn)
	if e == nil {
		return
	}
	switch kingClanFromCape(0, e.Equip[reinoCapeSlot].Index) {
	case clanHekalotia:
		d.doTeleport(w, s, kingDestHekalotia[0], kingDestHekalotia[1])
	case clanAkelonia:
		d.doTeleport(w, s, kingDestAkelonia[0], kingDestAkelonia[1])
	}
}

// teleportKingdom roteia /reino e /kingdom pela capa equipada. Capas de reino
// aterrissam nos NPCs de comércio delas; estados de capa neutros aterrissam entre reinos.
func (d *Dispatcher) teleportKingdom(w *world.World, s *world.Session) {
	e := w.Entity(s.Conn)
	if e == nil {
		return
	}
	dest := kingdomDestNeutral
	switch kingClanFromCape(0, e.Equip[reinoCapeSlot].Index) {
	case clanHekalotia:
		dest = kingdomDestHekalotia
	case clanAkelonia:
		dest = kingdomDestAkelonia
	}
	d.doTeleport(w, s, dest[0], dest[1])
}

// clearBuffs remove todo buff/debuff ativo (o comando /buffs), recalcula o score
// — derrubando por ex. os +20% do Divino — e atualiza o cliente (_MSG_MessageWhisper.cpp:42).
// Uma transformação limpa (affect 16) também precisa reverter sua malha de fera em visão, então
// esse caminho retransmite o equipamento em vez de só o score.
func (d *Dispatcher) clearBuffs(w *world.World, s *world.Session) {
	e := w.Entity(s.Conn)
	if e == nil {
		return
	}
	_, _, wasTransformed := activeTransform(e)
	e.ResetAffects()
	if wasTransformed {
		d.refreshEquip(w, s, e) // recalcula equipamento/brilho visual + score, transmite os dois
	} else {
		d.refreshScore(e)
		d.sendScore(w, s, e)
	}
	d.sendAffect(w, s, e)
}

// Constantes de destrave Celestial (_MSG_MessageWhisper.cpp:628/645/676).
const (
	celestialUnlockParm = 1    // Parm de _MSG_CombineComplete num destrave bem-sucedido
	cytheraMisticaItem  = 3502 // /destravar90 e recompensa Pedra da Furia Lv90 (Cythera_Mistica), concedida ao carry
	arcanaItemIndex     = 3507 // recompensa de /arcana, colocada em Equip[1]
	arcanaEquipSlot     = 1
)

// destravarCelestial trata /destravar40 (ninety=false) e /destravar90
// (ninety=true): o destrave de nível 40/90 Celestial (_MSG_MessageWhisper.cpp:628/645).
// Define a flag de gate QuestInfo.Celestial para CheckGetLevel deixar o personagem passar
// do nível 40/90 (CMob.cpp:1107), sinaliza o cliente e persiste a flag. /destravar90
// adicionalmente entrega a Cythera Mistica (item 3502) e toca a emoção de destrave.
//
// O comando só é efetivo para um Celestial — para qualquer outro é no-op
// (não há gate a destravar na curva Mortal/Arch), casando com o desenho de
// celestial-system-plan.md. Rodar de novo com a flag já definida é idempotente.
func (d *Dispatcher) destravarCelestial(w *world.World, s *world.Session, ninety bool) {
	e := w.Entity(s.Conn)
	if e == nil {
		return
	}
	d.destravarCelestialFor(w, s, e, ninety)
}

// destravarCelestialFor é o corpo de destravarCelestial com a entidade passada
// explicitamente, para outros chamadores que já têm uma (ex.: a receita
// Destrave Lv40 do combineOdin) não precisarem dela registrada na tabela de conexões
// do mundo — a mesma divisão que useItem/useClasseItem e combineItemOdin/combineOdin
// já usam.
func (d *Dispatcher) destravarCelestialFor(w *world.World, s *world.Session, e *world.Entity, ninety bool) {
	if e.ClassMaster != classMasterCelestial {
		d.log.Debug("destravar: not a celestial", "conn", s.Conn, "ninety", ninety)
		return
	}
	if ninety {
		if e.CelLv90 != 0 {
			return
		}
		e.CelLv90 = 1
		d.grantCarry(w, s, e, cytheraMisticaItem)
	} else {
		if e.CelLv40 != 0 {
			return
		}
		e.CelLv40 = 1
	}
	sendCombineComplete(w, s, celestialUnlockParm)
	if ninety {
		d.playUnlockEmote(w, s)
	}
	w.SaveCharacterThen(s, func(*world.World, *world.Session) {})
	d.log.Info("celestial unlock", "conn", s.Conn, "name", e.Name, "ninety", ninety)
}

// arcana trata /arcana: a entrega da quest Cythera Arcana (_MSG_MessageWhisper.cpp:676).
// Define QuestInfo.Circle, coloca a recompensa (item 3507) em Equip[1], sinaliza o
// cliente, toca a emoção de destrave e persiste. Só efetivo para um Celestial.
func (d *Dispatcher) arcana(w *world.World, s *world.Session) {
	e := w.Entity(s.Conn)
	if e == nil || e.ClassMaster != classMasterCelestial {
		d.log.Debug("arcana: not a celestial", "conn", s.Conn)
		return
	}
	if e.CelCircle != 0 {
		return
	}
	e.CelCircle = 1
	e.Equip[arcanaEquipSlot] = world.Item{Index: arcanaItemIndex}
	w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceEquip, arcanaEquipSlot, itemToSel(e.Equip[arcanaEquipSlot])))
	sendCombineComplete(w, s, celestialUnlockParm)
	d.playUnlockEmote(w, s)
	w.SaveCharacterThen(s, func(*world.World, *world.Session) {})
	d.log.Info("arcana quest", "conn", s.Conn, "name", e.Name)
}

// grantCarry coloca um item no primeiro slot de carry livre do jogador e empurra o
// SendItem de atualização. Se o inventário está cheio a concessão é descartada (logada) — a mudança
// de flag continua valendo, igual ao comportamento best-effort do PutItem legado.
func (d *Dispatcher) grantCarry(w *world.World, s *world.Session, e *world.Entity, index int16) {
	i := firstEmptyAccessibleCarry(e)
	if i >= 0 {
		e.Carry[i] = world.Item{Index: index}
		w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceCarry, i, itemToSel(e.Carry[i])))
		return
	}
	// Sem slot livre o item e perdido (comportamento do PutItem legado), mas o
	// jogador precisa saber: antes o premio sumia sem nenhuma linha.
	d.log.Info("grantCarry: inventory full", "conn", s.Conn, "index", index)
	d.notify(w, s, NoticeNoSpaceToTrade)
}

// playUnlockEmote envia a emoção de destrave celestial (MsgMotion 14/3) ao chamador
// e a todos em visão (a mesma emoção que o cliente toca ao subir de nível).
func (d *Dispatcher) playUnlockEmote(w *world.World, s *world.Session) {
	motion := protocol.EncodeMotion(motionLevelUp, motionLevelUpParm)
	w.Send(s, protocol.MsgMotion, motion)
	w.BroadcastInView(s.Conn, protocol.MsgMotion, motion)
}

// showChaosPoints trata o comando /cp: informa os pontos de caos atuais
// do jogador (_MSG_MessageWhisper.cpp:33-39, _DN_Show_Chao = "Pontos Caos atual: %d").
// GetPKPoint(conn)-75 mapeia para pkPoint(e)-75, agora os contadores reais persistidos
// de PKPoint/Guilty (issue #210), não uma aproximação.
func (d *Dispatcher) showChaosPoints(w *world.World, s *world.Session) {
	e := w.Entity(s.Conn)
	if e == nil {
		return
	}
	chaos := int(pkPoint(e)) - 75
	msg := fmt.Sprintf("Pontos Caos atual: %d", chaos)
	d.sendChatText(w, s, msg)
}

// itemMagicTrumpet é a Trombeta Mágica (ItemList.csv:5031 —
// "3330,Trombeta_Mágica,…,EF_VOLATILE,0,EF_GRID,0"): um consumível empilhável sem
// efeito de uso próprio; existe só para pagar /gritar.
const itemMagicTrumpet = 3330

// magicTrumpet trata /gritar (alias legado /spk): queima uma Trombeta Mágica
// e grita "[Nome]> texto" para todo jogador online
// (_MSG_MessageWhisper.cpp:114-162, issue #258).
//
// Desvios do legado, todos deliberados:
// - O legado repassava MSG_MagicTrumpet pelo DBSrv para todo canal
// receber (DBSrv/CFileDB.cpp:1616, TMSrv/ProcessDBMessage.cpp:136). A pilha
// Go tem um tmServer por canal e o dbServer não tem RPC de anúncio, então
// o grito alcança só os jogadores deste servidor. Fan-out entre canais é trabalho
// futuro.
// - Sem uma trombeta o legado retornava em silêncio (:139); aqui o jogador é
// avisado de por que nada aconteceu.
// - Uma mensagem vazia é recusada em vez de gritar "[Nome]> " e comer uma
// trombeta de qualquer modo.
// - O gate MuteChat (:118) não tem contraparte — mutar ainda não é modelado.
func (d *Dispatcher) magicTrumpet(w *world.World, s *world.Session, args []byte) {
	e := w.Entity(s.Conn)
	if e == nil {
		return
	}
	text := strings.TrimSpace(cstr(args))
	if text == "" {
		return
	}
	slot := -1
	for i, limit := 0, activeCarryLimit(e); i < limit; i++ {
		if e.Carry[i].Index == itemMagicTrumpet {
			slot = i
			break
		}
	}
	if slot < 0 {
		d.sendChatText(w, s, "Voc\xea precisa de uma Trombeta M\xe1gica para gritar.")
		return
	}
	consumeOneItem(&e.Carry[slot])
	d.sendSlot(w, s, world.ItemPlaceCarry, slot, e.Carry[slot])

	// HEADER.ID = 0 como o legado (sm_mt.ID = 0): o grito carrega seu falante
	// dentro do texto, não no header. Ninguém é excluído — quem grita também vê,
	// como o SyncMulticast(0, …) fazia.
	payload := protocol.EncodeMagicTrumpetBody(fmt.Sprintf("[%s]> %s", e.Name, text))
	w.ForEachPlaying(-1, func(vs *world.Session, _ *world.Entity) {
		w.SendTo(vs, protocol.Header{Type: protocol.MsgMagicTrumpet, ID: 0}, payload)
	})
	d.log.Info("magic trumpet", "conn", s.Conn, "account", s.AccountName, "name", e.Name, "text", text)
}

// sendChatText envia uma linha crua de MsgMessageChat só para s — a família
// SendClientMessage de avisos legados (_DN_Show_Chao, _DN_CantKillUser, _DD_PKPointPlus/
// Minus, _DN_lose_D_exp, …), em oposição à família Notice de caixa de diálogo
// (notice.go).
func (d *Dispatcher) sendChatText(w *world.World, s *world.Session, text string) {
	w.SendTo(s, protocol.Header{Type: protocol.MsgMessageChat, ID: uint16(s.Conn)}, protocol.EncodeMessageChatBody(text))
}

// expPanelColor é TNColor::Default (Basedef.h:611): 0xFF000000 | 0xCCAAFF.
const expPanelColor uint32 = 0xFFCCAAFF

// sendMsgExp espelha o SendMsgExp legado (Server.cpp:5940-5968): o ganho de EXP
// vai em MSG_Exp_Msg_Panel_ (0x5000) com Msg[96] + Color32, e o cliente pinta
// como a notificação de EXP em vez de o personagem falar a linha.
func (d *Dispatcher) sendMsgExp(w *world.World, s *world.Session, text string, color uint32) {
	w.SendTo(s, protocol.Header{Type: protocol.MsgExpPanel, ID: uint16(s.Conn)}, protocol.EncodeExpPanelBody(text, color))
}

// sendNPCChatText usa o ID do falante como o SendSay legado, mas mantém respostas
// de serviço pessoal privadas à sessão solicitante em vez de transmitir.
// O corpo é o MSG_MessageChat.String de largura fixa: um frame curto de um falante
// NPC derrubava a conexão do cliente (issue #344).
func (d *Dispatcher) sendNPCChatText(w *world.World, s *world.Session, npcID int, text string) {
	w.SendTo(s, protocol.Header{Type: protocol.MsgMessageChat, ID: uint16(npcID)}, protocol.EncodeMessageChatBody(text))
}

// notifyPKPointDelta formata a linha de chat _DD_PKPointPlus/_DD_PKPointMinus:
// os novos Pontos de Caos exibidos do jogador (PKPoint-75) mais a mudança com sinal
// que acabou de acontecer. O texto exato da tabela de strings legada não está disponível nesta
// árvore de código (Language.h só tem índices #define numéricos), então isto casa
// o tom/formato da linha _DN_Show_Chao existente acima.
func notifyPKPointDelta(newPKPoint uint8, delta int) string {
	sign := "+"
	if delta < 0 {
		sign = "-"
		delta = -delta
	}
	return fmt.Sprintf("Pontos Caos atual: %d (%s%d)", int(newPKPoint)-75, sign, delta)
}

// showNick trata o comando /nick <jogador> (issue #131): responde só ao
// chamador, com o Nick do alvo, Guilda (nome + fama, se registrada —
// world/guild.go), Cidadania e Fama do personagem. Espelha a linha de info legada
// _MSG_MessageWhisper.cpp:1618-1636 (Language.txt _NN_Check_User_Info
// "Cidadania: %d / Fama: %d"), estendida com a fama de guilda pela issue #131, e
// enviada de volta como linha de chat auto-endereçada (como o caminho de texto livre do gmNotice)
// já que ainda não há um formato de pacote de mensagem de sistema dedicado capturado.
func (d *Dispatcher) showNick(w *world.World, s *world.Session, rest string) {
	name := firstToken(strings.TrimSpace(rest))
	if name == "" {
		return
	}
	_, te := w.SessionByName(name)
	if te == nil {
		d.notify(w, s, NoticeNotConnected)
		return
	}
	guildPart := "Sem guilda"
	if te.Guild != 0 {
		if gi, ok := w.GuildInfo(te.Guild); ok {
			guildPart = fmt.Sprintf("%s (Fama guilda: %d)", gi.Name, gi.Fame)
		} else {
			guildPart = fmt.Sprintf("Guild #%d (nao registrada)", te.Guild)
		}
	}
	msg := fmt.Sprintf("%s [%s] Cidadania: %d / Fama: %d", te.Name, guildPart, te.Citizen, te.Fame)
	w.Send(s, protocol.MsgMessageChat, protocol.EncodeMessageChatBody(msg))
}

// firstToken devolve o primeiro token separado por espaço de s.
func firstToken(s string) string {
	if i := strings.IndexByte(s, ' '); i >= 0 {
		return s[:i]
	}
	return s
}
