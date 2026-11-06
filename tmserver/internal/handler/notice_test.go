package handler

import (
	"net"
	"testing"

	"openwyd-nextgen/tmserver/internal/protocol"
)

// readPanel espera um MSG_MessagePanel e devolve o texto. O cliente 7662 só
// desenha o painel quando HEADER.ID == 0 (TMScene.cpp:1371) — o mesmo motivo que
// fez os avisos de login sumirem antes da issue 32.
func readPanel(t *testing.T, c net.Conn) string {
	t.Helper()
	for i := 0; i < 20; i++ {
		h, payload, ok := readMaybeHeader(t, c)
		if !ok {
			t.Fatalf("nenhum frame chegou (esperava MSG_MessagePanel)")
		}
		if h.Type != protocol.MsgMessagePanel {
			continue
		}
		if h.ID != 0 {
			t.Fatalf("MSG_MessagePanel com HEADER.ID = %d, quer 0 (o cliente descarta)", h.ID)
		}
		return cstr(payload)
	}
	t.Fatal("nenhum MSG_MessagePanel entre os frames")
	return ""
}

// TestAutoTradeBuyWithoutGoldShowsMessage reproduz o relato do teste ao vivo de
// 06/11: na loja pessoal, comprar sem gold não comprava e **não dizia nada**. O
// servidor já recusava (NoticeNotEnoughMoney); o aviso é que ia como caixa
// numérica vazia. Agora tem de sair o texto de Language.txt 113.
func TestAutoTradeBuyWithoutGoldShowsMessage(t *testing.T) {
	const sellItem = int16(1030)
	const price, tax = int32(200_000), int32(5)

	db := autotradeDB(sellItem)
	// Comprador sem nenhum gold.
	buyerState := db.loads[11]
	buyerState.Coin = 0
	db.loads[11] = buyerState

	addr, stop, _ := startServerClock(t, db)
	defer stop()
	seller := enterWorldAs(t, addr, "tester") // conn 1
	defer seller.Close()
	buyer := enterWorldAs(t, addr, "tradeb") // conn 2
	defer buyer.Close()

	send(t, seller, protocol.MsgSendAutoTrade, openShopPayload("Loja Sem Gold", sellItem, 0, price))
	readUntil(t, seller, protocol.MsgSendAutoTrade)

	// O comprador abre a loja e tenta comprar o slot 0.
	send(t, buyer, protocol.MsgReqTradeList, protocol.EncodeStandardParm(1))
	readUntil(t, buyer, protocol.MsgSendAutoTrade)
	send(t, buyer, protocol.MsgReqBuy, reqBuyPayload(1, 0, sellItem, price, tax))

	if got := readPanel(t, buyer); got != notEnoughMoney {
		t.Fatalf("aviso = %q, quer %q", got, notEnoughMoney)
	}
}

// TestNoticeTextCoversShopAndTradeTexts trava a padronização: os avisos que já
// têm texto confirmado em Language.txt não podem voltar a cair no caminho
// numérico invisível.
func TestNoticeTextCoversShopAndTradeTexts(t *testing.T) {
	cases := map[Notice]string{
		NoticeNotEnoughMoney: notEnoughMoney,   // _NN_Not_Enough_Money (113)
		NoticeNoSpaceToTrade: tradeNoSpace,     // _NN_You_Have_No_Space_To_Trade (86)
		NoticeNotConnected:   partyTargetGone,  // _NN_Not_Connected (91)
		NoticeItemSold:       itemSoldText,     // _NN_ItemSold (171)
		NoticeOnlyVillage:    onlyVillageText,  // _NN_OnlyVillage (172)
		NoticeCantDropHere:   cantDropHereText, // _NN_Cant_Drop_Here (117)
	}
	for n, want := range cases {
		if got := noticeText(n); got != want {
			t.Errorf("noticeText(%d) = %q, quer %q", uint32(n), got, want)
		}
	}
}
