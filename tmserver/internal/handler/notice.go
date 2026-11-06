package handler

import (
	"encoding/binary"
	"fmt"
	"strings"

	"openwyd-nextgen/tmserver/internal/protocol"
	"openwyd-nextgen/tmserver/internal/world"
)

// Notice identifies a client-facing notification (the legacy _NN_*/_DN_* codes).
type Notice uint32

// Notification codes used by batch 1 (handlers/*.md). The numeric values are
// local identifiers, NOT the original notification numbers.
const (
	NoticeVersionMismatch     Notice = iota // _NN_Version_Not_Match_Rerun
	NoticeLoginNow                          // "Login now, wait a moment."
	Notice3WrongPass                        // _NN_3_Tims_Wrong_Pass
	NoticeBadPass                           // _MSG_DBAccountLoginFail_Pass
	NoticeNoAccount                         // _MSG_DBAccountLoginFail_Account
	NoticeBlocked                           // _MSG_DBAccountLoginFail_Block
	NoticeSelectCharacter                   // _NN_SelectCharacter
	NoticeDeletingWait                      // "Deleting Character. wait a moment."
	NoticeDBError                           // DB call failed
	NoticeCantDropHere                      // _NN_Cant_Drop_Here
	NoticeCantAutoTrade                     // _NN_CantWhenAutoTrade
	NoticeNotConnected                      // whisper target offline
	NoticeDenyWhisper                       // whisper target blocked whispers
	NoticeBillingDenied                     // binServer denied entry (expired/blocked)
	NoticeCargoFull                         // deposit/withdraw would exceed the 2G ceiling
	NoticeReqNotMet                         // equip requirement (level/attributes) not met
	NoticeCantEatMore                       // _NN_CantEatMore (a buff of this kind is already active)
	NoticeOtherClassSkill                   // _NN_Cant_Learn_Other_Class_Skill
	NoticeNotEnoughSkillPoint               // _NN_Not_Enough_Skill_Point
	NoticeOnlyOneEighthSkill                // _NN_Only_OneSkillLearn (8th skill is exclusive)
	NoticeLearnPrereq                       // _NN_Befor_LearnSkill (learn the tree's 7 first)
	NoticeAlreadyLearned                    // _NN_Already_Learned_It
	NoticeNotEnoughCoin                     // _DN_D_Cost (8th skill costs 50M gold)
	NoticeMaxPoint                          // _NN_Maximum_Point_Now / _200_Now (mastery cap)

	// Dust refine (_MSG_UseItem.cpp dust path; Language.h ids in the comments).
	NoticeOnlyToEquips   // _NN_Only_To_Equips (74): a dust only refines equipment
	NoticeCantRefineMore // _NN_Cant_Refine_More (75): at the cap, or EF_NOSANC
	NoticeFailToRefine   // _NN_Fail_To_Refine (76)
	NoticeRefineSuccess  // _NN_Refine_Success (176)
	NoticeIncubated      // _NN_INCUBATED (253): the egg hatched into its mount
	NoticeIncuWaitMore   // _NN_Incu_Wait_More (261): the egg's incubation timer
	NoticeMountLevel     // _NN_Mount_Level (263): a cria gained one mount level

	NoticeNoEmptySlot // _NN_NoEmptySlot
	NoticeMaxBag      // _NN_MAX_BAG: both Bolsa do Andarilho slots are already active

	NoticeNoKey // _NN_No_Key: a locked gate needs a key the player doesn't hold

	// Duel codes (issue #118). Obsolete since issue #424: the duel now sends the
	// legacy Language.txt texts (battleInProgress, duelWinBy, duelDraw) and has no
	// city restriction, so nothing sends these. Kept so the iota codes below them
	// — which the client and the handler tests match on — do not shift.
	NoticeDuelBattleInProgress
	NoticeDuelWin
	NoticeDuelLose
	NoticeDuelDraw
	NoticeDuelInCity

	// NoticeCantUseHere (_NN_Cant_Use_That_Here) is sent for consumables whose
	// real _MSG_UseItem.cpp behavior depends on data absent from this repo, and
	// for Gema Estelar / Portal Scroll zone gates (issues #135 and #140).
	NoticeCantUseHere

	// Gema Estelar / Portal Scroll (_MSG_UseItem.cpp Vol 12/13, issue #140).
	NoticeSetWarp // _NN_Set_Warp (186): warp save-point recorded

	// NoticeReinoCapeRequired was the /reino cape gate (issue #127). Obsolete since
	// issue #208: /reino now routes every cape to a destination instead of refusing
	// kingdom capes, so nothing sends this code. Kept so the iota codes below it —
	// which the client and the handler tests match on — do not shift.
	NoticeReinoCapeRequired

	// Personal shop / autotrade (issue #115, _MSG_SendAutoTrade.cpp / _MSG_ReqBuy.cpp).
	NoticeOnlyVillage    // _NN_OnlyVillage: a shop can only open inside a village
	NoticeNotEnoughMoney // _NN_Not_Enough_Money: buyer can't afford the item
	NoticeCantGetMore2G  // _NN_Cant_get_more_than_2G: seller would exceed the 2G cap
	NoticeNoSpaceToTrade // _NN_You_Have_No_Space_To_Trade: buyer inventory full
	NoticeItemSold       // _NN_ItemSold: an item left the seller's shop

	// NoticeAlreadyDone (_NN_Youve_Done_It_Already) — a one-shot quest hand-in
	// (e.g. CAPAVERDE_TRADE) was already completed.
	NoticeAlreadyDone
)

