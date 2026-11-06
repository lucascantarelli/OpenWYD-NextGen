package content

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"openwyd-nextgen/internal/domain"
)

// Parsers dos arquivos de configuração de jogo que ainda não tinham leitor Go
// (issue 59): Guard.txt, LevelItem.txt, MobMerc.txt, gameconfig.txt e
// Regions.txt.
//
// Princípio: import lossless. Onde o formato é inequívoco o campo é tipado;
// onde o cabeçalho legado não bate com o arquivo (Guard.txt declara 13 campos
// por regra e traz 8), o dado vai para jsonb e a linha crua é preservada em
// `raw`, para o importador nunca perder informação por modelagem.

var (
	bracketRE  = regexp.MustCompile(`\[([^\]]*)\]`)
	identRE    = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	mercSlotRE = regexp.MustCompile(`(?i)^item_(\d+)$`)
	mercIdxRE  = regexp.MustCompile(`\[(\d+)\]`)
)

// readContentLines opens a legacy content file and yields its lines with CRLF
// stripped. The files are ASCII/Windows-1252 config, so the text is kept
// byte-for-byte apart from the line terminator.
func readContentLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("ler %s: %w", path, err)
	}
	defer f.Close()

	var lines []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r")
		line = strings.TrimPrefix(line, "\ufeff")
		lines = append(lines, line)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("ler %s: %w", path, err)
	}
	return lines, nil
}

// isComment reports the two comment styles the legacy configs use: "#" and
// "//".
func isComment(line string) bool {
	t := strings.TrimSpace(line)
	return strings.HasPrefix(t, "#") || strings.HasPrefix(t, "//")
}

// LoadGuardSpawns parses Guard.txt. Each data line is
// "<guard index>,<generator>[a,b,c,d,e,f,g,h][...]" — the bracketed groups are
// kept as-is because the legacy header describes 13 fields and the file carries
// 8, so typed columns would be a guess.
func LoadGuardSpawns(path string) ([]domain.GuardSpawnEntry, error) {
	lines, err := readContentLines(path)
	if err != nil {
		return nil, err
	}
	var out []domain.GuardSpawnEntry
	for i, line := range lines {
		if strings.TrimSpace(line) == "" || isComment(line) {
			continue
		}
		open := strings.IndexByte(line, '[')
		if open < 0 {
			return nil, fmt.Errorf("%s:%d: linha sem bloco de regras", path, i+1)
		}
		head := strings.Split(line[:open], ",")
		if len(head) < 2 {
			return nil, fmt.Errorf("%s:%d: cabeçalho esperado \"indice,gerador\"", path, i+1)
		}
		index, err := parseInt16(head[0])
		if err != nil {
			return nil, fmt.Errorf("%s:%d: índice: %w", path, i+1, err)
		}
		gen, err := parseInt32(head[1])
		if err != nil {
			return nil, fmt.Errorf("%s:%d: gerador: %w", path, i+1, err)
		}
		var rules [][]int16
		for _, m := range bracketRE.FindAllStringSubmatch(line, -1) {
			var group []int16
			for _, tok := range strings.Split(m[1], ",") {
				tok = strings.TrimSpace(tok)
				if tok == "" {
					continue
				}
				v, err := parseInt16(tok)
				if err != nil {
					return nil, fmt.Errorf("%s:%d: regra: %w", path, i+1, err)
				}
				group = append(group, v)
			}
			rules = append(rules, group)
		}
		out = append(out, domain.GuardSpawnEntry{
			Index:          index,
			GeneratorIndex: gen,
			Rules:          rules,
			Raw:            strings.TrimSpace(line),
		})
	}
	return out, nil
}

