package handler

import (
	"openwyd-nextgen/tmserver/internal/protocol"
	"openwyd-nextgen/tmserver/internal/world"
)

// pkResetGuilty é o valor de Guilty aplicado ao acertar (ou receber) um golpe
// PvP caótico (_MSG_Attack.cpp "PK - War - Miss": SetGuilty(conn,8) / SetGuilty(idx,8)).
const pkResetGuilty uint8 = 8

// guiltyDecayPeriod escalona o decaimento de Guilty por jogador do mesmo jeito que
// affectTickPeriod escalona affects (affect_tick.go): o RegenMob legado roda
// uma vez por conexão a cada ~8 segundos reais (Server.cpp RegenMob,
// a cadência por conexão escalonada pelo Sec16 de ProcessSecMinTimer.cpp), e
// decrementa Guilty em 1 ali, incondicionalmente. Uma constante distinta de
// affectTickPeriod de propósito — os dois portam campos legados sem relação que só
// por coincidência compartilham um período.
const guiltyDecayPeriod = 8

// pkPointRecoverPeriod é a cadência de recuperação natural de PKPoint: o legado a põe
// atrás de um contador que incrementa uma vez por chamada de RegenMob e dispara a cada 450
// delas (Server.cpp: `if (!(Unk_2736 % 450))`), ou seja, 450×8s = 3600s = 1h.
// Global (não escalonada por conexão) — um gate de escala de hora não precisa de
// distribuição de carga, ao contrário do decaimento de Guilty de ~8s.
const pkPointRecoverPeriod = 3600

// Coloração do nick de PK: o cliente lê MobName[12] (PKPoint) do
// MSG_CreateMob de um jogador para colorir o nick — 75 é NEUTRO (branco), 0 é vermelho/piscando
// (caos/culpado), <75 é caos (GetFunc.cpp:1082-1101). Este é o driver real,
// NÃO o _MSG_PKInfo (que só carrega o estado "atacável/guerra").
const (
	pkPointNeutral uint8 = 75 // nick branco
	pkPointChaos   uint8 = 0  // nick vermelho piscando
)

// pkPointPerLevel é o Ponto de Caos pago de volta por nível ganho (issue #279).
//
// DIVERGÊNCIA DELIBERADA DO LEGADO — não "conserte" isto de volta à paridade
// sem ler a issue. O original nunca amarrou PKPoint a subir de nível: sua
// única recuperação natural é o gate horário do RegenMob (Server.cpp:4872-4881,
// portado abaixo como pkPointRecoverPeriod), e é por isso que a issue #210 foi fechada
// sem isto. A issue #279 a reabriu como decisão de design para este servidor:
// subir de nível também precisa devolver caos, para um personagem saindo do caos ver
// o nick ficar branco. Mesmo passo de +1 e mesmo teto de 75 do gate horário, e ele
// se soma a ele em vez de substituí-lo.
const pkPointPerLevel = 1

// pkGuilty informa se e está atualmente no estado caótico (nick vermelho piscando):
// seu contador de decaimento de Guilty (slot KILL_MARK de GetFunc.cpp) não chegou a 0.
func pkGuilty(e *world.Entity) bool {
	return e.Guilty > 0
}

// pkPoint é o byte MobName[12] para o estado atual de e (GetFunc.cpp
// GetCreateMob:1094-1099): o contador real de PKPoint, a menos que Guilty > 0, caso em que
// é 0 (caos/vermelho) independente do PKPoint. Empacotado em todo
// MSG_CreateMob de jogador e no blob de CNFCharacterLogin para o nick renderizar a cor
// certa.
func pkPoint(e *world.Entity) uint8 {
	if pkGuilty(e) {
		return pkPointChaos
	}
	return e.PKPoint
}

// pkInfoParm é o Parm de PKInfo S→C para e: 1 = estado PK/atacável, 0 = limpo.
// Isto comanda a flag "pode ser atacado / em guerra", NÃO a cor do nick (essa é
// MobName[12]); ainda o enviamos por paridade com o fluxo de login/view original.
func pkInfoParm(e *world.Entity) int32 {
	if pkGuilty(e) || e.PKMode {
		return 1
	}
	return 0
}

// broadcastPKState reenvia o estado PK atual de e para si mesmo e todos à vista
// sempre que ele muda (um golpe PvP entra, ou Guilty cai a 0): um
// MSG_CreateMob novo (cujo MobName[12..15] recolore o nick e atualiza os
// bytes de sequência de mortes — Server.cpp:4933-4941 faz exatamente isso quando Guilty chega
// a 0) mais a flag atacável do MSG_PKInfo.
func (d *Dispatcher) broadcastPKState(w *world.World, s *world.Session, e *world.Entity) {
	mob := protocol.EncodeCreateMobBody(d.createMobFrom(e, 0))
	info := protocol.EncodeStandardParm(pkInfoParm(e))
	send := func(vs *world.Session) {
		w.SendTo(vs, protocol.Header{Type: protocol.MsgCreateMob, ID: protocol.IDScene}, mob)
		w.SendTo(vs, protocol.Header{Type: protocol.MsgPKInfo, ID: uint16(s.Conn)}, info)
	}
	send(s) // o cliente do próprio jogador (recolore o próprio nick)
	w.ForEachInView(s.Conn, func(vs *world.Session, _ *world.Entity) { send(vs) })
}

