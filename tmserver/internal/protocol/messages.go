package protocol

import (
	"encoding/binary"
	"fmt"
)

// Este arquivo mantém codecs explícitos, offset por offset, das mensagens
// críticas de protocol-spec.md §3.5. Os corpos são os bytes DEPOIS do cabeçalho
// de 12 bytes (Frame.Payload), então os offsets abaixo são o do doc menos HeaderSize.
//
// Mensagens de rede são pack(1) (sem padding) — diferente das structs de save da Fase 2
// (alinhamento natural). Todo campo é lido/escrito por offset explícito,
// little-endian (general-rule 4). Variable-length bodies (Attack Dam[13], Trade)
// e o _AUTH_GAME de billing (196 B, UNVERIFIED) fica adiado — ver o fim
// deste arquivo.

// le é a ordem de bytes do wire para todas as mensagens CPSock (little-endian, x86).
var le = binary.LittleEndian

// MsgAccountLoginBody é o corpo de MSG_AccountLogin (C→S, Type 0x020D),
// protocol-spec.md §3.5 / Basedef.h:1545-1561, pack(1). Total packet 116 bytes.
//
// AccountPassword é texto plano no wire — uma dívida de segurança resolvida com hash no
// import (Fase 2/7). ClientVersion precisa ser igual a AppVersion (7640); essa checagem fica no
// lógica (Fase 4), não codec.
type MsgAccountLoginBody struct {
	AccountPassword [12]byte
	AccountName     [16]byte
	Zero            [52]byte
	ClientVersion   int32
	DBNeedSave      int32
	AdapterName     [4]int32
}

// MsgAccountLoginBodySize é o tamanho do corpo; o pacote completo é HeaderSize+isto.
const MsgAccountLoginBodySize = 104

// Decode lê um corpo MSG_AccountLogin de b (tamanho MsgAccountLoginBodySize).
func (m *MsgAccountLoginBody) Decode(b []byte) error {
	if len(b) < MsgAccountLoginBodySize {
		return fmt.Errorf("protocol: MsgAccountLoginBody.Decode: have %d, need %d", len(b), MsgAccountLoginBodySize)
	}
	copy(m.AccountPassword[:], b[0:12])
	copy(m.AccountName[:], b[12:28])
	copy(m.Zero[:], b[28:80])
	m.ClientVersion = int32(le.Uint32(b[80:84]))
	m.DBNeedSave = int32(le.Uint32(b[84:88]))
	for i := 0; i < 4; i++ {
		m.AdapterName[i] = int32(le.Uint32(b[88+i*4 : 92+i*4]))
	}
	return nil
}

// Encode escreve o corpo MSG_AccountLogin num novo slice MsgAccountLoginBodySize.
func (m *MsgAccountLoginBody) Encode() []byte {
	b := make([]byte, MsgAccountLoginBodySize)
	copy(b[0:12], m.AccountPassword[:])
	copy(b[12:28], m.AccountName[:])
	copy(b[28:80], m.Zero[:])
	le.PutUint32(b[80:84], uint32(m.ClientVersion))
	le.PutUint32(b[84:88], uint32(m.DBNeedSave))
	for i := 0; i < 4; i++ {
		le.PutUint32(b[88+i*4:92+i*4], uint32(m.AdapterName[i]))
	}
	return b
}

// MsgCreateCharacterBody — MSG_CreateCharacter (C→S, 0x020F), Basedef.h:1598-1605.
// Total packet 36 bytes.
type MsgCreateCharacterBody struct {
	Slot     int32
	MobName  [16]byte
	MobClass int32
}

// MsgCreateCharacterBodySize é o tamanho do corpo.
const MsgCreateCharacterBodySize = 24

// Decode lê um corpo MSG_CreateCharacter de b.
func (m *MsgCreateCharacterBody) Decode(b []byte) error {
	if len(b) < MsgCreateCharacterBodySize {
		return fmt.Errorf("protocol: MsgCreateCharacterBody.Decode: have %d, need %d", len(b), MsgCreateCharacterBodySize)
	}
	m.Slot = int32(le.Uint32(b[0:4]))
	copy(m.MobName[:], b[4:20])
	m.MobClass = int32(le.Uint32(b[20:24]))
	return nil
}

// Encode escreve o corpo MSG_CreateCharacter num novo slice.
func (m *MsgCreateCharacterBody) Encode() []byte {
	b := make([]byte, MsgCreateCharacterBodySize)
	le.PutUint32(b[0:4], uint32(m.Slot))
	copy(b[4:20], m.MobName[:])
	le.PutUint32(b[20:24], uint32(m.MobClass))
	return b
}

// MsgDeleteCharacterBody — MSG_DeleteCharacter (C→S, 0x0211), Basedef.h:1608-1616.
// Pacote total de 44 bytes. A senha confirma a exclusão.
type MsgDeleteCharacterBody struct {
	Slot     int32
	MobName  [16]byte
	Password [12]byte
}

// MsgDeleteCharacterBodySize é o tamanho do corpo.
const MsgDeleteCharacterBodySize = 32

// Decode lê um corpo MSG_DeleteCharacter de b.
func (m *MsgDeleteCharacterBody) Decode(b []byte) error {
	if len(b) < MsgDeleteCharacterBodySize {
		return fmt.Errorf("protocol: MsgDeleteCharacterBody.Decode: have %d, need %d", len(b), MsgDeleteCharacterBodySize)
	}
	m.Slot = int32(le.Uint32(b[0:4]))
	copy(m.MobName[:], b[4:20])
	copy(m.Password[:], b[20:32])
	return nil
}

// Encode escreve o corpo MSG_DeleteCharacter num novo slice.
func (m *MsgDeleteCharacterBody) Encode() []byte {
	b := make([]byte, MsgDeleteCharacterBodySize)
	le.PutUint32(b[0:4], uint32(m.Slot))
	copy(b[4:20], m.MobName[:])
	copy(b[20:32], m.Password[:])
	return b
}

