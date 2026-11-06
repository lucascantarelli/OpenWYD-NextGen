// Package ratetables é o único parser das tabelas de taxa editáveis em
// tmserver/content/Common/Settings — a tabela de refino (SancRate), a de
// combine/receita (CompRate) e as faixas de recompensa de quest (QuestsRate).
//
// Existe para que a web-api e o tmServer não divirjam: a UI do moderador serve
// as mesmas linhas que o tmServer carrega, com os mesmos defaults e as mesmas
// regras de exceção (SancRate só sobrescreve os índices que o arquivo lista, e
// Âmago cai na própria linha em vez do erro de cópia do legado na linha Ori).
//
// Uma célula é identificada por (file, family, name) e NÃO por posição
// numérica: CompRate tem muitas chaves por família ("Compositor Item_+7",
// "Compositor Item_+8"), então uma chave posicional as faria colidir. name é
// estável quando o arquivo ganha linhas novas:
//
//	SancRate   family "PO"/"PL"/"AMAGO"      name "0".."11"  (índice do refino)
//	CompRate   family "Compositor"/"Ehre"/…  name "Item_+7"   (chave do arquivo)
//	QuestsRate family "EXP"/"COIN"/"LEVEL"   name "<tier>.<coluna>"
package ratetables

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Nomes de arquivo, como guardados em content_rate_override.file.
const (
	FileSancRate   = "SancRate"
	FileCompRate   = "CompRate"
	FileQuestsRate = "QuestsRate"
)

// MaxPercent é o limite de toda coluna de taxa de SancRate/CompRate.
const MaxPercent = 100

// MaxReward limita as colunas EXP/COIN de QuestsRate (o loader usa o mesmo teto:
// 2_000_000_000).
const MaxReward = 2_000_000_000

// MaxLevel limita as janelas de nível de QuestsRate.
const MaxLevel = 399

// Row é uma célula editável: (file, family, name) → valor, mais a faixa que o
// backend impõe e um rótulo para a UI.
type Row struct {
	File   string
	Family string
	Name   string
	Value  int64
	Min    int64
	Max    int64
	Label  string
}

// Override é uma edição do moderador carregada de content_rate_override.
type Override struct {
	File   string
	Family string
	Name   string
	Value  int64
}

// FileRefineRate é a tabela de teto do refino com poeira (RefineRate.txt). As
// linhas são "POLICY <CHAVE> <VALOR>" e o tmServer as consome como política, não
// como tabela de percentuais.
const FileRefineRate = "RefineRate"

// Files lista as tabelas que o editor expõe.
func Files() []string { return []string{FileSancRate, FileCompRate, FileQuestsRate, FileRefineRate} }

// refinePolicyKeys são as células editáveis de RefineRate, com a faixa que o
// moderador pode definir (o tmServer limita os tetos de poeira a HARD_CAP ao
// carregar).
var refinePolicyKeys = []struct {
	Name  string
	Label string
}{
	{"PO_MAX", "Poeira de Ori · refino máximo"},
	{"PL_MAX", "Poeira de Lac · refino máximo"},
	{"GEM_LEVEL", "Nível do passo especial da gema"},
	{"HARD_CAP", "Teto absoluto do refino"},
}

// RefineRateMax limita todo valor de RefineRate (nenhum caminho do legado refina
// além de +12).
const RefineRateMax = 15

// refineRows lê RefineRate.txt do mesmo jeito que o tmServer: "POLICY chave
// valor", ignorando comentários e linhas malformadas. Arquivo ausente rende os
// defaults do legado, para o editor ainda mostrar o teto efetivo.
func refineRows(path string) ([]Row, error) {
	values := map[string]int64{"PO_MAX": 6, "PL_MAX": 9, "GEM_LEVEL": 10, "HARD_CAP": 11}
	f, err := os.Open(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("ratetables: open %s: %w", path, err)
		}
	} else {
		defer f.Close()
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			fields := strings.Fields(sc.Text())
			if len(fields) != 3 || !strings.EqualFold(fields[0], "POLICY") {
				continue
			}
			v, err := strconv.ParseInt(fields[2], 10, 64)
			key := strings.ToUpper(fields[1])
			if err != nil || v < 0 || v > RefineRateMax {
				continue
			}
			if _, ok := values[key]; !ok {
				continue
			}
			values[key] = v
		}
		if err := sc.Err(); err != nil {
			return nil, fmt.Errorf("ratetables: read RefineRate: %w", err)
		}
	}
	out := make([]Row, 0, len(refinePolicyKeys))
	for _, k := range refinePolicyKeys {
		out = append(out, Row{
			File: FileRefineRate, Family: "POLICY", Name: k.Name,
			Value: values[k.Name], Min: 0, Max: RefineRateMax, Label: k.Label,
		})
	}
	return out, nil
}

