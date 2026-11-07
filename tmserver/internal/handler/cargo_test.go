package handler

import (
	"context"
	"encoding/binary"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"

	"openwyd-nextgen/tmserver/internal/protocol"
	"openwyd-nextgen/tmserver/internal/world"
)

func le(b []byte) uint32   { return binary.LittleEndian.Uint32(b) }
func le16(b []byte) uint16 { return binary.LittleEndian.Uint16(b) }

// cargoDB semeia a conta "tester" com um personagem carregando charCoin (e um
// item opcional no slot 0 do carry) mais um cargo de conta carregando cargoCoin (e um
// item opcional no slot 0 do cargo).
func cargoDB(charCoin, cargoCoin int32, carry0, cargo0 int16) *fakeDB {
	db := newDB()
	st := world.CharacterState{Slot: 0, Name: "Hero", X: 5, Y: 5, HP: 1000, MaxHP: 1000, Coin: charCoin}
	if carry0 != 0 {
		st.Carry[0] = world.Item{Index: carry0}
	}
	db.loadResult = st
	cargo := world.CargoState{Coin: cargoCoin}
	if cargo0 != 0 {
		cargo.Items[0] = world.Item{Index: cargo0}
	}
	db.accounts["tester"].cargo = cargo
	return db
}

func depositFrame(t *testing.T, c net.Conn, coin int32) {
	t.Helper()
	send(t, c, protocol.MsgDeposit, protocol.EncodeStandardParm2(coin, 0))
}

func withdrawFrame(t *testing.T, c net.Conn, coin int32) {
	t.Helper()
	send(t, c, protocol.MsgWithdraw, protocol.EncodeStandardParm2(coin, 0))
}

// expect drena frames até ver want, devolvendo o payload daquele frame. Ele
// falha se o stream ficar mudo antes (o ack nunca chegou).
//
// Desde a padronização das mensagens (docs/migration/game-messages.md) um aviso
// vai como TEXTO em MSG_MessagePanel com HEADER.ID 0, não mais como a caixa
// numérica de 4 bytes. Os testes que pedem MsgMessageBoxOk aceitam o painel e o
// `noticeCode` traduz o texto de volta para o Notice — assim a asserção continua
// falando em código de aviso, e um texto desconhecido reprova.
func expect(t *testing.T, c net.Conn, want protocol.Type) []byte {
	t.Helper()
	for {
		ty, payload, ok := readMaybe(t, c)
		if !ok {
			t.Fatalf("expected %#x, stream went quiet", want)
		}
		if ty == want {
			return payload
		}
		if want == protocol.MsgMessageBoxOk && ty == protocol.MsgMessagePanel {
			return payload
		}
	}
}

func TestCargoDepositOK(t *testing.T) {
	addr, stop, _ := startServerClock(t, cargoDB(1000, 0, 0, 0))
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	depositFrame(t, c, 600)
	// O ack ecoa MsgDeposit, então as duas exibições de ouro atualizam.
	expect(t, c, protocol.MsgDeposit)
	cargoCoin := expect(t, c, protocol.MsgUpdateCargoCoin)
	if got := int32(le(cargoCoin[0:4])); got != 600 {
		t.Errorf("cargo coin = %d, want 600", got)
	}
	etc := expect(t, c, protocol.MsgUpdateEtc)
	if got := int32(le(etc[28:32])); got != 400 { // coin do personagem @body28
		t.Errorf("char coin = %d, want 400", got)
	}
}

func TestCargoDepositInsufficient(t *testing.T) {
	addr, stop, _ := startServerClock(t, cargoDB(100, 0, 0, 0))
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	depositFrame(t, c, 500) // mais que os 100 carregados
	expectPanel(t, c, cantDepositThatMuch)
	if ty, _, ok := readMaybe(t, c); ok {
		t.Errorf("over-deposit also produced %#x; should be rejected", ty)
	}
}

// TestCargoAmountRefusals: o legado recusa valor negativo com o mesmo
// painel de um valor acima do saldo (_MSG_Deposit/_MSG_Withdraw.cpp:34),
// e um valor zero fica silencioso aqui.
func TestCargoAmountRefusals(t *testing.T) {
	for _, tc := range []struct {
		name  string
		send  func(*testing.T, net.Conn, int32)
		coin  int32
		panel string
	}{
		{"deposit-negative", depositFrame, -1, cantDepositThatMuch},
		{"withdraw-negative", withdrawFrame, -1, cantWithdrawThatMuch},
		{"deposit-zero", depositFrame, 0, ""},
		{"withdraw-zero", withdrawFrame, 0, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			addr, stop, _ := startServerClock(t, cargoDB(100, 100, 0, 0))
			defer stop()
			c := enterWorld(t, addr)
			defer c.Close()

			tc.send(t, c, tc.coin)
			if tc.panel != "" {
				expectPanel(t, c, tc.panel)
			}
			if ty, _, ok := readMaybe(t, c); ok {
				t.Errorf("refused amount %d produced %#x", tc.coin, ty)
			}
		})
	}
}