// LoadLevelItems parses LevelItem.txt: "<seq> <level> <class> <type> <item>
// [extras...]". Comment lines and the header are skipped; a line that is not
// fully numeric is reported instead of silently dropped.
func LoadLevelItems(path string) ([]domain.LevelItemEntry, error) {
	lines, err := readContentLines(path)
	if err != nil {
		return nil, err
	}
	var out []domain.LevelItemEntry
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || isComment(line) {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) < 5 {
			continue // texto de cabeçalho ("#ID LEVEL CLASSE TYPE ITEM")
		}
		nums := make([]int16, 0, len(fields))
		ok := true
		for _, f := range fields {
			v, err := strconv.ParseInt(f, 10, 16)
			if err != nil {
				ok = false
				break
			}
			nums = append(nums, int16(v))
		}
		if !ok {
			return nil, fmt.Errorf("%s:%d: linha não numérica: %q", path, i+1, trimmed)
		}
		out = append(out, domain.LevelItemEntry{
			Ordinal:   int16(len(out)),
			Seq:       nums[0],
			Level:     nums[1],
			Class:     nums[2],
			Type:      nums[3],
			ItemIndex: int32(nums[4]),
			Extra:     nums[5:],
			Raw:       trimmed,
		})
	}
	return out, nil
}

// LoadMobMercs parses MobMerc.txt blocks: a "#<n>" header followed by
// key:value lines (MobName, GenerateIndex, Item_<slot>, RenewTime, RebornTime).
// Item values keep every numeric token after the item index; "-" is a literal
// separator in the file, not a value.
func LoadMobMercs(path string) ([]domain.MobMercEntry, error) {
	lines, err := readContentLines(path)
	if err != nil {
		return nil, err
	}
	var out []domain.MobMercEntry
	var cur *domain.MobMercEntry
	flush := func() {
		if cur != nil {
			out = append(out, *cur)
			cur = nil
		}
	}
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			flush()
			cur = &domain.MobMercEntry{}
			if m := mercIdxRE.FindStringSubmatch(trimmed); m != nil {
				block, _ := strconv.ParseInt(m[1], 10, 32)
				cur.TemplateName = fmt.Sprintf("block-%d", block)
			}
			cur.Raw = trimmed
			continue
		}
		if cur == nil || isComment(line) {
			continue
		}
		key, value, found := strings.Cut(trimmed, ":")
		if !found {
			return nil, fmt.Errorf("%s:%d: esperado \"chave: valor\": %q", path, i+1, trimmed)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		switch {
		case strings.EqualFold(key, "MobName"):
			cur.TemplateName = value
		case strings.EqualFold(key, "GenerateIndex"):
			v, err := parseInt32(value)
			if err != nil {
				return nil, fmt.Errorf("%s:%d: GenerateIndex: %w", path, i+1, err)
			}
			cur.GeneratorIndex = v
		case strings.EqualFold(key, "RenewTime"):
			v, err := parseInt32(value)
			if err != nil {
				return nil, fmt.Errorf("%s:%d: RenewTime: %w", path, i+1, err)
			}
			cur.RenewSeconds = v
		case strings.EqualFold(key, "RebornTime"):
			v, err := parseInt32(value)
			if err != nil {
				return nil, fmt.Errorf("%s:%d: RebornTime: %w", path, i+1, err)
			}
			cur.RebornSeconds = v
		case mercSlotRE.MatchString(key):
			slot, _ := strconv.ParseInt(mercSlotRE.FindStringSubmatch(key)[1], 10, 16)
			var vals []int16
			for _, tok := range strings.Fields(value) {
				if tok == "-" {
					continue
				}
				v, err := parseInt16(tok)
				if err != nil {
					return nil, fmt.Errorf("%s:%d: %s: %w", path, i+1, key, err)
				}
				vals = append(vals, v)
			}
			cur.Items = append(cur.Items, domain.MobMercItem{Slot: int16(slot), Values: vals})
		}
	}
	flush()
	return out, nil
}

// GameConfig is the parsed gameconfig.txt: settings, the drop-bonus matrix and
// the sections kept verbatim.
type GameConfig struct {
	Settings []domain.GameSetting
	Matrix   []domain.GameDropBonus
}

