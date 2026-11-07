package handler

import (
	"encoding/binary"
	"testing"

	"openwyd-nextgen/tmserver/internal/protocol"
	"openwyd-nextgen/tmserver/internal/world"
)

// autotradeDB coloca um vendedor (tester, conn 1) e um comprador (tradeb, conn 2)
// no mundo com HP>0 (o login cria os dois em Armia, uma vila, dentro da visão). O
// Cargo da conta do vendedor tem um item para pôr à venda; o comprador leva ouro.
func autotradeDB(sellItem int16) *fakeDB {
	db := newDB()
	db.loads = map[int64]world.CharacterState{
		7:  {Slot: 0, Name: "Seller", HP: 1000, MaxHP: 1000, Coin: 1000},
		11: {Slot: 0, Name: "Buyer", HP: 1000, MaxHP: 1000, Coin: 1_000_000},
	}
	var cargo world.CargoState
	cargo.Items[0] = world.Item{Index: sellItem}
	db.accounts["tester"].cargo = cargo
	return db
}

func autotradeViewDB(sellItem int16) *fakeDB {
	db := newDB()
	db.loads = map[int64]world.CharacterState{
		7:  {Slot: 0, Name: "Seller", X: 2086, Y: 2093, HP: 1000, MaxHP: 1000, Coin: 1000},
		11: {Slot: 0, Name: "Buyer", X: 2086, Y: 2130, HP: 1000, MaxHP: 1000, Coin: 1_000_000},
	}
	var cargo world.CargoState
	cargo.Items[0] = world.Item{Index: sellItem}
	db.accounts["tester"].cargo = cargo
	return db
}

func createMobTradeFields(t *testing.T, payload []byte) (x, y int16, mobID int, title string) {
	t.Helper()
	if len(payload) != 240 {
		t.Fatalf("CreateMobTrade body = %d bytes, want 240", len(payload))
	}
	x = int16(binary.LittleEndian.Uint16(payload[0:2]))
	y = int16(binary.LittleEndian.Uint16(payload[2:4]))
	mobID = int(binary.LittleEndian.Uint16(payload[4:6]))
	title = cstr(payload[216:240])
	return x, y, mobID, title
}

// openShopPayload monta um MSG_SendAutoTrade do cliente vendendo Cargo[cargoPos]
// pelo preço informado no slot 0 da loja.
func openShopPayload(title string, item int16, cargoPos int, price int32) []byte {
	body := protocol.MsgSendAutoTradeBody{Title: title}
	for i := range body.Slots {
		body.Slots[i].CarryPos = -1
	}
	body.Slots[0] = protocol.AutoTradeWireItem{
		Item:     protocol.WireItem{Index: item},
		CarryPos: int8(cargoPos),
		Coin:     price,
	}
	return body.Encode()
}

// reqBuyPayload monta um MSG_ReqBuy do cliente para o slot pos da loja do
// vendedor targetID.
func reqBuyPayload(targetID, pos int, item int16, price, tax int32) []byte {
	body := protocol.MsgReqBuyBody{
		Pos:      int32(pos),
		TargetID: uint16(targetID),
		Price:    price,
		Tax:      tax,
		Item:     protocol.WireItem{Index: item},
	}
	return body.Encode()
}

func TestAutoTradeShopPoseWhenBuyerEntersView(t *testing.T) {
	const sellItem = int16(1030)
	const title = "Loja Longe"
	addr, stop := startServerView(t, autotradeViewDB(sellItem), world.DefaultGridDim)
	defer stop()
	seller := enterWorldAs(t, addr, "tester") // conn 1, inside Armia at (2086,2093)
	defer seller.Close()
	buyer := enterWorldAs(t, addr, "tradeb") // conn 2, outside ViewRange at (2086,2130)
	defer buyer.Close()
	drainRaw(t, seller)
	drainRaw(t, buyer)

	send(t, seller, protocol.MsgSendAutoTrade, openShopPayload(title, sellItem, 0, 1000))
	readUntil(t, seller, protocol.MsgSendAutoTrade)
	if ty, _, ok := readMaybeRaw(t, buyer); ok {
		t.Fatalf("out-of-view buyer got %#x after shop opened, want no shop pose yet", ty)
	}

	actionFrameXY(t, buyer, protocol.MsgAction, serverTime, 2086, 2130, 2086, 2100)
	ty, payload, ok := readMaybeRaw(t, buyer)
	if !ok || ty != protocol.MsgCreateMobTrade {
		t.Fatalf("buyer entering view got %#x ok=%v, want CreateMobTrade", ty, ok)
	}
	if x, y, id, gotTitle := createMobTradeFields(t, payload); id != 1 || x != 2086 || y != 2093 || gotTitle != title {
		t.Fatalf("shop pose = id %d at (%d,%d) title %q, want id 1 at (2086,2093) title %q", id, x, y, gotTitle, title)
	}
	if ty, _, ok = readMaybeRaw(t, buyer); !ok || ty != protocol.MsgPKInfo {
		t.Fatalf("frame after shop pose = %#x ok=%v, want PKInfo", ty, ok)
	}
}

