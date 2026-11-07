package handler

import (
	"encoding/binary"
	"net"
	"testing"
	"time"

	"openwyd-nextgen/tmserver/internal/protocol"
	"openwyd-nextgen/tmserver/internal/world"
)

// tradeDB dá às duas contas inventários distintos de um item só, para uma troca ser
// observável: tester(id 7) carrega o item 1100, tradeb(id 11) carrega o item 2200.
func tradeDB() *fakeDB {
	db := newDB()
	mk := func(idx int16) world.CharacterState {
		st := world.CharacterState{Slot: 0, Name: "Hero", X: 5, Y: 5, HP: 1000, MaxHP: 1000, Coin: 1000}
		st.Carry[0] = world.Item{Index: idx}
		return st
	}
	db.loads = map[int64]world.CharacterState{7: mk(1100), 11: mk(2200)}
	return db
}

func enterWorldAs(t *testing.T, addr, account string) net.Conn {
	t.Helper()
	c := dial(t, addr)
	if ty := loginRetry(t, c, account); ty != protocol.MsgCNFAccountLogin {
		t.Fatalf("login %s failed: %#x", account, ty)
	}
	var body protocol.MsgCharacterLoginBody
	send(t, c, protocol.MsgCharacterLogin, body.Encode())
	if ty, _ := read(t, c); ty != protocol.MsgCNFCharacterLogin {
		t.Fatalf("char login %s failed: %#x", account, ty)
	}
	drainLoginScore(t, c)
	return c
}

// tradeBody monta um corpo MSG_Trade ofertando item (do slot de carry) e dinheiro ao
// oponente; um item zero deixa toda entrada vazia (InvenPos 0xFF, o -1 do legado).
func tradeBody(opponent int, item world.Item, slot int, money int32, check bool) protocol.MsgTradeBody {
	var body protocol.MsgTradeBody
	for i := range body.InvenPos {
		body.InvenPos[i] = tradeEmptyPos
	}
	if item.Index != 0 {
		body.Item[0] = protocol.WireItem{Index: item.Index}
		body.InvenPos[0] = byte(slot)
	}
	body.TradeMoney = money
	if check {
		body.MyCheck = 1
	}
	body.OpponentID = uint16(opponent)
	return body
}

func sendTrade(t *testing.T, c net.Conn, body protocol.MsgTradeBody) {
	t.Helper()
	send(t, c, protocol.MsgTrade, body.Encode())
}

// readForwarded lê o próximo frame em c e exige que seja um MSG_Trade
// repassado no layout clássico.
func readForwarded(t *testing.T, c net.Conn) protocol.MsgTradeBody {
	t.Helper()
	ty, p, ok := readMaybe(t, c)
	if !ok || ty != protocol.MsgTrade {
		t.Fatalf("got %#x ok=%v, want forwarded MsgTrade", ty, ok)
	}
	if len(p) != protocol.MsgTradeBodySize {
		t.Fatalf("forwarded body = %d bytes, want %d", len(p), protocol.MsgTradeBodySize)
	}
	var body protocol.MsgTradeBody
	if err := body.Decode(p); err != nil {
		t.Fatal(err)
	}
	return body
}

func expectType(t *testing.T, c net.Conn, want protocol.Type) []byte {
	t.Helper()
	ty, p, ok := readMaybe(t, c)
	if !ok || ty != want {
		t.Fatalf("got %#x ok=%v, want %#x", ty, ok, want)
	}
	return p
}

func expectSilence(t *testing.T, c net.Conn) {
	t.Helper()
	if ty, _, ok := readMaybe(t, c); ok {
		t.Fatalf("unexpected frame %#x", ty)
	}
}

// openTrade faz a e b trocarem suas ofertas iniciais: a (conn 1) oferece
// aItem/aMoney, b (conn 2) responde com bItem/bMoney; cada oferta chega só
// ao outro lado.
func openTrade(t *testing.T, a, b net.Conn, aItem, bItem world.Item, aMoney, bMoney int32) {
	t.Helper()
	sendTrade(t, a, tradeBody(2, aItem, 0, aMoney, false))
	readForwarded(t, b)
	sendTrade(t, b, tradeBody(1, bItem, 0, bMoney, false))
	readForwarded(t, a)
	expectSilence(t, a)
	expectSilence(t, b)
}

