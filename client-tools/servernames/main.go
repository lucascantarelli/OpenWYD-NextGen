// Command servernames lê e grava os nomes e os contadores do sn.bin do cliente.
//
// O WYD.exe carrega "sn.bin" e monta o rótulo de cada canal como "%s-%d" (nome
// do servidor + número do canal), então renomear um servidor renomeia os canais
// junto: "UP" vira "UP-1"/"UP-2" na tela.
//
// Layout medido no arquivo real (143 bytes), confirmado no disassembly do
// WYD.exe (0x54b688, que lê 0x63 bytes de nomes e depois 0x2c bytes de
// contadores):
//
//	bytes 0..98    11 posições de nome de 9 bytes (8 caracteres úteis + NUL):
//	               "Reserved" (placeholder), depois um nome por grupo da serverlist
//	bytes 99..142  11 int32 little-endian: contador de cada índice
//
// O que o contador significa foi medido, não deduzido: com os valores de fábrica
// `2,3,4` a lista mostra três linhas; zerando o índice 0 e deixando `1,1` o
// cliente não mostra NADA. Ou seja, o índice 0 precisa ficar diferente de zero
// para o arquivo valer, e a quantidade de linhas acompanha o valor do índice do
// grupo. Trate os contadores como "quantos canais este grupo tem" e mantenha o
// índice 0 como está de fábrica.
//
// Uso:
//
//	go run ./client-tools/servernames -print client/sn.bin
//	go run ./client-tools/servernames -in client/sn.bin -set 1=NextGen -counts 2,3
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"strings"
)

const (
	// slotSize is the fixed width of one name: 8 characters plus the NUL.
	slotSize = 9
	// nameBytes is how many characters actually fit in a slot.
	nameBytes = slotSize - 1
	// countsOffset is where the counter table starts: right after the 11 name
	// slots (11 × 9 = 99 bytes).
	countsOffset = 11 * slotSize
	// countsCount is how many counters the table holds (MAX_SERVERNUMBER).
	countsCount = 11
)

// parseSlots splits the name table into its 9-byte slots, stopping at the first
// slot that is entirely NUL (empty position). The counter table that follows
// never parses as text, so it is left out.
func parseSlots(b []byte) []string {
	var names []string
	for off := 0; off+slotSize <= len(b) && off < countsOffset; off += slotSize {
		slot := b[off : off+slotSize]
		empty := true
		for _, c := range slot {
			if c != 0 {
				empty = false
				break
			}
		}
		if empty {
			break
		}
		s := string(slot)
		if i := strings.IndexByte(s, 0); i >= 0 {
			s = s[:i]
		}
		names = append(names, s)
	}
	return names
}

// setName writes name into slot idx, NUL-padding the rest of the slot. It never
// resizes or reorders the file: the counter table stays untouched.
//
// A posição 0 é especial: o primeiro byte é a assinatura do build (0xD6 no nosso
// cliente, 0xAB no build do WYD Global) e ocupa o lugar do primeiro caractere do
// nome — é por isso que o original diz "\xd6eserved" para "Reserved" e o do WYD
// Global diz "\xablobal" para "Global". Aqui a assinatura nunca é sobrescrita e
// o nome entra a partir do byte 1, perdendo o primeiro caractere, exatamente como
// nos arquivos de fábrica.
func setName(b []byte, idx int, name string) error {
	if idx < 0 {
		return fmt.Errorf("índice inválido: %d", idx)
	}
	if idx == 0 {
		if len(b) < slotSize {
			return fmt.Errorf("arquivo curto (%d bytes) para a posição 0", len(b))
		}
		if len(name) == 0 {
			return fmt.Errorf("o nome da posição 0 não pode ser vazio: a assinatura ficaria sozinha")
		}
		slot := b[:slotSize]
		for i := 1; i < slotSize; i++ {
			slot[i] = 0
		}
		copy(slot[1:], name[1:])
		return nil
	}
	off := idx * slotSize
	if off+slotSize > len(b) || off >= countsOffset {
		return fmt.Errorf("índice %d fora da tabela de nomes (%d bytes)", idx, len(b))
	}
	if len(name) > nameBytes {
		return fmt.Errorf("nome %q tem %d caracteres; o limite é %d (o canal vira \"%s-1\")",
			name, len(name), nameBytes, name)
	}
	if strings.IndexByte(name, 0) >= 0 {
		return fmt.Errorf("nome com byte NUL")
	}
	slot := b[off : off+slotSize]
	for i := range slot {
		slot[i] = 0
	}
	copy(slot, name)
	return nil
}