func TestAutoTradeOpenBrowseBuy(t *testing.T) {
	const sellItem = int16(1030)
	const price, tax = int32(200_000), int32(5)
	addr, stop, _ := startServerClock(t, autotradeDB(sellItem))
	defer stop()
	seller := enterWorldAs(t, addr, "tester") // conn 1
	defer seller.Close()
	buyer := enterWorldAs(t, addr, "tradeb") // conn 2
	defer buyer.Close()

	// O vendedor abre a loja; o servidor devolve ao dono a própria lista.
	send(t, seller, protocol.MsgSendAutoTrade, openShopPayload("Minha Loja", sellItem, 0, price))
	list, _ := readUntil(t, seller, protocol.MsgSendAutoTrade)
	if got := cstr(list[0:24]); got != "Minha Loja" {
		t.Errorf("own list title = %q, want Minha Loja", got)
	}
	if got := int16(binary.LittleEndian.Uint16(list[24:26])); got != sellItem {
		t.Errorf("own list item = %d, want %d", got, sellItem)
	}

	// O comprador navega na loja do vendedor (Parm = conexão 1 do vendedor).
	send(t, buyer, protocol.MsgReqTradeList, protocol.EncodeStandardParm(1))
	blist, _ := readUntil(t, buyer, protocol.MsgSendAutoTrade)
	if got := int16(binary.LittleEndian.Uint16(blist[24:26])); got != sellItem {
		t.Errorf("browsed list item = %d, want %d", got, sellItem)
	}
	if got := int32(binary.LittleEndian.Uint32(blist[132:136])); got != price {
		t.Errorf("browsed list price = %d, want %d", got, price)
	}

	// O comprador compra o slot 0.
	send(t, buyer, protocol.MsgReqBuy, reqBuyPayload(1, 0, sellItem, price, tax))

	// O comprador recebe o item (MSG_SendItem: invType@0, slot@2, item.Index@4).
	si, _ := readUntil(t, buyer, protocol.MsgSendItem)
	if place := binary.LittleEndian.Uint16(si[0:2]); place != protocol.ItemPlaceCarry {
		t.Errorf("bought item place = %d, want Carry(%d)", place, protocol.ItemPlaceCarry)
	}
	if got := int16(binary.LittleEndian.Uint16(si[4:6])); got != sellItem {
		t.Errorf("bought item index = %d, want %d", got, sellItem)
	}
	// O ouro do comprador é debitado (MSG_UpdateEtc Coin @body28).
	etc, _ := readUntil(t, buyer, protocol.MsgUpdateEtc)
	if coin := int32(binary.LittleEndian.Uint32(etc[28:32])); coin != 1_000_000-price {
		t.Errorf("buyer coin = %d, want %d", coin, 1_000_000-price)
	}

	// O vendedor recebe o slot do cargo limpo + a moeda do cargo já com o imposto.
	// Imposto: (price/100)*tax = 10000; líquido = 190000.
	scc, _ := readUntil(t, seller, protocol.MsgUpdateCargoCoin)
	if coin := int32(binary.LittleEndian.Uint32(scc[0:4])); coin != price-(price/100)*tax {
		t.Errorf("seller cargo coin = %d, want %d", coin, price-(price/100)*tax)
	}
}

