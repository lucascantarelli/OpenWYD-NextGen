// Package textenc converts the legacy text encoding of the original Release/
// files to UTF-8.
//
// The shipped CSVs (ItemList.csv, SkillData.csv, NPCGener.txt, Guard.txt…) are
// Windows-1252 — the same encoding the win32 server wrote. PostgreSQL rejects
// invalid UTF-8 sequences outright ("invalid byte sequence for encoding UTF8:
// 0xe3"), so anything that leaves Go's byte-string world for the database has to
// be converted first. Go strings hold arbitrary bytes, which is why nothing
// noticed until the first insert.
package textenc

import (
	"strings"
	"unicode/utf8"
)

// cp1252 maps 0x80-0x9F. Those bytes are *not* Latin-1 control characters in
// Windows-1252: they are the printable punctuation the legacy data actually
// uses (’ “ ” – — •), so a plain byte→rune widening would put control chars in
// item names.
var cp1252 = [32]rune{
	'€', 0x81, '‚', 'ƒ', '„', '…', '†', '‡',
	'ˆ', '‰', 'Š', '‹', 'Œ', 0x8D, 'Ž', 0x8F,
	0x90, '‘', '’', '“', '”', '•', '–', '—',
	'˜', '™', 'š', '›', 'œ', 0x9D, 'ž', 'Ÿ',
}

// ToUTF8 returns s as valid UTF-8. A string that already is valid UTF-8 is
// returned unchanged (so re-importing a file that was fixed by hand does not
// double-encode it); anything else is decoded as Windows-1252.
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

// Slice applies ToUTF8 to every element, for a CSV row stored as JSON.
func Slice(fields []string) []string {
	out := make([]string, len(fields))
	for i, f := range fields {
		out[i] = ToUTF8(f)
	}
	return out
}
