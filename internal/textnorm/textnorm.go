// Package textnorm normaliza texto que entra no Postgres.
//
// A conversão de codificação (Windows-1252 → UTF-8) é do pacote `textenc`. Este
// pacote trata o que sobra depois dela e que `utf8.ValidString` não pega:
//
//   - **Forma Unicode**: o mesmo "é" pode ser um code point (NFC) ou dois (NFD).
//     Igualdade, UNIQUE, busca e `ORDER BY` tratam os dois como diferentes. O
//     banco inteiro usa NFC (o padrão SQL e o que quase todo sistema assume).
//   - **Espaço**: `" Espada "` e `"Espada  Longa"` são invisíveis na tela, mas
//     quebram busca por prefixo e deduplicação.
//   - **Controle**: C0/C1 vindos de CSV truncado ou dump. Não são dados.
//   - **Duplo encoding**: bytes UTF-8 lidos como CP1252 ("Cajado de AmÃ©m" em vez
//     de "Amém"). Reparar é possível enquanto os bytes originais existirem;
//     depois que o texto virou U+FFFD, não há o que recuperar.
//
// O que este pacote NÃO faz: adivinhar. Não aplica NFKC (perde distinção — é
// lossy), não troca maiúsculas/minúsculas, não remove acento. Dado de jogo tem
// que continuar idêntico ao que o cliente desenha.
package textnorm

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// Clean é o pipeline de entrada. A ordem importa:
//
//  1. controle sai primeiro, porque um NUL no meio invalida a leitura de bytes
//     do reparo;
//  2. reparo de duplo encoding;
//  3. colapso de espaço (resolve NBSP e espaço nas pontas);
//  4. NFC por último, sobre o texto já enxuto.
//
// É idempotente — rodar duas vezes dá o mesmo resultado.
func Clean(s string) string {
	return NFC(CollapseSpace(RepairMojibake(StripControl(s))))
}

// NFC devolve s na forma composta canônica (UAX #15).
func NFC(s string) string {
	if norm.NFC.IsNormalString(s) {
		return s
	}
	return norm.NFC.String(s)
}

// Slice aplica Clean a cada elemento.
func Slice(fields []string) []string {
	out := make([]string, len(fields))
	for i, field := range fields {
		out[i] = Clean(field)
	}
	return out
}

// RepairMojibake desfaz o duplo encoding UTF-8/CP1252 quando ele é inequívoco.
//
// Critério: a string precisa conter um marcador de duplo encoding ("Ã", "Â" ou
// "â"), e os bytes reinterpretados precisam formar UTF-8 válido. O português
// legítimo passa intacto porque seus bytes não formam UTF-8: "Âmbar" tem "Â"
// seguido de "m" (0xC2 0x6D, inválido) e "âmbar" tem "â" seguido de "m"
// (0xE2 0x6D, inválido).
//
// O reparo é tentado no texto inteiro e, se o texto misturar byte não-CP1252
// (emoji, CJK) ou espaço exótico, palavra por palavra — foi assim que um NBSP no
// fim da string impedia o reparo de "AmÃ©m" no meio dela.
func RepairMojibake(s string) string {
	if !hasMojibakeMarker(s) {
		return s
	}
	if fixed, ok := repairOnce(s); ok {
		return fixed
	}

	fields := strings.FieldsFunc(s, unicode.IsSpace)
	var b strings.Builder
	b.Grow(len(s))
	for i, field := range fields {
		if i > 0 {
			b.WriteByte(' ')
		}
		if fixed, ok := repairOnce(field); ok {
			b.WriteString(fixed)
			continue
		}
		b.WriteString(field)
	}
	return b.String()
}

func hasMojibakeMarker(s string) bool {
	return strings.Contains(s, "Ã") || strings.Contains(s, "Â") || strings.Contains(s, "â")
}

// repairOnce devolve o texto reinterpretado e true só quando o resultado muda.
func repairOnce(s string) (string, bool) {
	bytes, ok := cp1252Bytes(s)
	if !ok || !utf8.Valid(bytes) {
		return s, false
	}
	fixed := string(bytes)
	if fixed == s {
		return s, false
	}
	// Verificação de segurança: o reparo precisa reduzir marcadores. Sem isso,
	// um texto que por acaso formasse UTF-8 válido poderia ser trocado à toa.
	if markerCount(fixed) > markerCount(s) {
		return s, false
	}
	return fixed, true
}

func markerCount(s string) int {
	return strings.Count(s, "Ã") + strings.Count(s, "Â")
}

// cp1252High mapeia o rune de volta para o byte 0x80-0x9F do Windows-1252.
// O duplo encoding nasce de leitura CP1252 (não Latin-1 puro): o byte 0x80 vira
// "€", 0x93 vira "“", 0x94 vira "”". Sem esta tabela, um travessão ou um euro
// mojibake nunca seriam reconhecidos.
var cp1252High = map[rune]byte{
	'€': 0x80, '‚': 0x82, 'ƒ': 0x83, '„': 0x84, '…': 0x85, '†': 0x86, '‡': 0x87,
	'ˆ': 0x88, '‰': 0x89, 'Š': 0x8A, '‹': 0x8B, 'Œ': 0x8C, 'Ž': 0x8E,
	'‘': 0x91, '’': 0x92, '“': 0x93, '”': 0x94, '•': 0x95, '–': 0x96, '—': 0x97,
	'˜': 0x98, '™': 0x99, 'š': 0x9A, '›': 0x9B, 'œ': 0x9C, 'ž': 0x9E, 'Ÿ': 0x9F,
	// Os cinco bytes indefinidos do CP1252 não têm rune de volta.
}

// cp1252Bytes reinterpreta cada rune como um byte Windows-1252. Falso quando
// algum rune não existe nessa codificação — sinal de que a string não veio
// desse caminho (ou já foi reparada).
func cp1252Bytes(s string) ([]byte, bool) {
	out := make([]byte, 0, len(s))
	for _, r := range s {
		switch {
		case r <= 0xFF:
			out = append(out, byte(r))
		default:
			b, ok := cp1252High[r]
			if !ok {
				return nil, false
			}
			out = append(out, b)
		}
	}
	return out, true
}

// StripControl remove caracteres de controle (C0/C1), exceto tabulação, que
// vira espaço antes do colapso. U+FFFD é mantido de propósito: apagar perderia
// a evidência de que houve dano de codificação.
func StripControl(s string) string {
	if !strings.ContainsFunc(s, isControl) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r == '\t' || r == '\n' || r == '\r':
			b.WriteByte(' ')
		case isControl(r):
			// descartado
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func isControl(r rune) bool {
	return r < 0x20 || (r >= 0x7F && r <= 0x9F) || unicode.Is(unicode.Cc, r)
}

// CollapseSpace troca qualquer sequência de espaço Unicode (inclusive NBSP,
// tab e quebra de linha) por um espaço simples e remove as pontas.
func CollapseSpace(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	pendingSpace := false

	for _, r := range s {
		if unicode.IsSpace(r) {
			pendingSpace = b.Len() > 0
			continue
		}
		if pendingSpace {
			b.WriteByte(' ')
			pendingSpace = false
		}
		b.WriteRune(r)
	}
	return b.String()
}