func carryCoin(payload []byte) int32 {
	return int32(binary.LittleEndian.Uint32(payload[64*protocol.ItemSize:]))
}

func TestTradeSlotsAccessibleRequiresUnlockedCarry(t *testing.T) {
	e := &world.Entity{}
	e.Carry[44] = world.Item{Index: 1100}
	e.Carry[45] = world.Item{Index: 2200}

	if tradeSlotsAccessible(e, []int{44}) {
		t.Fatal("slot 44 was tradeable without a Wanderer Bag marker")
	}

	e.Carry[wandererBagSlot1] = world.Item{Index: itemWandererBag}
	if !tradeSlotsAccessible(e, []int{44}) {
		t.Fatal("slot 44 was not tradeable with one active Wanderer Bag marker")
	}
	if tradeSlotsAccessible(e, []int{45}) {
		t.Fatal("slot 45 was tradeable with only one active Wanderer Bag marker")
	}

	e.Carry[wandererBagSlot2] = world.Item{Index: itemWandererBag}
	if !tradeSlotsAccessible(e, []int{45}) {
		t.Fatal("slot 45 was not tradeable with two active Wanderer Bag markers")
	}
	if tradeSlotsAccessible(e, []int{wandererBagSlot1}) {
		t.Fatal("marker slot 60 must never be tradeable")
	}
}

// TestTradeForwardsOffer: a primeira oferta abre a janela do oponente — ela é
// repassada no layout clássico com OpponentID = sender e a cópia do servidor
// do item; o remetente não recebe nada de volta.
func TestTradeForwardsOffer(t *testing.T) {
	addr, stop, _ := startServerClock(t, tradeDB())
	defer stop()
	a := enterWorldAs(t, addr, "tester") // conn 1, item 1100
	defer a.Close()
	b := enterWorldAs(t, addr, "tradeb") // conn 2, item 2200
	defer b.Close()

	sendTrade(t, a, tradeBody(2, world.Item{Index: 1100}, 0, 100, false))
	got := readForwarded(t, b)
	if got.OpponentID != 1 || got.Item[0].Index != 1100 || got.InvenPos[0] != 0 || got.TradeMoney != 100 || got.MyCheck != 0 {
		t.Fatalf("forwarded offer = opp %d item %d pos %d money %d check %d", got.OpponentID, got.Item[0].Index, got.InvenPos[0], got.TradeMoney, got.MyCheck)
	}
	for i := 1; i < protocol.MaxTrade; i++ {
		if got.Item[i].Index != 0 || got.InvenPos[i] != tradeEmptyPos {
			t.Fatalf("entry %d = item %d pos %d, want empty", i, got.Item[i].Index, got.InvenPos[i])
		}
	}
	expectSilence(t, a)
}

