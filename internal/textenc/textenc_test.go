package textenc

import (
	"testing"
	"unicode/utf8"
)

func TestToUTF8CP1252(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"ascii intacto", "Garra", "Garra"},
		{"a-til", "Garr\xe3o", "Garrão"},
		{"cedilha", "Espada M\xe1gica \xe7\xe3", "Espada Mágica çã"},
		{"aspas tipograficas", "Anel \x93Lua\x94", "Anel “Lua”"},
		{"travessao", "N\xedvel \x96 alto", "Nível – alto"},
		{"já utf8 não dobra", "Garrão", "Garrão"},
		{"vazio", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ToUTF8(tc.in)
			if got != tc.want {
				t.Fatalf("ToUTF8(%q) = %q, want %q", tc.in, got, tc.want)
			}
			if !utf8.ValidString(got) {
				t.Fatalf("ToUTF8(%q) não é UTF-8 válido", tc.in)
			}
		})
	}
}

func TestSlice(t *testing.T) {
	got := Slice([]string{"831", "Garr\xe3o", "530"})
	if len(got) != 3 || got[1] != "Garrão" || got[0] != "831" {
		t.Fatalf("Slice = %q", got)
	}
}
