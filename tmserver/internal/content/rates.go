// Package content carrega os arquivos de conteúdo de jogo somente leitura (data-formats.md §2-3)
// de tmserver/content/: tabelas de taxa de combine/refino, os catálogos de item/skill e os
// mapas de terreno. Os loaders são validados contra os arquivos reais (content_test).
package content

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

const questTierCount = 5

// QuestTierRate é uma linha do QuestsRate.txt. Os níveis mínimos são inclusivos e os
// fora dela, casando com o gate Vol-191 do _MSG_UseItem.cpp.
type QuestTierRate struct {
	MortalExp, ArchExp                     int64
	Coin                                   int32
	MortalMin, MortalMax, ArchMin, ArchMax int32
}

// QuestRates guarda os cinco tiers de recompensa de item de quest (item 4117 + tier).
type QuestRates struct {
	tiers [questTierCount]QuestTierRate
}

var questRateDefaults = [questTierCount]QuestTierRate{
	{MortalExp: 1000, ArchExp: 500, Coin: 2000, MortalMin: 39, MortalMax: 115, ArchMin: 39, ArchMax: 115},
	{MortalExp: 2000, ArchExp: 1000, Coin: 4000, MortalMin: 115, MortalMax: 190, ArchMin: 115, ArchMax: 190},
	{MortalExp: 3000, ArchExp: 1500, Coin: 6000, MortalMin: 190, MortalMax: 265, ArchMin: 190, ArchMax: 265},
	{MortalExp: 4000, ArchExp: 2000, Coin: 8000, MortalMin: 265, MortalMax: 320, ArchMin: 265, ArchMax: 320},
	{MortalExp: 5000, ArchExp: 2500, Coin: 10000, MortalMin: 320, MortalMax: 350, ArchMin: 320, ArchMax: 350},
}

// DefaultQuestRates devolve os defaults de CReadFiles.cpp usados sem -content.
func DefaultQuestRates() *QuestRates { return &QuestRates{tiers: questRateDefaults} }

// Tier devolve um tier de quest e se o índice é válido.
func (q *QuestRates) Tier(tier int) (QuestTierRate, bool) {
	if q == nil || tier < 0 || tier >= len(q.tiers) {
		return QuestTierRate{}, false
	}
	return q.tiers[tier], true
}

// LoadQuestRates lê Common/Settings/QuestsRate.txt sobre os defaults legados.
func LoadQuestRates(path string) (*QuestRates, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("content: open QuestsRate: %w", err)
	}
	defer f.Close()
	return parseQuestRates(f)
}

func parseQuestRates(r io.Reader) (*QuestRates, error) {
	q := DefaultQuestRates()
	sc := bufio.NewScanner(r)
	for line := 1; sc.Scan(); line++ {
		fields := strings.Fields(sc.Text())
		if len(fields) == 0 || strings.HasPrefix(fields[0], "#") {
			continue
		}
		kind := asciiUpper(fields[0])
		var want int
		switch kind {
		case "EXP":
			want = 4
		case "COIN":
			want = 3
		case "LEVEL":
			want = 6
		default:
			return nil, fmt.Errorf("content: QuestsRate line %d: malformed directive", line)
		}
		if len(fields) != want {
			return nil, fmt.Errorf("content: QuestsRate line %d: malformed directive", line)
		}
		values := make([]int64, len(fields)-1)
		for i := 1; i < len(fields); i++ {
			v, err := strconv.ParseInt(fields[i], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("content: QuestsRate line %d: invalid integer %q", line, fields[i])
			}
			values[i-1] = v
		}
		tier := int(values[0])
		if tier < 0 || tier >= questTierCount {
			return nil, fmt.Errorf("content: QuestsRate line %d: tier %d out of range", line, tier)
		}
		r := &q.tiers[tier]
		switch kind {
		case "EXP":
			if values[1] < 0 || values[1] >= 2_000_000_000 || values[2] < 0 || values[2] >= 2_000_000_000 {
				return nil, fmt.Errorf("content: QuestsRate line %d: experience out of range", line)
			}
			r.MortalExp, r.ArchExp = values[1], values[2]
		case "COIN":
			if values[1] < 0 || values[1] > 2_000_000_000 {
				return nil, fmt.Errorf("content: QuestsRate line %d: coin out of range", line)
			}
			r.Coin = int32(values[1])
		case "LEVEL":
			for _, v := range values[1:] {
				if v < 0 || v >= 400 {
					return nil, fmt.Errorf("content: QuestsRate line %d: level out of range", line)
				}
			}
			if values[1] >= values[2] || values[3] >= values[4] {
				return nil, fmt.Errorf("content: QuestsRate line %d: empty level range", line)
			}
			r.MortalMin, r.MortalMax = int32(values[1]), int32(values[2])
			r.ArchMin, r.ArchMax = int32(values[3]), int32(values[4])
		}
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("content: read QuestsRate: %w", err)
	}
	return q, nil
}

