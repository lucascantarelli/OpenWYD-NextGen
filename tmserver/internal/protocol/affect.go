package protocol

// MaxAffect é o número de slots de affect (buff/debuff) por entidade (MAX_AFFECT).
const MaxAffect = 32

// sendAffectSize é o MSG_SendAffect completo (0x03B9): HEADER + STRUCT_AFFECT[32].
// STRUCT_AFFECT tem 8 bytes (Type u8, Value u8, Level u16, Time u32), então o
// corpo é 32*8 = 256 bytes e o pacote é 268 (captura-wyd-affect-divina.md §A,D).
const sendAffectSize = 268

// AffectData espelha STRUCT_AFFECT (8 bytes): um slot de buff/debuff com tempo.
type AffectData struct {
	Type  uint8
	Value uint8
	Level uint16
	Time  uint32
}

// EncodeSendAffect monta MSG_SendAffect (0x03B9): um snapshot completo dos 32
// slots de affect da entidade (slots vazios = zero). Envie com HEADER.ID = conn
func EncodeSendAffect(affects [MaxAffect]AffectData) []byte {
	b := make([]byte, sendAffectSize-HeaderSize) // 256
	for i, a := range affects {
		o := i * 8
		b[o] = a.Type
		b[o+1] = a.Value
		le.PutUint16(b[o+2:], a.Level)
		le.PutUint32(b[o+4:], a.Time)
	}
	return b
}
