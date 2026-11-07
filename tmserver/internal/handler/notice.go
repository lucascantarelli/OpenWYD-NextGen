package handler

import (
	"fmt"
	"strings"

	"openwyd-nextgen/tmserver/internal/protocol"
	"openwyd-nextgen/tmserver/internal/world"
)

// Notice identifica uma notificação voltada ao cliente (os códigos legados _NN_*/_DN_*).
type Notice uint32

// Códigos de notificação usados pelo lote 1 (handlers/*.md). Os valores numéricos são
// identificadores locais, NÃO os números de notificação originais.
const (
	NoticeVersionMismatch     Notice = iota // _NN_Version_Not_Match_Rerun
	NoticeLoginNow                          // "Login now, wait a moment."
	Notice3WrongPass                        // _NN_3_Tims_Wrong_Pass
	NoticeBadPass                           // _MSG_DBAccountLoginFail_Pass
	NoticeNoAccount                         // _MSG_DBAccountLoginFail_Account
	NoticeBlocked                           // _MSG_DBAccountLoginFail_Block
	NoticeSelectCharacter                   // _NN_SelectCharacter
	NoticeDeletingWait                      // "Deleting Character. wait a moment."
	NoticeDBError                           // Chamada ao banco falhou
	NoticeCantDropHere                      // _NN_Cant_Drop_Here
	NoticeCantAutoTrade                     // _NN_CantWhenAutoTrade
	NoticeNotConnected                      // alvo do sussurro offline
	NoticeDenyWhisper                       // alvo do sussurro bloqueou sussurros
	NoticeBillingDenied                     // binServer recusou a entrada (expirada/bloqueada)
	NoticeCargoFull                         // depósito/saque passaria do teto de 2G
	NoticeReqNotMet                         // requisito de equipar (nível/atributos) não cumprido
	NoticeCantEatMore                       // _NN_CantEatMore (um buff deste tipo já está ativo)
	NoticeOtherClassSkill                   // _NN_Cant_Learn_Other_Class_Skill
	NoticeNotEnoughSkillPoint               // _NN_Not_Enough_Skill_Point
	NoticeOnlyOneEighthSkill                // _NN_Only_OneSkillLearn (a 8ª skill é exclusiva)
	NoticeLearnPrereq                       // _NN_Befor_LearnSkill (aprenda as 7 da árvore primeiro)
	NoticeAlreadyLearned                    // _NN_Already_Learned_It
	NoticeNotEnoughCoin                     // _DN_D_Cost (a 8ª skill custa 50M de ouro)
	NoticeMaxPoint                          // _NN_Maximum_Point_Now / _200_Now (teto de maestria)

	// Refino com poeira (caminho de poeira de _MSG_UseItem.cpp; ids do Language.h nos comentários).
	NoticeOnlyToEquips   // _NN_Only_To_Equips (74): uma poeira só refina equipamento
	NoticeCantRefineMore // _NN_Cant_Refine_More (75): no teto, ou EF_NOSANC
	NoticeFailToRefine   // _NN_Fail_To_Refine (76)
	NoticeRefineSuccess  // _NN_Refine_Success (176)
	NoticeIncubated      // _NN_INCUBATED (253): o ovo chocou na montaria
	NoticeIncuWaitMore   // _NN_Incu_Wait_More (261): o timer de incubação do ovo
	NoticeMountLevel     // _NN_Mount_Level (263): uma cria ganhou um nível de montaria

	NoticeNoEmptySlot // _NN_NoEmptySlot
	NoticeMaxBag      // _NN_MAX_BAG: os dois slots da Bolsa do Andarilho já estão ativos

	NoticeNoKey // _NN_No_Key: um portão trancado precisa de uma chave que o jogador não tem

	// Códigos de duelo (issue #118). Obsoletos desde a issue #424: o duelo agora envia os
	// textos legados do Language.txt (battleInProgress, duelWinBy, duelDraw) e não tem
	// restrição de cidade, então nada envia estes. Mantidos para os códigos iota abaixo deles
	// — que o cliente e os testes de handler casam — não deslocarem.
	NoticeDuelBattleInProgress
	NoticeDuelWin
	NoticeDuelLose
	NoticeDuelDraw
	NoticeDuelInCity

	// NoticeCantUseHere (_NN_Cant_Use_That_Here) é enviado para consumíveis cujo
	// comportamento real em _MSG_UseItem.cpp depende de dados ausentes deste repositório, e
	// para os gates de zona da Gema Estelar / Portal Scroll (issues #135 e #140).
	NoticeCantUseHere

	// Gema Estelar / Portal Scroll (_MSG_UseItem.cpp Vol 12/13, issue #140).
	NoticeSetWarp // _NN_Set_Warp (186): ponto de save do warp registrado

	// NoticeReinoCapeRequired era o gate de capa de /reino (issue #127). Obsoleto desde
	// a issue #208: /reino agora roteia toda capa para um destino em vez de recusar
	// capas de reino, então nada envia este código. Mantido para os códigos iota abaixo dele —
	// que o cliente e os testes de handler casam — não deslocarem.
	NoticeReinoCapeRequired

	// Loja pessoal / trade automático (issue #115, _MSG_SendAutoTrade.cpp / _MSG_ReqBuy.cpp).
	NoticeOnlyVillage    // _NN_OnlyVillage: uma loja só abre dentro de uma vila
	NoticeNotEnoughMoney // _NN_Not_Enough_Money: o comprador não tem dinheiro para o item
	NoticeCantGetMore2G  // _NN_Cant_get_more_than_2G: o vendedor passaria do teto de 2G
	NoticeNoSpaceToTrade // _NN_You_Have_No_Space_To_Trade: inventário do comprador cheio
	NoticeItemSold       // _NN_ItemSold: um item saiu da loja do vendedor

	// NoticeAlreadyDone (_NN_Youve_Done_It_Already) — uma entrega de quest de uso único
	// (ex.: CAPAVERDE_TRADE) já foi concluída.
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
	// Nenhum aviso vaza calado: se o mapa estiver sem texto, o jogador recebe um
	// texto generico de projeto e o log registra o codigo. A caixa numerica de 4
	// bytes nao e desenhada pelo cliente 7662 e por isso nao e mais enviada.
	d.log.Error("aviso sem texto: enviando texto generico", "conn", s.Conn, "code", uint32(n))
	d.sendClientMessage(w, s, genericRefusalText)
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

	// Guilda: recusas que o legado deixava caladas (o clique no convite, no
	// cadastro e na criacao nao respondia nada). Os textos 144/147/390/429/535/537
	// sao da tabela; os demais sao de projeto, curtos e so informativos.
	guildMasterOnlyText    = "Uso restrito ao Mestre de Guilda."                                      // 144
	guildNoMedalText       = "O usu\xe1rio n\xe3o possui a medalha de l\xedder de guilda."            // 147
	sundayText             = "N\xe3o \xe9 possivel utilizar domingo."                                 // 390
	guildBadNameText       = "Nome incorreto, utilize de 5 a 10 caracteres para o nome da guilda."    // 429
	guildNeedsKingdomText  = "Voc\xea Ter\xe1 que pertencer a um dos Reinos para poder criar guilda!" // 535
	guildNeedsCitizenText  = "Voc\xea precisa pertencer a algum reino antes de criar uma Guild."      // 537
	guildInviteRefusedText = "Este jogador n\xe3o pode ingressar nesta guilda."
	guildAlreadyText       = "Voc\xea j\xe1 pertence a uma guilda."

	// Ultimo recurso do notifyf quando um aviso novo nao tem texto: o jogador
	// nunca fica sem resposta, e o log em Error aponta o codigo a mapear.
	genericRefusalText = "A\xe7\xe3o n\xe3o permitida."
)