// Rows lê todas as tabelas editáveis de contentDir/Common/Settings numa ordem
// estável (arquivo, depois família, depois nome), para a UI e os testes
// compararem fatias.
func Rows(contentDir string) ([]Row, error) {
	settings := filepath.Join(contentDir, "Common", "Settings")
	sanc, err := sancRows(filepath.Join(settings, "SancRate.txt"))
	if err != nil {
		return nil, err
	}
	comp, err := compRows(filepath.Join(settings, "CompRate.txt"))
	if err != nil {
		return nil, err
	}
	quests, err := questsRows(filepath.Join(settings, "QuestsRate.txt"))
	if err != nil {
		return nil, err
	}
	refine, err := refineRows(filepath.Join(settings, "RefineRate.txt"))
	if err != nil {
		return nil, err
	}
	out := make([]Row, 0, len(sanc)+len(comp)+len(quests)+len(refine))
	out = append(out, sanc...)
	out = append(out, comp...)
	out = append(out, quests...)
	out = append(out, refine...)
	return out, nil
}

// Apply aplica os overrides nas linhas lidas e informa quantos entraram. Chave
// sem linha correspondente é ignorada (e contada), nunca inventada: o arquivo é
// a fonte de quais células existem.
func Apply(rows []Row, overrides []Override) ([]Row, int, int) {
	if len(overrides) == 0 {
		return rows, 0, 0
	}
	byKey := make(map[key]int, len(rows))
	out := make([]Row, len(rows))
	copy(out, rows)
	for i, row := range out {
		byKey[key{row.File, row.Family, row.Name}] = i
	}
	applied, ignored := 0, 0
	for _, o := range overrides {
		idx, ok := byKey[key{o.File, o.Family, o.Name}]
		if !ok {
			ignored++
			continue
		}
		out[idx].Value = o.Value
		applied++
	}
	return out, applied, ignored
}

// Validate confere um valor contra a faixa da linha.
func Validate(row Row, value int64) error {
	if value < row.Min || value > row.Max {
		return fmt.Errorf("%s/%s/%s: valor %d fora da faixa %d..%d",
			row.File, row.Family, row.Name, value, row.Min, row.Max)
	}
	return nil
}

type key struct {
	File, Family, Name string
}

// SancAnvils são as três linhas de refino, na ordem da tabela.
var SancAnvils = []string{"PO", "PL", "AMAGO"}

// SancRowCount é a forma fixa [3][12] da tabela de refino.
const SancRowCount = 12

func sancRows(path string) ([]Row, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("ratetables: open %s: %w", path, err)
	}
	defer f.Close()
	return parseSanc(f)
}

func parseSanc(r io.Reader) ([]Row, error) {
	// A tabela é fixa: toda célula (bigorna, índice) existe mesmo quando o arquivo
	// a omite, porque ReadSancRate nunca limpa g_pSancRate. Emitir a forma completa
	// é o que torna "sem linha = valor compilado" visível na UI.
	const anvils = 3
	defaults := [anvils][SancRowCount]int64{
		{100, 100, 100, 100, 100, 100, 100, 0, 0, 0, 0, 0},
		{100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 0, 0},
		{100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 80, 80},
	}
	values := defaults

	sc := bufio.NewScanner(r)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) != 3 {
			continue
		}
		idx, err1 := strconv.Atoi(fields[1])
		rate, err2 := strconv.ParseInt(fields[2], 10, 64)
		if err1 != nil || err2 != nil {
			continue
		}
		anvil, ok := sancAnvil(fields[0])
		if !ok || idx < 0 || idx >= SancRowCount {
			continue
		}
		values[anvil][idx] = rate
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("ratetables: read SancRate: %w", err)
	}

	out := make([]Row, 0, anvils*SancRowCount)
	for anvil, family := range SancAnvils {
		for idx := 0; idx < SancRowCount; idx++ {
			out = append(out, Row{
				File: FileSancRate, Family: family, Name: strconv.Itoa(idx),
				Value: values[anvil][idx], Min: 0, Max: MaxPercent,
				Label: fmt.Sprintf("%s · índice %d", family, idx),
			})
		}
	}
	return out, nil
}

// sancAnvil mapeia o token da bigorna para a sua linha com a mesma comparação de
// bytes que o jogo faz (o arquivo é cp1252, então strings.ToUpper estragaria
// 0xC2).
func sancAnvil(token string) (int, bool) {
	switch asciiUpper(token) {
	case "PO":
		return 0, true
	case "PL":
		return 1, true
	case "\xC2MAGO", "\xC3\x82MAGO":
		return 2, true
	}
	return 0, false
}

