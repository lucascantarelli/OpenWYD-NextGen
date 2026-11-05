// Command serverlist gera o serverlist.bin que o cliente lê na tela inicial, a
// partir de Release/Common/serverlist.txt. Substitui o "serverlist editor.exe"
// (Windows, fechado) por código nosso — issue 52.
//
// O formato é o mesmo de BASE_InitializeServerList (Source/Code/Basedef.cpp:6874):
//
//	MAX_SERVERGROUP(10) × MAX_SERVERNUMBER(11) × 64 bytes
//
// cada célula é o endereço em ASCII, terminado em NUL, com cada byte somado de
// obfuscationKey[63-i] (o legado soma e depois subtrai a mesma tabela em memória).
//
// O terceiro campo é o conteúdo da CÉLULA, não necessariamente um IP: nos arquivos
// gerados pelo editor do cliente cada grupo usa a célula 0 para a URL de status
// ("http://<ip>:8281/serv00.htm") e as células 1..N para os endereços dos canais
// (ver docs/migration/canais.md §8).
//
// Uso:
//
//	go run ./client-tools/serverlist -in Release/Common/serverlist.txt -out serverlist.bin
//	go run ./client-tools/serverlist -print serverlist.bin
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

const (
	// GroupCount is MAX_SERVERGROUP (Basedef.h:109): how many "servers" (groups
	// of channels) the client can list.
	GroupCount = 10
	// ChannelCount is MAX_SERVERNUMBER = MAX_SERVER+1 (Basedef.h:108-110).
	ChannelCount = 11
	// AddrLen is the fixed width of one entry (g_pServerList[..][..][64]).
	AddrLen = 64
	// BinSize is sizeof(g_pServerList) and the exact size of the generated file.
	BinSize = GroupCount * ChannelCount * AddrLen
)

// obfuscationKey is the static table the legacy adds to every byte, reversed
// (Basedef.cpp:6927: g_pServerList[k][j][i] += szList[63-i]).
var obfuscationKey = [AddrLen]byte{
	0xA4, 0xA1, 0xA4, 0xA4, 0xA4, 0xA7, 0xA4, 0xA9, 0xA4, 0xB1, 0xA4, 0xB2, 0xA4, 0xB5, 0xA4, 0xB7,
	0xA4, 0xB8, 0xA4, 0xBA, 0xA4, 0xBB, 0xA4, 0xBC, 0xA4, 0xBD, 0xA4, 0xBE, 0xA4, 0xBF, 0xA4, 0xC1,
	0xA4, 0xC3, 0xA4, 0xC5, 0xA4, 0xC7, 0xA4, 0xCB, 0xA4, 0xCC, 0xA4, 0xD0, 0xA4, 0xD1, 0xA4, 0xD3,
	0xA4, 0xBF, 0xA4, 0xC4, 0xA4, 0xD3, 0xA4, 0xC7, 0xA4, 0xCC, 0xB0, 0xA1, 0xB3, 0xAA, 0xB4, 0xD9,
}

// Entry is one line of serverlist.txt: "<grupo> <canal> <endereço>".
type Entry struct {
	Group   int
	Channel int
	Addr    string
}

