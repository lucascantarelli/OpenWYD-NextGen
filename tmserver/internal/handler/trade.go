package handler

import (
	"openwyd-nextgen/tmserver/internal/protocol"
	"openwyd-nextgen/tmserver/internal/world"
)

// maxTradeSlot é o limite superior absoluto para slots normais de carry do jogador. O
// limite ativo por personagem pode ser menor quando a Bolsa do Andarilho está inativa.
const maxTradeSlot = maxUnlockedCarry

// tradeEmptyPos marca uma entrada de oferta não usada (o char InvenPos = -1 do legado).
const tradeEmptyPos = 0xFF

// maxTradeCoin é o teto legado de 2G sobre um valor de troca e sobre o ouro
// resultante de cada lado (_MSG_Trade.cpp: TradeMoney > 2000000000, _NN_Cant_get_more_than_2G).
const maxTradeCoin = 2_000_000_000

// trade trata _MSG_Trade (0x0383), portando o fluxo legado de
// TMSrv/_MSG_Trade.cpp:
//
//   - toda oferta aceita é repassada ao oponente com ID = opponent e
//     OpponentID = sender, que é o que abre/atualiza a janela do oponente;
//   - uma mudança de oferta reseta MyCheck dos dois lados; um item já ofertado ou um
//     valor não zero não pode mudar (só anexa, anti troca de item);
//   - o primeiro MyCheck responde ao remetente com _MSG_CNFCheck e repassa a
//     oferta conferida; o segundo faz a troca atômica, reenvia os dois carries
//     (_MSG_UpdateCarry leva Coin também), persiste os dois e fecha a troca.
//
// Qualquer falha de validação cancela a troca (_MSG_QuitTrade para os dois lados ligados).
// Divergências em relação ao legado: os gates de PK-mode/bloqueio de sussurro e as
// regras de item de guilda não foram portadas (o servidor também não as tinha antes), e uma
// troca que falha (espaço/ouro) cancela a troca em vez de deixar as duas janelas
// conferidas.
func (d *Dispatcher) trade(w *world.World, s *world.Session, _ protocol.Header, payload []byte) {
	e := w.Entity(s.Conn)
	if e == nil || e.HP == 0 || s.Mode != world.UserPlay {
		w.AddCrackError(s, 5, 18)
		d.rejectTrade(w, s)
		return
	}
	var body protocol.MsgTradeBody
	if err := body.Decode(payload); err != nil {
		d.rejectTrade(w, s)
		return
	}
	opp := int(body.OpponentID)
	other := w.Session(opp)
	oe := w.Entity(opp)
	if opp <= 0 || opp >= world.MaxUser || opp == s.Conn || other == nil || oe == nil || other.Mode != world.UserPlay {
		d.log.Warn("trade: opponent not in play", "conn", s.Conn, "opponent", opp,
			"money", body.TradeMoney, "mycheck", body.MyCheck,
			"first_pos", body.InvenPos[0], "first_item", body.Item[0].Index)
		d.tradeRefuse(w, s, tradeWrongPacket)
		return
	}
	if body.TradeMoney < 0 || body.TradeMoney > maxTradeCoin || body.TradeMoney > e.Coin {
		d.tradeRefuse(w, s, notEnoughMoney)
		return
	}
	offer, ok := d.tradeOffer(e, &body)
	if !ok {
		d.log.Warn("trade: offer does not match the carry", "conn", s.Conn, "opponent", opp,
			"money", body.TradeMoney)
		d.tradeRefuse(w, s, tradeWrongPacket)
		return
	}
	linked := other.Trade.Active && other.Trade.OpponentID == s.Conn
	if linked && !tradeOfferHeld(oe, &other.Trade) {
		d.log.Warn("trade: opponent offer changed", "conn", s.Conn, "opponent", opp)
		d.tradeRefuse(w, s, tradeWrongPacket) // o item ofertado do oponente saiu do lugar ou mudou
		return
	}

	if s.Trade.Active {
		if s.Trade.OpponentID != opp {
			d.tradeRefuse(w, s, tradeAlreadyOn) // _NN_Already_Trading
			return
		}
		for i := range s.Trade.Items {
			if s.Trade.Items[i].Index != 0 && (s.Trade.Items[i] != offer.Items[i] || s.Trade.InvenPos[i] != offer.InvenPos[i]) {
				d.tradeRefuse(w, s, tradeWrongPacket)
				return
			}
		}
		if s.Trade.Money != 0 && s.Trade.Money != offer.Money {
			d.tradeRefuse(w, s, tradeWrongPacket)
			return
		}
	}

	if other.Trade.Active && !linked {
		d.tradeRefuse(w, s, tradeAlreadyOn) // o oponente está negociando com outra pessoa
		return
	}

	if body.MyCheck == 1 {
		// Conferir só é válido sobre a oferta exata que o oponente já viu.
		if !linked || !s.Trade.Active || s.Trade.Items != offer.Items || s.Trade.InvenPos != offer.InvenPos || s.Trade.Money != offer.Money {
			d.tradeRefuse(w, s, tradeWrongPacket)
			return
		}
		s.Trade.Confirmed = true
		if !other.Trade.Confirmed {
			w.Send(s, protocol.MsgCNFCheck, nil)
			d.forwardTrade(w, s, other)
			return
		}
		d.executeSwap(w, s, other)
		return
	}

	offer.Active = true
	offer.OpponentID = opp
	s.Trade = offer
	other.Trade.Confirmed = false
	d.forwardTrade(w, s, other)
} // tradeRefuse é rejectTrade mais o motivo mostrado ao jogador. Toda recusa costumava
// ser silenciosa, então uma troca recusada só fechava a janela e soava como "a troca não
// funciona" (QA 05/11).
func (d *Dispatcher) tradeRefuse(w *world.World, s *world.Session, text string) {
	d.sendClientMessage(w, s, text)
	d.rejectTrade(w, s)
}

