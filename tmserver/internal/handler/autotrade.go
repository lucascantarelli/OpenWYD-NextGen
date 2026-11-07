package handler

import (
	"openwyd-nextgen/tmserver/internal/protocol"
	"openwyd-nextgen/tmserver/internal/world"
)

// Loja pessoal / autotrade (issue #115, TMSrv/_MSG_SendAutoTrade.cpp,
// _MSG_ReqBuy.cpp, _MSG_ReqTradeList.cpp, SendFunc.cpp:SendAutoTrade). O jogador
// abre uma banca AFK que vende itens do Cargo (armazém) da conta por CargoPos; o
// item ofertado continua referenciado no Cargo e só sai numa compra concluída,
// então fechar a loja nunca devolve nada. Todo o estado vive na Session (só de
// sessão, nunca persistido) e cada handler roda dentro da goroutine do loop, o
// que torna a transação de compra atômica sem locks. Fechar a loja é
// RemoveTrade (handler.removeTrade), NÃO _MSG_Deprivate (que é operação de guild).

// autoTradePriceMax é o teto de preço por item (_MSG_SendAutoTrade.cpp:76).
const autoTradePriceMax = 1_999_999_999

// autoTradeViewRange é VIEWGRIDX/VIEWGRIDY (33): quem navega ou compra precisa
// estar dentro desta caixa em relação ao dono da loja (_MSG_ReqBuy.cpp:62,
// _MSG_ReqTradeList.cpp:40).
const autoTradeViewRange = world.NoViewRange

// autoTradeTaxThreshold é o preço abaixo do qual não há imposto de cidade
// (_MSG_ReqBuy.cpp:138): imposto = (Price/100)*Tax só quando Price >= 100000.
const autoTradeTaxThreshold = 100000

// autoTradeBlacklist é o conjunto de sIndex que o original recusa pôr à venda
// (_MSG_SendAutoTrade.cpp:85): itens de quest/vinculados/evento, que nunca podem
// ser negociados.
var autoTradeBlacklist = map[int16]bool{
	508: true, 3993: true, 747: true, 509: true, 522: true,
	526: true, 527: true, 528: true, 529: true, 530: true, 531: true, 446: true,
}

// inAutoTradeForbiddenRect diz se (x,y) está dentro do retângulo fixo onde não se
// abre loja dentro da vila (_MSG_SendAutoTrade.cpp:57).
func inAutoTradeForbiddenRect(x, y int16) bool {
	return x >= 2123 && x <= 2148 && y >= 2139 && y <= 2157
}

// autoTradeInRange diz se b está dentro da caixa de visão de autotrade de a.
func autoTradeInRange(a, b *world.Entity) bool {
	dx := int(a.X) - int(b.X)
	dy := int(a.Y) - int(b.Y)
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	return dx <= autoTradeViewRange && dy <= autoTradeViewRange
}