// notify envia um aviso sem valor. Ver notifyf.
func (d *Dispatcher) notify(w *world.World, s *world.Session, n Notice) {
	d.notifyf(w, s, n)
}

// notifyf envia um aviso ao cliente como **texto** (`MSG_MessagePanel`,
// HEADER.ID 0), pelo caminho do `SendClientMessage` legado: é o único formato
// que o cliente 7662 desenha. `args` preenche os textos com `%d` (hoje só o
// Language.txt 204, "Você precisa de %d Gold.").
//
// Regra do projeto (docs/migration/game-messages.md): **todo aviso tem texto**. O
// corpo numérico (`MSG_MessageBoxOk` com 4 bytes) deixava o `String[96]` vazio —
// o cliente abria uma caixa em branco, ou nada —, e foi por isso que "comprar
// sem gold" e dezenas de outras recusas pareciam "não fazer nada" (QA 05/11,
// loja pessoal 06/11). Ele fica só como último recurso, com log em Warn: uma
// lacuna tem de aparecer no log, não sumir calada.
func (d *Dispatcher) notifyf(w *world.World, s *world.Session, n Notice, args ...any) {
	if text := noticeText(n); text != "" {
		if len(args) > 0 {
			text = fmt.Sprintf(text, args...)
		}
		d.sendClientMessage(w, s, text)
		return
	}
	d.log.Warn("aviso sem texto: a caixa numerica e invisivel no cliente 7662",
		"conn", s.Conn, "code", uint32(n))
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], uint32(n))
	w.Send(s, protocol.MsgMessageBoxOk, b[:])
}