// CompRate guarda as taxas de sucesso das receitas de combine (Common/Settings/CompRate.txt,
// CReadFiles::ReadCompRate, game-rules.md §3.2). As linhas são "Family Key Rate".
type CompRate struct {
	rates map[string]map[string]int // family → key → rate(0..100)
}

// Rate devolve a taxa de uma família/chave, e se ela foi encontrada.
func (c *CompRate) Rate(family, key string) (int, bool) {
	if m, ok := c.rates[family]; ok {
		r, ok := m[key]
		return r, ok
	}
	return 0, false
}

// Families devolve a contagem de famílias carregadas (para diagnóstico/testes).
func (c *CompRate) Families() int { return len(c.rates) }

// AnctChance devolve os bônus de sacrifício ITEM_+7/+8/+9 do Compositor somados à
// taxa base do Anct (g_pAnctChance, Basedef.cpp:113, defaults {2,4,10}). As chaves
// são comparadas sem diferenciar caixa porque o CompRate.txt usa caixa mista.
func (c *CompRate) AnctChance() [3]int {
	def := [3]int{2, 4, 10}
	if c == nil {
		return def
	}
	keys := []string{"ITEM_+7", "ITEM_+8", "ITEM_+9"}
	for i, want := range keys {
		if r, ok := c.rateCI("Compositor", want); ok {
			def[i] = r
		}
	}
	return def
}

// ehreRateDefaults é g_pEhreRate (Basedef.cpp:84), indexado pelo id de receita
// que GetMatchCombineEhre devolve. O índice 0 não é usado (0 = "sem receita") e
// o índice 5 (Refinação abençoada purificada) NÃO tem chave no CompRate — o handler
// recalcula aquela a partir do nível do personagem, então o default compilado só fica ali.
var ehreRateDefaults = [10]int{0, 100, 100, 40, 10, 10, 100, 100, 100, 100}

// ehreRateKeys mapeia os tokens "Ehre <key>" do CompRate.txt para seu slot em
// g_pEhreRate (CReadFiles.cpp:414-434). O arquivo só SOBRESCREVE o que lista, então os
// defaults acima são o ponto de partida — mesmo contrato do SancRate.
var ehreRateKeys = map[string]int{
	"PACOTE_ORI":             1,
	"MISTERIOSA":             2,
	"ESPIRITUAL":             3,
	"AMUNRA":                 4,
	"TRAJE_MONTARIA":         6,
	"RETIRAR_TRAJE_MONTARIA": 7,
	"SOUL":                   8,
}

// EhreRates devolve as taxas de sucesso por receita da família Ehre, indexadas pelo
// id de receita (1..8) que combine.MatchEhre devolve.
func (c *CompRate) EhreRates() [10]int {
	out := ehreRateDefaults
	if c == nil {
		return out
	}
	for key, idx := range ehreRateKeys {
		if r, ok := c.rateCI("Ehre", key); ok {
			out[idx] = r
		}
	}
	return out
}

// chanceBaseDefaults são g_pTinyBase/g_pShanyBase/g_pAilynBase/g_pAgathaBase
// (Basedef.cpp:108-111), a única taxa "ChanceBase" que cada uma dessas famílias
// carrega no CompRate.txt.
var chanceBaseDefaults = map[string]int{
	"TINY":   15,
	"SHANY":  35,
	"AILYN":  100,
	"AGATHA": 15,
}

// ChanceBase devolve a taxa "ChanceBase" de uma família do CompRate.txt, caindo
// para o default compilado de Basedef. Uma família desconhecida devolve 0, o que
// faz suas receitas sempre falharem em vez de passarem em silêncio.
func (c *CompRate) ChanceBase(family string) int {
	def := chanceBaseDefaults[asciiUpper(family)]
	if c == nil {
		return def
	}
	if r, ok := c.rateCI(family, "ChanceBase"); ok {
		return r
	}
	return def
}

func (c *CompRate) rateCI(family, key string) (int, bool) {
	if r, ok := c.Rate(family, key); ok {
		return r, true
	}
	fu := strings.ToUpper(family)
	ku := strings.ToUpper(key)
	for fam, m := range c.rates {
		if strings.ToUpper(fam) != fu {
			continue
		}
		for k, r := range m {
			if strings.ToUpper(k) == ku {
				return r, true
			}
		}
	}
	return 0, false
}

// LoadCompRate lê CompRate.txt.
func LoadCompRate(path string) (*CompRate, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("content: open CompRate: %w", err)
	}
	defer f.Close()
	return parseCompRate(f)
}

func parseCompRate(r io.Reader) (*CompRate, error) {
	c := &CompRate{rates: make(map[string]map[string]int)}
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) != 3 { // blank or malformed → skip
			continue
		}
		rate, err := strconv.Atoi(fields[2])
		if err != nil {
			continue
		}
		family, key := fields[0], fields[1]
		if c.rates[family] == nil {
			c.rates[family] = make(map[string]int)
		}
		c.rates[family][key] = rate
	}
	return c, sc.Err()
}

