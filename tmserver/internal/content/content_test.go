package content

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// release devolve um caminho na árvore tmserver/content/ do repositório, pulando o teste
// quando o arquivo não está lá: os catálogos migraram para o Postgres e saíram do
// repositório (issue 58, fase 5), então estas fixtures precisam de cópia externa.
func release(t *testing.T, parts ...string) string {
	t.Helper()
	p := filepath.Join(append([]string{"..", "..", "..", "tmserver", "content"}, parts...)...)
	if _, err := os.Stat(p); err != nil {
		t.Skipf("motivo: conteúdo ausente — árvore tmserver/content/ não encontrada nesta árvore: %v", err)
	}
	return p
}

func TestCombineVariantRatesFromRelease(t *testing.T) {
	path := filepath.Join("..", "..", "..", "tmserver", "content", "Common", "Settings", "CompRate.txt")
	c, err := LoadCompRate(path)
	if err != nil {
		t.Skipf("motivo: conteúdo ausente — árvore tmserver/content/ não encontrada nesta árvore: %v", err)
	}
	if got := c.ChanceBase("Tiny"); got != 15 {
		t.Errorf("Tiny ChanceBase=%d, want 15", got)
	}
	if got := c.ChanceBase("Ailyn"); got != 10 {
		t.Errorf("Ailyn ChanceBase=%d, want 10", got)
	}
	if got := c.ChanceBase("Agatha"); got != 15 {
		t.Errorf("Agatha ChanceBase=%d, want 15", got)
	}
	if got := c.EhreRates()[3]; got != 40 {
		t.Errorf("Ehre Espiritual=%d, want 40", got)
	}
}

func TestLoadCompRate(t *testing.T) {
	c, err := LoadCompRate(release(t, "Common", "Settings", "CompRate.txt"))
	if err != nil {
		t.Skipf("motivo: conteúdo ausente — tmserver/content/Common/Settings/CompRate.txt não foi lido: %v", err)
	}
	cases := []struct {
		family, key string
		want        int
	}{
		{"Tiny", "ChanceBase", 15},
		{"Shany", "ChanceBase", 30},
		{"Ehre", "Espiritual", 40},
		{"Ehre", "Amunra", 10},
		{"Odin", "Item_12_Ref_8", 12},
		{"Odin", "Item_12_Secreta", 1},
	}
	for _, tt := range cases {
		got, ok := c.Rate(tt.family, tt.key)
		if !ok || got != tt.want {
			t.Errorf("Rate(%s,%s) = %d,%v, want %d", tt.family, tt.key, got, ok, tt.want)
		}
	}
	if c.Families() < 4 {
		t.Errorf("families = %d, want >= 4", c.Families())
	}
	ch := c.AnctChance()
	if ch != [3]int{2, 4, 10} {
		t.Errorf("AnctChance = %v, want [2 4 10]", ch)
	}
}

func TestLoadSancRate(t *testing.T) {
	s, err := LoadSancRate(release(t, "Common", "Settings", "SancRate.txt"))
	if err != nil {
		t.Skipf("motivo: conteúdo ausente — tmserver/content/Common/Settings/SancRate.txt não foi lido: %v", err)
	}
	cases := []struct {
		name      string
		anvil     int
		idx, want int
	}{
		{"PO 0 from file", sancAnvilOri, 0, 100},
		{"PO 3 from file", sancAnvilOri, 3, 85},
		{"PO 5 from file", sancAnvilOri, 5, 40},
		{"PL 6 from file", sancAnvilLac, 6, 80},
		{"PL 9 from file", sancAnvilLac, 9, 10},

		// O arquivo lista só PO 0..5, então a cauda mantém os defaults de Basedef.cpp.
		// Índice 6 = 100 torna o refino Ori +5→+6 grátis; um operador que queira isso
		// controlado adiciona uma linha "PO 6 <rate>" (issue #103).
		{"PO 6 falls back to the compiled default", sancAnvilOri, 6, 100},
		{"PO 7 falls back to the compiled default", sancAnvilOri, 7, 0},
		{"PL 10 falls back to the compiled default", sancAnvilLac, 10, 0},

		// As linhas de Âmago caem na linha 2. No legado elas se sobrepunham à linha 0 e
		// destruíam toda taxa PO acima (CReadFiles.cpp:157); a issue #103 corrigiu isso,
		// então PO 1 continua sendo o 100 do arquivo e NÃO o 80 do Âmago.
		{"Amago 1 on its own row", sancAnvilAma, 1, 80},
		{"Amago 11 on its own row", sancAnvilAma, 11, 5},
		{"PO 1 not clobbered by Amago", sancAnvilOri, 1, 100},
		{"PO 4 not clobbered by Amago", sancAnvilOri, 4, 70},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := s.Rate(tt.anvil, tt.idx); got != tt.want {
				t.Errorf("Rate(%d,%d) = %d, want %d", tt.anvil, tt.idx, got, tt.want)
			}
		})
	}
}