// sendAutoTrade trata _MSG_SendAutoTrade (0x0397): abre uma loja pessoal. Os itens
// são validados contra o Cargo da conta do vendedor (memcmp anti troca de item) e
// contra a blacklist de sIndex; a loja só abre em vila. No sucesso guarda a loja,
// marca TradeMode, envia ao dono a própria lista e multicasta a pose da banca.
func (d *Dispatcher) sendAutoTrade(w *world.World, s *world.Session, _ protocol.Header, payload []byte) {
	e := w.Entity(s.Conn)
	if e == nil || e.HP <= 0 || s.Mode != world.UserPlay {
		w.AddCrackError(s, 10, 88)
		d.removeTrade(w, s)
		return
	}
	// Não dá para abrir loja no meio de uma troca nem com uma já aberta.
	if s.Trade.Active || s.TradeMode != 0 {
		d.notify(w, s, NoticeCantAutoTrade)
		return
	}
	village := world.Village(e.X, e.Y)
	if village < 0 || village > 4 || inAutoTradeForbiddenRect(e.X, e.Y) {
		d.removeTrade(w, s)
		d.notify(w, s, NoticeOnlyVillage)
		return
	}
	var body protocol.MsgSendAutoTradeBody
	if err := body.Decode(payload); err != nil {
		return
	}
	cargo := w.Cargo(s.AccountID)
	if cargo == nil {
		// Sem armazém carregado — não há de onde vender. Fala em vez de sumir: uma
		// loja que não abre sem explicar é o defeito que o QA reporta como
		// "clicou e não aconteceu nada".
		d.sendClientMessage(w, s, tradeWrongPacket)
		return
	}

	// Valida cada slot montando a loja; nada é aplicado antes de a mensagem
	// inteira passar (valida-tudo-depois-aplica, como o loop + memcpy legado).
	shop := &world.AutoTradeState{Title: body.Title, Tax: world.CityTax(village)}
	for i := 0; i < protocol.MaxAutoTradeWire; i++ {
		ws := body.Slots[i]
		shop.Slots[i].CargoPos = -1
		if ws.Item.Index == 0 {
			if ws.Coin != 0 {
				return // moeda sem item (_MSG_SendAutoTrade.cpp:79)
			}
			continue
		}
		if ws.Coin <= 0 || ws.Coin > autoTradePriceMax {
			return // preço 0 ou acima do teto é recusado
		}
		if autoTradeBlacklist[ws.Item.Index] {
			return
		}
		pos := int(ws.CarryPos)
		if pos < 0 || pos >= world.MaxCargo {
			return
		}
		// EF_NOTRADE (127) nunca entra em loja: o efeito pode estar no item do cargo
		// ou no catálogo (itemEffects), então a pergunta é feita pelo mesmo
		// `itemAbility` que o trade usa. Sem isto, um item de quest/único virava
		// mercadoria e o comprador recebia algo que o servidor recusa equipar.
		if d.itemAbility(cargo.Items[pos], efNoTrade) != 0 {
			return
		}
		// memcmp da oferta contra o item real do Cargo (anti troca de item). Também
		// bloqueia listar o mesmo slot duas vezes: um slot posterior que reusasse pos
		// ainda casaria com o item (inalterado) do Cargo, mas o caminho de compra
		// limpa os dois na venda, então a referência duplicada esvazia na primeira
		// compra.
		if !sameItem(ws.Item, cargo.Items[pos]) {
			d.removeTrade(w, s)
			return
		}
		shop.Slots[i].Item = cargo.Items[pos]
		shop.Slots[i].CargoPos = pos
		shop.Slots[i].Price = ws.Coin
	}

	s.AutoTrade = shop
	s.TradeMode = 1
	d.sendShopList(w, s, s.Conn) // SendAutoTrade(conn, conn): devolve ao dono a própria lista
	d.sendShopPose(w, s, e)
	d.log.Info("autotrade opened", "conn", s.Conn, "title", shop.Title, "tax", shop.Tax)
}

// reqTradeList trata _MSG_ReqTradeList (0x039A): navega na loja de outro jogador.
func (d *Dispatcher) reqTradeList(w *world.World, s *world.Session, _ protocol.Header, payload []byte) {
	e := w.Entity(s.Conn)
	if e == nil || e.HP == 0 || s.Mode != world.UserPlay {
		w.AddCrackError(s, 10, 87)
		return
	}
	autoID, ok := protocol.StandardParm(payload)
	if !ok || autoID <= 0 || int(autoID) >= world.MaxUser {
		return
	}
	seller := w.Session(int(autoID))
	te := w.Entity(int(autoID))
	if seller == nil || te == nil || seller.TradeMode == 0 || seller.AutoTrade == nil {
		return
	}
	if !autoTradeInRange(e, te) {
		d.log.Info("autotrade list too far", "conn", s.Conn, "seller", autoID)
		d.notify(w, s, NoticeCantUseHere)
		return
	}
	d.sendShopList(w, s, int(autoID))
}

