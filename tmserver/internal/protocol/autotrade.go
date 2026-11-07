package protocol

import "fmt"

// Codecs de wire da loja pessoal / autotrade (issue #115), byte a byte contra o
// compiler-verified capture (captura-wyd-autotrade.md, MSVC x86 natural align).
//
//   MSG_SendAutoTrade (0x0397, 196B, C↔S): o conteúdo da loja — Title + 12
//     slots (STRUCT_ITEM + slot de Cargo de origem + preço) + imposto da cidade. Enviado C→S para abrir
//     uma loja, e S→C (SendAutoTrade) para listar uma loja a quem navega.
//   MSG_ReqBuy        (0x0398, 36B,  C→S): compra um slot. Note o padding MSVC de 2
//     bytes depois de TargetID (Price fica no offset 20 do pacote, não 18).
//   MSG_CreateMobTrade(0x0363, 252B, S→C): a pose de "banca" do dono da loja — um
//     MSG_CreateMob (232B) mais Tab[26] + Desc[24] (o título da loja).

// MaxAutoTradeWire é MAX_AUTOTRADE (Basedef.h:140): slots de item da loja.
const MaxAutoTradeWire = 12

// autoTradeTitleLen é MAX_AUTOTRADETITLE (Basedef.h:141): o buffer do título da loja.
const autoTradeTitleLen = 24

// AutoTradeWireItem é um slot da loja: o item ofertado, seu slot de origem no
// Cargo da conta do vendedor (-1 = vazio) e seu preço.
type AutoTradeWireItem struct {
	Item     WireItem
	CarryPos int8
	Coin     int32
}

// MsgSendAutoTradeBody é MSG_SendAutoTrade (Basedef.h:2293). Layout de bytes (corpo,
// depois do cabeçalho de 12 bytes): Title[24]@0, Item[12]×8@24, CarryPos[12]@120,
// Coin[12]×4@132, Tax@180, Index@182 — 184 bytes no total (196 no wire).
type MsgSendAutoTradeBody struct {
	Title string
	Slots [MaxAutoTradeWire]AutoTradeWireItem
	Tax   int16
	Index int16
}

// MsgSendAutoTradeBodySize é o tamanho significativo do corpo (196 − 12).
const MsgSendAutoTradeBodySize = autoTradeTitleLen + MaxAutoTradeWire*ItemSize + MaxAutoTradeWire + MaxAutoTradeWire*4 + 2 + 2

// sendAutoTradeListWireSize é o tamanho de corpo no wire que SendAutoTrade
// realmente emite: o legado passa sizeof(MSG_UpdateCarry)=528 para AddMessage
// (SendFunc.cpp:635), então o pacote de lista S→C leva 528 bytes no total — os 184
// bytes significativos seguidos de 332 zeros ao final. Replicado para paridade de wire;
// o cliente lê só o MSG_SendAutoTrade inicial e ignora a cauda.
const sendAutoTradeListWireSize = 528 - HeaderSize // 516

// Decode lê um MSG_SendAutoTrade do cliente (pedido de abrir loja).
func (m *MsgSendAutoTradeBody) Decode(b []byte) error {
	if len(b) < MsgSendAutoTradeBodySize {
		return fmt.Errorf("protocol: MsgSendAutoTradeBody.Decode: have %d, need %d", len(b), MsgSendAutoTradeBodySize)
	}
	m.Title = cTrimNUL(b[0:autoTradeTitleLen])
	itemOff := autoTradeTitleLen                  // 24
	posOff := itemOff + MaxAutoTradeWire*ItemSize // 120
	coinOff := posOff + MaxAutoTradeWire          // 132
	taxOff := coinOff + MaxAutoTradeWire*4        // 180
	for i := 0; i < MaxAutoTradeWire; i++ {
		m.Slots[i].Item = decodeWireItem(b[itemOff+i*ItemSize:])
		m.Slots[i].CarryPos = int8(b[posOff+i])
		m.Slots[i].Coin = int32(le.Uint32(b[coinOff+i*4:]))
	}
	m.Tax = int16(le.Uint16(b[taxOff:]))
	m.Index = int16(le.Uint16(b[taxOff+2:]))
	return nil
}