// rejectTrade cancela a troca de s depois de um _MSG_Trade recusado. Diferente de removeTrade, ele
// também responde a um remetente que ainda não tinha troca registrada, então uma primeira oferta recusada
// ainda fecha a janela que o cliente abriu localmente (o RemoveTrade legado
// sempre sinaliza _MSG_QuitTrade para conn).
func (d *Dispatcher) rejectTrade(w *world.World, s *world.Session) {
	silent := !s.Trade.Active && s.AutoTrade == nil && s.TradeMode == 0
	d.removeTrade(w, s)
	if silent {
		w.Send(s, protocol.MsgQuitTrade, nil)
	}
}

// tradeOffer normaliza e valida a oferta em body contra o carry de e. Uma
// entrada é usada quando seu InvenPos é um slot e seu item está definido; o item precisa
// casar exatamente com o item do carry (o memcmp legado) e não ser EF_NOTRADE, e nenhum
// slot pode repetir.
func (d *Dispatcher) tradeOffer(e *world.Entity, body *protocol.MsgTradeBody) (world.TradeState, bool) {
	t := world.TradeState{Money: body.TradeMoney}
	var used [world.MaxCarry]bool
	for i := 0; i < protocol.MaxTrade; i++ {
		pos := int(body.InvenPos[i])
		if pos == tradeEmptyPos || body.Item[i].Index == 0 {
			t.InvenPos[i] = tradeEmptyPos
			continue
		}
		if pos >= maxTradeSlot || !carrySlotAccessible(e, pos) || used[pos] || e.Carry[pos].Empty() {
			return world.TradeState{}, false
		}
		// Oferta server-authoritative: o slot é a verdade, os bytes do item que o
		// cliente enviou são só uma dica. O legado reconstrói do mesmo jeito
		// (memcpy de MOB.Carry, _MSG_Trade.cpp:199) e recusa só por um
		// slot ruim. Exigir casamento exato do STRUCT_ITEM enviado recusava toda
		// troca real com "Erro na transação" (QA 05/11), porque os bytes de
		// efeito do cliente para o mesmo item diferem dos armazenados.
		if !sameItem(body.Item[i], e.Carry[pos]) {
			d.log.Debug("trade: client item bytes differ from carry, using carry",
				"conn", e.ID, "slot", pos, "client", body.Item[i].Index, "carry", e.Carry[pos].Index)
		}
		if d.itemAbility(e.Carry[pos], efNoTrade) != 0 {
			return world.TradeState{}, false
		}
		used[pos] = true
		t.Items[i] = e.Carry[pos]
		t.InvenPos[i] = uint8(pos)
		t.Slots = append(t.Slots, pos)
	}
	return t, true
}