// reqBuy trata _MSG_ReqBuy (0x0398): compra um item de uma loja. A transação
// inteira roda numa única passada do loop (valida-tudo-depois-aplica): o item vai
// do Cargo do vendedor para o Carry do comprador, o ouro vai da moeda do
// comprador para a moeda do Cargo do vendedor (menos o imposto da cidade) e os
// slots da loja/Cargo são limpos. Como o loop serializa eventos, um segundo
// comprador do mesmo slot o encontra vazio.
func (d *Dispatcher) reqBuy(w *world.World, s *world.Session, _ protocol.Header, payload []byte) {
	e := w.Entity(s.Conn)
	if e == nil || e.HP == 0 || s.Mode != world.UserPlay {
		w.AddCrackError(s, 10, 86)
		d.removeTrade(w, s)
		return
	}
	if s.TradeMode != 0 || s.Trade.Active {
		return // quem vende ou troca não pode comprar
	}
	var m protocol.MsgReqBuyBody
	if err := m.Decode(payload); err != nil {
		return
	}
	targetID := int(m.TargetID)
	if targetID <= 0 || targetID >= world.MaxUser {
		return
	}
	seller := w.Session(targetID)
	te := w.Entity(targetID)
	if seller == nil || te == nil || seller.AutoTrade == nil || seller.TradeMode == 0 {
		return
	}
	if !autoTradeInRange(e, te) {
		d.log.Info("autotrade buy too far", "conn", s.Conn, "seller", targetID)
		d.notify(w, s, NoticeCantUseHere)
		return
	}
	pos := int(m.Pos)
	if pos < 0 || pos >= world.MaxAutoTrade {
		return
	}
	slot := &seller.AutoTrade.Slots[pos]
	cpos := slot.CargoPos
	if cpos < 0 || cpos >= world.MaxCargo || slot.Item.Empty() {
		return // já vendido / slot vazio
	}
	// Anti-adulteração: a oferta (imposto + preço + item) precisa casar exatamente
	// E a oferta guardada precisa ainda casar com o item vivo do Cargo (dois memcmp,
	// _MSG_ReqBuy.cpp:72-88).
	sellerCargo := w.Cargo(seller.AccountID)
	if sellerCargo == nil {
		return
	}
	cargoItem := sellerCargo.Items[cpos]
	if m.Tax != int32(seller.AutoTrade.Tax) || m.Price != slot.Price ||
		!sameItem(m.Item, slot.Item) || !itemsEqual(slot.Item, cargoItem) {
		// Anti-tamper: a oferta não confere com a loja. O legado fecha o comércio em
		// silêncio; aqui o jogador recebe _NN_Wrong_Trade_Packet1 (80), senão a
		// compra recusada parece "o servidor travou".
		d.sendClientMessage(w, s, tradeWrongPacket)
		d.removeTrade(w, s)
		return
	}
	if e.Coin < m.Price {
		d.notify(w, s, NoticeNotEnoughMoney)
		return
	}
	if int64(sellerCargo.Coin)+int64(m.Price) > maxCoin {
		d.notify(w, s, NoticeCantGetMore2G)
		return
	}
	dst := firstFreeTradeSlot(e)
	if dst < 0 {
		d.notify(w, s, NoticeNoSpaceToTrade)
		return
	}

	// Aplica de forma atômica. O imposto só incide a partir do limiar
	// (_MSG_ReqBuy.cpp:138).
	e.Carry[dst] = cargoItem
	e.Coin -= m.Price
	var imposto int32
	if m.Price >= autoTradeTaxThreshold {
		imposto = (m.Price / 100) * m.Tax
	}
	sellerCargo.Coin += m.Price - imposto
	sellerCargo.Items[cpos] = world.Item{}
	*slot = world.AutoTradeSlot{CargoPos: -1} // limpa o slot CERTO (não repete o bug de memset do legado)

	// S→C: o comprador recebe o item + a moeda atualizada; o vendedor recebe o slot
	// do cargo limpo + a moeda do cargo; quem está vendo a loja recebe
	// _MSG_ItemSold para tirar o item da listagem.
	w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceCarry, dst, itemToSel(cargoItem)))
	d.sendEtc(w, s, e)
	w.Send(seller, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceCargo, cpos, protocol.SelItem{}))
	w.Send(seller, protocol.MsgUpdateCargoCoin, protocol.EncodeUpdateCargoCoin(sellerCargo.Coin))
	w.BroadcastInView(targetID, protocol.MsgItemSold, protocol.EncodeStandardParm2(int32(targetID), int32(pos)))
	d.notify(w, seller, NoticeItemSold)
	d.log.Info("autotrade buy", "buyer", s.Conn, "seller", targetID, "item", cargoItem.Index, "price", m.Price, "tax", imposto)
}

