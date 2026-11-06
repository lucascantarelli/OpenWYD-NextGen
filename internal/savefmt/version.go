package savefmt

import "strings"

// Version identifica o layout de um arquivo de conta pelo tamanho em bytes. O
// DBSrv original aceita (fsize>=7500 && fsize<=7600) ||
// fsize==sizeof(STRUCT_ACCOUNTFILE); a forma de 4294 bytes é um layout ainda mais
// antigo, anterior à expansão, visto em account/A/antonio.
type Version int

// Versões conhecidas de arquivo de conta.
const (
	VersionUnknown      Version = iota
	VersionLegacy4294           // 4294 B — anterior à expansão; layout não verificado
	VersionIntermediate         // 7500–7600 B — layout não verificado
	VersionCurrent              // 7952 B — modelado por inteiro (Decode/Encode)
)

// String rende a versão para log.
func (v Version) String() string {
	switch v {
	case VersionLegacy4294:
		return "legacy-4294"
	case VersionIntermediate:
		return "intermediate-7500..7600"
	case VersionCurrent:
		return "current-7952"
	default:
		return "unknown"
	}
}

// Modelled informa se esta stack decodifica a versão por inteiro. Só o formato
// atual de 7952 bytes é modelado; os outros são detectados, mas seu layout de
// campo não está verificado e precisa ser revertido de um build/captura de
// referência.
func (v Version) Modelled() bool { return v == VersionCurrent }

// DetectVersion mapeia um tamanho de arquivo para a versão do layout.
func DetectVersion(size int) Version {
	switch {
	case size == AccountFileSize:
		return VersionCurrent
	case size >= 7500 && size <= 7600:
		return VersionIntermediate
	case size == 4294:
		return VersionLegacy4294
	default:
		return VersionUnknown
	}
}

// AccountNameAt0 lê o campo AccountName no offset 0 — o único campo estável em
// todas as versões conhecidas (STRUCT_ACCOUNTINFO abre o arquivo em todo layout).
// Remove espaços e NULs à direita (arquivos legados preenchem com espaço; ver a
// amostra antonio). Devolve "" se b for curto demais.
func AccountNameAt0(b []byte) string {
	if len(b) < 16 {
		return ""
	}
	return strings.TrimRight(string(b[0:16]), " \x00")
}