// tradeOfferHeld informa se todo item de uma oferta registrada ainda está no
// slot de carry de onde foi ofertado (anti troca de item durante a confirmação).
func tradeOfferHeld(e *world.Entity, t *world.TradeState) bool {
	for i := range t.Items {
		if t.Items[i].Index == 0 {
			continue
		}
		pos := int(t.InvenPos[i])
		if pos >= maxTradeSlot || !carrySlotAccessible(e, pos) || e.Carry[pos] != t.Items[i] {
			return false
		}
	}
	return true
}

// forwardTrade envia a oferta registrada de from ao seu oponente no layout
// clássico de MSG_Trade, com OpponentID = from (legado: m->ID = OpponentID;
// m->OpponentID = conn; AddMessage).
func (d *Dispatcher) forwardTrade(w *world.World, from, to *world.Session) {
	var body protocol.MsgTradeBody
	for i := range from.Trade.Items {
		body.Item[i] = wireFromItem(from.Trade.Items[i])
		body.InvenPos[i] = from.Trade.InvenPos[i]
	}
	body.TradeMoney = from.Trade.Money
	if from.Trade.Confirmed {
		body.MyCheck = 1
	}
	body.OpponentID = uint16(from.Conn)
	w.Send(to, protocol.MsgTrade, body.Encode())
}

// executeSwap move as duas ofertas confirmadas de forma atômica (valida-tudo-e-
// aplica-tudo, BASE_CanTrade legado): os slots ofertados de cada lado são esvaziados, os
// itens que chegam preenchem os primeiros slots livres acessíveis, e o ouro anda nos dois sentidos.
// Qualquer falta faz rollback e cancela a troca. No sucesso os dois carries são
// reenviados (_MSG_UpdateCarry, com Coin), os dois personagens são persistidos, e a
// troca é fechada dos dois lados (_MSG_QuitTrade), como SendCarry + SaveUser +
// RemoveTrade no legado.
func (d *Dispatcher) executeSwap(w *world.World, a, b *world.Session) {
	ea, eb := w.Entity(a.Conn), w.Entity(b.Conn)
	if ea == nil || eb == nil || !tradeOfferHeld(ea, &a.Trade) || !tradeOfferHeld(eb, &b.Trade) {
		d.removeTrade(w, a)
		return
	}
	if a.Trade.Money > ea.Coin || b.Trade.Money > eb.Coin {
		d.removeTrade(w, a)
		return
	}
	coinA := int64(ea.Coin) - int64(a.Trade.Money) + int64(b.Trade.Money)
	coinB := int64(eb.Coin) - int64(b.Trade.Money) + int64(a.Trade.Money)
	if coinA > maxTradeCoin || coinB > maxTradeCoin {
		d.removeTrade(w, a)
		return
	}

	aItems := takeItems(ea, a.Trade.Slots)
	bItems := takeItems(eb, b.Trade.Slots)
	if freeCarry(eb) < len(aItems) || freeCarry(ea) < len(bItems) {
		noRoomA, noRoomB := freeCarry(ea) < len(bItems), freeCarry(eb) < len(aItems)
		putBack(ea, a.Trade.Slots, aItems) // espaço insuficiente → rollback
		putBack(eb, b.Trade.Slots, bItems)
		if noRoomA {
			d.notify(w, a, NoticeNoEmptySlot)
		}
		if noRoomB {
			d.notify(w, b, NoticeNoEmptySlot)
		}
		d.removeTrade(w, a)
		return
	}
	for _, it := range aItems {
		if dst := firstEmptyAccessibleCarry(eb); dst >= 0 {
			eb.Carry[dst] = it
		}
	}
	for _, it := range bItems {
		if dst := firstEmptyAccessibleCarry(ea); dst >= 0 {
			ea.Carry[dst] = it
		}
	}
	ea.Coin = int32(coinA)
	eb.Coin = int32(coinB)
	d.log.Info("trade completed", "conn", a.Conn, "opponent", b.Conn,
		"items", len(aItems), "opponentItems", len(bItems), "coin", a.Trade.Money, "opponentCoin", b.Trade.Money)

	d.sendCarry(w, a, ea)
	d.sendCarry(w, b, eb)
	w.SaveCharacterAsync(a)
	w.SaveCharacterAsync(b)
	d.removeTrade(w, a)
}