// MsgCharacterLoginBody — MSG_CharacterLogin (C→S, 0x0213), Basedef.h:1674-1680.
// Total packet 20 bytes.
type MsgCharacterLoginBody struct {
	Slot  int32
	Force int32
}

// MsgCharacterLoginBodySize é o tamanho do corpo.
const MsgCharacterLoginBodySize = 8

// Decode lê um corpo MSG_CharacterLogin de b.
func (m *MsgCharacterLoginBody) Decode(b []byte) error {
	if len(b) < MsgCharacterLoginBodySize {
		return fmt.Errorf("protocol: MsgCharacterLoginBody.Decode: have %d, need %d", len(b), MsgCharacterLoginBodySize)
	}
	m.Slot = int32(le.Uint32(b[0:4]))
	m.Force = int32(le.Uint32(b[4:8]))
	return nil
}

// Encode escreve o corpo MSG_CharacterLogin num novo slice.
func (m *MsgCharacterLoginBody) Encode() []byte {
	b := make([]byte, MsgCharacterLoginBodySize)
	le.PutUint32(b[0:4], uint32(m.Slot))
	le.PutUint32(b[4:8], uint32(m.Force))
	return b
}

// MsgActionBody — MSG_Action (movement, C↔S, 0x036C/0x0366/0x0368),
// Basedef.h:2070-2082. Total packet 52 bytes. Effect: 0=walking, 1=teleporting.
type MsgActionBody struct {
	PosX    int16
	PosY    int16
	Effect  int32
	Speed   int32
	Route   [24]byte
	TargetX int16
	TargetY int16
}

// MsgActionBodySize é o tamanho do corpo.
const MsgActionBodySize = 40

// Decode lê um corpo MSG_Action de b.
func (m *MsgActionBody) Decode(b []byte) error {
	if len(b) < MsgActionBodySize {
		return fmt.Errorf("protocol: MsgActionBody.Decode: have %d, need %d", len(b), MsgActionBodySize)
	}
	m.PosX = int16(le.Uint16(b[0:2]))
	m.PosY = int16(le.Uint16(b[2:4]))
	m.Effect = int32(le.Uint32(b[4:8]))
	m.Speed = int32(le.Uint32(b[8:12]))
	copy(m.Route[:], b[12:36])
	m.TargetX = int16(le.Uint16(b[36:38]))
	m.TargetY = int16(le.Uint16(b[38:40]))
	return nil
}

// Encode escreve o corpo MSG_Action num novo slice.
func (m *MsgActionBody) Encode() []byte {
	b := make([]byte, MsgActionBodySize)
	le.PutUint16(b[0:2], uint16(m.PosX))
	le.PutUint16(b[2:4], uint16(m.PosY))
	le.PutUint32(b[4:8], uint32(m.Effect))
	le.PutUint32(b[8:12], uint32(m.Speed))
	copy(b[12:36], m.Route[:])
	le.PutUint16(b[36:38], uint16(m.TargetX))
	le.PutUint16(b[38:40], uint16(m.TargetY))
	return b
}

// MsgDropItemBody — MSG_DropItem (C→S, 0x0272), handlers/_MSG_DropItem.md.
//
// UNVERIFIED: os offsets exatos dos campos não estão em protocol-spec.md §3.5; este é um
// layout de melhor esforço (autoconsistente para o codec). Fixe por captura.
type MsgDropItemBody struct {
	SourType int32
	SourPos  int32
	Rotate   int32
	GridX    uint16
	GridY    uint16
}

// MsgDropItemBodySize é o tamanho do corpo.
const MsgDropItemBodySize = 16

// Decode lê um corpo MSG_DropItem.
func (m *MsgDropItemBody) Decode(b []byte) error {
	if len(b) < MsgDropItemBodySize {
		return fmt.Errorf("protocol: MsgDropItemBody.Decode: have %d, need %d", len(b), MsgDropItemBodySize)
	}
	m.SourType = int32(le.Uint32(b[0:4]))
	m.SourPos = int32(le.Uint32(b[4:8]))
	m.Rotate = int32(le.Uint32(b[8:12]))
	m.GridX = le.Uint16(b[12:14])
	m.GridY = le.Uint16(b[14:16])
	return nil
}

// Encode escreve o corpo MSG_DropItem num novo slice.
func (m *MsgDropItemBody) Encode() []byte {
	b := make([]byte, MsgDropItemBodySize)
	le.PutUint32(b[0:4], uint32(m.SourType))
	le.PutUint32(b[4:8], uint32(m.SourPos))
	le.PutUint32(b[8:12], uint32(m.Rotate))
	le.PutUint16(b[12:14], m.GridX)
	le.PutUint16(b[14:16], m.GridY)
	return b
}

// MsgGetItemBody — MSG_GetItem (C→S, 0x0270), handlers/_MSG_GetItem.md. ItemID é
// o id no chão + GroundItemIDOffset (10000).
//
// UNVERIFIED: offsets exatos dos campos não estão em §3.5; layout de melhor esforço autoconsistente.
type MsgGetItemBody struct {
	ItemID   int32
	DestType int32
	DestPos  int32
}

// MsgGetItemBodySize é o tamanho do corpo.
const MsgGetItemBodySize = 12

// Decode lê um corpo MSG_GetItem.
func (m *MsgGetItemBody) Decode(b []byte) error {
	if len(b) < MsgGetItemBodySize {
		return fmt.Errorf("protocol: MsgGetItemBody.Decode: have %d, need %d", len(b), MsgGetItemBodySize)
	}
	m.ItemID = int32(le.Uint32(b[0:4]))
	m.DestType = int32(le.Uint32(b[4:8]))
	m.DestPos = int32(le.Uint32(b[8:12]))
	return nil
}

