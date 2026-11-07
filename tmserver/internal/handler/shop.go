package handler

import (
	"encoding/binary"

	"openwyd-nextgen/tmserver/internal/protocol"
	"openwyd-nextgen/tmserver/internal/world"
)

// reqShopList trata _MSG_REQShopList (0x027B): o cliente clicou num NPC
// mercador. A lista da loja é o próprio Carry[] do NPC (SendFunc.cpp:SendShopList).
// _MSG_REQShopList.cpp: Merchant 1 → ShopType 1, Merchant 19 → ShopType 3.
func (d *Dispatcher) reqShopList(w *world.World, s *world.Session, _ protocol.Header, payload []byte) {
	if s.Mode != world.UserPlay || len(payload) < 2 {
		return
	}
	target := int(binary.LittleEndian.Uint16(payload[0:2]))
	npc := w.Entity(target)
	if npc == nil || npc.Mode == world.MobEmpty || npc.Merchant == 0 {
		return // não é um mercador
	}
	// Merchant==2 é o guarda de cargo (Guarda-Carga): ele abre o
	// armazém da conta, não uma lista de compra/venda. UNVERIFIED: a marcação Merchant==2 dos
	// NPCs de tmserver/content/ ainda não está confirmada por captura.
	if npc.Merchant == 2 {
		d.openCargo(w, s)
		return
	}
	shopType := int32(1)
	if npc.Merchant == 19 {
		shopType = 3
	}
	var list [27]protocol.SelItem
	for i := 0; i < 27; i++ {
		c := npc.Carry[protocol.ShopSlot(i)]
		list[i] = protocol.SelItem{
			Index: uint16(c.Index),
			Eff: [3][2]uint8{
				{c.Effects[0].Effect, c.Effects[0].Value},
				{c.Effects[1].Effect, c.Effects[1].Value},
				{c.Effects[2].Effect, c.Effects[2].Value},
			},
		}
	}
	body := protocol.EncodeShopListBody(shopType, list, 0) // Tax 0 (tabela de imposto de cidade UNVERIFIED)
	w.SendTo(s, protocol.Header{Type: protocol.MsgShopList, ID: protocol.IDScene}, body)
	d.log.Info("shop opened", "conn", s.Conn, "npc", target, "merchant", npc.Merchant)
}

// buy trata _MSG_Buy (0x0379): compra um item de loja de um NPC. Só o
// mercador simples (Merchant 1) em view vende (merchantInView). Preço =
// itemPrices[index] (sem imposto de cidade — atalho de vila). O original aceita
// Price==0 (item grátis) e só rejeita preço negativo / ouro insuficiente.
// Debita o ouro, põe o item no Carry do jogador, ecoa MSG_Buy (novo Coin) +
// MSG_UpdateEtc.
func (d *Dispatcher) buy(w *world.World, s *world.Session, _ protocol.Header, payload []byte) {
	if s.Mode != world.UserPlay || len(payload) < 6 {
		return
	}
	target := int(binary.LittleEndian.Uint16(payload[0:2]))
	npcPos := int(int16(binary.LittleEndian.Uint16(payload[2:4])))
	myPos := int(int16(binary.LittleEndian.Uint16(payload[4:6])))
	npc := w.Entity(target)
	e := w.Entity(s.Conn)
	if npc == nil || e == nil || target < world.MaxUser {
		return
	}
	if npcPos < 0 || npcPos >= world.MaxCarry || !carrySlotAccessible(e, myPos) {
		return
	}
	if !d.merchantInView(w, s, e, npc, onlyMerchantSells) {
		return
	}
	item := npc.Carry[npcPos]
	if item.Index == 0 {
		return // slot vazio da loja
	}
	// Slot de destino ocupado: ressincroniza o cliente com o que REALMENTE está naquele slot,
	// exatamente como o original (_MSG_Buy.cpp:158-162). Sem isso o cliente mantém uma
	// visão "vazia" obsoleta (por exemplo, um item de classe/corpo que ele não renderiza na bolsa) e
	// repete o mesmo slot ocupado para sempre — toda compra falha em silêncio (B11).
	if e.Carry[myPos].Index != 0 {
		d.log.Info("buy resync (dest occupied)", "conn", s.Conn, "npcPos", npcPos, "wantItem", item.Index, "myPos", myPos, "destItem", e.Carry[myPos].Index)
		w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceCarry, myPos, itemToSel(e.Carry[myPos])))
		return
	}
	price, ok := d.itemPrices[int(item.Index)]
	if !ok || price < 0 || price > e.Coin {
		d.log.Info("buy denied", "conn", s.Conn, "item", item.Index, "price", price, "gold", e.Coin)
		// Só a falta de gold responde no legado (_MSG_Buy.cpp:147-151); preço
		// negativo e item sem preço voltavam em silêncio. Aqui os três casos falam:
		// o jogador precisa saber que a compra foi recusada, não que o NPC travou.
		switch {
		case ok && price >= 0:
			d.sendClientMessage(w, s, notEnoughMoney)
		default:
			d.sendClientMessage(w, s, tradeWrongPacket)
		}
		return
	}
	e.Coin -= price
	e.Carry[myPos] = item
	d.log.Info("buy ok", "conn", s.Conn, "npcPos", npcPos, "item", item.Index, "myPos", myPos, "price", price, "gold", e.Coin)
	// Responde na ordem EXATA do original (_MSG_Buy.cpp:271-296): o eco de MSG_Buy
	// (ID=ESCENE_FIELD, novo Coin) primeiro, depois SendEtc, e o SendItem POR ÚLTIMO —
	// o cliente confirma o item comprado na bolsa no SendItem, então ele precisa chegar
	// depois do ack de compra/ouro ou o item aparece uma compra atrás (B?).
	echo := make([]byte, len(payload))
	copy(echo, payload)
	if len(echo) >= 12 {
		binary.LittleEndian.PutUint32(echo[8:12], uint32(e.Coin)) // Coin @body8
	}
	w.SendTo(s, protocol.Header{Type: protocol.MsgBuy, ID: protocol.IDScene}, echo)
	d.sendEtc(w, s, e)
	w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceCarry, myPos, itemToSel(item)))
}