// LoadGameConfig parses gameconfig.txt. The file is six labelled sections:
// three of "key value..." pairs, one "Item Drop Bonus" matrix and one "Treasure"
// block kept as raw text. The matrix measured 8 rows of 16 values, so it is
// stored with explicit row/col indices instead of assuming the 8x8 the issue
// predicted.
func LoadGameConfig(path string) (GameConfig, error) {
	lines, err := readContentLines(path)
	if err != nil {
		return GameConfig{}, err
	}

	const (
		sectionMatrix   = "Item Drop Bonus Settings"
		sectionTreasure = "Treasure Settings"
	)

	var cfg GameConfig
	section := ""
	matrixRow := int16(0)
	var rawLines []string

	flushRaw := func() {
		if section != sectionTreasure || len(rawLines) == 0 {
			return
		}
		cfg.Settings = append(cfg.Settings, domain.GameSetting{
			Key:       "treasure",
			Section:   section,
			Kind:      "raw",
			ValueText: strings.Join(rawLines, "\n"),
		})
		rawLines = nil
	}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasSuffix(trimmed, ":") {
			// Cabeçalho de seção: "Nome da Seção:".
			flushRaw()
			section = strings.TrimSuffix(trimmed, ":")
			matrixRow = 0
			continue
		}
		if section == sectionTreasure {
			rawLines = append(rawLines, trimmed)
			continue
		}
		if section == sectionMatrix {
			fields := strings.Fields(trimmed)
			for col, f := range fields {
				v, err := parseInt16(f)
				if err != nil {
					return GameConfig{}, fmt.Errorf("%s:%d: matriz: %w", path, i+1, err)
				}
				cfg.Matrix = append(cfg.Matrix, domain.GameDropBonus{
					Row: matrixRow, Col: int16(col), Pct: v,
				})
			}
			matrixRow++
			continue
		}
		// Seção de pares: cada identificador abre uma chave e os tokens
		// seguintes são valores até o próximo identificador ("evstart 0 0 0 1").
		fields := strings.Fields(trimmed)
		for pos := 0; pos < len(fields); {
			if !identRE.MatchString(fields[pos]) {
				return GameConfig{}, fmt.Errorf("%s:%d: chave inválida %q", path, i+1, fields[pos])
			}
			key := fields[pos]
			pos++
			start := pos
			for pos < len(fields) && !identRE.MatchString(fields[pos]) {
				pos++
			}
			values := fields[start:pos]
			setting := domain.GameSetting{Key: key, Section: section}
			if len(values) == 1 {
				if v, err := strconv.ParseInt(values[0], 10, 64); err == nil {
					setting.Kind, setting.ValueInt = "int", &v
				} else {
					setting.Kind, setting.ValueText = "string", values[0]
				}
			} else if len(values) > 1 {
				setting.Kind, setting.ValueText = "list", strings.Join(values, " ")
			} else {
				return GameConfig{}, fmt.Errorf("%s:%d: chave %q sem valor", path, i+1, key)
			}
			cfg.Settings = append(cfg.Settings, setting)
		}
	}
	flushRaw()
	return cfg, nil
}

// LoadRegions parses Regions.txt: "x1, y1, x2, y2 = Nome".
func LoadRegions(path string) ([]domain.RegionEntry, error) {
	lines, err := readContentLines(path)
	if err != nil {
		return nil, err
	}
	var out []domain.RegionEntry
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || isComment(line) {
			continue
		}
		coords, name, found := strings.Cut(trimmed, "=")
		if !found {
			return nil, fmt.Errorf("%s:%d: esperado \"x1, y1, x2, y2 = nome\": %q", path, i+1, trimmed)
		}
		fields := strings.Split(coords, ",")
		if len(fields) != 4 {
			return nil, fmt.Errorf("%s:%d: esperados 4 números, veio %d", path, i+1, len(fields))
		}
		var v [4]int32
		for j, f := range fields {
			n, err := parseInt32(strings.TrimSpace(f))
			if err != nil {
				return nil, fmt.Errorf("%s:%d: coordenada %d: %w", path, i+1, j+1, err)
			}
			v[j] = n
		}
		out = append(out, domain.RegionEntry{
			Name: strings.TrimSpace(name), X1: v[0], Y1: v[1], X2: v[2], Y2: v[3],
		})
	}
	return out, nil
}

func parseInt16(s string) (int16, error) {
	v, err := strconv.ParseInt(strings.TrimSpace(s), 10, 16)
	if err != nil {
		return 0, fmt.Errorf("inteiro inválido %q: %w", s, err)
	}
	return int16(v), nil
}

func parseInt32(s string) (int32, error) {
	v, err := strconv.ParseInt(strings.TrimSpace(s), 10, 32)
	if err != nil {
		return 0, fmt.Errorf("inteiro inválido %q: %w", s, err)
	}
	return int32(v), nil
}
