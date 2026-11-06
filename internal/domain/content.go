package domain

// ItemCatalogEntry é uma linha do catálogo de itens, persistida em item_catalog.
// Fields guarda as colunas cruas do CSV para a importação ficar sem perda; os
// campos derivados espelham o que content.ItemList calcula delas
// (Prices/Grades/Requirements/BaseEffects…).
type ItemCatalogEntry struct {
	Index    int
	Name     string
	Price    int32
	Grade    int16
	UniqueID int16
	Extra    int32
	Pos      int16

	ReqLevel int16
	ReqStr   int16
	ReqInt   int16
	ReqDex   int16
	ReqCon   int16

	// Volatile é o valor de EF_VOLATILE (classe de ação do item); nil = não volátil.
	Volatile *int16
	// AttackRange é EF_RANGE; nil = o item não concede alcance próprio.
	AttackRange *int16

	Effects []ItemCatalogEffect
	Fields  []string
	// SHA256 hasheia a linha canônica do CSV, para uma reimportação relatar
	// divergência entre arquivo e banco em vez de sobrescrever em silêncio.
	SHA256 string
}

// ItemCatalogEffect é um efeito EF_* resolvido dos atributos base do item.
type ItemCatalogEffect struct {
	Eff uint8 `json:"eff"`
	Val int16 `json:"val"`
}

// SkillCatalogEntry é uma linha de skill_catalog. AffectTime é gravado já
// dividido pelo divisor do loader legado, como em content.Spell — o banco guarda
// o valor que o runtime usa.
type SkillCatalogEntry struct {
	Index             int
	Name              string
	SkillPoint        int16
	TargetType        int16
	ManaSpent         int16
	Delay             int16
	Range             int16
	InstanceType      int16
	InstanceValue     int16
	TickType          int16
	TickValue         int16
	AffectType        int16
	AffectValue       int16
	AffectTime        int16
	InstanceAttribute int16
	TickAttribute     int16
	Aggressive        int16
	MaxTarget         int16
	BParty            int16
	AffectResist      int16
	Passive           int16
}

// NPCTemplateEntry é um template STRUCT_MOB persistido em npc_template. Blob é
// sempre o layout canônico de 816 bytes; SourceSize registra o tamanho em disco
// (756/816/920) para auditoria.
type NPCTemplateEntry struct {
	Name       string
	FileName   string
	SourceSize int16
	SHA256     string
	Blob       []byte
}

// WorldItemSeedEntry é um objeto estático do mundo vindo de InitItem.csv.
type WorldItemSeedEntry struct {
	Index  int16
	X      int16
	Y      int16
	Rotate int16
}