// noticeTexts é o registro único aviso → texto de
// tmserver/content/TMsrv/run/Language.txt, nos bytes Windows-1252 da tabela
// (`\xe3` = ã, `\xe7` = ç, …). O número no comentário é o id do Language.h.
//
// Só entram textos copiados da tabela: inventar string é pior que a caixa vazia,
// porque parece funcionar. Um aviso sem texto fica de fora de propósito e o
// notifyf loga em Warn — e `TestEveryLiveNoticeHasText` reprova a lacuna.
//
// Os códigos marcados obsoletos no bloco de constantes (NoticeDuel*,
// NoticeReinoCapeRequired) não entram: nada os envia.
var noticeTexts = map[Notice]string{
	// Login, conta e seleção de personagem.
	NoticeVersionMismatch: versionMismatchText,                        // 36
	NoticeLoginNow:        "Realizando login. Aguarde...",             // 470
	Notice3WrongPass:      threeWrongPassText,                         // 37
	NoticeBadPass:         wrongPasswordText,                          // 133
	NoticeNoAccount:       noAccountText,                              // 131
	NoticeBlocked:         blockedAccountText,                         // 132
	NoticeSelectCharacter: "Por favor selecione um personagem.",       // 34
	NoticeDeletingWait:    "Excluindo personagem. Aguarde...",         // 469
	NoticeDBError:         "Resultou em falha.",                       // 476
	NoticeBillingDenied:   "Voc\xea usou todo o tempo de servi\xe7o.", // 126

	// Comércio, grupo e chat.
	NoticeNotConnected:   partyTargetGone,                                                    // 91
	NoticeDenyWhisper:    denyWhisper,                                                        // 165
	NoticeCargoFull:      "N\xe3o \xe9 poss\xedvel armazenar mais que 2 bilh\xf5es de Gold.", // 273
	NoticeCantDropHere:   cantDropHereText,                                                   // 117
	NoticeCantAutoTrade:  "N\xe3o \xe9 poss\xedvel durante a auto venda.",                    // 170
	NoticeOnlyVillage:    onlyVillageText,                                                    // 172
	NoticeNotEnoughMoney: notEnoughMoney,                                                     // 113
	NoticeCantGetMore2G:  "N\xe3o \xe9 poss\xedvel receber mais que 2 bilh\xf5es de Gold.",   // 273 (lado de quem recebe)
	NoticeNoSpaceToTrade: tradeNoSpace,                                                       // 86
	NoticeItemSold:       itemSoldText,                                                       // 171

	// Itens, refino e montaria.
	NoticeReqNotMet:      "N\xedvel Insuficiente. Isto n\xe3o pode ser utilizado.",    // 298
	NoticeCantUseHere:    "N\xe3o est\xe1 dispon\xedvel nesta \xe1rea.",               // 96
	NoticeCantEatMore:    "N\xe3o \xe9 poss\xedvel comer mais.",                       // 345
	NoticeOnlyToEquips:   "Poss\xedvel somente com armas e armaduras equipadas.",      // 74
	NoticeCantRefineMore: "Este item n\xe3o pode ser mais refinado.",                  // 75
	NoticeFailToRefine:   "Refina\xe7\xe3o falhou.",                                   // 76
	NoticeRefineSuccess:  "Obteve sucesso na refina\xe7\xe3o.",                        // 176
	NoticeIncubated:      "Incuba\xe7\xe3o sucedida.",                                 // 253
	NoticeIncuWaitMore:   "Incuba\xe7\xe3o n\xe3o est\xe1 pronta.",                    // 261
	NoticeMountLevel:     "Sua montaria cresceu.",                                     // 263
	NoticeNoEmptySlot:    "N\xe3o h\xe1 espa\xe7o para o Arch. Exclua um personagem.", // 307
	NoticeMaxBag:         "Voc\xea j\xe1 possui as 2 bolsas ativadas.",                // 531
	NoticeNoKey:          "Voc\xea n\xe3o possui uma chave.",                          // 79
	NoticeSetWarp:        "Localiza\xe7\xe3o foi armazenada.",                         // 186

	// Skills e progressão.
	NoticeOtherClassSkill:     "N\xe3o pode aprender habilidades de outra classe.",                 // 107
	NoticeNotEnoughSkillPoint: "N\xe3o possui pontos suficientes.",                                 // 108
	NoticeAlreadyLearned:      "Voc\xea j\xe1 aprendeu essa habilidade.",                           // 109
	NoticeOnlyOneEighthSkill:  "8\xba Skill pode ser somente da 1\xaa Classe.",                     // 402
	NoticeLearnPrereq:         "\xc9 necess\xe1rio aprender todas as skills antes da 8\xaa Skill.", // 403
	NoticeMaxPoint:            "N\xe3o pode colocar mais pontos.",                                  // 105
	NoticeNotEnoughCoin:       "Voc\xea precisa de %d Gold.",                                       // 204

	// Quests.
	NoticeAlreadyDone: "Voc\xea j\xe1 completou esta Quest.", // 71
}

// noticeText devolve o texto do aviso (vazio = sem texto confirmado).
func noticeText(n Notice) string { return noticeTexts[n] }