func compRows(path string) ([]Row, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("ratetables: open %s: %w", path, err)
	}
	defer f.Close()
	return parseComp(f)
}

func parseComp(r io.Reader) ([]Row, error) {
	var out []Row
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) != 3 {
			continue
		}
		rate, err := strconv.ParseInt(fields[2], 10, 64)
		if err != nil {
			continue
		}
		out = append(out, Row{
			File: FileCompRate, Family: fields[0], Name: fields[1],
			Value: rate, Min: 0, Max: MaxPercent,
			Label: fields[0] + " · " + fields[1],
		})
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("ratetables: read CompRate: %w", err)
	}
	return out, nil
}

func questsRows(path string) ([]Row, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("ratetables: open %s: %w", path, err)
	}
	defer f.Close()
	return parseQuests(f)
}

// questColumns descreve os quatro campos de uma linha "Level", na ordem do
// arquivo.
var questColumns = []string{"mortal min", "mortal max", "arch min", "arch max"}

func parseQuests(r io.Reader) ([]Row, error) {
	const tiers = 5
	type tier struct {
		expMortal, expArch int64
		coin               int64
		levels             [4]int64
	}
	defaults := [tiers]tier{
		{1000, 500, 2000, [4]int64{39, 115, 39, 115}},
		{2000, 1000, 4000, [4]int64{115, 190, 115, 190}},
		{3000, 1500, 6000, [4]int64{190, 265, 190, 265}},
		{4000, 2000, 8000, [4]int64{265, 320, 265, 320}},
		{5000, 2500, 10000, [4]int64{320, 350, 320, 350}},
	}
	values := defaults

	sc := bufio.NewScanner(r)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) == 0 || strings.HasPrefix(fields[0], "#") {
			continue
		}
		switch asciiUpper(fields[0]) {
		case "EXP":
			if len(fields) != 4 {
				continue
			}
			t, err1 := strconv.Atoi(fields[1])
			mortal, err2 := strconv.ParseInt(fields[2], 10, 64)
			arch, err3 := strconv.ParseInt(fields[3], 10, 64)
			if err1 != nil || err2 != nil || err3 != nil || t < 0 || t >= tiers {
				continue
			}
			values[t].expMortal, values[t].expArch = mortal, arch
		case "COIN":
			if len(fields) != 3 {
				continue
			}
			t, err1 := strconv.Atoi(fields[1])
			coin, err2 := strconv.ParseInt(fields[2], 10, 64)
			if err1 != nil || err2 != nil || t < 0 || t >= tiers {
				continue
			}
			values[t].coin = coin
		case "LEVEL":
			if len(fields) != 6 {
				continue
			}
			t, err1 := strconv.Atoi(fields[1])
			if err1 != nil || t < 0 || t >= tiers {
				continue
			}
			for i := 0; i < 4; i++ {
				v, err := strconv.ParseInt(fields[2+i], 10, 64)
				if err != nil {
					break
				}
				values[t].levels[i] = v
			}
		}
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("ratetables: read QuestsRate: %w", err)
	}

	out := make([]Row, 0, tiers*7)
	for t := 0; t < tiers; t++ {
		prefix := strconv.Itoa(t)
		out = append(out,
			Row{File: FileQuestsRate, Family: "EXP", Name: prefix + ".0", Value: values[t].expMortal,
				Min: 0, Max: MaxReward, Label: fmt.Sprintf("tier %d · EXP mortal", t)},
			Row{File: FileQuestsRate, Family: "EXP", Name: prefix + ".1", Value: values[t].expArch,
				Min: 0, Max: MaxReward, Label: fmt.Sprintf("tier %d · EXP arch", t)},
			Row{File: FileQuestsRate, Family: "COIN", Name: prefix + ".0", Value: values[t].coin,
				Min: 0, Max: MaxReward, Label: fmt.Sprintf("tier %d · moedas", t)},
		)
		for i, column := range questColumns {
			out = append(out, Row{
				File: FileQuestsRate, Family: "LEVEL", Name: prefix + "." + strconv.Itoa(i),
				Value: values[t].levels[i], Min: 0, Max: MaxLevel,
				Label: fmt.Sprintf("tier %d · %s", t, column),
			})
		}
	}
	return out, nil
}

// asciiUpper converte só a-z para maiúscula, espelhando _strupr do runtime C no
// locale "C": o byte cp1252 0xC2 não é UTF-8 válido, então strings.ToUpper o
// substituiria.
func asciiUpper(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'a' && c <= 'z' {
			b[i] = c - ('a' - 'A')
		}
	}
	return string(b)
}