// Encode escreve o corpo MSG_GetItem num novo slice.
func (m *MsgGetItemBody) Encode() []byte {
	b := make([]byte, MsgGetItemBodySize)
	le.PutUint32(b[0:4], uint32(m.ItemID))
	le.PutUint32(b[4:8], uint32(m.DestType))
	le.PutUint32(b[8:12], uint32(m.DestPos))
	return b
}

// MsgUseItemBody — MSG_UseItem (C↔S, 0x0373), protocol-spec.md §3.5
// (Basedef.h:2196-2206). Documented layout.
type MsgUseItemBody struct {
	SourType int32
	SourPos  int32
	DestType int32
	DestPos  int32
	GridX    uint16
	GridY    uint16
	WarpID   uint16
}

// MsgUseItemBodySize é o tamanho do corpo (total 34 no doc − HeaderSize 12).
const MsgUseItemBodySize = 22

// Decode lê um corpo MSG_UseItem.
func (m *MsgUseItemBody) Decode(b []byte) error {
	if len(b) < MsgUseItemBodySize {
		return fmt.Errorf("protocol: MsgUseItemBody.Decode: have %d, need %d", len(b), MsgUseItemBodySize)
	}
	m.SourType = int32(le.Uint32(b[0:4]))
	m.SourPos = int32(le.Uint32(b[4:8]))
	m.DestType = int32(le.Uint32(b[8:12]))
	m.DestPos = int32(le.Uint32(b[12:16]))
	m.GridX = le.Uint16(b[16:18])
	m.GridY = le.Uint16(b[18:20])
	m.WarpID = le.Uint16(b[20:22])
	return nil
}

// Encode escreve o corpo MSG_UseItem num novo slice.
func (m *MsgUseItemBody) Encode() []byte {
	b := make([]byte, MsgUseItemBodySize)
	le.PutUint32(b[0:4], uint32(m.SourType))
	le.PutUint32(b[4:8], uint32(m.SourPos))
	le.PutUint32(b[8:12], uint32(m.DestType))
	le.PutUint32(b[12:16], uint32(m.DestPos))
	le.PutUint16(b[16:18], m.GridX)
	le.PutUint16(b[18:20], m.GridY)
	le.PutUint16(b[20:22], m.WarpID)
	return b
}

// WireEffect é um par efeito/valor de um STRUCT_ITEM do wire.
type WireEffect struct {
	Effect uint8
	Value  uint8
}

// WireItem é um STRUCT_ITEM como aparece nas mensagens (8 bytes): sIndex + 3
// pares de efeito (Basedef.h:500-522). Index==0 é um slot vazio.
type WireItem struct {
	Index   int16
	Effects [3]WireEffect
}

func decodeWireItem(b []byte) WireItem {
	var it WireItem
	it.Index = int16(le.Uint16(b[0:2]))
	for i := 0; i < 3; i++ {
		it.Effects[i] = WireEffect{Effect: b[2+i*2], Value: b[3+i*2]}
	}
	return it
}

func encodeWireItem(b []byte, it WireItem) {
	le.PutUint16(b[0:2], uint16(it.Index))
	for i := 0; i < 3; i++ {
		b[2+i*2] = it.Effects[i].Effect
		b[3+i*2] = it.Effects[i].Value
	}
}

// MaxTrade é MAX_TRADE (Basedef.h:139): máximo de itens ofertados numa troca.
const MaxTrade = 15

// MsgTradingItemBody — MSG_TradingItem (C→S, 0x0376), protocol-spec.md §3.5
// (Basedef.h:2103-2113): move um item para dentro/fora da janela de troca. WarpID é o
// opponent's id.
type MsgTradingItemBody struct {
	DestPlace uint8
	DestSlot  uint8
	SrcPlace  uint8
	SrcSlot   uint8
	WarpID    int32
}

// MsgTradingItemBodySize é o tamanho do corpo.
const MsgTradingItemBodySize = 8

// Decode lê um corpo MSG_TradingItem.
func (m *MsgTradingItemBody) Decode(b []byte) error {
	if len(b) < MsgTradingItemBodySize {
		return fmt.Errorf("protocol: MsgTradingItemBody.Decode: have %d, need %d", len(b), MsgTradingItemBodySize)
	}
	m.DestPlace, m.DestSlot, m.SrcPlace, m.SrcSlot = b[0], b[1], b[2], b[3]
	m.WarpID = int32(le.Uint32(b[4:8]))
	return nil
}

// Encode escreve o corpo MSG_TradingItem num novo slice.
func (m *MsgTradingItemBody) Encode() []byte {
	b := make([]byte, MsgTradingItemBodySize)
	b[0], b[1], b[2], b[3] = m.DestPlace, m.DestSlot, m.SrcPlace, m.SrcSlot
	le.PutUint32(b[4:8], uint32(m.WarpID))
	return b
}

// MsgTradeBody — MSG_Trade (C↔S, 0x0383), protocol-spec.md §3.5
// (Basedef.h:2435-2445): a confirmação da troca, carregando a oferta inteira. Uma
// entrada de oferta i está ativa quando Item[i].Index != 0; InvenPos[i] é seu slot no carry.
type MsgTradeBody struct {
	Item       [MaxTrade]WireItem
	InvenPos   [MaxTrade]uint8
	TradeMoney int32
	MyCheck    uint8
	OpponentID uint16
}

// MsgTradeBodySize é o corpo com alinhamento natural MSVC x86: 15*8 (itens) +
// 15 (posições) + 1 (pad até o int) + 4 (moeda) + 1 (check) + 1 (pad até o
// short) + 2 (oponente) = 144. O struct do cliente não está num bloco pack(1);
// ler em offsets empacotados devolvia OpponentID = 0 (o padding antes do
// short) e toda troca caía em "oponente fora de jogo".
const MsgTradeBodySize = MaxTrade*ItemSize + MaxTrade + 1 + 4 + 1 + 1 + 2

// ItemSize é o tamanho do STRUCT_ITEM no wire (8 bytes).
const ItemSize = 8