func TestCargoDepositOverflow(t *testing.T) {
	addr, stop, _ := startServerClock(t, cargoDB(maxCoin, maxCoin, 0, 0))
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	depositFrame(t, c, 1) // cargo já no teto de 2G
	// Aviso padrão: texto de Language.txt (273), não a caixa numérica invisível.
	if got := readPanel(t, c); got != noticeText(NoticeCargoFull) {
		t.Errorf("overflow = %q, quer %q", got, noticeText(NoticeCargoFull))
	}
}

func TestCargoWithdrawOK(t *testing.T) {
	addr, stop, _ := startServerClock(t, cargoDB(0, 1000, 0, 0))
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	withdrawFrame(t, c, 250)
	expect(t, c, protocol.MsgWithdraw)
	cargoCoin := expect(t, c, protocol.MsgUpdateCargoCoin)
	if got := int32(le(cargoCoin[0:4])); got != 750 {
		t.Errorf("cargo coin = %d, want 750", got)
	}
	etc := expect(t, c, protocol.MsgUpdateEtc)
	if got := int32(le(etc[28:32])); got != 250 {
		t.Errorf("char coin = %d, want 250", got)
	}
}

func TestCargoWithdrawTooMuch(t *testing.T) {
	addr, stop, _ := startServerClock(t, cargoDB(0, 100, 0, 0))
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	withdrawFrame(t, c, 500) // o cargo só tem 100
	expectPanel(t, c, cantWithdrawThatMuch)
	if ty, _, ok := readMaybe(t, c); ok {
		t.Errorf("over-withdraw also produced %#x; should be rejected", ty)
	}
}

// tradeItemFrame envia uma troca de slot _MSG_TradingItem (0x0376).
func tradeItemFrame(t *testing.T, c net.Conn, srcPlace, srcSlot, dstPlace, dstSlot, warpID int) {
	t.Helper()
	body := protocol.MsgTradingItemBody{
		SrcPlace: uint8(srcPlace), SrcSlot: uint8(srcSlot),
		DestPlace: uint8(dstPlace), DestSlot: uint8(dstSlot),
		WarpID: int32(warpID),
	}
	send(t, c, protocol.MsgTradingItem, body.Encode())
}

// cargoGuardID é o id que SpawnMob atribui ao primeiro NPC (MaxUser).
const cargoGuardID = world.MaxUser

// startServerCargoGuard é startServerClock com um NPC guarda de cargo Merchant==2
// gerado em (5,5) — ao lado de onde o personagem de teste loga — para os movimentos
// de slot do cargo passarem o gate de proximidade.
func startServerCargoGuard(t *testing.T, persist world.Persistence) (string, func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	d := New(Config{Log: log})
	w := world.New(world.Config{GridDim: 16}, log, persist, d.Handle)

	tmpl := make([]byte, 816)
	copy(tmpl[0:16], "CargoGuard")
	tmpl[92+12] = 2                                  // CurrentScore.Merchant = guarda de cargo
	binary.LittleEndian.PutUint32(tmpl[92+16:], 100) // MaxHp
	binary.LittleEndian.PutUint32(tmpl[92+24:], 100) // Hp
	if id := w.SpawnMob(tmpl, 5, 5); id != cargoGuardID {
		t.Fatalf("cargo guard spawned as id %d, want %d", id, cargoGuardID)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { _ = w.Serve(ctx, ln); close(done) }()
	return ln.Addr().String(), func() {
		cancel()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Error("server did not stop")
		}
	}
}

