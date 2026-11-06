package main

import (
	"encoding/binary"
	"testing"
)

// snFixture mirrors the head of the real client sn.bin: 11 name slots of 9 bytes
// (only the first four used) followed by the 11-int32 counter table.
func snFixture() []byte {
	b := make([]byte, countsOffset+countsCount*4)
	copy(b[0:], "\xd6eserved")
	copy(b[9:], "UP")
	copy(b[18:], "PVP")
	copy(b[27:], "WYDCDK")
	// The shipped file carries 2, 3, 4.
	binary.LittleEndian.PutUint32(b[countsOffset+0:], 2)
	binary.LittleEndian.PutUint32(b[countsOffset+4:], 3)
	binary.LittleEndian.PutUint32(b[countsOffset+8:], 4)
	return b
}

func TestParseSlots(t *testing.T) {
	names := parseSlots(snFixture())
	want := []string{"\xd6eserved", "UP", "PVP", "WYDCDK"}
	if len(names) != len(want) {
		t.Fatalf("posições = %d (%q), quer %d", len(names), names, len(want))
	}
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("posição %d = %q, quer %q", i, names[i], want[i])
		}
	}
}

func TestSetNameKeepsTail(t *testing.T) {
	b := snFixture()
	if err := setName(b, 1, "NextGen"); err != nil {
		t.Fatalf("setName: %v", err)
	}
	if names := parseSlots(b); names[1] != "NextGen" || names[2] != "PVP" {
		t.Errorf("nomes = %q", names)
	}
	table, err := channelCounts(b)
	if err != nil {
		t.Fatalf("channelCounts: %v", err)
	}
	if table[0] != 2 || table[1] != 3 || table[2] != 4 {
		t.Errorf("tabela de contadores foi tocada: %v", table)
	}
	if b[9+7] != 0 || b[9+8] != 0 {
		t.Errorf("slot não foi preenchido com NUL: %v", b[9:18])
	}
}

// TestSetNameClearsSlot: nome vazio apaga a posição (usado para tirar de cena
// servidores que não existem mais, como PVP e WYDCDK), sem tocar na cauda.
func TestSetNameClearsSlot(t *testing.T) {
	b := snFixture()
	if err := setName(b, 2, ""); err != nil {
		t.Fatalf("setName vazio: %v", err)
	}
	for i := 18; i < 27; i++ {
		if b[i] != 0 {
			t.Fatalf("byte %d do slot 2 = %#x, quer 0", i, b[i])
		}
	}
	if names := parseSlots(b); len(names) != 2 || names[1] != "UP" {
		t.Errorf("nomes após limpar o 2 = %q, quer só os dois primeiros", names)
	}
	table, err := channelCounts(b)
	if err != nil {
		t.Fatalf("channelCounts: %v", err)
	}
	if table[0] != 2 || table[1] != 3 || table[2] != 4 {
		t.Errorf("contadores mudaram: %v", table)
	}
}

// TestSetNameSlot0KeepsSignature: a posição 0 guarda a assinatura do build no
// primeiro byte, no lugar do primeiro caractere do nome ("Global" → "\xablobal").
func TestSetNameSlot0KeepsSignature(t *testing.T) {
	b := snFixture()
	if err := setName(b, 0, "NextGen"); err != nil {
		t.Fatalf("setName slot 0: %v", err)
	}
	if b[0] != 0xd6 {
		t.Fatalf("assinatura = %#x, quer 0xd6 (não pode ser sobrescrita)", b[0])
	}
	if got := string(b[1:7]); got != "extGen" {
		t.Errorf("nome no slot 0 = %q, quer \"extGen\" (o primeiro caractere é a assinatura)", got)
	}
	for i := 7; i < 9; i++ {
		if b[i] != 0 {
			t.Errorf("byte %d do slot 0 = %#x, quer NUL", i, b[i])
		}
	}
	if err := setName(b, 0, ""); err == nil {
		t.Error("nome vazio na posição 0 deveria falhar")
	}
	table, err := channelCounts(b)
	if err != nil {
		t.Fatalf("channelCounts: %v", err)
	}
	if table[0] != 2 || table[1] != 3 {
		t.Errorf("contadores mudaram: %v", table)
	}
}

func TestSetNameLimits(t *testing.T) {
	b := snFixture()
	if err := setName(b, 1, "NomeLongo1"); err == nil {
		t.Error("nome de 9 caracteres deveria falhar (limite 8)")
	}
	if err := setName(b, 11, "X"); err == nil {
		t.Error("índice além da tabela de nomes deveria falhar")
	}
}

func TestParseSet(t *testing.T) {
	if idx, name, err := parseSet("NextGen"); err != nil || idx != 1 || name != "NextGen" {
		t.Errorf("parseSet(NextGen) = %d, %q, %v", idx, name, err)
	}
	if idx, name, err := parseSet("2=Canal 2"); err != nil || idx != 2 || name != "Canal 2" {
		t.Errorf("parseSet(2=Canal 2) = %d, %q, %v", idx, name, err)
	}
}

func TestParseCounts(t *testing.T) {
	got, err := parseCounts("2,3")
	if err != nil {
		t.Fatalf("parseCounts: %v", err)
	}
	if len(got) != 2 || got[0] != 2 || got[1] != 3 {
		t.Errorf("parseCounts(2,3) = %v", got)
	}
	if _, err := parseCounts("-1"); err == nil {
		t.Error("contador negativo deveria falhar")
	}
	if _, err := parseCounts("1,2,3,4,5,6,7,8,9,10,11,12"); err == nil {
		t.Error("mais de 11 contadores deveria falhar")
	}
}

// TestSetCountsReproducesDisplayedRows: os valores de fábrica 2,3,4 são os que
// fazem a lista aparecer, então gravar 2,3 tem de preservar o índice 0.
func TestSetCountsReproducesDisplayedRows(t *testing.T) {
	b := snFixture()
	if err := setCounts(b, []int32{2, 3}); err != nil {
		t.Fatalf("setCounts: %v", err)
	}
	table, err := channelCounts(b)
	if err != nil {
		t.Fatalf("channelCounts: %v", err)
	}
	want := []int32{2, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	for i := range want {
		if table[i] != want[i] {
			t.Errorf("contador[%d] = %d, quer %d", i, table[i], want[i])
		}
	}
	if names := parseSlots(b); names[1] != "UP" {
		t.Errorf("nome 1 = %q, quer UP", names[1])
	}
}

// TestSetCountsRejectsZeroAtZero is the regression for the run that emptied the
// server list: o índice 0 em zero faz o cliente não listar nada.
func TestSetCountsRejectsZeroAtZero(t *testing.T) {
	b := snFixture()
	if err := setCounts(b, []int32{0, 3}); err == nil {
		t.Error("contador 0 = 0 deveria ser recusado")
	}
}
