package domain

// Configuração de jogo importada dos arquivos legados.
//
// Estes tipos carregam o arquivo inteiro para o banco sem inventar semântica:
// onde o formato é claro (região, item por nível, pares chave/valor) há campo
// tipado; onde o cabeçalho legado e o arquivo divergem (Guard.txt) o dado vai
// como jsonb + a linha crua.

// GuardSpawnEntry é uma linha de Guard.txt: o índice da guarda, o gerador de NPC
// a que se aplica e os grupos de regra entre colchetes exatamente como escritos.
type GuardSpawnEntry struct {
	Index          int16
	GeneratorIndex int32
	Rules          [][]int16
	Raw            string
}

// LevelItemEntry é uma linha de LevelItem.txt: qual item um nível/classe recebe.
// Extra guarda os números depois do item (colunas de refino/efeito que o
// cabeçalho legado não documenta).
type LevelItemEntry struct {
	// Ordinal é a posição da linha de dados no arquivo. É a chave de import:
	// o arquivo repete blocos, então (classe, sequência) não é único.
	Ordinal   int16
	Seq       int16
	Level     int16
	Class     int16
	Type      int16
	ItemIndex int32
	Extra     []int16
	Raw       string
}

// MobMercItem é uma linha Item_n de um bloco MobMerc.txt.
type MobMercItem struct {
	Slot   int16
	Values []int16
}

// MobMercEntry é um bloco MobMerc.txt: tempos de reabastecimento/renascimento e
// estoque.
type MobMercEntry struct {
	GeneratorIndex int32
	TemplateName   string
	RenewSeconds   int32
	RebornSeconds  int32
	Items          []MobMercItem
	Raw            string
}

// GameSetting é um par chave/valor de gameconfig.txt. Kind é "int" para um único
// valor numérico, "list" para vários números, "string" para texto livre e "raw"
// para uma seção inteira guardada como está (Treasure).
type GameSetting struct {
	Key       string
	Section   string
	Kind      string
	ValueInt  *int64
	ValueText string
}

// GameDropBonus é uma célula da matriz "Item Drop Bonus".
type GameDropBonus struct {
	Row int16
	Col int16
	Pct int16
}

// RegionEntry é um retângulo de Regions.txt.
type RegionEntry struct {
	Name string
	X1   int32
	Y1   int32
	X2   int32
	Y2   int32
}

// ContentSourceFile é o registro de proveniência da importação de um arquivo de
// conteúdo.
type ContentSourceFile struct {
	Path      string
	SHA256    string
	SizeBytes int64
	Rows      int32
}