// Decode lê um corpo MSG_Trade.
func (m *MsgTradeBody) Decode(b []byte) error {
	if len(b) < MsgTradeBodySize {
		return fmt.Errorf("protocol: MsgTradeBody.Decode: have %d, need %d", len(b), MsgTradeBodySize)
	}
	for i := 0; i < MaxTrade; i++ {
		m.Item[i] = decodeWireItem(b[i*ItemSize:])
	}
	off := MaxTrade * ItemSize
	copy(m.InvenPos[:], b[off:off+MaxTrade])
	off += MaxTrade + 1 // padding até o int de 4 bytes
	m.TradeMoney = int32(le.Uint32(b[off : off+4]))
	m.MyCheck = b[off+4]
	m.OpponentID = le.Uint16(b[off+6 : off+8]) // +1 de padding antes do short
	return nil
}

// Encode escreve o corpo MSG_Trade num novo slice.
func (m *MsgTradeBody) Encode() []byte {
	b := make([]byte, MsgTradeBodySize)
	for i := 0; i < MaxTrade; i++ {
		encodeWireItem(b[i*ItemSize:], m.Item[i])
	}
	off := MaxTrade * ItemSize
	copy(b[off:off+MaxTrade], m.InvenPos[:])
	off += MaxTrade + 1
	le.PutUint32(b[off:off+4], uint32(m.TradeMoney))
	b[off+4] = m.MyCheck
	le.PutUint16(b[off+6:off+8], m.OpponentID)
	return b
}

// Tipos e detalhes de bônus para MSG_ApplyBonus (protocol-spec.md §3.5).
const (
	BonusScore   = 0 // distribute a stat point
	BonusSpecial = 1
	BonusSkill   = 2

	DetailStr = 0
	DetailInt = 1
	DetailDex = 2
	DetailCon = 3
)

// MsgApplyBonusBody — MSG_ApplyBonus (C→S, 0x0277), protocol-spec.md §3.5
// (Basedef.h:2144-2151): distribute a free attribute point.
type MsgApplyBonusBody struct {
	BonusType int16
	Detail    int16
	TargetID  uint16
}

// MsgApplyBonusBodySize é o tamanho do corpo.
const MsgApplyBonusBodySize = 6

// Decode lê um corpo MSG_ApplyBonus.
func (m *MsgApplyBonusBody) Decode(b []byte) error {
	if len(b) < MsgApplyBonusBodySize {
		return fmt.Errorf("protocol: MsgApplyBonusBody.Decode: have %d, need %d", len(b), MsgApplyBonusBodySize)
	}
	m.BonusType = int16(le.Uint16(b[0:2]))
	m.Detail = int16(le.Uint16(b[2:4]))
	m.TargetID = le.Uint16(b[4:6])
	return nil
}

// Encode escreve o corpo MSG_ApplyBonus num novo slice.
func (m *MsgApplyBonusBody) Encode() []byte {
	b := make([]byte, MsgApplyBonusBodySize)
	le.PutUint16(b[0:2], uint16(m.BonusType))
	le.PutUint16(b[2:4], uint16(m.Detail))
	le.PutUint16(b[4:6], m.TargetID)
	return b
}

// MsgWhisperBody — MSG_MessageWhisper (C→S, 0x0334): MobName é o jogador de
// destino (ou uma palavra-chave de comando); String é o texto da mensagem (terminado
// em NUL, o resto do corpo).
type MsgWhisperBody struct {
	MobName [16]byte
	String  []byte
}

// Decode lê um corpo MSG_MessageWhisper (MobName + string final).
func (m *MsgWhisperBody) Decode(b []byte) error {
	if len(b) < 16 {
		return fmt.Errorf("protocol: MsgWhisperBody.Decode: have %d, need >= 16", len(b))
	}
	copy(m.MobName[:], b[0:16])
	m.String = append([]byte(nil), b[16:]...)
	return nil
}

// Encode escreve o corpo MSG_MessageWhisper num novo slice.
func (m *MsgWhisperBody) Encode() []byte {
	b := make([]byte, 16+len(m.String))
	copy(b[0:16], m.MobName[:])
	copy(b[16:], m.String)
	return b
}

// MagicTrumpetBodySize é MSG_MagicTrumpet menos o cabeçalho (Basedef.h:1739-1745):
// char String[MESSAGE_LENGTH] seguido de int Color — 100 bytes, então o frame no
// wire é 112.
const MagicTrumpetBodySize = MessageLength + 4

// magicTrumpetMarker é o offset do byte que o legado marca como 1 logo após
// copiar o texto (_MSG_MessageWhisper.cpp:154, "sm_mt.String[94] = 1").
const magicTrumpetMarker = 94

// EncodeMagicTrumpetBody devolve o corpo MSG_MagicTrumpet carregando text (já
// formatado como "[Nome]> mensagem" por quem chama).
//
// Notas de paridade: o texto é truncado em magicTrumpetMarker-1 bytes para a string C
// sempre terminar antes do byte marcador — o strncpy(String, temp, 96) legado
// podia enviar uma string sem terminador e depois sobrescrever o byte 94 no meio do texto. Color
// fica 0: o legado só fez memset dos primeiros sizeof(MSG_STANDARDPARM) bytes da
// struct, então enviava lixo de pilha não inicializado ali, e o cliente corrigido
// escolhe a cor sozinho (ClientPatch_v7662/Functions.cpp:51).
func EncodeMagicTrumpetBody(text string) []byte {
	b := make([]byte, MagicTrumpetBodySize)
	copy(b[:magicTrumpetMarker-1], text)
	b[magicTrumpetMarker] = 1
	return b
}

