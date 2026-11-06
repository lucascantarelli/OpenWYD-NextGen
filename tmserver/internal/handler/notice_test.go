package handler

import (
	"fmt"
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

// readNotice lê o próximo aviso e devolve o seu código, aceitando as duas formas
// (painel de texto — o padrão — e a caixa numérica legada). Substitui as
// asserções que exigiam MsgMessageBoxOk cru, que passaram a falhar quando os
// avisos viraram texto visível.
func readNotice(t *testing.T, c net.Conn) Notice {
	t.Helper()
	for i := 0; i < 20; i++ {
		ty, payload, ok := readMaybe(t, c)
		if !ok {
			t.Fatal("nenhum aviso chegou (esperava painel de texto ou caixa numérica)")
		}
		switch ty {
		case protocol.MsgMessagePanel, protocol.MsgMessageBoxOk:
			return noticeCode(t, payload)
		}
	}
	t.Fatal("nenhum aviso entre os frames")
	return 0
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

// TestEveryLiveNoticeHasText é a trava da regra "todo aviso tem texto"
// (docs/migration/game-messages.md): cada código que o servidor realmente envia
// precisa de uma linha em noticeTexts, senão o cliente 7662 abre uma caixa
// vazia — o defeito que fez "comprar sem gold" parecer travamento.
//
// A lista é explícita porque Go não itera constantes: ao declarar um aviso novo,
// acrescente-o aqui e ao mapa. Os códigos marcados obsoletos no bloco de
// constantes ficam de fora.
func TestEveryLiveNoticeHasText(t *testing.T) {
	live := map[string]Notice{
		"VersionMismatch":     NoticeVersionMismatch,
		"LoginNow":            NoticeLoginNow,
		"3WrongPass":          Notice3WrongPass,
		"BadPass":             NoticeBadPass,
		"NoAccount":           NoticeNoAccount,
		"Blocked":             NoticeBlocked,
		"SelectCharacter":     NoticeSelectCharacter,
		"DeletingWait":        NoticeDeletingWait,
		"DBError":             NoticeDBError,
		"BillingDenied":       NoticeBillingDenied,
		"CantDropHere":        NoticeCantDropHere,
		"CantAutoTrade":       NoticeCantAutoTrade,
		"NotConnected":        NoticeNotConnected,
		"DenyWhisper":         NoticeDenyWhisper,
		"CargoFull":           NoticeCargoFull,
		"ReqNotMet":           NoticeReqNotMet,
		"CantEatMore":         NoticeCantEatMore,
		"OtherClassSkill":     NoticeOtherClassSkill,
		"NotEnoughSkillPoint": NoticeNotEnoughSkillPoint,
		"OnlyOneEighthSkill":  NoticeOnlyOneEighthSkill,
		"LearnPrereq":         NoticeLearnPrereq,
		"AlreadyLearned":      NoticeAlreadyLearned,
		"NotEnoughCoin":       NoticeNotEnoughCoin,
		"MaxPoint":            NoticeMaxPoint,
		"OnlyToEquips":        NoticeOnlyToEquips,
		"CantRefineMore":      NoticeCantRefineMore,
		"FailToRefine":        NoticeFailToRefine,
		"RefineSuccess":       NoticeRefineSuccess,
		"Incubated":           NoticeIncubated,
		"IncuWaitMore":        NoticeIncuWaitMore,
		"MountLevel":          NoticeMountLevel,
		"NoEmptySlot":         NoticeNoEmptySlot,
		"MaxBag":              NoticeMaxBag,
		"NoKey":               NoticeNoKey,
		"CantUseHere":         NoticeCantUseHere,
		"SetWarp":             NoticeSetWarp,
		"OnlyVillage":         NoticeOnlyVillage,
		"NotEnoughMoney":      NoticeNotEnoughMoney,
		"CantGetMore2G":       NoticeCantGetMore2G,
		"NoSpaceToTrade":      NoticeNoSpaceToTrade,
		"ItemSold":            NoticeItemSold,
		"AlreadyDone":         NoticeAlreadyDone,
	}
	for name, n := range live {
		if got := noticeText(n); got == "" {
			t.Errorf("Notice%s (%d) sem texto em noticeTexts: o cliente não desenha a caixa numérica", name, uint32(n))
		}
	}
}

// TestNotEnoughCoinCarriesTheAmount garante que o aviso de custo sai com o valor
// (_DN_D_Cost 204): "Você precisa de %d Gold." sem número não informa nada.
func TestNotEnoughCoinCarriesTheAmount(t *testing.T) {
	// O texto é o _DN_D_Cost (204) de Language.txt; o %d tem de ser preenchido.
	want := "Voc\xea precisa de 30000000 Gold."
	if got := fmt.Sprintf(noticeText(NoticeNotEnoughCoin), 30000000); got != want {
		t.Fatalf("formatação do aviso = %q, quer %q", got, want)
	}
}
