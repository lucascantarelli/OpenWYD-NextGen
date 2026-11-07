package exptool

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"openwyd-nextgen/internal/savefmt"
	"openwyd-nextgen/tmserver/internal/content"
	"openwyd-nextgen/tmserver/internal/level"
	"openwyd-nextgen/tmserver/internal/protocol"
)

// template monta um STRUCT_MOB cru de 816 bytes preenchido com um padrão não
// zero, para a checagem de round-trip pegar qualquer byte que o stamp não pode tocar.
func template(name string, lvl uint32, exp uint64, merchant byte) []byte {
	b := make([]byte, content.BaseMobSize)
	for i := range b {
		b[i] = byte(i % 251)
	}
	for i := 0; i < 16; i++ {
		b[i] = 0
	}
	copy(b, name)
	binary.LittleEndian.PutUint64(b[32:], exp)
	const cs = 92
	binary.LittleEndian.PutUint32(b[cs+0:], lvl)
	b[cs+12] = merchant
	return b
}

func write(t *testing.T, dir, name string, data []byte) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, dir, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestStampRoundTrip(t *testing.T) {
	dir := t.TempDir()
	monster := template("Cobaia", 10, 68571, 0) // garbage authored exp
	merchant := template("Vendedor", 5, 25000, 100)
	levelZero := template("Base", 0, 123, 0)
	write(t, dir, "Cobaia", monster)
	write(t, dir, "Vendedor", merchant)
	write(t, dir, "Base", levelZero)
	write(t, dir, "Nota.txt", []byte("not a template"))

	res, err := Stamp(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Stamped) != 1 || res.Skipped != 2 || res.SkippedNonTemplate != 1 {
		t.Fatalf("stamped %d / skipped %d / non-template %d, want 1 / 2 / 1",
			len(res.Stamped), res.Skipped, res.SkippedNonTemplate)
	}
	e := res.Stamped[0]
	want := level.MobExpForLevel(10)
	if e.Name != "Cobaia" || e.OldExp != 68571 || e.NewExp != want {
		t.Fatalf("entry = %+v, want Cobaia 68571 -> %d", e, want)
	}

	// O arquivo do monstro mudou exatamente nos 8 bytes de Exp.
	got := read(t, dir, "Cobaia")
	if b := protocol.ParseMobBasics(got); b.Exp != want {
		t.Errorf("stamped Exp = %d, want %d", b.Exp, want)
	}
	patched := append([]byte(nil), monster...)
	binary.LittleEndian.PutUint64(patched[32:], uint64(want))
	if !bytes.Equal(got, patched) {
		t.Error("stamp touched bytes outside the Exp field")
	}

	// Non-monster files stay byte-identical.
	if !bytes.Equal(read(t, dir, "Vendedor"), merchant) {
		t.Error("merchant template was modified")
	}
	if !bytes.Equal(read(t, dir, "Base"), levelZero) {
		t.Error("level-0 template was modified")
	}
}

// legacyTemplate monta um STRUCT_MOB legado cru de 756 bytes (data-formats.md
// §1.4.1) com o mesmo truque de padrão não zero do template: Exp é um campo
// de 32 bits no offset 28 e o score compacto fica em 64 (Level u16 @+0,
// Merchant @+6). size seleciona o layout plano de 756 ou o de 920 com padding,
// cujo lixo final precisa sobreviver ao stamp intocado.
func legacyTemplate(name string, lvl uint16, exp int32, merchant byte, size int) []byte {
	b := make([]byte, size)
	for i := range b {
		b[i] = byte(i % 251)
	}
	for i := 0; i < 16; i++ {
		b[i] = 0
	}
	copy(b, name)
	binary.LittleEndian.PutUint32(b[28:], uint32(exp))
	const cs = 64
	binary.LittleEndian.PutUint16(b[cs+0:], lvl)
	b[cs+6] = merchant
	return b
}

// TestStampCoversLegacyTemplates é a correção do resíduo que a issue #244 deixou
// para trás: os layouts legados eram pulados, então 222 monstros distribuídos mantinham
// seu Exp autorado enquanto o log de boot do tmServer seguia dizendo aos operadores
// para rodar esta ferramenta. Eles são restampados no lugar agora — 4 bytes no
// offset 28 — e o arquivo mantém seu tamanho, que era o motivo original de pulá-los.
func TestStampCoversLegacyTemplates(t *testing.T) {
	dir := t.TempDir()
	legacy := legacyTemplate("Antigo", 20, 999, 0, savefmt.MobSizeLegacy756)
	padded := legacyTemplate("Preenchido", 30, 0, 0, savefmt.MobSizeLegacy756Padded)
	shop := legacyTemplate("Loja", 2, 1, 1, savefmt.MobSizeLegacy756)
	monster := template("Cobaia", 10, 0, 0)
	write(t, dir, "Antigo", legacy)
	write(t, dir, "Preenchido", padded)
	write(t, dir, "Loja", shop)
	write(t, dir, "Cobaia", monster)

	res, err := Stamp(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Stamped) != 3 || res.StampedVariant != 2 || res.Skipped != 1 || res.SkippedNonTemplate != 0 {
		t.Fatalf("stamped %d (variant %d) / skipped %d / non-template %d, want 3 (2) / 1 / 0",
			len(res.Stamped), res.StampedVariant, res.Skipped, res.SkippedNonTemplate)
	}

	// Cada arquivo legado mudou exatamente nos 4 bytes de Exp — o lixo final do
	// padded form included.
	for _, tc := range []struct {
		file   string
		before []byte
		lvl    int32
	}{
		{"Antigo", legacy, 20},
		{"Preenchido", padded, 30},
	} {
		got := read(t, dir, tc.file)
		if len(got) != len(tc.before) {
			t.Errorf("%s: size = %d, want %d", tc.file, len(got), len(tc.before))
			continue
		}
		patched := append([]byte(nil), tc.before...)
		binary.LittleEndian.PutUint32(patched[28:], uint32(level.MobExpForLevel(tc.lvl)))
		if !bytes.Equal(got, patched) {
			t.Errorf("%s: stamp touched bytes outside the legacy Exp field", tc.file)
		}
	}

	// Um mercador legado continua mercador: sua flag Merchant vive no
	// score compacto, então lê-la pelos offsets canônicos o teria
	// estampado por engano.
	if !bytes.Equal(read(t, dir, "Loja"), shop) {
		t.Error("legacy merchant template was modified")
	}
}

func TestStampDryRun(t *testing.T) {
	dir := t.TempDir()
	monster := template("Cobaia", 10, 0, 0)
	write(t, dir, "Cobaia", monster)

	res, err := Stamp(dir, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Stamped) != 1 || res.Stamped[0].NewExp != level.MobExpForLevel(10) {
		t.Fatalf("dry-run report = %+v, want the would-be stamp", res.Stamped)
	}
	if !bytes.Equal(read(t, dir, "Cobaia"), monster) {
		t.Error("dry-run wrote to disk")
	}
}