// MsgSendReqPartyBody é MSG_SendReqParty (C↔S, 0x037F): PartyID é o líder
// e Unk carrega o jogador convidado no pedido do cliente. Esta struct está
// fora dos blocos pack(1) legados, então o campo int mantém o padding do MSVC x86
// depois de MobName (Basedef.h:2261-2279).
type MsgSendReqPartyBody struct {
	Class    uint8
	PartyPos uint8
	Level    uint16
	MaxHP    uint16
	HP       uint16
	PartyID  int16
	MobName  [16]byte
	Unk      int32
	Target   int16
}

// MsgSendReqPartyBodySize é o corpo com alinhamento natural MSVC x86:
// Class+PartyPos+Level+MaxHp+Hp+PartyID (10) + MobName (16) + 2 de padding +
// unk (4) + Target (2) + 2 de padding final = 36 (40 com o header). O cliente
// 7662 envia o alvo em `unk` no offset 28; lendo em 26 (pack(1)) um convite
// para a conn 2 virava 131072 (2<<16), porque os dois bytes lidos eram o
// padding — daí a resposta "jogador não conectado".
const MsgSendReqPartyBodySize = 36

// Decode lê um corpo MSG_SendReqParty.
func (m *MsgSendReqPartyBody) Decode(b []byte) error {
	if len(b) < MsgSendReqPartyBodySize {
		return fmt.Errorf("protocol: MsgSendReqPartyBody.Decode: have %d, need %d", len(b), MsgSendReqPartyBodySize)
	}
	m.Class = b[0]
	m.PartyPos = b[1]
	m.Level = le.Uint16(b[2:4])
	m.MaxHP = le.Uint16(b[4:6])
	m.HP = le.Uint16(b[6:8])
	m.PartyID = int16(le.Uint16(b[8:10]))
	copy(m.MobName[:], b[10:26])
	m.Unk = int32(le.Uint32(b[28:32]))
	m.Target = int16(le.Uint16(b[32:34]))
	return nil
}

// Encode escreve o corpo MSG_SendReqParty num novo slice.
func (m *MsgSendReqPartyBody) Encode() []byte {
	b := make([]byte, MsgSendReqPartyBodySize)
	b[0] = m.Class
	b[1] = m.PartyPos
	le.PutUint16(b[2:4], m.Level)
	le.PutUint16(b[4:6], m.MaxHP)
	le.PutUint16(b[6:8], m.HP)
	le.PutUint16(b[8:10], uint16(m.PartyID))
	copy(b[10:26], m.MobName[:])
	le.PutUint32(b[28:32], uint32(m.Unk))
	le.PutUint16(b[32:34], uint16(m.Target))
	return b
}

// MsgAcceptPartyBody é MSG_AcceptParty (C↔S, 0x03AB): LeaderID é quem convidou
// o jogador. pack(1): sizeof 30, corpo 18 — o tamanho por alinhamento natural que o port
// assumiu (20) fazia o Decode recusar todo accept, então a party nunca se formava.
type MsgAcceptPartyBody struct {
	LeaderID int16
	MobName  [16]byte
}

// MsgAcceptPartyBodySize é o tamanho do corpo.
const MsgAcceptPartyBodySize = 18

// Decode lê um corpo MSG_AcceptParty.
func (m *MsgAcceptPartyBody) Decode(b []byte) error {
	if len(b) < MsgAcceptPartyBodySize {
		return fmt.Errorf("protocol: MsgAcceptPartyBody.Decode: have %d, need %d", len(b), MsgAcceptPartyBodySize)
	}
	m.LeaderID = int16(le.Uint16(b[0:2]))
	copy(m.MobName[:], b[2:18])
	return nil
}

// Encode escreve o corpo MSG_AcceptParty num novo slice.
func (m *MsgAcceptPartyBody) Encode() []byte {
	b := make([]byte, MsgAcceptPartyBodySize)
	le.PutUint16(b[0:2], uint16(m.LeaderID))
	copy(b[2:18], m.MobName[:])
	return b
}

// MsgRemovePartyBody é MSG_RemoveParty (S→C, 0x037E), enviado pelo
// SendRemoveParty legado. LeaderConn é 0 ao limpar toda a UI de party do
// destinatário, ou a conn de quem saiu/foi expulso ao notificar os membros restantes.
type MsgRemovePartyBody struct {
	LeaderConn int16
	Unk        int16
}

// MsgRemovePartyBodySize é sizeof(MSG_RemoveParty) menos o cabeçalho comum.
const MsgRemovePartyBodySize = 4

// Encode escreve o corpo MSG_RemoveParty num novo slice.
func (m *MsgRemovePartyBody) Encode() []byte {
	b := make([]byte, MsgRemovePartyBodySize)
	le.PutUint16(b[0:2], uint16(m.LeaderConn))
	le.PutUint16(b[2:4], uint16(m.Unk))
	return b
}

// MsgCNFAddPartyBody é MSG_CNFAddParty (S→C, 0x037D), emitido por SendAddParty
// para sincronizar um slot da UI de party. Para PartyPos 0 o legado escreve LeaderConn
// como id de entidade exibido; para slots não zero escreve 30000 e carrega o
// id de entidade em PartyID.
type MsgCNFAddPartyBody struct {
	LeaderConn uint16
	Level      uint16
	MaxHP      uint16
	HP         uint16
	PartyID    uint16
	MobName    [16]byte
	Target     uint16
}

// MsgCNFAddPartyBodySize é o tamanho do corpo após o cabeçalho comum de 12 bytes.
const MsgCNFAddPartyBodySize = 28

// Encode escreve o corpo MSG_CNFAddParty num novo slice.
func (m *MsgCNFAddPartyBody) Encode() []byte {
	b := make([]byte, MsgCNFAddPartyBodySize)
	le.PutUint16(b[0:2], m.LeaderConn)
	le.PutUint16(b[2:4], m.Level)
	le.PutUint16(b[4:6], m.MaxHP)
	le.PutUint16(b[6:8], m.HP)
	le.PutUint16(b[8:10], m.PartyID)
	copy(b[10:26], m.MobName[:])
	le.PutUint16(b[26:28], m.Target)
	return b
}

