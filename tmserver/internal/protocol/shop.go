package protocol

// Pacotes de loja de NPC (byte a byte, verificados pelo compilador).
//
//   MSG_REQShopList (0x027B, 16B, C→S): Target@12 = MobID do NPC — abre a loja dele.
//   MSG_ShopList    (0x017C, 236B, S→C): a lista de 27 itens vem do
//     próprio Carry[] (SendFunc.cpp: List[i] = NPC.Carry[(i%9)+(i/9)*27]).

const (
	maxShopList    = 27
	shopListSize   = 236
	structMobEquip = 140 // Equip[16] offset within STRUCT_MOB
	structMobCarry = 268 // Carry[64] offset within STRUCT_MOB
)

// MobCarry lê os 64 slots de inventário (STRUCT_ITEM[64] @268) de um
// template STRUCT_MOB cru — para um NPC este é o estoque da loja.
func MobCarry(mob816 []byte) [64]SelItem {
	var c [64]SelItem
	for i := 0; i < 64; i++ {
		o := structMobCarry + i*8
		c[i].Index = le.Uint16(mob816[o : o+2])
		c[i].Eff[0] = [2]uint8{mob816[o+2], mob816[o+3]}
		c[i].Eff[1] = [2]uint8{mob816[o+4], mob816[o+5]}
		c[i].Eff[2] = [2]uint8{mob816[o+6], mob816[o+7]}
	}
	return c
}

// ShopSlot mapeia um índice da lista de loja (0..26) para o slot do Carry do NPC de onde lê
// (3 abas de 9: Carry[0..8], [27..35], [54..62]).
func ShopSlot(i int) int { return (i % 9) + (i/9)*maxShopList }

// EncodeShopListBody monta o corpo de MSG_ShopList (0x017C). list guarda os 27
// itens da loja (já mapeados via ShopSlot). Envie com HEADER.ID = IDScene (30000).
func EncodeShopListBody(shopType int32, list [maxShopList]SelItem, tax int32) []byte {
	b := make([]byte, shopListSize-HeaderSize) // 224
	le.PutUint32(b[0:], uint32(shopType))      // ShopType @abs12 → body0
	for i := 0; i < maxShopList; i++ {         // List[27] @abs16 → body4 (8B each)
		writeSelItem(b[4+i*8:], list[i])
	}
	le.PutUint32(b[220:], uint32(tax)) // Tax @abs232 → body220
	return b
}

// ITEM_PLACE_* (Basedef.h:146) — onde um item mora.
const (
	ItemPlaceEquip = 0
	ItemPlaceCarry = 1
	ItemPlaceCargo = 2
)

// EncodeSendItemBody monta MSG_SendItem (0x0182, 24B): atualiza um slot de inventário
// no cliente. invType = ITEM_PLACE_*, slot = índice, item = o STRUCT_ITEM
// (sIndex 0 limpa o slot). Envie com HEADER.ID = conn.
func EncodeSendItemBody(invType, slot int, item SelItem) []byte {
	b := make([]byte, 24-HeaderSize)     // 12
	le.PutUint16(b[0:], uint16(invType)) // invType @abs12 → body0
	le.PutUint16(b[2:], uint16(slot))    // Slot @abs14 → body2
	writeSelItem(b[4:], item)            // item @abs16 → body4 (8B)
	return b
}

const updateEtcSize = 48

// EncodeUpdateEtcCoin monta MSG_UpdateEtc (0x0337) carregando o ouro do jogador.
// Coin fica no offset 40 do pacote (corpo 28). Outros campos (Exp/Learn/bônus) são 0 —
// serve para atualizar o ouro após uma transação de loja. Envie com HEADER.ID = conn.
func EncodeUpdateEtcCoin(coin int32) []byte {
	b := make([]byte, updateEtcSize-HeaderSize) // 36
	le.PutUint32(b[28:], uint32(coin))          // Coin @abs40 → body28
	return b
}

// UpdateEtcData espelha MSG_UpdateEtc (SendFunc.cpp SendEtc, Basedef.h): os campos
// de status que o cliente lê FORA do STRUCT_SCORE. Notavelmente ScoreBonus (pontos
// livres de atributo) vive aqui, NÃO no MSG_UpdateScore — então o cliente só fica
// sabendo dos pontos ganhos no level-up por este pacote. O original sempre envia a
// struct inteira, então uma atualização parcial (só ouro) zeraria o ScoreBonus/Exp.
type UpdateEtcData struct {
	Hold         uint32
	Exp          int64
	Learn        int64
	ScoreBonus   uint16
	SpecialBonus uint16
	SkillBonus   uint16
	Magic        uint16
	Coin         int32
}

// EncodeUpdateEtc monta o MSG_UpdateEtc completo (0x0337). Os offsets de campo são os
// layout MSVC alinhado naturalmente de MSG_UpdateEtc (Basedef.h): Hold@corpo0, Exp@corpo4,
// Learn@body12, ScoreBonus@body20, SpecialBonus@body22, SkillBonus@body24,
// Magic@corpo26, Coin@corpo28. Envie com HEADER.ID = conn.
func EncodeUpdateEtc(d UpdateEtcData) []byte {
	b := make([]byte, updateEtcSize-HeaderSize) // 36
	le.PutUint32(b[0:], d.Hold)                 // Hold @abs12 → body0
	le.PutUint64(b[4:], uint64(d.Exp))          // Exp @abs16 → body4
	le.PutUint64(b[12:], uint64(d.Learn))       // Learn @abs24 → body12
	le.PutUint16(b[20:], d.ScoreBonus)          // ScoreBonus @abs32 → body20
	le.PutUint16(b[22:], d.SpecialBonus)        // SpecialBonus @abs34 → body22
	le.PutUint16(b[24:], d.SkillBonus)          // SkillBonus @abs36 → body24
	le.PutUint16(b[26:], d.Magic)               // Magic @abs38 → body26
	le.PutUint32(b[28:], uint32(d.Coin))        // Coin @abs40 → body28
	return b
}