// quitTrade trata _MSG_QuitTrade (0x0384): cancela a troca.
func (d *Dispatcher) quitTrade(w *world.World, s *world.Session, _ protocol.Header, _ []byte) {
	if e := w.Entity(s.Conn); e == nil || e.HP <= 0 || s.Mode != world.UserPlay {
		w.AddCrackError(s, 10, 17)
	}
	d.removeTrade(w, s)
}

// removeTrade cancela qualquer troca ativa em s e no seu oponente, avisando os dois.
// É também o gancho anti-dup chamado quando um jogador dropa/usa/ataca no meio de uma troca.
func (d *Dispatcher) removeTrade(w *world.World, s *world.Session) {
	// O RemoveTrade do original também fecha uma loja pessoal aberta (Server.cpp:8124);
	// é isso que faz andar/comprar/operações de item/sair da troca derrubarem a banca.
	d.closeAutoTrade(w, s)
	if !s.Trade.Active {
		return
	}
	opp := s.Trade.OpponentID
	s.Trade = world.TradeState{}
	w.Send(s, protocol.MsgQuitTrade, nil)
	if other := w.Session(opp); other != nil && other.Trade.OpponentID == s.Conn {
		other.Trade = world.TradeState{}
		w.Send(other, protocol.MsgQuitTrade, nil)
	}
}

// sameItem informa se um item do wire é igual ao item do inventário (o memcmp
// usado para detectar um item trocado durante a confirmação).
func sameItem(wi protocol.WireItem, it world.Item) bool {
	if wi.Index != it.Index {
		return false
	}
	for i := 0; i < 3; i++ {
		if wi.Effects[i].Effect != it.Effects[i].Effect || wi.Effects[i].Value != it.Effects[i].Value {
			return false
		}
	}
	return true
}

func freeCarry(e *world.Entity) int {
	return freeAccessibleCarry(e)
}

func tradeSlotsAccessible(e *world.Entity, slots []int) bool {
	for _, sl := range slots {
		if sl < 0 || sl >= maxTradeSlot || !carrySlotAccessible(e, sl) || e.Carry[sl].Empty() {
			return false
		}
	}
	return true
}

// takeItems remove os itens dos slots de carry dados, devolvendo-os alinhados aos
// slots (para o putBack restaurá-los no rollback).
func takeItems(e *world.Entity, slots []int) []world.Item {
	out := make([]world.Item, len(slots))
	for i, sl := range slots {
		if sl >= 0 && sl < world.MaxCarry {
			out[i] = e.Carry[sl]
			e.Carry[sl] = world.Item{}
		}
	}
	return out
}

func putBack(e *world.Entity, slots []int, items []world.Item) {
	for i, sl := range slots {
		if sl >= 0 && sl < world.MaxCarry {
			e.Carry[sl] = items[i]
		}
	}
}