// StandardParm lê o único campo int32 inicial (Parm) de um corpo
// MSG_STANDARDPARM (usado por Deposit/Withdraw: Parm = a quantidade de ouro).
func StandardParm(b []byte) (parm int32, ok bool) {
	if len(b) < 4 {
		return 0, false
	}
	return int32(le.Uint32(b[0:4])), true
}

// StandardParm2 lê os dois campos int32 iniciais (Parm1, Parm2) de um
// corpo MSG_STANDARDPARM2 (usado por InviteGuild, War, ChallangeConfirm).
func StandardParm2(b []byte) (parm1, parm2 int32, ok bool) {
	if len(b) < 8 {
		return 0, 0, false
	}
	return int32(le.Uint32(b[0:4])), int32(le.Uint32(b[4:8])), true
}

// EncodeStandardParm monta um corpo MSG_STANDARDPARM (Parm int32 único), ex.:
// MSG_PKInfo (SendPKInfo, SendFunc.cpp:1869).
func EncodeStandardParm(parm int32) []byte {
	b := make([]byte, 4)
	le.PutUint32(b, uint32(parm))
	return b
}

// EncodeSetShortSkill monta o corpo MSG_SetShortSkill (Basedef.h:2115):
// Skill1[4] (MOB.SkillBar) depois Skill2[16] (CUser.CharShortSkill), os mesmos
// 20 bytes que o cliente envia ao editar a hotbar.
func EncodeSetShortSkill(skill1 [4]uint8, skill2 [16]uint8) []byte {
	b := make([]byte, 0, len(skill1)+len(skill2))
	b = append(b, skill1[:]...)
	return append(b, skill2[:]...)
}

// EncodeStandardParm2 monta um corpo MSG_STANDARDPARM2.
func EncodeStandardParm2(parm1, parm2 int32) []byte {
	b := make([]byte, 8)
	le.PutUint32(b[0:4], uint32(parm1))
	le.PutUint32(b[4:8], uint32(parm2))
	return b
}

// EncodeEnvEffect monta um corpo MSG_EnvEffect (x1,y1,x2,y2,Effect,EffectParm —
// seis campos int16, Basedef.h:2529), usado para o muro de fechamento da arena de duelo
// visual (SendEnvEffect, SendFunc.cpp:456).
func EncodeEnvEffect(x1, y1, x2, y2, effect, effectParm int16) []byte {
	b := make([]byte, 12)
	le.PutUint16(b[0:2], uint16(x1))
	le.PutUint16(b[2:4], uint16(y1))
	le.PutUint16(b[4:6], uint16(x2))
	le.PutUint16(b[6:8], uint16(y2))
	le.PutUint16(b[8:10], uint16(effect))
	le.PutUint16(b[10:12], uint16(effectParm))
	return b
}

// MsgUpdateWeatherBodySize é o payload do MSG_UpdateWeather: um único int32
// CurrentWeather (Basedef.h:2167-2173), então 16 bytes no wire com o cabeçalho.
const MsgUpdateWeatherBodySize = 4

// EncodeUpdateWeather monta um corpo MSG_UpdateWeather. O legado o transmite
// para toda conexão USER_PLAY com HEADER.ID = ESCENE_FIELD, não por célula de
// (SendWeather, SendFunc.cpp:1669-1696).
//
// Note que a divergência de largura é legado, não bug aqui: este pacote carrega um
// int32, enquanto o mesmo valor vai em MSG_CNFCharacterLogin como unsigned short
// (Basedef.h:1682-1695). Só 0/1/2 são enviados, então ambos são equivalentes.
func EncodeUpdateWeather(weather int32) []byte {
	b := make([]byte, MsgUpdateWeatherBodySize)
	le.PutUint32(b, uint32(weather))
	return b
}

// EncodeStandardParm3 monta um corpo de sinal de três int32, usado por SendWarInfo
// (Guild, castle clan, ally/war target).
func EncodeStandardParm3(parm1, parm2, parm3 int32) []byte {
	b := make([]byte, 12)
	le.PutUint32(b[0:4], uint32(parm1))
	le.PutUint32(b[4:8], uint32(parm2))
	le.PutUint32(b[8:12], uint32(parm3))
	return b
}

// MaxCombine é o número de slots de entrada num pacote de combine: MAX_COMBINE
// (Basedef.h:173). A receita base Anct só usa Item[0]/Item[1], mas a
// receita "+12+" da família Odin (GetMatchCombineOdin, GetFunc.cpp:551) lê até
// Item[6] — 7 dos 8 slots — então o tamanho completo é exigido para paridade.
const MaxCombine = 8

// MsgCombineItemBody — MSG_CombineItem (C→S, 0x03A6 e as
// variantes baseadas em Item[]), game-rules.md §3.1. Uma entrada i fica ativa quando
// Item[i].Index != 0; InvenPos[i] é seu slot no carry.
//
// O layout é a struct C literal (Basedef.h:2344-2351):
//
//	struct MSG_CombineItem { _MSG; STRUCT_ITEM Item[MAX_COMBINE]; char InvenPos[MAX_COMBINE]; };
//
// então o corpo é 8*8 + 8 = 72 bytes e o frame 84.
type MsgCombineItemBody struct {
	Item     [MaxCombine]WireItem
	InvenPos [MaxCombine]uint8
}

// MsgCombineItemBodySize é o tamanho do corpo.
const MsgCombineItemBodySize = MaxCombine*ItemSize + MaxCombine

// Decode lê um corpo MSG_CombineItem.
func (m *MsgCombineItemBody) Decode(b []byte) error {
	if len(b) < MsgCombineItemBodySize {
		return fmt.Errorf("protocol: MsgCombineItemBody.Decode: have %d, need %d", len(b), MsgCombineItemBodySize)
	}
	for i := 0; i < MaxCombine; i++ {
		m.Item[i] = decodeWireItem(b[i*ItemSize:])
	}
	copy(m.InvenPos[:], b[MaxCombine*ItemSize:MaxCombine*ItemSize+MaxCombine])
	return nil
}

