package content

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"openwyd-nextgen/internal/npctemplate"
)

// NPCGenerator é um bloco de spawn do NPCGener.txt (CNPCGene.cpp ParseString):
// ele gera de MinGroup a MaxGroup mobs de Leader (mais Followers) patrulhando os
// waypoints SegX/SegY conforme o RouteType, regenerando a cada MinuteGenerate minutos.
//
// O mapeamento de índice de waypoint é o do original: Start*→[0], Segment1..3*→[1..3],
// Dest*→[4]. Waypoints não usados ficam 0 e o andador de segmento os pula
// (SetSegment, CMob.cpp:608) — então o bloco comum só com Start/Dest patrulha
// 0→4→0. Os acessores StartX/StartY mantêm o nome de ponto de spawn usado em outros lugares.
type NPCGenerator struct {
	Leader         string
	Follower       string
	MinuteGenerate int // respawn period in minutes; <=0 = never regenerate
	MinGroup       int
	MaxGroup       int
	MaxNumMob      int
	RouteType      int
	Formation      int
	SegX, SegY     [5]int16
	SegRange       [5]int
	SegWait        [5]int
	FightAction    [4]string
	DieAction      [4]string
}

// LoadNPCGenerators lê NPCGener.txt. Os blocos começam com '#'; as linhas são
// "Key:\tvalue"; linhas '//' são comentários.
func LoadNPCGenerators(path string) ([]NPCGenerator, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("content: open NPCGener: %w", err)
	}
	defer f.Close()

	var out []NPCGenerator
	var cur *NPCGenerator
	flush := func() {
		if cur != nil && cur.Leader != "" {
			out = append(out, *cur)
		}
	}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		if strings.HasPrefix(line, "#") {
			flush()
			cur = &NPCGenerator{MinGroup: 1}
			continue
		}
		if cur == nil {
			continue
		}
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		switch key {
		case "Leader":
			cur.Leader = val
		case "Follower":
			if val != "0" { // "0" = no follower template (ParseString rejects it)
				cur.Follower = val
			}
		case "MinuteGenerate":
			cur.MinuteGenerate = atoi(val)
		case "MinGroup":
			cur.MinGroup = atoi(val)
		case "MaxGroup":
			cur.MaxGroup = atoi(val)
		case "MaxNumMob":
			cur.MaxNumMob = atoi(val)
		case "RouteType":
			cur.RouteType = atoi(val)
		case "Formation":
			cur.Formation = atoi(val)
		case "FightAction":
			setAction(&cur.FightAction, val)
		case "DieAction":
			setAction(&cur.DieAction, val)
		// Waypoints: Start→[0], Segment1..3→[1..3], Dest→[4] (ParseString).
		case "StartX":
			cur.SegX[0] = int16(atoi(val))
		case "StartY":
			cur.SegY[0] = int16(atoi(val))
		case "StartRange":
			cur.SegRange[0] = atoi(val)
		case "StartWait":
			cur.SegWait[0] = atoi(val)
		case "DestX":
			cur.SegX[4] = int16(atoi(val))
		case "DestY":
			cur.SegY[4] = int16(atoi(val))
		case "DestRange":
			cur.SegRange[4] = atoi(val)
		case "DestWait":
			cur.SegWait[4] = atoi(val)
		default:
			if setIndexedAction(key, "FightAction", &cur.FightAction, val) ||
				setIndexedAction(key, "DieAction", &cur.DieAction, val) {
				continue
			}
			// Segment1..3{X,Y,Range,Wait} → índices 1..3. As chaves Segment*Action são
			// texto de sabor da rota e continuam sem mapeamento; FightAction/DieAction acima são
			// linhas de chat visíveis em combate.
			if strings.HasPrefix(key, "Segment") && len(key) > 8 {
				if i := int(key[7] - '0'); i >= 1 && i <= 3 {
					switch key[8:] {
					case "X":
						cur.SegX[i] = int16(atoi(val))
					case "Y":
						cur.SegY[i] = int16(atoi(val))
					case "Range":
						cur.SegRange[i] = atoi(val)
					case "Wait":
						cur.SegWait[i] = atoi(val)
					}
				}
			}
		}
	}
	flush()
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("content: scan NPCGener: %w", err)
	}
	return out, nil
}

// LoadNPCTemplate lê um template de mob (tmserver/content/TMsrv/run/npc/<name>) e
// o devolve como STRUCT_MOB canônico cru de 816 bytes — os layouts legados de
// 756/920 bytes daquele diretório são alargados por npctemplate.Load (data-formats.md
// §1.4.1). Nomes legados do NPCGener são resolvidos com compatibilidade de caminho
// no estilo Windows (sem diferenciar caixa, pontos finais ignorados). Os templates são
// cacheados por quem chama (muitos geradores compartilham um nome).
func LoadNPCTemplate(dir, name string) ([]byte, error) {
	b, _, err := npctemplate.Load(dir, name)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func atoi(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}

func setAction(dst *[4]string, val string) {
	if len(val) >= 79 {
		return // CNPCGene::SetAct rejects action strings that would overflow [80].
	}
	for i := range dst {
		if dst[i] == "" {
			dst[i] = val
			return
		}
	}
}

func setIndexedAction(key, prefix string, dst *[4]string, val string) bool {
	if !strings.HasPrefix(key, prefix) || key == prefix {
		return false
	}
	n, err := strconv.Atoi(key[len(prefix):])
	if err != nil {
		return false
	}
	if n < 1 || n > 4 {
		return false
	}
	if len(val) < 79 {
		dst[n-1] = val
	}
	return true
}