// merchantInView é o gate que _MSG_Buy.cpp:55-64 e _MSG_Sell.cpp:103-112 compartilham:
// só Merchant 1 negocia itens. Todo outro NPC com tag Merchant (o
// mestre de skills 19, NPCs de quest 100+) ainda abre sua própria janela via REQShopList,
// mas seu Carry não está à venda — comprar do mestre de skills Foema entregava
// a skill como item grátis (#391). Um NPC fora do alcance GetInView (±VIEWGRIDX)
// fecha a loja em vez disso. Informa se a negociação pode seguir.
func (d *Dispatcher) merchantInView(w *world.World, s *world.Session, e, npc *world.Entity, refusal string) bool {
	if npc.Merchant != 1 {
		d.sendClientMessage(w, s, refusal)
		return false
	}
	if chebyshev(e.X, e.Y, npc.X, npc.Y) > world.NoViewRange {
		w.SendTo(s, protocol.Header{Type: protocol.MsgCloseShop, ID: protocol.IDScene}, nil)
		return false
	}
	return true
}

// itemToSel converte um item de inventário do mundo para a forma STRUCT_ITEM do wire.
func itemToSel(it world.Item) protocol.SelItem {
	return protocol.SelItem{
		Index: uint16(it.Index),
		Eff: [3][2]uint8{
			{it.Effects[0].Effect, it.Effects[0].Value},
			{it.Effects[1].Effect, it.Effects[1].Value},
			{it.Effects[2].Effect, it.Effects[2].Value},
		},
	}
}

// sell trata _MSG_Sell (0x037A): vende um item do Carry a um NPC. Só o
// mercador simples em view compra (merchantInView), exceto uma Huntress com a
// skill 0x1000, que vende em qualquer lugar (_MSG_Sell.cpp:94-95). Preço de venda =
// Price/4 (→/2 se >10000, →*2/3 se 5000<x<=10000); sem imposto de cidade. Credita ouro,
// limpa o slot, ecoa MSG_Sell + MSG_UpdateEtc. Só MyType=1 (Carry) por enquanto.
func (d *Dispatcher) sell(w *world.World, s *world.Session, _ protocol.Header, payload []byte) {
	if s.Mode != world.UserPlay || len(payload) < 6 {
		return
	}
	target := int(binary.LittleEndian.Uint16(payload[0:2]))
	myType := int(int16(binary.LittleEndian.Uint16(payload[2:4])))
	myPos := int(int16(binary.LittleEndian.Uint16(payload[4:6])))
	npc := w.Entity(target)
	e := w.Entity(s.Conn)
	if e == nil || myType != 1 {
		return // só venda de inventário (Carry) suportada nesta passagem
	}
	if !carrySlotAccessible(e, myPos) {
		return
	}
	if e.Class != 3 || e.LearnedSkill&0x1000 == 0 {
		if npc == nil || target < world.MaxUser || !d.merchantInView(w, s, e, npc, onlyMerchantBuyYours) {
			return
		}
	}
	if e.Carry[myPos].Index == 0 {
		return
	}
	price := d.itemPrices[int(e.Carry[myPos].Index)]
	sp := price / 4
	if sp > 10000 {
		sp /= 2
	} else if sp > 5000 {
		sp = 2 * sp / 3
	}
	e.Coin += sp
	e.Carry[myPos] = world.Item{}
	d.log.Info("sell ok", "conn", s.Conn, "slot", myPos, "gain", sp, "gold", e.Coin)
	w.SendTo(s, protocol.Header{Type: protocol.MsgSell, ID: protocol.IDScene}, payload)
	// Limpa o slot vendido no cliente (sIndex 0) + atualiza o ouro.
	w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceCarry, myPos, protocol.SelItem{}))
	d.sendEtc(w, s, e)
}
