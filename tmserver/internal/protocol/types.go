package protocol

import "encoding/binary"

// Type é o campo HEADER.Type de 2 bytes: uma base de mensagem com OR de
// flags de direção (protocol-spec.md §2, Basedef.h:1212-1221). O dispatcher compara o
// valor completo do wire (base + flags) direto, então uma constante com várias
// flags tem um único valor de wire que casa em qualquer contexto.
type Type uint16

// MessageLength é MSG_MessageChat.String[MESSAGE_LENGTH] (Basedef.h:189,
// 1789): o chat de ação de mob e o chat público normal são uma string C fixa de 96 bytes.
const MessageLength = 96

// MessagePanelLength é a string C de largura fixa de MSG_MessagePanel.String
// (Basedef.h:1528). Diferente das mensagens de chat, os avisos ao cliente reservam 128 bytes.
const MessagePanelLength = 128

// EncodeMessagePanelBody devolve o payload fixo de MSG_MessagePanel.String.
// A cauda preenchida com zeros fornece o terminador de string C esperado pelo cliente.
func EncodeMessagePanelBody(text string) []byte {
	body := make([]byte, MessagePanelLength)
	copy(body[:MessagePanelLength-1], text)
	return body
}

// Bits de flag de direção (protocol-spec.md §2, Basedef.h:1212-1221).
const (
	FlagGame2Client Type = 0x0100 // TMSrv → client
	FlagClient2Game Type = 0x0200 // client → TMSrv
	FlagDB2Game     Type = 0x0400 // DBSrv → TMSrv
	FlagGame2DB     Type = 0x0800 // TMSrv → DBSrv
	FlagDB2NP       Type = 0x1000 // DBSrv → NPServer
	FlagNP2DB       Type = 0x2000 // NPServer → DBSrv
	FlagNew         Type = 0x4000 // família "new" de mensagens (ranking grind/exp)

	// flagMask cobre todos os bits de direção; os bits baixos restantes são a base.
	flagMask Type = 0xFF00
)

// EncodeMessageChatBody devolve o payload fixo de MSG_MessageChat.String. O
// cliente espera a struct inteira (o SendSay legado envia sizeof(MSG_MessageChat)),
// e o último byte fica zero para texto longo demais continuar sendo string C terminada.
func EncodeMessageChatBody(text string) []byte {
	body := make([]byte, MessageLength)
	copy(body[:MessageLength-1], text)
	return body
}

// MsgExpPanelBodySize é o corpo empacotado de MSG_Exp_Msg_Panel_: Msg[96]
// seguido de sua cor de texto ARGB de 32 bits.
const MsgExpPanelBodySize = MessageLength + 4

// EncodeExpPanelBody monta o corpo de largura fixa exibido no painel de
// notificação de EXP do cliente. Texto maior que a string C legada é truncado.
func EncodeExpPanelBody(text string, color uint32) []byte {
	body := make([]byte, MsgExpPanelBodySize)
	copy(body[:MessageLength], text)
	binary.LittleEndian.PutUint32(body[MessageLength:], color)
	return body
}

// Base devolve a base da mensagem (o valor de wire sem as flags de direção).
func (t Type) Base() Type { return t &^ flagMask }

// HasFlag informa se t carrega a flag de direção dada.
func (t Type) HasFlag(f Type) bool { return t&f == f }