// TestTradeAtomicSwap: ofertas trocadas, A confere (CNFCheck para A, oferta conferida
// para B), B confere → os dois carries reenviados com os itens trocados e o ouro, os dois
// personagens salvos, e a troca fechada dos dois lados.
func TestTradeAtomicSwap(t *testing.T) {
	db := tradeDB()
	addr, stop, _ := startServerClock(t, db)
	defer stop()
	a := enterWorldAs(t, addr, "tester")
	defer a.Close()
	b := enterWorldAs(t, addr, "tradeb")
	defer b.Close()

	openTrade(t, a, b, world.Item{Index: 1100}, world.Item{Index: 2200}, 100, 50)
	_, savesBefore := db.lastSavedChar()

	sendTrade(t, a, tradeBody(2, world.Item{Index: 1100}, 0, 100, true))
	expectType(t, a, protocol.MsgCNFCheck)
	if got := readForwarded(t, b); got.MyCheck != 1 || got.OpponentID != 1 {
		t.Fatalf("checked offer to B: check %d opp %d", got.MyCheck, got.OpponentID)
	}
	expectSilence(t, a)

	sendTrade(t, b, tradeBody(1, world.Item{Index: 2200}, 0, 50, true))
	ca := expectType(t, a, protocol.MsgUpdateCarry)
	cb := expectType(t, b, protocol.MsgUpdateCarry)
	if updateCarryIndex(ca, 0) != 2200 || updateCarryIndex(cb, 0) != 1100 {
		t.Fatalf("carry slot 0 after swap: A=%d B=%d, want 2200/1100", updateCarryIndex(ca, 0), updateCarryIndex(cb, 0))
	}
	if carryCoin(ca) != 950 || carryCoin(cb) != 1050 {
		t.Fatalf("coin after swap: A=%d B=%d, want 950/1050", carryCoin(ca), carryCoin(cb))
	}
	expectType(t, a, protocol.MsgQuitTrade)
	expectType(t, b, protocol.MsgQuitTrade)
	expectSilence(t, a)
	expectSilence(t, b)

	deadline := time.Now().Add(2 * time.Second)
	for {
		if _, n := db.lastSavedChar(); n >= savesBefore+2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("both characters were not saved after the trade")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// TestTradeChangeResetsCheck: depois que A confere, B mudar sua oferta a repassa
// sem conferência e limpa a conferência de A, então a conferência de B sozinha não completa a troca.
func TestTradeChangeResetsCheck(t *testing.T) {
	addr, stop, _ := startServerClock(t, tradeDB())
	defer stop()
	a := enterWorldAs(t, addr, "tester")
	defer a.Close()
	b := enterWorldAs(t, addr, "tradeb")
	defer b.Close()

	openTrade(t, a, b, world.Item{Index: 1100}, world.Item{}, 0, 0)
	sendTrade(t, a, tradeBody(2, world.Item{Index: 1100}, 0, 0, true))
	expectType(t, a, protocol.MsgCNFCheck)
	readForwarded(t, b)

	sendTrade(t, b, tradeBody(1, world.Item{Index: 2200}, 0, 0, false))
	if got := readForwarded(t, a); got.MyCheck != 0 || got.Item[0].Index != 2200 {
		t.Fatalf("changed offer to A: check %d item %d", got.MyCheck, got.Item[0].Index)
	}

	sendTrade(t, b, tradeBody(1, world.Item{Index: 2200}, 0, 0, true))
	expectType(t, b, protocol.MsgCNFCheck) // a conferência de A foi resetada: sem troca ainda
	if got := readForwarded(t, a); got.MyCheck != 1 {
		t.Fatalf("B's checked offer to A: check %d", got.MyCheck)
	}
	expectSilence(t, a)
	expectSilence(t, b)
}

// TestTradeRejectsTamperedOffers: o servidor reconstrói toda oferta a partir do slot
// de carry (os bytes de item do cliente são só uma dica, como o memcpy legado em
// _MSG_Trade.cpp:199), então bytes forjados não podem contrabandear outro item; uma conferência sobre
// dados que o oponente nunca viu, remover um item ofertado e uma oferta de ouro acima
// da bolsa cancelam tudo.
func TestTradeRejectsTamperedOffers(t *testing.T) {
	addr, stop, _ := startServerClock(t, tradeDB())
	defer stop()
	a := enterWorldAs(t, addr, "tester")
	defer a.Close()
	b := enterWorldAs(t, addr, "tradeb")
	defer b.Close()

	// Bytes de item forjados: o oponente recebe o que o slot tem (1100), nunca
	// o 2200 que o cliente alegou. Exigir casamento exato, em vez disso, recusava toda
	// troca real com "Erro na transação".
	sendTrade(t, a, tradeBody(2, world.Item{Index: 2200}, 0, 0, false))
	if got := readForwarded(t, b); got.Item[0].Index != 1100 {
		t.Fatalf("forwarded item = %d, want the carry item 1100", got.Item[0].Index)
	}
	send(t, a, protocol.MsgQuitTrade, nil)
	expectType(t, a, protocol.MsgQuitTrade)
	expectSilence(t, b)

	// Conferir um valor diferente da oferta repassada cancela os dois.
	openTrade(t, a, b, world.Item{Index: 1100}, world.Item{}, 0, 0)
	sendTrade(t, a, tradeBody(2, world.Item{Index: 1100}, 0, 5, true))
	expectType(t, a, protocol.MsgMessagePanel)
	expectType(t, a, protocol.MsgQuitTrade)
	expectType(t, b, protocol.MsgQuitTrade)

	// Remover um item já ofertado cancela os dois.
	openTrade(t, a, b, world.Item{Index: 1100}, world.Item{}, 0, 0)
	sendTrade(t, a, tradeBody(2, world.Item{}, 0, 0, false))
	expectType(t, a, protocol.MsgMessagePanel)
	expectType(t, a, protocol.MsgQuitTrade)
	expectType(t, b, protocol.MsgQuitTrade)

	// Uma oferta de ouro acima da bolsa é recusada.
	sendTrade(t, a, tradeBody(2, world.Item{}, 0, 5000, false))
	expectType(t, a, protocol.MsgMessagePanel) // _NN_Not_Enough_Money
	expectType(t, a, protocol.MsgQuitTrade)
	expectSilence(t, b)
}

// TestTradeNoRoomRollsBack: o carry acessível de B está cheio, então o item de A não
// cabe; a troca faz rollback, B recebe o motivo, e as duas janelas fecham.
func TestTradeNoRoomRollsBack(t *testing.T) {
	db := tradeDB()
	full := db.loads[11]
	for i := 0; i < baseCarrySlots; i++ {
		full.Carry[i] = world.Item{Index: 2200}
	}
	db.loads[11] = full
	addr, stop, _ := startServerClock(t, db)
	defer stop()
	a := enterWorldAs(t, addr, "tester")
	defer a.Close()
	b := enterWorldAs(t, addr, "tradeb")
	defer b.Close()

	openTrade(t, a, b, world.Item{Index: 1100}, world.Item{}, 0, 0)
	sendTrade(t, a, tradeBody(2, world.Item{Index: 1100}, 0, 0, true))
	expectType(t, a, protocol.MsgCNFCheck)
	readForwarded(t, b)
	sendTrade(t, b, tradeBody(1, world.Item{}, 0, 0, true))

	if n := readNotice(t, b); n != NoticeNoEmptySlot {
		t.Fatalf("B notice = %d, want NoticeNoEmptySlot", n)
	}
	expectType(t, b, protocol.MsgQuitTrade)
	expectType(t, a, protocol.MsgQuitTrade)
	expectSilence(t, a)
	expectSilence(t, b)

	// A ainda tem seu item: ofertá-lo de novo é aceito e repassado.
	sendTrade(t, a, tradeBody(2, world.Item{Index: 1100}, 0, 0, false))
	readForwarded(t, b)
}

func TestTradeCancel(t *testing.T) {
	addr, stop, _ := startServerClock(t, tradeDB())
	defer stop()
	a := enterWorldAs(t, addr, "tester")
	defer a.Close()
	b := enterWorldAs(t, addr, "tradeb")
	defer b.Close()

	openTrade(t, a, b, world.Item{}, world.Item{}, 0, 0)

	// A cancela → os dois recebem QuitTrade.
	send(t, a, protocol.MsgQuitTrade, nil)
	expectType(t, a, protocol.MsgQuitTrade)
	expectType(t, b, protocol.MsgQuitTrade)
}

// TestTradeDisconnectCancels: a janela do oponente fecha quando o outro lado
// desconecta no meio da troca.
func TestTradeDisconnectCancels(t *testing.T) {
	addr, stop, _ := startServerClock(t, tradeDB())
	defer stop()
	a := enterWorldAs(t, addr, "tester")
	defer a.Close()
	b := enterWorldAs(t, addr, "tradeb")

	openTrade(t, a, b, world.Item{Index: 1100}, world.Item{}, 0, 0)
	_ = b.Close()
	_, _ = readUntil(t, a, protocol.MsgQuitTrade)
}

// TestTradeDupCancelsOnDrop: dropar um item no meio da troca cancela a troca dos dois
// lados (a regra anti-dup, Fase 8 §2.7).
func TestTradeDupCancelsOnDrop(t *testing.T) {
	addr, stop, _ := startServerClock(t, tradeDB())
	defer stop()
	a := enterWorldAs(t, addr, "tester")
	defer a.Close()
	b := enterWorldAs(t, addr, "tradeb")
	defer b.Close()

	openTrade(t, a, b, world.Item{}, world.Item{}, 0, 0)

	// A dropa um item enquanto negocia → troca cancelada para os dois.
	dropFrame(t, a, 0, 5, 5)
	if ty, _, ok := readMaybe(t, a); !ok || ty != protocol.MsgQuitTrade {
		t.Errorf("A got %#x ok=%v, want QuitTrade (dup cancel)", ty, ok)
	}
	if ty, _, ok := readMaybe(t, b); !ok || ty != protocol.MsgQuitTrade {
		t.Errorf("B got %#x ok=%v, want QuitTrade (dup cancel)", ty, ok)
	}
}
