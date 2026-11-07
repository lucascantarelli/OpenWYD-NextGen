package protocol

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// transportVector espelha o esquema de fixture de parity-tests.md §7a. wire_hex é
// capturado byte a byte do servidor AO VIVO, então estes vetores provam paridade
// que um round-trip não prova. plain_hex/wire_hex incluem o cabeçalho de 12 bytes;
// ignored.
type transportVector struct {
	Kind     string `json:"kind"`
	IKeyWord uint8  `json:"iKeyWord"`
	PlainHex string `json:"plain_hex"`
	WireHex  string `json:"wire_hex"`
	Checksum uint8  `json:"checksum"`
	Note     string `json:"note"`
}

func unhex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(strings.ReplaceAll(s, " ", ""))
	if err != nil {
		t.Fatalf("bad hex %q: %v", s, err)
	}
	return b
}

// TestTransportVectors valida o codec contra frames reais capturados
// (parity-tests.md §3/§7a). Até existir uma captura, a única fixture versionada é
// um exemplo de esquema com hex vazio, que o teste pula. Coloque capturas reais em
// test/fixtures/transport/ para torná-lo assertivo — nenhuma mudança de código é necessária.
func TestTransportVectors(t *testing.T) {
	dir := filepath.Join("..", "..", "test", "fixtures", "transport")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("motivo: conteúdo ausente — tmserver/test/fixtures/transport/ não existe: %v", err)
	}
	ran := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		t.Run(e.Name(), func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				t.Fatal(err)
			}
			var v transportVector
			if err := json.Unmarshal(raw, &v); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if v.WireHex == "" || v.PlainHex == "" {
				t.Skip("motivo: conteúdo ausente — exemplo só de schema (sem hex); precisa de captura real para virar vetor")
			}
			plain := unhex(t, v.PlainHex)
			wire := unhex(t, v.WireHex)

			// encode(plain, iKeyWord) == wire
			h, err := DecodeHeader(plain)
			if err != nil {
				t.Fatal(err)
			}
			gotWire, err := Encode(h, plain[HeaderSize:], v.IKeyWord)
			if err != nil {
				t.Fatal(err)
			}
			if string(gotWire) != string(wire) {
				t.Errorf("encode mismatch:\n got % x\nwant % x", gotWire, wire)
			}
			if gotWire[3] != v.Checksum {
				t.Errorf("checksum = %d, want %d", gotWire[3], v.Checksum)
			}

			// decode(wire) == plain
			gotH, gotBody, _, err := Decode(wire)
			if err != nil {
				t.Fatal(err)
			}
			gotPlain := make([]byte, HeaderSize+len(gotBody))
			_ = EncodeHeader(gotPlain, gotH)
			copy(gotPlain[HeaderSize:], gotBody)
			// Os bytes de checksum/keyword do cabeçalho diferem por construção; compare a partir
			// do campo Type em diante, mais o corpo.
			if string(gotPlain[4:]) != string(plain[4:]) {
				t.Errorf("decode mismatch:\n got % x\nwant % x", gotPlain[4:], plain[4:])
			}
			ran++
		})
	}
	if ran == 0 {
		t.Log("no real transport vectors yet — only schema example present (expected until capture)")
	}
}