// Tipos de mensagem Cliente → TMSrv que o servidor precisa tratar (protocol-spec.md §3.1).
// Estes são os únicos com case no dispatcher original
// (ProcessClientMessage.cpp); o dump completo de 198 Types vive em protocol-spec.md
// Apêndice B e ainda não estão todos nomeados aqui.
const (
	MsgAccountLogin        Type = 0x020D // 525  MSG_AccountLogin
	MsgCharacterLogin      Type = 0x0213 // 531  MSG_CharacterLogin
	MsgCharacterLogout     Type = 0x0215 // 533
	MsgDeleteCharacter     Type = 0x0211 // 529
	MsgCreateCharacter     Type = 0x020F // 527
	MsgAccountSecure       Type = 0x0FDE // 4062 PIN token
	MsgMessageChat         Type = 0x0333 // 819
	MsgMessageWhisper      Type = 0x0334 // 820
	MsgAction              Type = 0x036C // 876  movimento (também 0x0366/0x0368)
	MsgAction2             Type = 0x0366 // 870
	MsgAction3             Type = 0x0368 // 872
	MsgMotion              Type = 0x036A // 874
	MsgNoViewMob           Type = 0x0369 // 873  ressincroniza a visibilidade de uma entidade
	MsgChangeCity          Type = 0x0291 // 657  define a cidade de spawn
	MsgReqTeleport         Type = 0x0290 // 656  teleporte pago
	MsgAttack              Type = 0x0367 // 871  (também AttackOne 0x039D / AttackTwo 0x039E)
	MsgAttackOne           Type = 0x039D // 925
	MsgAttackTwo           Type = 0x039E // 926
	MsgDropItem            Type = 0x0272 // 626
	MsgGetItem             Type = 0x0270 // 624
	MsgUseItem             Type = 0x0373 // 883
	MsgDeleteItem          Type = 0x02E4 // 228  destrói um item do carry (_MSG_DeleteItem)
	MsgSplitItem           Type = 0x02E5 // 229  divide uma pilha (_MSG_SplitItem)
	MsgUpdateItem          Type = 0x0374 // 116  C↔S abre portão/porta (_MSG_UpdateItem)
	MsgSendAffect          Type = 0x03B9 // 953  snapshot de buff/debuff (STRUCT_AFFECT[32])
	MsgTradingItem         Type = 0x0376 // 886
	MsgTrade               Type = 0x0383 // 899
	MsgQuitTrade           Type = 0x0384 // 900
	MsgCNFCheck            Type = 0x0386 // 902  S→C troca: verificação da própria parte aceita (MSG_STANDARD)
	MsgCombineItem         Type = 0x03A6 // 934  refino base (Anct)
	MsgCombineItemEhre     Type = 0x02D3 // 723
	MsgCombineItemTiny     Type = 0x03C0 // 960
	MsgCombineItemShany    Type = 0x02C4 // 708
	MsgCombineItemAilyn    Type = 0x03B5 // 949
	MsgCombineItemAgatha   Type = 0x03BA // 954
	MsgCombineItemOdin     Type = 0x02D2 // 722
	MsgCombineItemLindy    Type = 0x02C3 // 707
	MsgCombineItemAlquimia Type = 0x02E1 // 737
	MsgCombineItemExtracao Type = 0x02D4 // 724  (MSG_STANDARDPARM2)
	MsgApplyBonus          Type = 0x0277 // 631  distribui pontos de atributo
	MsgSetShortSkill       Type = 0x0378 // 888  define a hotbar de skills (Skill1[4]+Skill2[16])
	MsgQuest               Type = 0x028B // 651  NPC de quest (MSG_STANDARDPARM2)
	MsgMasterGriff         Type = 0x0AD9 // 2777 Mestre Grifo warp request (MSG_MasterGriff)
	MsgReqRanking          Type = 0x039F // 927  duelo / ranking PvP (MSG_STANDARDPARM2)
	MsgCapsuleInfo         Type = 0x02CD // 717  info de cápsula/cash (repassada ao DB)
	MsgPutoutSeal          Type = 0x03CC // 972  selo
	MsgRestart             Type = 0x0289 // 649
	MsgRemoveParty         Type = 0x037E // 894  sair/expulsar (MSG_STANDARDPARM)
	MsgSendReqParty        Type = 0x037F // 895  convida para a party
	MsgAcceptParty         Type = 0x03AB // 939  aceita o convite
	MsgInviteGuild         Type = 0x03D5 // 981  convida para a guilda (MSG_STANDARDPARM2)
	MsgGuildAlly           Type = 0x0E12 // 3602 aliança de guilda
	MsgWar                 Type = 0x0E0E // 3598 declara guerra de guilda (MSG_STANDARDPARM2)
	MsgChallange           Type = 0x028E // 654  desafio de zona / taxa (MSG_STANDARDPARM)
	MsgChallangeConfirm    Type = 0x028F // 655  confirma o desafio (MSG_STANDARDPARM2)
	MsgPing                Type = 0x03A0 // 928  keepalive — sem efeito ao receber (§2)

	// Loja pessoal / autotrade (issue #115, Basedef.h:2165-2325). SendAutoTrade é
	// bidirecional: C→S abre a loja, S→C (SendAutoTrade) a lista de volta.
	MsgSendAutoTrade Type = 0x0397 // 919  C↔S abre loja / lista uma loja (MSG_SendAutoTrade)
	MsgReqBuy        Type = 0x0398 // 920  C→S compra um item de uma loja (MSG_ReqBuy)
	MsgReqTradeList  Type = 0x039A // 922  C→S navega por uma loja (MSG_STANDARDPARM, Parm=sellerID)
	// MsgDeprivate (0x028C, 140|FLAG_CLIENT2GAME) NÃO é autotrade — apesar do
	// título da issue, é uma op de GUILD (destitui um sub-líder, Server.cpp:8694
	// DoDeprivate). Uma loja pessoal é fechada por RemoveTrade (handler.removeTrade),
	// não por esta mensagem, então ela fica de propósito sem rota aqui.
	MsgDeprivate Type = 0x028C // 652  C→S destituição de guilda (fora do escopo da #115)
)

