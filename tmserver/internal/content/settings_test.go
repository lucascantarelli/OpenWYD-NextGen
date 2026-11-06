package content

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"openwyd-nextgen/internal/domain"
)

// Fixtures inline cobrem os detalhes de formato que já quebraram importadores
// antes: CRLF, comentários, espaçamento irregular e listas de tamanho variável.

func writeFixture(t *testing.T, name, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("escrever fixture: %v", err)
	}
	return path
}

func TestLoadRegionsAcceptsLooseSpacing(t *testing.T) {
	path := writeFixture(t, "Regions.txt", "# comentário\r\n1272, 1427,1403, 1532 = Big_Cubo\r\n\r\n3199, 1667, 3321, 1785 = Kefra_City\r\n")
	got, err := LoadRegions(path)
	if err != nil {
		t.Fatalf("LoadRegions: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("regiões = %d, want 2", len(got))
	}
	want := struct{ x1, y1, x2, y2 int32 }{1272, 1427, 1403, 1532}
	if r := got[0]; r.Name != "Big_Cubo" || r.X1 != want.x1 || r.Y1 != want.y1 || r.X2 != want.x2 || r.Y2 != want.y2 {
		t.Errorf("região[0] = %+v, want Big_Cubo %+v", r, want)
	}
}

func TestLoadLevelItemsSkipsCommentsAndHeader(t *testing.T) {
	path := writeFixture(t, "LevelItem.txt", strings.Join([]string{
		"# SEQUENCIA, LEVEL, CLASSE, TYPE, ITEM",
		"#ID LEVEL CLASSE TYPE ITEM",
		"",
		"0   29    0       1   1147 43 3 2 20 42 40",
		"1   34    2       2   1159 43 3 0 0 0 0",
	}, "\r\n"))
	got, err := LoadLevelItems(path)
	if err != nil {
		t.Fatalf("LoadLevelItems: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("linhas = %d, want 2", len(got))
	}
	if got[0].Ordinal != 0 || got[1].Ordinal != 1 {
		t.Errorf("ordinais = %d,%d, want 0,1", got[0].Ordinal, got[1].Ordinal)
	}
	if got[0].Seq != 0 || got[0].Level != 29 || got[0].Class != 0 || got[0].Type != 1 || got[0].ItemIndex != 1147 {
		t.Errorf("linha[0] = %+v", got[0])
	}
	if len(got[0].Extra) != 6 {
		t.Errorf("extras = %v, want 6 números", got[0].Extra)
	}
}

func TestLoadGuardSpawnsKeepsEveryRuleGroup(t *testing.T) {
	path := writeFixture(t, "Guard.txt", "#\r\n# NPC-Index, Req0 ...\r\n\r\n"+
		"0,5617[412,61,4,0,0,0,0,76][413,61,2,0,0,0,0,65]\r\n")
	got, err := LoadGuardSpawns(path)
	if err != nil {
		t.Fatalf("LoadGuardSpawns: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("guardas = %d, want 1", len(got))
	}
	if got[0].Index != 0 || got[0].GeneratorIndex != 5617 {
		t.Errorf("cabeçalho = %d/%d, want 0/5617", got[0].Index, got[0].GeneratorIndex)
	}
	if len(got[0].Rules) != 2 {
		t.Fatalf("grupos = %d, want 2", len(got[0].Rules))
	}
	if len(got[0].Rules[0]) != 8 || got[0].Rules[0][0] != 412 || got[0].Rules[0][7] != 76 {
		t.Errorf("grupo[0] = %v", got[0].Rules[0])
	}
}

func TestLoadMobMercsParsesBlock(t *testing.T) {
	path := writeFixture(t, "MobMerc.txt", strings.Join([]string{
		"// Merc Teste",
		"#\t[   0]",
		"\tMobName:\tMerc_Zakum",
		"\tGenerateIndex:\t6500",
		"\tItem_0:\t\t5338 0 0 0 0 0 0 - 5",
		"\tRenewTime:\t86400",
		"\tRebornTime:\t86400",
	}, "\r\n"))
	got, err := LoadMobMercs(path)
	if err != nil {
		t.Fatalf("LoadMobMercs: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("mercadores = %d, want 1", len(got))
	}
	m := got[0]
	if m.TemplateName != "Merc_Zakum" || m.GeneratorIndex != 6500 || m.RenewSeconds != 86400 || m.RebornSeconds != 86400 {
		t.Errorf("bloco = %+v", m)
	}
	if len(m.Items) != 1 || m.Items[0].Slot != 0 || m.Items[0].Values[0] != 5338 {
		t.Errorf("itens = %+v", m.Items)
	}
}

func TestLoadGameConfigParsesPairsListMatrixAndRaw(t *testing.T) {
	path := writeFixture(t, "gameconfig.txt", strings.Join([]string{
		"Drop Item Event Settings:",
		"   evindex 0   evdelete 0   evstart 0 0 0 1",
		"Etc Event Settings:",
		"   double 200   deadpoint 1",
		"Item Drop Bonus Settings:",
		"    50  50",
		"    60  70",
		"Treasure Settings:",
		"0      0     0",
		"       0",
	}, "\r\n"))
	cfg, err := LoadGameConfig(path)
	if err != nil {
		t.Fatalf("LoadGameConfig: %v", err)
	}
	byKey := make(map[string]domain.GameSetting, len(cfg.Settings))
	for _, s := range cfg.Settings {
		byKey[s.Key] = s
	}
	if got := byKey["double"]; got.Kind != "int" || got.ValueInt == nil || *got.ValueInt != 200 {
		t.Errorf("double = %+v, want int 200", got)
	}
	if got := byKey["evstart"]; got.Kind != "list" || got.ValueText != "0 0 0 1" {
		t.Errorf("evstart = %+v, want list \"0 0 0 1\"", got)
	}
	if got, ok := byKey["treasure"]; !ok || got.Kind != "raw" || !strings.Contains(got.ValueText, "0      0     0") {
		t.Errorf("treasure = %+v (existe=%v), want raw com as linhas", got, ok)
	}
	if len(cfg.Matrix) != 4 {
		t.Fatalf("células da matriz = %d, want 4", len(cfg.Matrix))
	}
	if cfg.Matrix[3].Row != 1 || cfg.Matrix[3].Col != 1 || cfg.Matrix[3].Pct != 70 {
		t.Errorf("matriz[3] = %+v, want 1/1/70", cfg.Matrix[3])
	}
}

// TestContentSettingsParityWithRealFiles is the golden check the importer relies
// on: the parsers must read the shipped tree without dropping or inventing rows.
func TestContentSettingsParityWithRealFiles(t *testing.T) {
	root := filepath.Join("..", "..", "..", "tmserver", "content")
	if _, err := os.Stat(root); err != nil {
		t.Skip("motivo: árvore tmserver/content ausente neste checkout")
	}

	guards, err := LoadGuardSpawns(filepath.Join(root, "TMsrv", "run", "Guard.txt"))
	if err != nil {
		t.Fatalf("Guard.txt: %v", err)
	}
	if len(guards) != 5 {
		t.Errorf("guardas = %d, want 5", len(guards))
	}

	items, err := LoadLevelItems(filepath.Join(root, "TMsrv", "run", "LevelItem.txt"))
	if err != nil {
		t.Fatalf("LevelItem.txt: %v", err)
	}
	if len(items) != 318 {
		t.Errorf("itens por nível = %d, want 318", len(items))
	}
	seenSeq := map[[2]int16]bool{}
	dupes := 0
	for i, it := range items {
		if int(it.Ordinal) != i {
			t.Fatalf("ordinal %d na posição %d", it.Ordinal, i)
		}
		key := [2]int16{it.Class, it.Seq}
		if seenSeq[key] {
			dupes++
		}
		seenSeq[key] = true
	}
	if want := len(items) - len(seenSeq); dupes != want {
		t.Errorf("duplicatas = %d, want %d", dupes, want)
	}
	if dupes == 0 {
		t.Log("nota: o arquivo não tem mais blocos repetidos; o ordinal continua sendo a chave estável")
	}

	mercs, err := LoadMobMercs(filepath.Join(root, "Common", "Settings", "MobMerc.txt"))
	if err != nil {
		t.Fatalf("MobMerc.txt: %v", err)
	}
	if len(mercs) != 1 || mercs[0].GeneratorIndex != 6500 {
		t.Errorf("mercadores = %+v, want 1 bloco com gerador 6500", mercs)
	}

	cfg, err := LoadGameConfig(filepath.Join(root, "TMsrv", "run", "gameconfig.txt"))
	if err != nil {
		t.Fatalf("gameconfig.txt: %v", err)
	}
	if len(cfg.Settings) != 26 {
		t.Errorf("chaves do gameconfig = %d, want 26", len(cfg.Settings))
	}
	if len(cfg.Matrix) != 64 {
		t.Errorf("células da matriz = %d, want 64 (4x16 medido)", len(cfg.Matrix))
	}
	rows, cols := 0, 0
	for _, c := range cfg.Matrix {
		if int(c.Row)+1 > rows {
			rows = int(c.Row) + 1
		}
		if int(c.Col)+1 > cols {
			cols = int(c.Col) + 1
		}
	}
	if rows != 4 || cols != 16 {
		t.Errorf("matriz = %dx%d, want 4x16", rows, cols)
	}

	regions, err := LoadRegions(filepath.Join(root, "TMsrv", "run", "Regions.txt"))
	if err != nil {
		t.Fatalf("Regions.txt: %v", err)
	}
	if len(regions) != 52 {
		t.Errorf("regiões = %d, want 52", len(regions))
	}
}
