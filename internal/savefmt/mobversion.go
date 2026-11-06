package savefmt

// MobVersion identifica o layout de um template STRUCT_MOB avulso pelo tamanho em
// bytes, do mesmo jeito que DetectVersion roteia arquivos de conta (version.go).
//
// tmserver/content/TMsrv/run/npc/ mistura três tamanhos: 1769 arquivos no
// canônico de 816, 207 em 756 e 15 em 920. A forma de 756 bytes é um STRUCT_MOB
// mais antigo LEGÍTIMO — Exp de 32 bits, sem Quest/Magic/GuildLevel e com
// STRUCT_SCORE compacta de 28 bytes — e não um registro de 816 truncado. A forma
// de 920 bytes é esse mesmo registro de 756 seguido de 164 bytes de lixo de heap
// não inicializado da ferramenta legada que o escreveu, então um decodificador
// serve aos dois.
type MobVersion int

// Versões conhecidas de template STRUCT_MOB.
const (
	MobVersionUnknown MobVersion = iota
	// MobVersionLegacy756 é o layout anterior à expansão (offsets L756).
	MobVersionLegacy756
	// MobVersionLegacy756Padded é MobVersionLegacy756 mais lixo no fim.
	MobVersionLegacy756Padded
	// MobVersionCurrent é o layout de 816 bytes modelado por decodeMob/encodeMob.
	MobVersionCurrent
)

// String rende a versão para log e relatório de inventário.
func (v MobVersion) String() string {
	switch v {
	case MobVersionLegacy756:
		return "legacy-756"
	case MobVersionLegacy756Padded:
		return "legacy-756+pad920"
	case MobVersionCurrent:
		return "current-816"
	default:
		return "unknown"
	}
}

// DetectMobVersion mapeia o tamanho de um arquivo de template para a versão do
// layout.
func DetectMobVersion(size int) MobVersion {
	switch size {
	case MobSize:
		return MobVersionCurrent
	case MobSizeLegacy756:
		return MobVersionLegacy756
	case MobSizeLegacy756Padded:
		return MobVersionLegacy756Padded
	default:
		return MobVersionUnknown
	}
}