// sendShopList envia para s a loja de sellerConn (SendFunc.cpp:SendAutoTrade).
// Reaproveita a struct S→C de MSG_SendAutoTrade com a esquisitice de Size=528 no
// wire, ID igual ao id de cena e Index igual à conexão do vendedor.
func (d *Dispatcher) sendShopList(w *world.World, s *world.Session, sellerConn int) {
	seller := w.Session(sellerConn)
	if seller == nil || seller.AutoTrade == nil {
		return
	}
	at := seller.AutoTrade
	body := protocol.MsgSendAutoTradeBody{Title: at.Title, Tax: at.Tax, Index: int16(sellerConn)}
	for i := range at.Slots {
		sl := at.Slots[i]
		if sl.CargoPos < 0 || sl.Item.Empty() {
			body.Slots[i].CarryPos = -1
			continue
		}
		body.Slots[i].Item = wireFromItem(sl.Item)
		body.Slots[i].CarryPos = int8(sl.CargoPos)
		body.Slots[i].Coin = sl.Price
	}
	w.SendTo(s, protocol.Header{Type: protocol.MsgSendAutoTrade, ID: protocol.IDScene}, body.EncodeList())
}

// sendShopPose mostra o vendedor na pose da banca para si e para quem está na
// visão (_MSG_SendAutoTrade.cpp:112-120). A pose é o Type MSG_CreateMobTrade, não
// um valor de CreateType; Score.Con é zerado por paridade.
func (d *Dispatcher) sendShopPose(w *world.World, s *world.Session, e *world.Entity) {
	data := d.createMobFrom(e, 0)
	data.Con = 0 // _MSG_SendAutoTrade.cpp:118
	tab := make([]byte, 26)
	body := protocol.EncodeCreateMobTradeBody(data, tab, s.AutoTrade.Title)
	w.SendTo(s, protocol.Header{Type: protocol.MsgCreateMobTrade, ID: protocol.IDScene}, body)
	w.BroadcastInView(s.Conn, protocol.MsgCreateMobTrade, body)
}

// closeAutoTrade fecha uma loja pessoal aberta: limpa o estado, fecha a UI da loja
// no dono (sinal _MSG_QuitTrade) e reemite um MSG_CreateMob NORMAL, para a pose da
// banca voltar ao normal no dono e em quem está na visão (RemoveTrade,
// Server.cpp:8138-8145). Não faz nada quando não há loja aberta. Chamada por
// removeTrade (e portanto por todo gatilho de RemoveTrade: sair da troca, andar,
// comprar, operações de item, …).
func (d *Dispatcher) closeAutoTrade(w *world.World, s *world.Session) {
	if s.AutoTrade == nil && s.TradeMode == 0 {
		return
	}
	s.AutoTrade = nil
	s.TradeMode = 0
	w.Send(s, protocol.MsgQuitTrade, nil)
	e := w.Entity(s.Conn)
	if e == nil || e.Mode != world.MobUser {
		return
	}
	body := protocol.EncodeCreateMobBody(d.createMobFrom(e, 0))
	w.SendTo(s, protocol.Header{Type: protocol.MsgCreateMob, ID: protocol.IDScene}, body)
	w.BroadcastInView(s.Conn, protocol.MsgCreateMob, body)
}

// firstFreeTradeSlot devolve o primeiro slot vazio do Carry na região
// negociável atualmente liberada, ou -1 se estiver cheia.
func firstFreeTradeSlot(e *world.Entity) int {
	for i := 0; i < activeCarryLimit(e); i++ {
		if e.Carry[i].Empty() {
			return i
		}
	}
	return -1
}

// wireFromItem converte um item do mundo para a forma STRUCT_ITEM do wire (WireItem).
func wireFromItem(it world.Item) protocol.WireItem {
	wi := protocol.WireItem{Index: it.Index}
	for i := 0; i < 3; i++ {
		wi.Effects[i] = protocol.WireEffect{Effect: it.Effects[i].Effect, Value: it.Effects[i].Value}
	}
	return wi
}

// itemsEqual diz se dois itens do mundo são idênticos byte a byte (índice +
// efeitos) — o memcmp que confirma que a oferta da loja ainda casa com o slot vivo
// do Cargo.
func itemsEqual(a, b world.Item) bool {
	if a.Index != b.Index {
		return false
	}
	for i := 0; i < 3; i++ {
		if a.Effects[i] != b.Effects[i] {
			return false
		}
	}
	return true
}