// TestCargoItemDeposit move um item do Carry para o cofre via _MSG_TradingItem e
// afirma que os dois slots afetados são atualizados no cliente.
func TestCargoItemDeposit(t *testing.T) {
	addr, stop := startServerCargoGuard(t, cargoDB(0, 0, 1100, 0))
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	tradeItemFrame(t, c, world.ItemPlaceCarry, 0, world.ItemPlaceCargo, 0, cargoGuardID)
	expect(t, c, protocol.MsgTradingItem) // eco do movimento
	// Dois refreshes de SendItem: o slot de carry agora vazio e o slot de cargo preenchido.
	carry := expect(t, c, protocol.MsgSendItem)
	if place := int(le16(carry[0:2])); place != world.ItemPlaceCarry {
		t.Errorf("first refresh place = %d, want carry", place)
	}
	cargoSlot := expect(t, c, protocol.MsgSendItem)
	if place := int(le16(cargoSlot[0:2])); place != world.ItemPlaceCargo {
		t.Errorf("second refresh place = %d, want cargo", place)
	}
	if idx := le16(cargoSlot[4:6]); idx != 1100 { // item @body4 (SelItem.Index)
		t.Errorf("cargo slot item = %d, want 1100", idx)
	}
}

// TestCargoItemWithdraw move um item do cargo de volta para o inventário.
func TestCargoItemWithdraw(t *testing.T) {
	addr, stop := startServerCargoGuard(t, cargoDB(0, 0, 0, 2200))
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	tradeItemFrame(t, c, world.ItemPlaceCargo, 0, world.ItemPlaceCarry, 5, cargoGuardID)
	expect(t, c, protocol.MsgTradingItem)
	expect(t, c, protocol.MsgSendItem) // slot do cargo (agora vazio)
	carry := expect(t, c, protocol.MsgSendItem)
	if place := int(le16(carry[0:2])); place != world.ItemPlaceCarry || le16(carry[2:4]) != 5 {
		t.Errorf("carry refresh place/slot = %d/%d, want carry/5", place, le16(carry[2:4]))
	}
	if idx := le16(carry[4:6]); idx != 2200 {
		t.Errorf("carry slot item = %d, want 2200", idx)
	}
}

// TestCargoLogoutSavesBoth é a regressão anti-dup do cenário entre
// personagens: um personagem saca um item do cargo compartilhado, então desloga para
// a seleção de personagem. O carry do personagem (agora com o item) E o cargo
// (agora sem ele) precisam AMBOS ser persistidos antes da confirmação do logout — senão
// a linha obsoleta de account_cargo mantém o item e ele é duplicado no próximo
// carregamento (por exemplo, quando outro personagem da conta loga).
func TestCargoLogoutSavesBoth(t *testing.T) {
	db := cargoDB(0, 0, 0, 2200) // o cargo tem o item 2200 no slot 0; carry vazio
	addr, stop := startServerCargoGuard(t, db)
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	// Saca o item do cargo para o slot 5 do carry.
	tradeItemFrame(t, c, world.ItemPlaceCargo, 0, world.ItemPlaceCarry, 5, cargoGuardID)
	expect(t, c, protocol.MsgTradingItem)
	expect(t, c, protocol.MsgSendItem)
	expect(t, c, protocol.MsgSendItem)

	// Desloga para a seleção de personagem — a confirmação só chega depois que os dois saves assentam.
	send(t, c, protocol.MsgCharacterLogout, nil)
	expect(t, c, protocol.MsgCNFCharacterLogout)

	// O personagem foi salvo com o item sacado...
	char, nc := db.lastSavedChar()
	if nc == 0 {
		t.Fatal("character not saved on logout")
	}
	if !hasItem(char.Carry, 2200) {
		t.Errorf("saved character carry missing withdrawn item 2200: %+v", char.Carry)
	}
	// ...e o cargo foi salvo SEM ele (nenhuma duplicata deixada no armazém).
	cargo, ng := db.lastSavedCargo()
	if ng == 0 {
		t.Fatal("cargo not saved on logout (item would duplicate on next load)")
	}
	if hasItem(cargo.Items, 2200) {
		t.Errorf("saved cargo still holds the withdrawn item 2200 (dup): %+v", cargo.Items)
	}
}

func hasItem(items []world.SavedItem, index int16) bool {
	for _, it := range items {
		if it.Index == index {
			return true
		}
	}
	return false
}

// TestCargoMoveNoGuard recusa um movimento de cargo quando nenhum NPC guarda de cargo é referenciado
// (WarpID 0): o gate de proximidade bloqueia e nada acontece.
func TestCargoMoveNoGuard(t *testing.T) {
	addr, stop := startServerCargoGuard(t, cargoDB(0, 0, 1100, 0))
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	tradeItemFrame(t, c, world.ItemPlaceCarry, 0, world.ItemPlaceCargo, 0, 0) // nenhum NPC
	if ty, _, ok := readMaybe(t, c); ok {
		t.Errorf("cargo move without guard produced %#x; should be blocked", ty)
	}
}