// SancRate guarda as taxas de sucesso do refino com pó por bigorna e índice de tabela
// (Common/Settings/SancRate.txt, ReadSancRate, game-rules.md §3.3). As linhas são
// "Anvil Index Rate".
//
// A tabela é fixa [3][12]: linha 0 = Ori (PO), 1 = Lac (PL), 2 = Âmago, e
// BASE_GetSuccessRate a indexa em level+1 (o slot 0 não é usado). O arquivo só
// SOBRESCREVE os índices que lista — ReadSancRate nunca limpa a tabela — então os
// defaults compilados de Basedef.cpp:68 são o ponto de partida.
type SancRate struct {
	rates [3][12]int
}

// sancRateDefaults é g_pSancRate (Basedef.cpp:68), a tabela antes de
// SancRate.txt ser aplicado sobre ela.
var sancRateDefaults = [3][12]int{
	{100, 100, 100, 100, 100, 100, 100, 0, 0, 0, 0, 0},         // PO  (Ori)
	{100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 0, 0},   // PL  (Lac)
	{100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 80, 80}, // Âmago
}

// Linhas de bigorna, espelhando refine.Ori/Lac/Amago (mantidas como ints simples para
// evitar um import content→refine).
const (
	sancAnvilOri = 0
	sancAnvilLac = 1
	sancAnvilAma = 2
)

// Rate devolve a taxa de sucesso de uma linha de bigorna num índice de tabela. Buscas
// fora da faixa e tabela nil devolvem 0 (sempre falha) em vez de dar panic no
// laço do jogo.
func (s *SancRate) Rate(anvil, idx int) int {
	if s == nil {
		return 0
	}
	if anvil < 0 || anvil >= len(s.rates) || idx < 0 || idx >= len(s.rates[anvil]) {
		return 0
	}
	return s.rates[anvil][idx]
}

// DefaultSancRate devolve a tabela só com os defaults de Basedef.cpp — o que
// o caminho de refino usa quando nenhuma árvore -content está montada.
func DefaultSancRate() *SancRate { return &SancRate{rates: sancRateDefaults} }

// LoadSancRate lê SancRate.txt.
func LoadSancRate(path string) (*SancRate, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("content: open SancRate: %w", err)
	}
	defer f.Close()
	return parseSancRate(f)
}

// sancAnvilRow mapeia um token de bigorna do SancRate.txt para sua linha na tabela.
//
// O arquivo é cp1252, não UTF-8: "Âmago" são os bytes C2 6D 61 67 6F. O legado
// também nunca o decodifica — ReadFiles aplica _strupr no token (só ASCII no
// locale C, então o 0xC2 sobrevive) e o compara com strcmp contra o literal cp1252
// "ÂMAGO". Comparar bytes crus do mesmo jeito é exato e dispensa decodificador; a
// grafia UTF-8 também é aceita caso o arquivo seja regravado.
func sancAnvilRow(token string) (int, bool) {
	switch asciiUpper(token) {
	case "PO":
		return sancAnvilOri, true
	case "PL":
		return sancAnvilLac, true
	case "\xC2MAGO", "\xC3\x82MAGO": // cp1252 / UTF-8 "ÂMAGO"
		// Divergência deliberada do legado: CReadFiles.cpp:157 escreve as
		// linhas de Âmago em g_pSancRate[0] — a linha PO (:123) — um bug de copiar-colar.
		// Como Âmago é listado por último no arquivo, ele sobrescrevia em silêncio toda
		// taxa Ori. A issue #103 resolveu isso como "corrigir", então as linhas PO do
		// arquivo são as que valem e Âmago cai na própria linha.
		return sancAnvilAma, true
	}
	return 0, false
}

// asciiUpper põe a-z em maiúscula byte a byte, deixando todo outro byte intocado — o
// _strupr do runtime C no locale "C". strings.ToUpper não pode ser usado: ele
// decodifica runes, e o 0xC2 cp1252 é UTF-8 inválido, então seria reescrito
// como 3 bytes U+FFFD.
func asciiUpper(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'a' && c <= 'z' {
			b[i] = c - ('a' - 'A')
		}
	}
	return string(b)
}

func parseSancRate(r io.Reader) (*SancRate, error) {
	s := &SancRate{rates: sancRateDefaults}
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) != 3 {
			continue
		}
		idx, err1 := strconv.Atoi(fields[1])
		rate, err2 := strconv.Atoi(fields[2])
		if err1 != nil || err2 != nil {
			continue
		}
		anvil, ok := sancAnvilRow(fields[0])
		if !ok || idx < 0 || idx >= len(s.rates[anvil]) {
			continue
		}
		s.rates[anvil][idx] = rate
	}
	return s, sc.Err()
}