// sendClientMessage espelha o SendClientMessage legado (SendFunc.cpp:27-43): um
// MSG_MessagePanel com HEADER.ID 0. O cliente 7662 só trata um painel de mensagem
// cujo ID é 0 (TMScene.cpp:1371); w.Send carimba o conn da própria sessão,
// que o cliente descarta.
func (d *Dispatcher) sendClientMessage(w *world.World, s *world.Session, text string) {
	w.SendTo(s, protocol.Header{Type: protocol.MsgMessagePanel}, protocol.EncodeMessagePanelBody(text))
}

// Painéis de recusa legados (tmserver/content/TMsrv/run/Language.txt), nos
// bytes Windows-1252 da tabela como o resto da tabela de strings legada.
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

	// Recusas de login, literalmente de tmserver/content/TMsrv/run/Language.txt (bytes
	// Windows-1252, como a tabela). O servidor legado envia estas por
	// SendClientMessage — um MSG_MessagePanel com HEADER.ID 0 — e depois fecha:
	// TMSrv/ProcessDBMessage.cpp:518-570 (_Account/_Block/_Pass) e
	// _MSG_AccountLogin.cpp:44-90 (versão/modo/bloqueio). Enviar MSG_MessageBoxOk
	// com um código numérico puro (o que o notify faz) faz o cliente 7662 abrir uma
	// caixa vazia, então o jogador não vê erro nenhum.
	versionMismatchText = "Vers\xe3o do jogo incompat\xedvel com os servidores. Atualize o jogo pelo launcher." // _NN_Version_Not_Match_Rerun (36)
	threeWrongPassText  = "3 tentativas de senha incorreta ocasiona bloqueio de acesso durante 5 min."          // _NN_3_Tims_Wrong_Pass (37)
	noAccountText       = "Conta inexistente."                                                                  // _NN_No_Account_With_That_Name (131)
	blockedAccountText  = "Conta bloqueada Grade X, entre em contato com Suporte."                              // _SN_Blocked_Account (132), Parm 0
	wrongPasswordText   = "Senha incorreta."                                                                    // _NN_Wrong_Password (133)
)

// Textos de PvP e duelo (Language.txt, Windows-1252), enviados por
// sendClientMessage como as chamadas legadas SendClientMessage/SendNoticeArea.
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

// Textos de jogabilidade que o legado envia por SendClientMessage (MSG_MessagePanel,
// HEADER.ID 0), literalmente de tmserver/content/TMsrv/run/Language.txt nos
// bytes Windows-1252 da tabela. O número entre parênteses é o id do Language.h.
//
// Eles existem porque o port costumava responder estes fluxos com nada de todo
// (os handlers retornavam em silêncio), então o jogador via um clique não fazer nada: grupo
// e troca foram reportados como "não funcionando" em 05/11 exatamente por isso.
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

// cstr corta um array C de char de tamanho fixo preenchido com NUL/espaço para uma string Go.
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
