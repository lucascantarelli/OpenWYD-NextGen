package protocol

// motionSize é sizeof(MSG_Motion) (0x036A): HEADER(12) + ushort Motion +
// ushort Parm + int NotUsed = 20 bytes (captura-wyd-levelup.md §7). É uma
// transmissão genérica de efeito/animação de entidade; o brilho de level-up é
// Motion=14, Parm=3 (o cliente toca a animação/som ao receber).
const motionSize = 20

// EncodeMotion monta o corpo de MSG_Motion (0x036A). Envie com HEADER.ID = a
// entidade onde o efeito toca (o próprio jogador ou um mob em vista).
func EncodeMotion(motion, parm uint16) []byte {
	b := make([]byte, motionSize-HeaderSize) // 8
	le.PutUint16(b[0:], motion)              // Motion @body0
	le.PutUint16(b[2:], parm)                // Parm @body2
	// NotUsed @corpo4 (int) fica zero.
	return b
}