// MsgDeleteCharacterFail é _MSG_DeleteCharacterFail (Basedef.h:1732): o
// dbServer recusou a exclusão (senha errada), respondida com corpo vazio.
const MsgDeleteCharacterFail Type = 0x011B // 283

// Tipos de mensagem TMSrv → cliente que o servidor precisa produzir (protocol-spec.md §3.2,
// actionable subset).
const (
	MsgMessagePanel       Type = 0x0101 // 257
	MsgMessageBoxOk       Type = 0x0102 // 258
	MsgCNFAccountLogin    Type = 0x010A // 266  → character selection
	MsgCNFNewCharacter    Type = 0x0110 // 272
	MsgCNFDeleteCharacter Type = 0x0112 // 274
	MsgCNFCharacterLogin  Type = 0x0114 // 276  entra no mundo (snapshot do personagem)
	MsgCNFCharacterLogout Type = 0x0116 // 278
	MsgCharacterLoginFail Type = 0x0119 // 281
	MsgNewCharacterFail   Type = 0x011A // 282
	MsgAlreadyPlaying     Type = 0x011C // 284
	MsgStillPlaying       Type = 0x011D // 285  a sessão anterior da conta foi fechada; tente de novo
	MsgCreateMob          Type = 0x0364 // 868  spawna no campo de visão
	MsgRemoveMob          Type = 0x0165 // 357  despawna
	MsgPKInfo             Type = 0x0166 // 358  S→C estado PK/guerra de um jogador (MSG_STANDARDPARM, Basedef.h:1954)
	MsgPKMode             Type = 0x0399 // 921  C→S alterna o PlayerKiller (tecla K; MSG_STANDARDPARM, Basedef.h:2164)
	MsgSendItem           Type = 0x0182 // 386  atualiza um slot
	MsgUpdateCarry        Type = 0x0185 // 389  S→C snapshot completo de Carry[64] + Coin
	MsgAccountSecureFail  Type = 0x0FDF // 4063 S→C PIN recusado (_MSG_AccountSecureFail)
	MsgCNFAddParty        Type = 0x037D // 893  S→C sincroniza o slot da lista de party (MSG_CNFAddParty)
	MsgUpdateEquip        Type = 0x036B // 875  S→C atualiza o equipamento visível (_MSG_UpdateEquip, Basedef.h:2094)
	MsgREQShopList        Type = 0x027B // 635  C→S abre a loja do NPC (Target=NPC)
	MsgShopList           Type = 0x017C // 380  S→C lista da loja (Carry do NPC)
	MsgBuy                Type = 0x0379 // 889  C↔S compra do NPC
	MsgSell               Type = 0x037A // 890  C↔S vende ao NPC
	MsgCloseShop          Type = 0x0196 // 406  S→C fecha a janela da loja (sinal, ID=ESCENE_FIELD; Basedef.h:2193)
	MsgDeposit            Type = 0x0388 // 904  C↔S deposita ouro no cargo (MSG_STANDARDPARM)
	MsgWithdraw           Type = 0x0387 // 903  C↔S saca ouro do cargo (MSG_STANDARDPARM)
	MsgUpdateCargoCoin    Type = 0x0339 // 825  S→C ouro do cargo da conta (_MSG_UpdateCargoCoin)
	MsgUpdateEtc          Type = 0x0337 // 823  S→C atualiza ouro/exp/etc (Coin@40)
	MsgCNFGetItem         Type = 0x0171 // 369
	MsgCNFDropItem        Type = 0x0175 // 373
	MsgDecayItem          Type = 0x016F // 367  item do chão sumiu
	MsgCombineComplete    Type = 0x03A7 // 935  resultado do combine (parm 0/1/2)
	MsgSendWarInfo        Type = 0x03A8 // 936  S→C estado de guerra/aliança da guilda
	MsgUpdateScore        Type = 0x0336 // 822  atualização de atributos/score
	MsgSetHpDam           Type = 0x018A // 394  S→C HP + cura/dano flutuantes (ticks de affect)
	MsgSetHpMp            Type = 0x0181 // 385
	MsgSendArchEffect     Type = 0x03B4 // 948  S↔C efeito criado por arch (MSG_STANDARDPARM)
	MsgStartTime          Type = 0x03A1 // 929  S→C sinal de contagem regressiva (MSG_STANDARDPARM, Basedef.h:2526)
	MsgEnvEffect          Type = 0x03A2 // 930  S→C caixa de efeito de área (MSG_EnvEffect, Basedef.h:2529)
	MsgMobLeft            Type = 0x03B0 // 944  S→C contador de mobs restantes (MSG_STANDARDPARM, Basedef.h:2570)
	MsgUpdateWeather      Type = 0x018B // 395  S→C mudança global de clima (MSG_UpdateWeather, Basedef.h:2167)

	// MsgMagicTrumpet é _MSG_MagicTrumpet (Basedef.h:1738, protocol-spec.md:685):
	// 29 | DB2Game | Game2DB | Game2Client. O grito Trombeta Mágica — uma
	// linha de chat para o servidor inteiro. O legado a repassava pelo DBSrv para todo
	// O ClientPatch o pinta de verde ao chegar (ClientPatch_v7662/Functions.cpp:51
	// escreve TNColor::Speak 0xFF00CD00 em 0x4A016A), que é por que o grito não
	// reusa o MsgMessageChat.
	MsgMagicTrumpet Type = 0x0D1D // 3357 S→C grito global (MSG_MagicTrumpet)

	// Loja pessoal / autotrade S→C (issue #115). CreateMobTrade é a pose da loja:
	// a pose é dirigida por este Type (0x0363), NÃO por um valor de CreateType.
	MsgCreateMobTrade Type = 0x0363 // 867  S→C spawna um dono de loja no campo de visão (MSG_CreateMobTrade)
	MsgItemSold       Type = 0x039B // 923  S→C um item saiu de uma loja (MSG_STANDARDPARM2, Parm1=seller Parm2=pos)
	MsgExpPanel       Type = 0x5000 // S→C MSG_Exp_Msg_Panel_ (painel custom de notificação de EXP)
)
