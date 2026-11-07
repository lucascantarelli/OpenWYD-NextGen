package world

// IsWaterGenerator identifica os blocos NPCGener do evento a partir de Basedef.h.
// Eles nunca podem povoar na inicialização nem entrar em nenhum caminho automático de respawn.
func IsWaterGenerator(index int) bool {
	return index >= 10 && index <= 21 || index >= 171 && index <= 194
}