// Textos de CONFIRMAÇÃO que o Language.txt não tem. A regra do catálogo (todo
// texto vem da tabela) vale para recusas, onde o cliente compara string; para
// confirmar que uma ação deu certo não existe linha na tabela, e ficar calado foi
// exatamente o defeito relatado no QA ("usei e não avisou nada"):
//
//   - bolsa do andarilho: 15 slots a mais, sem nenhum aviso;
//   - Perzen (místico/arcano): a troca da esfera da sorte entregava a montaria
//     em silêncio, e sem o item exigido também não dizia nada;
//   - jóia da recuperação sem nada para limpar;
//   - selo do guerreiro: +10 de fama sem confirmação.
//
// São textos de projeto, curtos, no mesmo encoding do resto (Windows-1252), e
// nunca são usados para impedir ação — só para informar.
const (
	bagActivatedText = "Bolsa do Andarilho ativada: mais 15 espa\xe7os no invent\xe1rio."
	perzenDoneText   = "Troca realizada: sua recompensa est\xe1 na mochila."
	perzenNeedText   = "Voc\xea precisa trazer o item exigido por este NPC."
	noCleanseText    = "Nenhum efeito para remover."
	fameGainedText   = "Voc\xea ganhou 10 pontos de Fama."
)

// sendClientMessage mirrors the legacy SendClientMessage (SendFunc.cpp:27-43): a
// MSG_MessagePanel with HEADER.ID 0. The 7662 client only handles a message
// panel whose ID is 0 (TMScene.cpp:1371); w.Send stamps the session's own conn,
// which the client drops.
func (d *Dispatcher) sendClientMessage(w *world.World, s *world.Session, text string) {
	w.SendTo(s, protocol.Header{Type: protocol.MsgMessagePanel}, protocol.EncodeMessagePanelBody(text))
}

// Legacy refusal panels (tmserver/content/TMsrv/run/Language.txt), in the table's
// Windows-1252 bytes like the rest of the legacy string table.
const (
	cantDepositThatMuch  = "Esta quantia n\xe3o pode ser armazenada."      // _NN_Cant_Deposit_That_Much (44)
	cantWithdrawThatMuch = "Esta quantia n\xe3o pode ser retirada."        // _NN_Cant_Withdraw_That_Much (45)
	notEnoughMoney       = "N\xe3o possui gold suficiente."                // _NN_Not_Enough_Money (113)
	onlyMerchantSells    = "Pode comprar itens apenas do mercador."        // _NN_Only_Merchant_Sells (112)
	onlyMerchantBuyYours = "Poss\xedvel vender somente para comerciantes." // _NN_Only_Merchant_Buy_Yours (116)

	cantWithArmor = "No momento voc\xea n\xe3o pode equipar arma e armadura." // _NN_Cant_with_armor (205)

	// Vendas/loja pessoal (Language.txt:171/172) e drop no chão.
	itemSoldText     = "O item foi vendido."                       // _NN_ItemSold (171)
	onlyVillageText  = "Apenas na cidade."                         // _NN_OnlyVillage (172)
	cantDropHereText = "N\xe3o \xe9 poss\xedvel jogar itens aqui." // _NN_Cant_Drop_Here (117)

	newbieZone = "Somente n\xedvel 35 ou inferior pode entrar no campo de treinamento." // _NN_Newbie_zone (46)

	// Login refusals, verbatim from tmserver/content/TMsrv/run/Language.txt (Windows-1252
	// bytes, like the table). The legacy server sends these through
	// SendClientMessage — a MSG_MessagePanel with HEADER.ID 0 — and then closes:
	// TMSrv/ProcessDBMessage.cpp:518-570 (_Account/_Block/_Pass) and
	// _MSG_AccountLogin.cpp:44-90 (version/mode/lockout). Sending MSG_MessageBoxOk
	// with a bare numeric code (what notify does) makes the 7662 client open an
	// empty box, so the player sees no error at all.
	versionMismatchText = "Vers\xe3o do jogo incompat\xedvel com os servidores. Atualize o jogo pelo launcher." // _NN_Version_Not_Match_Rerun (36)
	threeWrongPassText  = "3 tentativas de senha incorreta ocasiona bloqueio de acesso durante 5 min."          // _NN_3_Tims_Wrong_Pass (37)
	noAccountText       = "Conta inexistente."                                                                  // _NN_No_Account_With_That_Name (131)
	blockedAccountText  = "Conta bloqueada Grade X, entre em contato com Suporte."                              // _SN_Blocked_Account (132), Parm 0
	wrongPasswordText   = "Senha incorreta."                                                                    // _NN_Wrong_Password (133)
)

