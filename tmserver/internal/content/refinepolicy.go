package content

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"openwyd-nextgen/internal/ratetables"
)

// RefinePolicy é o teto do refino com pó por tipo de pó, movido do código
// (_MSG_UseItem.cpp tinha 6/9/10/11 fixo) para Common/Settings/RefineRate.txt e
// a sobreposição do moderador (content_rate_override, arquivo "RefineRate"). As
// chances continuam no SancRate.txt — esta tabela só diz até onde cada pó pode
// levar um item.
//
// OriMax/PLMax são o nível mais alto que cada pó alcança, GemLevel é o passo
// especial +10 (um sucesso ali vai para +11 e mantém a gema) e HardCap é a
// parede absoluta: nele ou acima dele o caminho do pó recusa direto.
type RefinePolicy struct {
	OriMax   int
	PLMax    int
	GemLevel int
	HardCap  int
}

// DefaultRefinePolicy é o comportamento legado (_MSG_UseItem.cpp:203/802/857),
// usado quando nenhuma árvore -content está montada.
func DefaultRefinePolicy() RefinePolicy {
	return RefinePolicy{OriMax: 6, PLMax: 9, GemLevel: 10, HardCap: 11}
}

// RefinePolicyKeys são os nomes editáveis, na ordem do arquivo.
var RefinePolicyKeys = []string{"PO_MAX", "PL_MAX", "GEM_LEVEL", "HARD_CAP"}

// RefinePolicyMax limita toda chave: REF_11 é 12 e nenhum caminho legado refina
// além de +12, então um valor acima de 15 nunca seria alcançado por um item de qualquer forma.
const RefinePolicyMax = 15

// LoadRefinePolicy lê Common/Settings/RefineRate.txt. Um arquivo ausente não é
// fatal — os defaults legados ficam e quem chama registra no log — porque esta
// tabela é conveniência do moderador, não conteúdo necessário para o mundo funcionar.
func LoadRefinePolicy(path string) (RefinePolicy, error) {
	f, err := os.Open(path)
	if err != nil {
		return DefaultRefinePolicy(), fmt.Errorf("content: open RefineRate: %w", err)
	}
	defer f.Close()
	return ParseRefinePolicy(f)
}

// ParseRefinePolicy lê linhas "POLICY <KEY> <VALUE>". Linhas malformadas e
// chaves desconhecidas são puladas: o boot não pode morrer porque um moderador
// digitou uma palavra perdida no arquivo.
func ParseRefinePolicy(r io.Reader) (RefinePolicy, error) {
	p := DefaultRefinePolicy()
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) != 3 || !strings.EqualFold(fields[0], "POLICY") {
			continue
		}
		v, err := strconv.Atoi(fields[2])
		if err != nil || v < 0 || v > RefinePolicyMax {
			continue
		}
		switch strings.ToUpper(fields[1]) {
		case "PO_MAX":
			p.OriMax = v
		case "PL_MAX":
			p.PLMax = v
		case "GEM_LEVEL":
			p.GemLevel = v
		case "HARD_CAP":
			p.HardCap = v
		}
	}
	return p.normalized(), sc.Err()
}

// normalized mantém a tabela coerente: os tetos de pó não podem exceder o
// limite absoluto, ou um pó poderia chegar a um nível que o item nunca poderia ter.
func (p RefinePolicy) normalized() RefinePolicy {
	if p.HardCap < 1 {
		p.HardCap = DefaultRefinePolicy().HardCap
	}
	if p.OriMax > p.HardCap {
		p.OriMax = p.HardCap
	}
	if p.PLMax > p.HardCap {
		p.PLMax = p.HardCap
	}
	if p.GemLevel > p.HardCap {
		p.GemLevel = p.HardCap
	}
	return p
}

// ApplyRefineOverrides aplica a sobreposição do moderador na política, com a mesma
// identidade (arquivo, família, nome) que todas as outras taxas usam.
func ApplyRefineOverrides(p RefinePolicy, overrides []ratetables.Override) (RefinePolicy, int, int) {
	applied, ignored := 0, 0
	for _, o := range overrides {
		if o.File != ratetables.FileRefineRate {
			continue
		}
		if o.Value < 0 || o.Value > RefinePolicyMax {
			ignored++
			continue
		}
		switch strings.ToUpper(o.Name) {
		case "PO_MAX":
			p.OriMax = int(o.Value)
		case "PL_MAX":
			p.PLMax = int(o.Value)
		case "GEM_LEVEL":
			p.GemLevel = int(o.Value)
		case "HARD_CAP":
			p.HardCap = int(o.Value)
		default:
			ignored++
			continue
		}
		applied++
	}
	return p.normalized(), applied, ignored
}