// Encode escreve o corpo significativo de 184 bytes do MSG_SendAutoTrade (usado nos
// testes e como base da lista S→C). Use EncodeList para o padding de paridade de wire.
func (m *MsgSendAutoTradeBody) Encode() []byte {
	b := make([]byte, MsgSendAutoTradeBodySize)
	copy(b[0:autoTradeTitleLen], m.Title)
	itemOff := autoTradeTitleLen
	posOff := itemOff + MaxAutoTradeWire*ItemSize
	coinOff := posOff + MaxAutoTradeWire
	taxOff := coinOff + MaxAutoTradeWire*4
	for i := 0; i < MaxAutoTradeWire; i++ {
		encodeWireItem(b[itemOff+i*ItemSize:], m.Slots[i].Item)
		b[posOff+i] = byte(m.Slots[i].CarryPos)
		le.PutUint32(b[coinOff+i*4:], uint32(m.Slots[i].Coin))
	}
	le.PutUint16(b[taxOff:], uint16(m.Tax))
	le.PutUint16(b[taxOff+2:], uint16(m.Index))
	return b
}

// EncodeList monta o corpo da lista de lojas S→C: a struct significativa com padding
// de zeros até os 516 bytes de corpo que o SendAutoTrade legado emite (a
// peculiaridade Size=528). Envie com HEADER.ID = IDScene e Index = a conn do vendedor.
func (m *MsgSendAutoTradeBody) EncodeList() []byte {
	b := make([]byte, sendAutoTradeListWireSize)
	copy(b, m.Encode())
	return b
}

// MsgReqBuyBody é MSG_ReqBuy (Basedef.h:2310). Layout do corpo: Pos@0, TargetID@4,
// (pad de 2 bytes @6), Price@8, Tax@12, item@16 — 24 bytes (36 no wire).
type MsgReqBuyBody struct {
	Pos      int32
	TargetID uint16
	Price    int32
	Tax      int32
	Item     WireItem
}

// MsgReqBuyBodySize é o tamanho do corpo (36 − 12).
const MsgReqBuyBodySize = 24

// Decode lê um MSG_ReqBuy do cliente. Respeita o padding MSVC de 2 bytes depois do
// uint16 TargetID (Price fica alinhado naturalmente no offset 8 do corpo).
func (m *MsgReqBuyBody) Decode(b []byte) error {
	if len(b) < MsgReqBuyBodySize {
		return fmt.Errorf("protocol: MsgReqBuyBody.Decode: have %d, need %d", len(b), MsgReqBuyBodySize)
	}
	m.Pos = int32(le.Uint32(b[0:4]))
	m.TargetID = le.Uint16(b[4:6])
	// b[6:8] padding
	m.Price = int32(le.Uint32(b[8:12]))
	m.Tax = int32(le.Uint32(b[12:16]))
	m.Item = decodeWireItem(b[16:24])
	return nil
}

// Encode escreve um corpo MSG_ReqBuy (round-trip / testes).
func (m *MsgReqBuyBody) Encode() []byte {
	b := make([]byte, MsgReqBuyBodySize)
	le.PutUint32(b[0:4], uint32(m.Pos))
	le.PutUint16(b[4:6], m.TargetID)
	le.PutUint32(b[8:12], uint32(m.Price))
	le.PutUint32(b[12:16], uint32(m.Tax))
	encodeWireItem(b[16:24], m.Item)
	return b
}

// createMobTradeSize é MSG_CreateMobTrade (Basedef.h:1890): MSG_CreateMob (232) +
// Tab[26] + Desc[24].
const createMobTradeSize = 252

// EncodeCreateMobTradeBody monta o corpo de MSG_CreateMobTrade (0x0363): a pose de
// banca do dono da loja. Reusa o layout de campos de MSG_CreateMob (offsets
// compartilhados) e anexa Tab[26]@190 e Desc[24]@216 (o título da loja). A pose é
// escolhida pelo Type da mensagem, não por CreateType (que fica 0 + bits de guild,
// como em GetCreateMobTrade). Envie com HEADER.ID = IDScene; o id da entidade é
// MobID. Quem chama zera Score.Con (d.Con) para paridade com _MSG_SendAutoTrade.cpp:118.
func EncodeCreateMobTradeBody(d CreateMobData, tab []byte, desc string) []byte {
	b := make([]byte, createMobTradeSize-HeaderSize) // 240
	// MSG_CreateMob compartilha seus primeiros 202 bytes (até AnctCode) com a
	// variante de troca; reuse aquele encoder e copie o resultado para o corpo maior.
	copy(b, EncodeCreateMobBody(d))
	copy(b[190:190+26], tab)                 // Tab[26] @abs202 → body190
	copy(b[216:216+autoTradeTitleLen], desc) // Desc[24] @abs228 → body216
	return b
}