// TestAutoTradeBuyNoDup: uma vez comprado, o slot é limpo, então uma segunda compra
// do mesmo slot não entrega nada (o loop serializa as compras — sem duplicação).
func TestAutoTradeBuyNoDup(t *testing.T) {
	const sellItem = int16(1030)
	const price, tax = int32(50_000), int32(5)
	addr, stop, _ := startServerClock(t, autotradeDB(sellItem))
	defer stop()
	seller := enterWorldAs(t, addr, "tester")
	defer seller.Close()
	buyer := enterWorldAs(t, addr, "tradeb")
	defer buyer.Close()

	send(t, seller, protocol.MsgSendAutoTrade, openShopPayload("Loja", sellItem, 0, price))
	readUntil(t, seller, protocol.MsgSendAutoTrade)

	// A primeira compra passa → item entregue.
	send(t, buyer, protocol.MsgReqBuy, reqBuyPayload(1, 0, sellItem, price, tax))
	si, _ := readUntil(t, buyer, protocol.MsgSendItem)
	if got := int16(binary.LittleEndian.Uint16(si[4:6])); got != sellItem {
		t.Fatalf("first buy item = %d, want %d", got, sellItem)
	}
	readUntil(t, buyer, protocol.MsgUpdateEtc)
	// O comprador está na visão do vendedor, então também recebe o broadcast
	// ItemSold da própria compra — drena para a asserção seguinte ver o socket limpo.
	readUntil(t, buyer, protocol.MsgItemSold)

	// A segunda compra do mesmo slot (agora vazio) não entrega nada.
	send(t, buyer, protocol.MsgReqBuy, reqBuyPayload(1, 0, sellItem, price, tax))
	if ty, _, ok := readMaybe(t, buyer); ok {
		t.Fatalf("second buy produced a %#x frame, want none (slot already sold)", ty)
	}
}

// TestAutoTradeCloseOnQuit: mandar _MSG_QuitTrade fecha a loja aberta via
// RemoveTrade — o dono recebe o sinal QuitTrade e um CreateMob normal reemitido, e
// depois navegar na loja (agora fechada) não devolve nada.
func TestAutoTradeCloseOnQuit(t *testing.T) {
	const sellItem = int16(1030)
	addr, stop, _ := startServerClock(t, autotradeDB(sellItem))
	defer stop()
	seller := enterWorldAs(t, addr, "tester")
	defer seller.Close()
	buyer := enterWorldAs(t, addr, "tradeb")
	defer buyer.Close()

	send(t, seller, protocol.MsgSendAutoTrade, openShopPayload("Loja", sellItem, 0, 1000))
	readUntil(t, seller, protocol.MsgSendAutoTrade)
	// O comprador, na visão, recebeu o CreateMobTrade da pose da banca na abertura —
	// drena para a asserção pós-fechamento ver o socket limpo.
	readUntil(t, buyer, protocol.MsgCreateMobTrade)

	// Fechar (QuitTrade → removeTrade → closeAutoTrade) sinaliza o dono.
	send(t, seller, protocol.MsgQuitTrade, nil)
	readUntil(t, seller, protocol.MsgQuitTrade)

	// A loja sumiu: navegar não devolve nada.
	send(t, buyer, protocol.MsgReqTradeList, protocol.EncodeStandardParm(1))
	if ty, _, ok := readMaybe(t, buyer); ok {
		t.Fatalf("browse after close produced a %#x frame, want none (shop closed)", ty)
	}
}

// TestAutoTradeOpenRejectsNoTrade: um item com EF_NOTRADE (127) no próprio cargo
// não pode ser listado. É a metade que faltava da checagem que o trade já fazia:
// o carimbo do efeito vale mesmo quando o catálogo não conhece o sIndex.
func TestAutoTradeOpenRejectsNoTrade(t *testing.T) {
	const sellItem = int16(1030)
	db := autotradeDB(sellItem)
	db.accounts["tester"].cargo.Items[0].Effects[0] = world.Effect{Effect: efNoTrade, Value: 1}
	addr, stop, _ := startServerClock(t, db)
	defer stop()
	seller := enterWorldAs(t, addr, "tester")
	defer seller.Close()

	send(t, seller, protocol.MsgSendAutoTrade, openShopPayload("Loja", sellItem, 0, 1000))
	if ty, _, ok := readMaybe(t, seller); ok {
		t.Fatalf("EF_NOTRADE open produced a %#x frame, want none (shop refused)", ty)
	}
}

// TestAutoTradeOpenRejectsBlacklist: um sIndex da blacklist é recusado, então
// nenhuma loja abre (o dono não recebe a lista de volta).
func TestAutoTradeOpenRejectsBlacklist(t *testing.T) {
	const blacklisted = int16(508)
	addr, stop, _ := startServerClock(t, autotradeDB(blacklisted))
	defer stop()
	seller := enterWorldAs(t, addr, "tester")
	defer seller.Close()

	send(t, seller, protocol.MsgSendAutoTrade, openShopPayload("Loja", blacklisted, 0, 1000))
	if ty, _, ok := readMaybe(t, seller); ok {
		t.Fatalf("blacklisted open produced a %#x frame, want none (shop refused)", ty)
	}
}