// ParseList reads the text serverlist. Blank lines and '#' comments are ignored;
// an address may carry a port ("host:8282") since the field is a free string.
func ParseList(r io.Reader) ([]Entry, error) {
	var out []Entry
	sc := bufio.NewScanner(r)
	for line := 1; sc.Scan(); line++ {
		text := strings.TrimSpace(sc.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		fields := strings.Fields(text)
		if len(fields) != 3 {
			return nil, fmt.Errorf("linha %d: esperado \"<grupo> <canal> <endereço>\", veio %q", line, text)
		}
		group, err := strconv.Atoi(fields[0])
		if err != nil {
			return nil, fmt.Errorf("linha %d: grupo inválido %q", line, fields[0])
		}
		channel, err := strconv.Atoi(fields[1])
		if err != nil {
			return nil, fmt.Errorf("linha %d: canal inválido %q", line, fields[1])
		}
		if group < 0 || group >= GroupCount {
			return nil, fmt.Errorf("linha %d: grupo %d fora de 0..%d", line, group, GroupCount-1)
		}
		if channel < 0 || channel >= ChannelCount {
			return nil, fmt.Errorf("linha %d: canal %d fora de 0..%d", line, channel, ChannelCount-1)
		}
		if len(fields[2]) >= AddrLen {
			return nil, fmt.Errorf("linha %d: endereço maior que %d bytes", line, AddrLen-1)
		}
		out = append(out, Entry{Group: group, Channel: channel, Addr: fields[2]})
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// Build lays the entries out in the fixed 10×11×64 grid and obfuscates every
// byte. Later entries win, like the sequential strcpy in the legacy loader.
func Build(entries []Entry) ([]byte, error) {
	b := make([]byte, BinSize)
	if len(entries) == 0 {
		return nil, errors.New("nenhuma entrada: a serverlist ficaria vazia")
	}
	for _, e := range entries {
		if e.Group < 0 || e.Group >= GroupCount || e.Channel < 0 || e.Channel >= ChannelCount {
			return nil, fmt.Errorf("entrada fora da grade: grupo %d canal %d", e.Group, e.Channel)
		}
		if e.Addr == "" || len(e.Addr) >= AddrLen {
			return nil, fmt.Errorf("endereço inválido para grupo %d canal %d: %q", e.Group, e.Channel, e.Addr)
		}
		slot := (e.Group*ChannelCount + e.Channel) * AddrLen
		copy(b[slot:slot+AddrLen-1], e.Addr) // NUL terminating, as strcpy would
	}
	for slot := 0; slot < BinSize; slot += AddrLen {
		for i := 0; i < AddrLen; i++ {
			b[slot+i] += obfuscationKey[AddrLen-1-i]
		}
	}
	return b, nil
}

// Decode reverses Build, so a generated file can be inspected (and the writer
// tested end-to-end).
func Decode(b []byte) ([][]string, error) {
	if len(b) != BinSize {
		return nil, fmt.Errorf("tamanho %d, esperado %d", len(b), BinSize)
	}
	out := make([][]string, GroupCount)
	for g := 0; g < GroupCount; g++ {
		out[g] = make([]string, ChannelCount)
		for c := 0; c < ChannelCount; c++ {
			slot := (g*ChannelCount + c) * AddrLen
			addr := make([]byte, AddrLen)
			for i := 0; i < AddrLen; i++ {
				addr[i] = b[slot+i] - obfuscationKey[AddrLen-1-i]
			}
			if i := strings.IndexByte(string(addr), 0); i >= 0 {
				addr = addr[:i]
			}
			out[g][c] = string(addr)
		}
	}
	return out, nil
}

func main() {
	in := flag.String("in", "Release/Common/serverlist.txt", "serverlist.txt de entrada")
	out := flag.String("out", "serverlist.bin", "arquivo .bin a gerar")
	print := flag.String("print", "", "imprime o conteúdo de um .bin existente")
	flag.Parse()

	if *print != "" {
		raw, err := os.ReadFile(*print)
		if err != nil {
			fmt.Fprintln(os.Stderr, "erro:", err)
			os.Exit(1)
		}
		grid, err := Decode(raw)
		if err != nil {
			fmt.Fprintln(os.Stderr, "erro:", err)
			os.Exit(1)
		}
		for g := range grid {
			for c, addr := range grid[g] {
				if addr != "" {
					fmt.Printf("%d %d %s\n", g, c, addr)
				}
			}
		}
		return
	}

	f, err := os.Open(*in)
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		os.Exit(1)
	}
	defer f.Close()
	entries, err := ParseList(f)
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		os.Exit(1)
	}
	bin, err := Build(entries)
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*out, bin, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		os.Exit(1)
	}
	fmt.Printf("%s: %d entradas, %d bytes\n", *out, len(entries), len(bin))
	for _, e := range entries {
		fmt.Printf("  grupo %d canal %d → %s\n", e.Group, e.Channel, e.Addr)
	}
}