// PvP and duel texts (Language.txt, Windows-1252), sent through
// sendClientMessage like the legacy SendClientMessage/SendNoticeArea calls.
const (
	inArenaNoExpLoss = "Na arena n\xe3o h\xe1 perda de EXP."                       // _NN_In_Arena_No_Exp_Loss (143)
	denyWhisper      = "N\xe3o \xe9 poss\xedvel sussurrar para este jogador."      // _NN_Deny_Whisper (165)
	battleInProgress = "H\xe1 uma partida em progresso."                           // _NN_Battle_In_Progress (193)
	battleStarted    = "Batalha entre %s e %s foi iniciada."                       // _NN_Battle_Started (194)
	duelWinBy        = "%s \xe9 o vencedor, %s \xe9 o perdedor desta partida."     // _SS_S_WinBy_S (196)
	duelDraw         = "Este jogo com %s e %s finalizou em empate."                // _SS_S_S_Draw (197)
	pkPointMinus     = "Seu ponto caos \xe9 %d (Menos_%d)."                        // _DD_PKPointMinus (208)
	cantKillUser     = "Voc\xea n\xe3o pode mais derrotar jogadores. (Atual : %d)" // _DN_CantKillUser (209)
	cantTradePKMode  = "A troca n\xe3o fica dispon\xedvel com o modo PK ativado."  // _NN_Cant_trade_pkmode (331)
)

// Gameplay texts the legacy sends through SendClientMessage (MSG_MessagePanel,
// HEADER.ID 0), verbatim from tmserver/content/TMsrv/run/Language.txt in the table's
// Windows-1252 bytes. The number in parentheses is the Language.h id.
//
// They exist because the port used to answer these flows with nothing at all
// (the handlers returned silently), so the player saw a click do nothing: party
// and trade were reported as "not working" on 05/11 precisely because of that.
const (
	levelUpText      = " + + + + SUBIU DE N\xcdVEL + + + + "                                              // _NN_Level_Up (53)
	expGainFormat    = "*** EXP + %d ***"                                                                 // _NS_GETEXP (473)
	joinedTownText   = "Voc\xea se cadastrou neste servidor."                                             // _DD_JOINTOWNPEP (427)
	partyAlreadyIn   = "N\xe3o pode ingressar no grupo pois voc\xea j\xe1 possui um."                     // _NN_Dropped_Current_Party_First (121)
	partyTargetGone  = "O jogador n\xe3o est\xe1 conectado."                                              // _NN_Not_Connected (91)
	partyOtherMember = "Este jogador j\xe1 \xe9 membro de um grupo."                                      // _NN_Other_Partys_Member (119)
	partyOwnGroup    = "Este jogador j\xe1 criou um grupo."                                               // _NN_Have_Own_Party_Already (120)
	partyLevelTooFar = "Grupo \xe9 poss\xedvel somente para jogadores com diferen\xe7a de 200 n\xedveis." // _NN_Party_Level_Limit (215)
	partyIsFull      = "O grupo est\xe1 cheio."                                                           // _NN_Partys_Full (122)
	tradeWrongPacket = "Erro na transa\xe7\xe3o."                                                         // _NN_Wrong_Trade_Packet1 (80)
	tradeAlreadyOn   = "J\xe1 h\xe1 uma transa\xe7\xe3o com outro jogador."                               // _NN_Already_Trading (82)
	tradeNoSpace     = "N\xe3o h\xe1 espa\xe7o no invent\xe1rio para realizar transa\xe7\xf5es."          // _NN_You_Have_No_Space_To_Trade (86)
)

// cstr trims a fixed-size NUL/space-padded C char array to a Go string.
func cstr(b []byte) string {
	if i := indexByte(b, 0); i >= 0 {
		b = b[:i]
	}
	return strings.TrimRight(string(b), " ")
}

func indexByte(b []byte, c byte) int {
	for i := range b {
		if b[i] == c {
			return i
		}
	}
	return -1
}