// Encode escreve o corpo MSG_CombineItem num novo slice.
func (m *MsgCombineItemBody) Encode() []byte {
	b := make([]byte, MsgCombineItemBodySize)
	for i := 0; i < MaxCombine; i++ {
		encodeWireItem(b[i*ItemSize:], m.Item[i])
	}
	copy(b[MaxCombine*ItemSize:], m.InvenPos[:])
	return b
}

// MaxTarget é MAX_TARGET (Basedef.h:233): o número máximo de alvos por ataque.
const MaxTarget = 13

// Layout de Dam[] do Attack no corpo: cada STRUCT_DAM é {int32 TargetID; int32
// Damage} at MsgAttackDamOffset + i*MsgAttackDamStride (protocol-spec.md §3.5).
const (
	MsgAttackDamOffset = 48 // doc offset 60 − HeaderSize 12
	MsgAttackDamStride = 8
)

// DamEntry é um STRUCT_DAM (alvo + dano resolvido). O dano é
// autoritativo do servidor; valores negativos são códigos de erro de acerto/bloqueio.
type DamEntry struct {
	TargetID int32
	Damage   int32
}

// MsgAttackBody decodifica os campos de MSG_Attack / MSG_AttackOne / MSG_AttackTwo
// que o servidor precisa (protocol-spec.md §3.5, Basedef.h:2400-2432). O número de
// entradas Dam é derivado do tamanho do corpo (as variantes diferem só em como
// quantos alvos eles carregam).
type MsgAttackBody struct {
	CurrentHp      int32
	CurrentExp     int64
	PosX           uint16
	PosY           uint16
	TargetX        uint16
	TargetY        uint16
	AttackerID     uint16
	Progress       uint16
	Motion         uint8
	DoubleCritical uint8
	CurrentMp      int32
	SkillIndex     int16
	ReqMp          int16
	Dam            []DamEntry
}

// Decode lê um corpo de ataque. b precisa ter ao menos MsgAttackDamOffset bytes.
func (m *MsgAttackBody) Decode(b []byte) error {
	if len(b) < MsgAttackDamOffset {
		return fmt.Errorf("protocol: MsgAttackBody.Decode: have %d, need >= %d", len(b), MsgAttackDamOffset)
	}
	m.CurrentHp = int32(le.Uint32(b[4:8]))
	m.CurrentExp = int64(le.Uint64(b[12:20]))
	m.PosX = le.Uint16(b[22:24])
	m.PosY = le.Uint16(b[24:26])
	m.TargetX = le.Uint16(b[26:28])
	m.TargetY = le.Uint16(b[28:30])
	m.AttackerID = le.Uint16(b[30:32])
	m.Progress = le.Uint16(b[32:34])
	m.Motion = b[34]
	m.DoubleCritical = b[36]
	m.CurrentMp = int32(le.Uint32(b[40:44]))
	m.SkillIndex = int16(le.Uint16(b[44:46]))
	m.ReqMp = int16(le.Uint16(b[46:48]))

	n := (len(b) - MsgAttackDamOffset) / MsgAttackDamStride
	if n > MaxTarget {
		n = MaxTarget
	}
	m.Dam = make([]DamEntry, n)
	for i := 0; i < n; i++ {
		off := MsgAttackDamOffset + i*MsgAttackDamStride
		m.Dam[i] = DamEntry{
			TargetID: int32(le.Uint32(b[off : off+4])),
			Damage:   int32(le.Uint32(b[off+4 : off+8])),
		}
	}
	return nil
}

// Encode serializa um corpo de ataque (campos fixos + entradas Dam). Reservado/desconhecido
// bytes são zero.
func (m *MsgAttackBody) Encode() []byte {
	b := make([]byte, MsgAttackDamOffset+len(m.Dam)*MsgAttackDamStride)
	le.PutUint32(b[4:8], uint32(m.CurrentHp))
	le.PutUint64(b[12:20], uint64(m.CurrentExp))
	le.PutUint16(b[22:24], m.PosX)
	le.PutUint16(b[24:26], m.PosY)
	le.PutUint16(b[26:28], m.TargetX)
	le.PutUint16(b[28:30], m.TargetY)
	le.PutUint16(b[30:32], m.AttackerID)
	le.PutUint16(b[32:34], m.Progress)
	b[34] = m.Motion
	b[36] = m.DoubleCritical
	le.PutUint32(b[40:44], uint32(m.CurrentMp))
	le.PutUint16(b[44:46], uint16(m.SkillIndex))
	le.PutUint16(b[46:48], uint16(m.ReqMp))
	for i, d := range m.Dam {
		off := MsgAttackDamOffset + i*MsgAttackDamStride
		le.PutUint32(b[off:off+4], uint32(d.TargetID))
		le.PutUint32(b[off+4:off+8], uint32(d.Damage))
	}
	return b
}

// AuthGameSize é sizeof(_AUTH_GAME), o pacote fixo de 196 bytes de billing TMSrv↔BISrv
// packet (CPSock.h:132, protocol-spec.md §1.6).
//
// UNVERIFIED: o layout interno de campos do _AUTH_GAME não está confirmado — a
// struct viva é char Unk[196] e o layout de referência "FROM TANTRA" não foi
// casado com este servidor (protocol-spec.md §4.3, README pendência). Ele precisa ser
// capturado do tráfego real TMSrv↔BISrv antes de ser decodificado; implementado na
// Fase 6. Ver o teste ignorado em messages_test.go.
const AuthGameSize = 196

// MsgDeleteItemBody — MSG_DeleteItem (C→S, 0x02E4), Basedef.h:2371-2378: destroy a
// item de carry. Slot é o índice no carry; SIndex é o id do template do item (o
// cliente envia para validação, mas o handler legado limpa o slot incondicionalmente).
type MsgDeleteItemBody struct {
	Slot   int32
	SIndex int32
}