// parseSet reads one "índice=NOME" or plain "NOME" argument. Without an index it
// targets slot 1, the first real server name (slot 0 is "Reserved").
func parseSet(arg string) (int, string, error) {
	idx := 1
	name := arg
	if i := strings.IndexByte(arg, '='); i > 0 {
		n := 0
		if _, err := fmt.Sscanf(arg[:i], "%d", &n); err != nil {
			return 0, "", fmt.Errorf("índice inválido em %q", arg)
		}
		idx = n
		name = arg[i+1:]
	}
	return idx, name, nil
}

// parseCounts reads raw counter values, e.g. "2,3": index 0 = 2, index 1 = 3 and
// every index past the list becomes 0.
func parseCounts(arg string) ([]int32, error) {
	var out []int32
	for _, part := range strings.Split(arg, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		var v int32
		if _, err := fmt.Sscanf(part, "%d", &v); err != nil {
			return nil, fmt.Errorf("contador inválido em %q", arg)
		}
		if v < 0 {
			return nil, fmt.Errorf("contador negativo em %q", arg)
		}
		out = append(out, v)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("lista de contadores vazia")
	}
	if len(out) > countsCount {
		return nil, fmt.Errorf("%d contadores; a tabela tem %d", len(out), countsCount)
	}
	return out, nil
}

// channelCounts reads the counter table.
func channelCounts(b []byte) ([]int32, error) {
	if len(b) < countsOffset+countsCount*4 {
		return nil, fmt.Errorf("arquivo curto (%d bytes) para a tabela de contadores", len(b))
	}
	out := make([]int32, countsCount)
	for i := range out {
		out[i] = int32(binary.LittleEndian.Uint32(b[countsOffset+i*4:]))
	}
	return out, nil
}

// setCounts writes values starting at index 0 and zeroes the remaining indexes.
// Index 0 must stay non-zero: zerar o índice 0 faz o cliente não listar nada.
func setCounts(b []byte, values []int32) error {
	if len(b) < countsOffset+countsCount*4 {
		return fmt.Errorf("arquivo curto (%d bytes) para a tabela de contadores", len(b))
	}
	if len(values) > countsCount {
		return fmt.Errorf("%d contadores; a tabela tem %d", len(values), countsCount)
	}
	for i := 0; i < countsCount; i++ {
		var v int32
		if i < len(values) {
			v = values[i]
		}
		binary.LittleEndian.PutUint32(b[countsOffset+i*4:], uint32(v))
	}
	if values[0] == 0 {
		return fmt.Errorf("contador 0 = 0: o cliente deixa de listar os servidores")
	}
	return nil
}

func main() {
	in := flag.String("in", "client/sn.bin", "sn.bin de entrada")
	out := flag.String("out", "", "sn.bin de saída (vazio: sobrescreve -in)")
	printOnly := flag.Bool("print", false, "só lista nomes e contadores e sai")
	set := flag.String("set", "", "\"índice=nome\" (ou só \"nome\" para o índice 1)")
	counts := flag.String("counts", "", "valores crus dos contadores, ex.: \"2,3\"")
	flag.Parse()

	b, err := os.ReadFile(*in)
	if err != nil {
		fmt.Fprintf(os.Stderr, "erro: %v\n", err)
		os.Exit(1)
	}

	if *printOnly || (*set == "" && *counts == "") {
		names := parseSlots(b)
		fmt.Printf("%s: %d bytes, %d nomes\n", *in, len(b), len(names))
		for i, n := range names {
			fmt.Printf("  nome %d %q\n", i, n)
		}
		if table, err := channelCounts(b); err == nil {
			fmt.Printf("  contadores: %v\n", table)
		}
		if len(b) > 0 {
			fmt.Printf("  assinatura (byte 0): 0x%02x\n", b[0])
		}
		if *set == "" && *counts == "" {
			return
		}
	}

	if *set != "" {
		idx, name, err := parseSet(*set)
		if err != nil {
			fmt.Fprintf(os.Stderr, "erro: %v\n", err)
			os.Exit(1)
		}
		if err := setName(b, idx, name); err != nil {
			fmt.Fprintf(os.Stderr, "erro: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("nome %d = %q\n", idx, name)
	}

	if *counts != "" {
		values, err := parseCounts(*counts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "erro: %v\n", err)
			os.Exit(1)
		}
		if err := setCounts(b, values); err != nil {
			fmt.Fprintf(os.Stderr, "erro: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("contadores = %v\n", values)
	}

	dst := *out
	if dst == "" {
		dst = *in
	}
	if err := os.WriteFile(dst, b, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "erro: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("%s: %d bytes gravados\n", dst, len(b))
}