func TestQuestRateDefaults(t *testing.T) {
	q := DefaultQuestRates()
	want := []QuestTierRate{
		{1000, 500, 2000, 39, 115, 39, 115},
		{2000, 1000, 4000, 115, 190, 115, 190},
		{3000, 1500, 6000, 190, 265, 190, 265},
		{4000, 2000, 8000, 265, 320, 265, 320},
		{5000, 2500, 10000, 320, 350, 320, 350},
	}
	for i := range want {
		if got, ok := q.Tier(i); !ok || got != want[i] {
			t.Errorf("Tier(%d) = %+v,%v, want %+v,true", i, got, ok, want[i])
		}
	}
	if _, ok := q.Tier(-1); ok {
		t.Error("Tier(-1) succeeded")
	}
}

func TestLoadQuestRates(t *testing.T) {
	q, err := LoadQuestRates(release(t, "Common", "Settings", "QuestsRate.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := q.Tier(2); got != (QuestTierRate{200000, 200000, 100000, 190, 265, 190, 265}) {
		t.Errorf("Tier(2) = %+v", got)
	}
}

func TestParseQuestRatesOverlayAndValidation(t *testing.T) {
	q, err := parseQuestRates(strings.NewReader("exp 0 30 20\nCoin 0 40\nLEVEL 0 1 2 3 4\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := q.Tier(0); got != (QuestTierRate{30, 20, 40, 1, 2, 3, 4}) {
		t.Errorf("Tier(0) = %+v", got)
	}
	if got, _ := q.Tier(1); got != questRateDefaults[1] {
		t.Errorf("missing Tier(1) override changed default: %+v", got)
	}
	bad := []string{"EXP 5 1 1", "EXP 0 nope 1", "COIN 0 -1", "LEVEL 0 4 4 1 2", "LEVEL 0 0 400 1 2", "wat 0 1"}
	for _, input := range bad {
		if _, err := parseQuestRates(strings.NewReader(input)); err == nil {
			t.Errorf("parseQuestRates(%q) succeeded", input)
		}
	}
}

func TestSancRateOutOfRange(t *testing.T) {
	s := DefaultSancRate()
	for _, tt := range []struct{ anvil, idx int }{{-1, 0}, {3, 0}, {0, -1}, {0, 12}} {
		if got := s.Rate(tt.anvil, tt.idx); got != 0 {
			t.Errorf("Rate(%d,%d) = %d, want 0", tt.anvil, tt.idx, got)
		}
	}
	var nilTable *SancRate
	if got := nilTable.Rate(0, 1); got != 0 {
		t.Errorf("nil Rate = %d, want 0", got)
	}
}

// O token de bigorna é comparado em bytes cp1252 crus, como _strupr+strcmp faz.
func TestSancAnvilRow(t *testing.T) {
	cases := []struct {
		token string
		want  int
		ok    bool
	}{
		{"PO", sancAnvilOri, true},
		{"po", sancAnvilOri, true},
		{"PL", sancAnvilLac, true},
		{"\xC2mago", sancAnvilAma, true}, // cp1252, como o arquivo real o escreve
		{"\xC2MAGO", sancAnvilAma, true},
		{"\xC3\x82mago", sancAnvilAma, true}, // UTF-8, caso o arquivo seja regravado algum dia
		{"Ori", 0, false},
		{"", 0, false},
	}
	for _, tt := range cases {
		got, ok := sancAnvilRow(tt.token)
		if ok != tt.ok || (ok && got != tt.want) {
			t.Errorf("sancAnvilRow(%q) = %d,%v, want %d,%v", tt.token, got, ok, tt.want, tt.ok)
		}
	}
}

func TestLoadItemList(t *testing.T) {
	l, err := LoadItemList(release(t, "Common", "ItemList.csv"))
	if err != nil {
		t.Skipf("motivo: conteúdo ausente — ItemList.csv saiu do repositório (o catálogo vive no banco): o importador só roda sobre árvore externa: %v", err)
	}
	if e, ok := l.Get(1); !ok || e.Name != "TransKnight" {
		t.Errorf("Get(1) = %+v, want name TransKnight", e)
	}
	if l.Len() < 1000 {
		t.Errorf("item count = %d, want >= 1000", l.Len())
	}
}

func TestBaseEffects(t *testing.T) {
	// Uma linha de boot (formato real) com EF_AC e EF_DAMAGE entre efeitos ignorados.
	const row = "168,Botas_de_Guarda(Az),17.0,0.0.0.0.0,0,0,32,0,0," +
		"EF_CLASS,1,EF_GRID,0,EF_AC,96,EF_RUNSPEED,2,EF_REGENMP,40,EF_DAMAGE,24,EF_ITEMLEVEL,5"
	l, err := parseItemList(strings.NewReader(row))
	if err != nil {
		t.Fatal(err)
	}
	eff := l.BaseEffects()[168]
	got := map[uint8]int16{}
	for _, e := range eff {
		got[e.Eff] = e.Val
	}
	// Efeitos relevantes ao score mais EF_ITEMLEVEL (usado pelo casamento de combine) e
	// EF_RUNSPEED (issue #64: bônus de velocidade de movimento das botas).
	if len(got) != 4 || got[3] != 96 || got[2] != 24 || got[87] != 5 || got[29] != 2 {
		t.Errorf("BaseEffects = %v, want AC(3):96, DAMAGE(2):24, ITEMLEVEL(87):5, RUNSPEED(29):2", got)
	}
}

func TestBaseEffectsSpecialAll(t *testing.T) {
	const row = "700,Anel_Aprendizagem,0.0,0.0.0.0.0,0,0,1024,0,0,EF_CLASS,255,EF_SPECIALALL,18"
	l, err := parseItemList(strings.NewReader(row))
	if err != nil {
		t.Fatal(err)
	}
	var got int16
	var found bool
	for _, e := range l.BaseEffects()[700] {
		if e.Eff == 74 {
			got, found = e.Val, true
		}
	}
	if !found || got != 18 {
		t.Errorf("BaseEffects()[700] EF_SPECIALALL = %d (found=%v), want 18", got, found)
	}
}

// TestBaseEffectsCritical: EF_CRITICAL/EF_CRITICAL2 são stats de score (Basedef.cpp:3209
// deriva MOB.Critical deles), não os efeitos visuais em que já foram arquivados —
// descartá-los aqui é o que deixou equipamento de crit valendo nada (issue #102).
// A maior parte do crit no catálogo vem das peças de conjunto e dos itens de corpo de classe.
func TestBaseEffectsCritical(t *testing.T) {
	// Linhas reais: uma peça de conjunto de armadura (ItemList.csv:225) e um item de corpo de classe (:1).
	const rows = "165,Armadura_de_Guarda(Az),17.0,0.0.0.0.0,0,0,4,0,0," +
		"EF_CLASS,1,EF_GRID,0,EF_AC,138,EF_CRITICAL,50,EF_HPADD,9,EF_ITEMLEVEL,5\n" +
		"1,TransKnight,0.0,0.0.0.0.0,0,0,1,0,0," +
		"EF_CLASS,1,EF_RANGE,1,EF_REGENHP,2,EF_REGENMP,2,EF_CRITICAL,10\n"
	l, err := parseItemList(strings.NewReader(rows))
	if err != nil {
		t.Fatal(err)
	}
	eff := l.BaseEffects()
	for _, tc := range []struct {
		idx  int
		want int16
	}{{165, 50}, {1, 10}} {
		var got int16
		for _, e := range eff[tc.idx] {
			if e.Eff == 42 { // EF_CRITICAL
				got = e.Val
			}
		}
		if got != tc.want {
			t.Errorf("BaseEffects()[%d] EF_CRITICAL = %d, want %d", tc.idx, got, tc.want)
		}
	}

	// O catálogo real precisa de fato carregar crit — o parser descartá-lo em silêncio é
	// o que era a issue #102. Protege contra o efeito sumir do efName de novo.
	full, err := LoadItemList(release(t, "Common", "ItemList.csv"))
	if err != nil {
		t.Skipf("motivo: conteúdo ausente — ItemList.csv saiu do repositório (o catálogo vive no banco): o importador só roda sobre árvore externa: %v", err)
	}
	var withCrit int
	for _, effs := range full.BaseEffects() {
		for _, e := range effs {
			if e.Eff == 42 || e.Eff == 71 {
				withCrit++
				break
			}
		}
	}
	if withCrit < 100 {
		t.Errorf("real catalog items carrying EF_CRITICAL = %d, want >= 100", withCrit)
	}
	// Pedra_Amunra (o amuleto da issue) equipa nos quatro slots de acessório: seu nPos
	// 0xF00 é o que dispara a promoção de refino +9 x2 em handler.itemCritical.
	if pos := full.Positions()[3464]; pos&0xF00 == 0 {
		t.Errorf("Pedra_Amunra nPos = %d, want the accessory mask 0xF00 set", pos)
	}
}

// TestBaseEffectsResist: EF_RESIST1..4/EF_RESISTALL também são stats de score (CMob.cpp:640-643
// deriva MOB.Resist[0..3] deles), não ignorados — faltar no efName é o que deixou
// todo item de imunidade/resist sem conceder nada (issue #211).
func TestBaseEffectsResist(t *testing.T) {
	// Linhas reais: Bolsa_Perfumada (ItemList.csv:1009, EF_RESISTALL) e Orb_de_Fogo
	// (ItemList.csv:1013, EF_RESIST1).
	const rows = "599,Bolsa_Perfumada,319.0,0.0.0.0.0,0,50000,1024,0,0,EF_CLASS,255,EF_RESISTALL,5\n" +
		"601,Orb_de_Fogo,63.0,37.0.0.0.0,0,30000,1024,0,0,EF_CLASS,255,EF_RESIST1,3,EF_RESIST2,0,EF_RESIST3,0,EF_RESIST4,0,EF_ITEMLEVEL,1\n"
	l, err := parseItemList(strings.NewReader(rows))
	if err != nil {
		t.Fatal(err)
	}
	eff := l.BaseEffects()
	for _, tc := range []struct {
		idx  int
		eff  uint8
		want int16
	}{{599, 54, 5}, {601, 49, 3}} {
		var got int16
		var found bool
		for _, e := range eff[tc.idx] {
			if e.Eff == tc.eff {
				got, found = e.Val, true
			}
		}
		if !found || got != tc.want {
			t.Errorf("BaseEffects()[%d] EF(%d) = %d (found=%v), want %d", tc.idx, tc.eff, got, found, tc.want)
		}
	}

	// O catálogo real precisa de fato carregar resist — o parser descartá-lo em silêncio é
	// o que era a issue #211. Protege contra os efeitos sumirem do efName de novo.
	full, err := LoadItemList(release(t, "Common", "ItemList.csv"))
	if err != nil {
		t.Skipf("motivo: conteúdo ausente — ItemList.csv saiu do repositório (o catálogo vive no banco): o importador só roda sobre árvore externa: %v", err)
	}
	var withResist int
	for _, effs := range full.BaseEffects() {
		for _, e := range effs {
			if e.Eff >= 49 && e.Eff <= 52 || e.Eff == 54 {
				withResist++
				break
			}
		}
	}
	if withResist < 20 {
		t.Errorf("real catalog items carrying EF_RESIST1..4/EF_RESISTALL = %d, want >= 20", withResist)
	}
}

// TestBaseEffectsMagic: EF_MAGIC/EF_MAGICADD também são stats de score (Basedef.cpp:3194-3195
// deriva o Magic do conjurador de BASE_GetMobAbility(EF_MAGIC)+BASE_GetMobAbility(EF_MAGICADD)),
// não ignorados — faltar no efName é o que deixou todo item "Ataque Mágico" sem conceder
// nada (issue #223).
func TestBaseEffectsMagic(t *testing.T) {
	// Linhas reais: Bracelete_de_Hecate (ItemList.csv:871, EF_MAGIC,6) e Colar_Mágico
	// (ItemList.csv:1075, EF_MAGIC,22).
	const rows = "514,Bracelete_de_Hecate,96.0,132.0.0.0.0,0,73000,256,0,0,EF_CLASS,255,EF_MAGIC,6,EF_ITEMLEVEL,2\n" +
		"642,Colar_Magico,2.6,306.0.0.0.0,0,125000,256,0,0,EF_CLASS,255,EF_MAGIC,22,EF_ITEMLEVEL,5\n"
	l, err := parseItemList(strings.NewReader(rows))
	if err != nil {
		t.Fatal(err)
	}
	eff := l.BaseEffects()
	for _, tc := range []struct {
		idx  int
		eff  uint8
		want int16
	}{{514, 60, 6}, {642, 60, 22}} {
		var got int16
		var found bool
		for _, e := range eff[tc.idx] {
			if e.Eff == tc.eff {
				got, found = e.Val, true
			}
		}
		if !found || got != tc.want {
			t.Errorf("BaseEffects()[%d] EF(%d) = %d (found=%v), want %d", tc.idx, tc.eff, got, found, tc.want)
		}
	}

	// O catálogo real precisa de fato carregar magic — o parser descartá-lo em silêncio é
	// o que era a issue #223. Protege contra o efeito sumir do efName de novo.
	full, err := LoadItemList(release(t, "Common", "ItemList.csv"))
	if err != nil {
		t.Skipf("motivo: conteúdo ausente — ItemList.csv saiu do repositório (o catálogo vive no banco): o importador só roda sobre árvore externa: %v", err)
	}
	realEffects := full.BaseEffects()
	for _, tc := range []struct {
		idx  int
		want int16
	}{
		{3582, 55},                                     // O Cajado Caotico comum mantém o valor legado.
		{3725, 70}, {3726, 70}, {3727, 70}, {3728, 70}, // Issue #281: variantes Anct rebalanceadas.
	} {
		var got int16
		for _, e := range realEffects[tc.idx] {
			if e.Eff == 60 {
				got = e.Val
				break
			}
		}
		if got != tc.want {
			t.Errorf("real catalog item %d EF_MAGIC = %d, want %d", tc.idx, got, tc.want)
		}
	}
	var withMagic int
	for _, effs := range realEffects {
		for _, e := range effs {
			if e.Eff == 60 || e.Eff == 68 {
				withMagic++
				break
			}
		}
	}
	if withMagic < 20 {
		t.Errorf("real catalog items carrying EF_MAGIC/EF_MAGICADD = %d, want >= 20", withMagic)
	}
}

// TestBaseEffectsNoSanc: pedras de quest/troféu (Almas, Pedras, Sephirot) são
// equipáveis mas não carregam EF_VOLATILE, então nada impedia o handler de refino
// com pó de plantar um par EF_SANC nelas (issue #133). O ItemList.csv
// agora as marca com EF_NOSANC, então o gate
// itemAbility(dst, efNoSanc) != 0 de handler.refineItem as rejeita.
func TestBaseEffectsNoSanc(t *testing.T) {
	full, err := LoadItemList(release(t, "Common", "ItemList.csv"))
	if err != nil {
		t.Skipf("motivo: conteúdo ausente — ItemList.csv saiu do repositório (o catálogo vive no banco): o importador só roda sobre árvore externa: %v", err)
	}
	for _, idx := range []int{1740, 1742, 1748, 1755, 1760, 1763} {
		var got int16
		for _, e := range full.BaseEffects()[idx] {
			if e.Eff == 126 { // EF_NOSANC
				got = e.Val
			}
		}
		if got == 0 {
			t.Errorf("BaseEffects()[%d] EF_NOSANC = %d, want nonzero", idx, got)
		}
	}
}

func TestRanges(t *testing.T) {
	// Linhas de modelo de mob (formato real): o item de corpo do arqueiro carrega
	// EF_RANGE,4 — a origem do alcance à distância de um mob (BASE_GetMobAbility).
	// Uma linha sem EF_RANGE precisa ficar ausente do mapa.
	const rows = "242,Ciclope_Arqueiro,6.0,0.0.0.0.0,0,0,1,0,0,EF_CLASS,21,EF_RANGE,4\n" +
		"168,Botas_de_Guarda(Az),17.0,0.0.0.0.0,0,0,32,0,0,EF_CLASS,1,EF_AC,96\n"
	l, err := parseItemList(strings.NewReader(rows))
	if err != nil {
		t.Fatal(err)
	}
	r := l.Ranges()
	if r[242] != 4 {
		t.Errorf("Ranges()[242] = %d, want 4", r[242])
	}
	if v, ok := r[168]; ok {
		t.Errorf("Ranges()[168] = %d, want absent (no EF_RANGE)", v)
	}

	// O catálogo real (quando montado) precisa concordar no modelo do arqueiro.
	full, err := LoadItemList(release(t, "Common", "ItemList.csv"))
	if err != nil {
		t.Skipf("motivo: conteúdo ausente — ItemList.csv saiu do repositório (o catálogo vive no banco): o importador só roda sobre árvore externa: %v", err)
	}
	if got := full.Ranges()[242]; got != 4 {
		t.Errorf("real ItemList Ranges()[242] = %d, want 4 (Ciclope_Arqueiro)", got)
	}
}

func TestRequirements(t *testing.T) {
	// Uma espada exigindo nível 100 + STR 50 (col 4 = ReqLvl.Str.Int.Dex.Con); mais uma
	// item sem requisito que precisa ser omitido.
	const rows = "900,Espada,0.0,100.50.0.0.0,0,0,0,0,0,EF_DAMAGE,80\n" +
		"901,Adaga,0.0,0.0.0.0.0,0,0,0,0,0,EF_DAMAGE,20"
	l, err := parseItemList(strings.NewReader(rows))
	if err != nil {
		t.Fatal(err)
	}
	reqs := l.Requirements()
	if r := reqs[900]; r.Lvl != 100 || r.Str != 50 || r.Int != 0 {
		t.Errorf("reqs[900] = %+v, want Lvl 100 Str 50", r)
	}
	if _, ok := reqs[901]; ok {
		t.Errorf("no-requirement item 901 should be omitted, got %+v", reqs[901])
	}
}

// TestRequirementsClasseD é a regressão da issue #278. Equipamento de classe D
// (EF_ITEMLEVEL 4) exigia até nível 254 enquanto seus próprios irmãos de classe D
// (Conjuradora/Aeon/Natureza) ficavam em 128-174, então toda linha em 185..260 teve seu
// nível cortado em 80 e seus atributos escalados na mesma razão. Ancorado no
// catálogo real: uma fixture sintética não pegaria o .csv sendo regerado
// a partir de uma fonte obsoleta.
func TestRequirementsClasseD(t *testing.T) {
	full, err := LoadItemList(release(t, "Common", "ItemList.csv"))
	if err != nil {
		t.Skipf("motivo: conteúdo ausente — ItemList.csv saiu do repositório (o catálogo vive no banco): o importador só roda sobre árvore externa: %v", err)
	}
	reqs := full.Requirements()
	for _, tc := range []struct {
		idx  int
		name string
		want ItemReq
	}{
		{1346, "Tunica_de_Mytril(A)", ItemReq{Lvl: 174, Str: 117, Int: 205}},
		{911, "Solaris", ItemReq{Lvl: 174, Str: 464, Dex: 140}},
		{1191, "Elmo_Anao(N)", ItemReq{Lvl: 115, Str: 146, Con: 98}},
		{1892, "Tunica_Conjuradora(A), the EF_BONUS variant", ItemReq{Lvl: 132, Str: 90, Int: 157}},
		// Issue #308: o cliente exibe ReqLvl com base 1, então 160 cru é nível 161.
		{661, "Ankh_da_Justica", ItemReq{Lvl: 160}},
		{662, "Ankh_da_Eternidade", ItemReq{Lvl: 160}},
		{663, "Ankh_da_Gloria", ItemReq{Lvl: 160}},
		// Issue #307: 100 cru permite o emblema no nível exibido 101.
		{4080, "Emblema_da_Aprendizagem", ItemReq{Lvl: 100}},
		// Intocado: abaixo da faixa reescalada, e acima dela (as capas especiais).
		{1331, "Tunica_Conjuradora(A), base set", ItemReq{Lvl: 174, Str: 119, Int: 208}},
		{1720, "Capa_da_Escuridao", ItemReq{Lvl: 399}},
	} {
		if got := reqs[tc.idx]; got != tc.want {
			t.Errorf("Requirements()[%d] (%s) = %+v, want %+v", tc.idx, tc.name, got, tc.want)
		}
	}

	// A faixa em si precisa ficar vazia: nenhum item de classe D pode exigir 181..260
	// de novo. Acima de 260 só sobrevivem os especiais preservados (capas, Pedra Amunra).
	effects := full.BaseEffects()
	for idx, req := range reqs {
		classeD := false
		for _, e := range effects[idx] {
			if e.Eff == 87 && e.Val == 4 { // EF_ITEMLEVEL 4 = classe D
				classeD = true
				break
			}
		}
		if classeD && req.Lvl >= 181 && req.Lvl <= 260 {
			t.Errorf("class D item %d requires level %d, want <= 180 (or a preserved special > 260)", idx, req.Lvl)
		}
	}
}

func TestLoadSkillData(t *testing.T) {
	s, err := LoadSkillData(release(t, "Common", "SkillData.csv"))
	if err != nil {
		t.Skipf("motivo: conteúdo ausente — SkillData.csv saiu do repositório (o catálogo vive no banco): o importador só roda sobre árvore externa: %v", err)
	}
	// Linhas golden direto do CSV (ordem do sscanf, AffectTime já ÷4).
	tests := []struct {
		index int
		want  Spell
	}{
		{0, Spell{Index: 0, SkillPoint: 24, TargetType: 3, ManaSpent: 15, Delay: 3,
			Range: 5, InstanceType: 4, InstanceValue: 5, InstanceAttribute: 4,
			Aggressive: 1, MaxTarget: 13, AffectResist: 3, Name: "Giro_da_F\xfaria"}},
		{5, Spell{Index: 5, SkillPoint: 81, ManaSpent: 53, TickType: 17, TickValue: 75,
			AffectTime: 150, MaxTarget: 1, Name: "Aura_da_Vida"}},
		// Escudo Dourado (affect 31): a cauda curta da tabela, AffectTime cru
		// 30. Fixado aqui porque o ÷8 da issue #92 truncava para 3 (issue #236);
		// o loader voltou a ser fiel ao legado e o ajuste migrou para
		// world.AffectDuration, que aplica piso aos buffs amistosos curtos.
		{85, Spell{Index: 85, SkillPoint: 90, ManaSpent: 120, Delay: 5, AffectType: 31,
			AffectValue: 150, AffectTime: 7, MaxTarget: 1, Name: "Explos\xe3o_Et\xe9rea"}},
	}
	for _, tt := range tests {
		got, ok := s.Get(tt.index)
		if !ok {
			t.Errorf("Get(%d): missing", tt.index)
			continue
		}
		if got != tt.want {
			t.Errorf("Get(%d) = %+v, want %+v", tt.index, got, tt.want)
		}
	}
	// Os índices são esparsos além da contagem de linhas: o loader legado usa a coluna 0 como chave.
	// Os nomes mantêm os bytes Latin-1 do arquivo (o charset do cliente) — sem reencodar.
	if e, ok := s.Get(200); !ok || e.Name != "Prote\xe7\xe3o_Divina" {
		t.Errorf("Get(200) = %+v, %v, want Proteção_Divina (Latin-1)", e, ok)
	}
	if s.Len() < 100 {
		t.Errorf("skill count = %d, want >= 100", s.Len())
	}
}

// TestParseSkillDataDividesAffectTime fixa a regra AffectTime ÷4 do loader numa
// linha sintética (sem depender do CSV do Release): 13 ints, as 2 strings Act
// pontuadas, 7 ints, depois o nome. A coluna 12 é o AffectTime cru.
func TestParseSkillDataDividesAffectTime(t *testing.T) {
	const row = "2,3,0,10,4,0,1,10,0,0,11,5,600,anim.a,anim.b,0,0,1,2,0,0,0,MySkill"
	s, err := parseSkillData(strings.NewReader(row))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := s.Get(2)
	if !ok {
		t.Fatal("skill 2 missing after parse")
	}
	if got.AffectTime != 150 {
		t.Errorf("AffectTime = %d, want 150 (600÷4)", got.AffectTime)
	}
	if got.AffectType != 11 || got.AffectValue != 5 {
		t.Errorf("parsed = %+v, want AffectType 11 / AffectValue 5", got)
	}
	if got.Name != "MySkill" {
		t.Errorf("Name = %q, want MySkill", got.Name)
	}
}

// TestParseSkillDataUsesLegacyAffectTimeDivisor fixa o divisor uniforme do loader
// legado. O ajuste de duração pertence a world.AffectDuration no momento da
// conjuração, então tanto a linha curta do escudo quanto uma linha idêntica carregam com ÷4.
func TestParseSkillDataUsesLegacyAffectTimeDivisor(t *testing.T) {
	const (
		shield = "85,90,0,120,5,0,0,0,0,0,31,150,30,anim.a,anim.b,0,0,0,1,0,0,0,Explosao_Eterea"
		other  = "84,90,0,120,5,0,0,0,0,0,31,150,30,anim.a,anim.b,0,0,0,1,0,0,0,Outra"
	)
	s, err := parseSkillData(strings.NewReader(shield + "\n" + other))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := s.Get(85)
	if !ok {
		t.Fatal("skill 85 missing after parse")
	}
	if got.AffectTime != 7 {
		t.Errorf("skill 85 AffectTime = %d, want 7 (30÷4, legacy divisor)", got.AffectTime)
	}
	if got, ok := s.Get(84); !ok || got.AffectTime != 7 {
		t.Errorf("skill 84 AffectTime = %d (ok=%v), want 7 (30÷4)", got.AffectTime, ok)
	}
}

func TestSkillKindAndClass(t *testing.T) {
	tests := []struct{ index, kind, class int }{
		{0, 1, 0},   // TK tree 1
		{8, 2, 0},   // TK tree 2
		{16, 3, 0},  // TK tree 3
		{25, 1, 1},  // Foema tree 1
		{95, 3, 3},  // Huntress last
		{100, 1, 4}, // Sephira range
	}
	for _, tt := range tests {
		if got := SkillKind(tt.index); got != tt.kind {
			t.Errorf("SkillKind(%d) = %d, want %d", tt.index, got, tt.kind)
		}
		if got := SkillClass(tt.index); got != tt.class {
			t.Errorf("SkillClass(%d) = %d, want %d", tt.index, got, tt.class)
		}
	}
}

func TestLoadMaps(t *testing.T) {
	attr, err := LoadGrid(release(t, "TMsrv", "run", "AttributeMap.dat"), AttributeMapDim)
	if err != nil {
		t.Skipf("motivo: conteúdo ausente — tmserver/content/TMsrv/run/AttributeMap.dat não está nesta árvore: %v", err)
	}
	if len(attr.Data) != AttributeMapDim*AttributeMapDim {
		t.Errorf("attribute map size %d", len(attr.Data))
	}

	hm, err := LoadHeightMap(release(t, "TMsrv", "run", "HeightMap.dat"))
	if err != nil {
		t.Skipf("motivo: conteúdo ausente — tmserver/content/TMsrv/run/HeightMap.dat não está nesta árvore: %v", err)
	}
	if hm.Dim != HeightMapDim || len(hm.Data) != HeightMapDim*HeightMapDim {
		t.Errorf("height map dim %d size %d", hm.Dim, len(hm.Data))
	}
}
