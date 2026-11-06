// Package textenc converte a codificação de texto legada dos arquivos originais
// de Release/ para UTF-8.
//
// Os CSVs distribuídos (ItemList.csv, SkillData.csv, NPCGener.txt, Guard.txt…) estão
// em Windows-1252, a mesma codificação que o servidor win32 escrevia. O PostgreSQL
// recusa sequência UTF-8 inválida de imediato ("invalid byte sequence for encoding
// UTF8: 0xe3"), então tudo que sai da string de bytes do Go para o banco precisa ser
// convertido antes. Strings do Go aceitam qualquer byte, e por isso ninguém notou
// até o primeiro insert.
package textenc

import (
	"strings"
	"unicode/utf8"
)

// cp1252 mapeia 0x80-0x9F. Esses bytes *não* são caracteres de controle Latin-1 em
// Windows-1252: são a pontuação imprimível que os dados legados realmente usam
// (’ “ ” – — •), então um widening byte→rune colocaria caracteres de controle em
// nome de item.
var cp1252 = [32]rune{
	'€', 0x81, '‚', 'ƒ', '„', '…', '†', '‡',
	'ˆ', '‰', 'Š', '‹', 'Œ', 0x8D, 'Ž', 0x8F,
	0x90, '‘', '’', '“', '”', '•', '–', '—',
	'˜', '™', 'š', '›', 'œ', 0x9D, 'ž', 'Ÿ',
}

// ToUTF8 devolve s como UTF-8 válido. Uma string que já é UTF-8 válido volta
// intacta (para reimportar um arquivo corrigido à mão não codificar duas vezes);
// qualquer outra coisa é decodificada como Windows-1252.
func ToUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + len(s)/4)
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c < 0x80:
			b.WriteByte(c)
		case c <= 0x9F:
			b.WriteRune(cp1252[c-0x80])
		default:
			b.WriteRune(rune(c))
		}
	}
	return b.String()
}

// Slice aplica ToUTF8 a cada elemento, para uma linha de CSV guardada em JSON.
func Slice(fields []string) []string {
	out := make([]string, len(fields))
	for i, f := range fields {
		out[i] = ToUTF8(f)
	}
	return out
}