// MsgDeleteItemBodySize é o tamanho do corpo.
const MsgDeleteItemBodySize = 8

// Decode lê um corpo MSG_DeleteItem.
func (m *MsgDeleteItemBody) Decode(b []byte) error {
	if len(b) < MsgDeleteItemBodySize {
		return fmt.Errorf("protocol: MsgDeleteItemBody.Decode: have %d, need %d", len(b), MsgDeleteItemBodySize)
	}
	m.Slot = int32(le.Uint32(b[0:4]))
	m.SIndex = int32(le.Uint32(b[4:8]))
	return nil
}

// Encode escreve o corpo MSG_DeleteItem num novo slice.
func (m *MsgDeleteItemBody) Encode() []byte {
	b := make([]byte, MsgDeleteItemBodySize)
	le.PutUint32(b[0:4], uint32(m.Slot))
	le.PutUint32(b[4:8], uint32(m.SIndex))
	return b
}

// MsgSplitItemBody — MSG_SplitItem (C→S, 0x02E5), Basedef.h:2381-2390: peel Num
// unidades da pilha no Slot do carry para uma pilha nova. SIndex é o id do template
// do item (validado contra uma lista de dividíveis no handler).
type MsgSplitItemBody struct {
	Slot   int32
	SIndex int32
	Num    int32
}

// MsgSplitItemBodySize é o tamanho do corpo.
const MsgSplitItemBodySize = 12

// Decode lê um corpo MSG_SplitItem.
func (m *MsgSplitItemBody) Decode(b []byte) error {
	if len(b) < MsgSplitItemBodySize {
		return fmt.Errorf("protocol: MsgSplitItemBody.Decode: have %d, need %d", len(b), MsgSplitItemBodySize)
	}
	m.Slot = int32(le.Uint32(b[0:4]))
	m.SIndex = int32(le.Uint32(b[4:8]))
	m.Num = int32(le.Uint32(b[8:12]))
	return nil
}

// Encode escreve o corpo MSG_SplitItem num novo slice.
func (m *MsgSplitItemBody) Encode() []byte {
	b := make([]byte, MsgSplitItemBodySize)
	le.PutUint32(b[0:4], uint32(m.Slot))
	le.PutUint32(b[4:8], uint32(m.SIndex))
	le.PutUint32(b[8:12], uint32(m.Num))
	return b
}

// MsgUpdateItemBody — MSG_UpdateItem (C↔S, 0x0374), Basedef.h:2214-2221: gate/door
// interação. ItemID é o id do item no mundo (id no chão + GroundItemIDOffset); State
// é o estado de gate pedido. Na entrada é um pedido de abertura; o servidor ecoa o
// mesmo corpo aos clientes próximos para refletir o novo estado (GridMulticast).
type MsgUpdateItemBody struct {
	ItemID int32
	State  int32
}

// MsgUpdateItemBodySize é o tamanho do corpo.
const MsgUpdateItemBodySize = 8

// Decode lê um corpo MSG_UpdateItem.
func (m *MsgUpdateItemBody) Decode(b []byte) error {
	if len(b) < MsgUpdateItemBodySize {
		return fmt.Errorf("protocol: MsgUpdateItemBody.Decode: have %d, need %d", len(b), MsgUpdateItemBodySize)
	}
	m.ItemID = int32(le.Uint32(b[0:4]))
	m.State = int32(le.Uint32(b[4:8]))
	return nil
}

// Encode escreve o corpo MSG_UpdateItem num novo slice.
func (m *MsgUpdateItemBody) Encode() []byte {
	b := make([]byte, MsgUpdateItemBodySize)
	le.PutUint32(b[0:4], uint32(m.ItemID))
	le.PutUint32(b[4:8], uint32(m.State))
	return b
}

// MsgAccountSecureBody — MSG_AccountSecure (C↔S, 0x0FDE), Basedef.h:1588-1595: o
// PIN numérico. NumericToken é o PIN de 6 caracteres (texto plano legado);
// ChangeNumeric é 0 para verificar, 1 para definir/trocar. O PIN nunca é gravado em
// texto plano — o dbServer o transforma em hash com argon2id (issue #120).
type MsgAccountSecureBody struct {
	NumericToken  [6]byte
	Unknown       [10]byte
	ChangeNumeric int32
}

// MsgAccountSecureBodySize é o tamanho do corpo.
const MsgAccountSecureBodySize = 20

// Decode lê um corpo MSG_AccountSecure.
func (m *MsgAccountSecureBody) Decode(b []byte) error {
	if len(b) < MsgAccountSecureBodySize {
		return fmt.Errorf("protocol: MsgAccountSecureBody.Decode: have %d, need %d", len(b), MsgAccountSecureBodySize)
	}
	copy(m.NumericToken[:], b[0:6])
	copy(m.Unknown[:], b[6:16])
	m.ChangeNumeric = int32(le.Uint32(b[16:20]))
	return nil
}

// Encode escreve o corpo MSG_AccountSecure num novo slice.
func (m *MsgAccountSecureBody) Encode() []byte {
	b := make([]byte, MsgAccountSecureBodySize)
	copy(b[0:6], m.NumericToken[:])
	copy(b[6:16], m.Unknown[:])
	le.PutUint32(b[16:20], uint32(m.ChangeNumeric))
	return b
}

// PIN devolve o token numérico como string, cortado no primeiro NUL. O PIN é
// entrada de verificação/definição para o dbServer; nunca é logado nem gravado em texto plano.
func (m *MsgAccountSecureBody) PIN() string {
	return cTrimNUL(m.NumericToken[:])
}

// cTrimNUL devolve o prefixo de b até o primeiro NUL — um auxiliar para virar
// vetores char de tamanho fixo preenchidos com NUL (nomes, senhas) em strings Go na
// handler boundary (Phase 4).
func cTrimNUL(b []byte) string {
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}