// markGuilty põe o contador Guilty de e em pkResetGuilty (acertar/receber um
// golpe PvP caótico fora de um duelo) e, se ainda não estava culpado, retransmite
// o estado PK para o nick ficar vermelho. s pode ser nil (por exemplo, a vítima se desconectou
// no meio da luta) — broadcastPKState precisa de sessão só para endereçar o envio ao "cliente
// próprio", então uma sessão nil apenas pula esse envio próprio pela guarda abaixo.
func (d *Dispatcher) markGuilty(w *world.World, s *world.Session, e *world.Entity) {
	wasGuilty := pkGuilty(e)
	e.Guilty = pkResetGuilty
	if !wasGuilty && s != nil {
		d.broadcastPKState(w, s, e)
	}
}

// grantLevelUpPKPoint devolve levels × pkPointPerLevel rumo ao neutro 75
// depois de um nível ganho (issue #279). Nunca empurra além de 75: a faixa 76..150 só
// vem dos caminhos de quest/item (_MSG_Quest.cpp:2569-2579, _MSG_UseItem.cpp:3867)
// e subir de nível não pode completá-los — mesmo teto que a recuperação horária usa.
//
// O ganho é reportado uma vez, não por nível, então um salto de vários níveis (Poeira de
// Fada, GM /setlevel) produz uma única linha de chat, e o estado PK é
// retransmitido para o nick recolorir sem relog. Enquanto Guilty > 0 o
// pkPoint(e) visível fica fixado em 0, então o CreateMob carregaria o mesmo byte de antes
// — pule a transmissão nesse caso e deixe o decaimento de Guilty no sweepGuilty fazê-lo.
// s pode ser nil (os chamadores GM/offline applyLevelUps).
func (d *Dispatcher) grantLevelUpPKPoint(w *world.World, s *world.Session, e *world.Entity, levels int32) {
	if levels <= 0 || e.PKPoint >= pkPointNeutral {
		return
	}
	// aritmética de int primeiro: levels pode estar na casa das centenas (/setlevel), o que
	// estouraria o contador uint8.
	gain := int(levels) * pkPointPerLevel
	if room := int(pkPointNeutral - e.PKPoint); gain > room {
		gain = room
	}
	e.PKPoint += uint8(gain)
	if s == nil {
		return
	}
	d.sendChatText(w, s, notifyPKPointDelta(e.PKPoint, gain))
	if !pkGuilty(e) {
		d.broadcastPKState(w, s, e)
	}
}

// pkMode trata _MSG_PKMode (0x0399): o toggle de consentimento Player-Killer da tecla K
// do cliente (Exec_MSG_PKMode, _MSG_PKMode.cpp). Alternar só vira o gate de consentimento
// verificado pelo attack() — cancela qualquer troca ativa (mesma guarda anti-dup do
// OpponentID legado) e ecoa o estado atual de PKInfo. Apertar K NÃO
// pisca o nick sozinho (isso precisa de um golpe PvP aplicado → culpado).
func (d *Dispatcher) pkMode(w *world.World, s *world.Session, _ protocol.Header, payload []byte) {
	if s.Mode != world.UserPlay {
		return
	}
	e := w.Entity(s.Conn)
	if e == nil {
		return
	}
	parm, _ := protocol.StandardParm(payload)
	e.PKMode = parm != 0

	// Uma troca mútua recebe _NN_Cant_trade_pkmode nos dois lados antes de fechar
	// (_MSG_PKMode.cpp:27-35, issue #422).
	if s.Trade.Active {
		if other := w.Session(s.Trade.OpponentID); other != nil && other.Mode == world.UserPlay && other.Trade.OpponentID == s.Conn {
			d.sendClientMessage(w, s, cantTradePKMode)
			d.sendClientMessage(w, other, cantTradePKMode)
		}
	}
	d.removeTrade(w, s)
	// Só a flag atacável (PKInfo) muda num toggle de K — a cor do nick
	// (MobName[12]) não é afetada, então nenhuma retransmissão de CreateMob é necessária.
	info := protocol.EncodeStandardParm(pkInfoParm(e))
	w.SendTo(s, protocol.Header{Type: protocol.MsgPKInfo, ID: uint16(s.Conn)}, info)
	w.ForEachInView(s.Conn, func(vs *world.Session, _ *world.Entity) {
		w.SendTo(vs, protocol.Header{Type: protocol.MsgPKInfo, ID: uint16(s.Conn)}, info)
	})
}

// sweepGuilty é o port do RegenMob para decaimento de Guilty + recuperação de PKPoint
// (Server.cpp RegenMob): Guilty decrementa 1 na fase escalonada de ~8s de cada
// jogador (só para jogadores conectados no momento — RegenMob nunca roda para uma
// conexão offline, então o decaimento pausa corretamente durante o logout sem
// contabilidade extra), e PKPoint recupera +1 rumo ao neutro (75) num gate horário
// global. Local de chamada inalterado (mobai.go Tick()).
func (d *Dispatcher) sweepGuilty(w *world.World) {
	phase := d.tickCount % guiltyDecayPeriod
	pkTick := d.tickCount%pkPointRecoverPeriod == 0
	w.ForEachPlayer(func(s *world.Session, e *world.Entity) {
		if s.Conn%guiltyDecayPeriod == phase && e.Guilty > 0 {
			e.Guilty--
			if e.Guilty == 0 {
				d.broadcastPKState(w, s, e)
			}
		}
		if pkTick && e.PKPoint < pkPointNeutral {
			e.PKPoint++
			d.sendChatText(w, s, notifyPKPointDelta(e.PKPoint, 1))
			// O ponto recuperado muda MobName[12], então o nick tem de ser
			// repintado aqui também — senão mantém a cor velha até o
			// próximo CreateMob (um relog, ou sair e reentrar na visão de alguém).
			// Fixado em 0 enquanto Guilty > 0, então nada a repintar nesse caso.
			if !pkGuilty(e) {
				d.broadcastPKState(w, s, e)
			}
		}
	})
}
