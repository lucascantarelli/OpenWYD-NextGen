package main

import (
	"strings"
	"testing"
)

func TestParseList(t *testing.T) {
	entries, err := ParseList(strings.NewReader("# canais\n0 0 192.168.0.10:8281\n\n0 1 192.168.0.10:8282\n"))
	if err != nil {
		t.Fatalf("ParseList: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("entradas = %d, quer 2", len(entries))
	}
	if entries[1].Channel != 1 || entries[1].Addr != "192.168.0.10:8282" {
		t.Errorf("segunda entrada = %+v", entries[1])
	}
}

func TestParseListRejectsBadInput(t *testing.T) {
	cases := map[string]string{
		"duas colunas":   "0 0",
		"grupo grande":   "10 0 1.2.3.4",
		"canal grande":   "0 11 1.2.3.4",
		"grupo inválido": "x 0 1.2.3.4",
	}
	for name, text := range cases {
		if _, err := ParseList(strings.NewReader(text)); err == nil {
			t.Errorf("%s: esperava erro", name)
		}
	}
}

func TestBuildRoundTrip(t *testing.T) {
	entries := []Entry{
		{Group: 0, Channel: 0, Addr: "192.168.0.10:8281"},
		{Group: 0, Channel: 1, Addr: "192.168.0.10:8282"},
	}
	bin, err := Build(entries)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(bin) != BinSize {
		t.Fatalf("tamanho = %d, quer %d", len(bin), BinSize)
	}
	grid, err := Decode(bin)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if grid[0][0] != "192.168.0.10:8281" || grid[0][1] != "192.168.0.10:8282" {
		t.Errorf("round-trip = %q / %q", grid[0][0], grid[0][1])
	}
	if grid[3][5] != "" {
		t.Errorf("célula vazia decodificou %q", grid[3][5])
	}
	// The obfuscation must actually change the bytes, or the client cannot read it.
	if bin[0] == '1' {
		t.Error("primeiro byte não foi ofuscado")
	}
}

func TestBuildRejectsEmpty(t *testing.T) {
	if _, err := Build(nil); err == nil {
		t.Fatal("esperava erro para lista vazia")
	}
}
