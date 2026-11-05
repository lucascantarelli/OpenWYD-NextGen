package domain

// ItemCatalogEntry is one row of the item catalog (Release/Common/ItemList.csv)
// as persisted in item_catalog (issue 58, fase 1). Fields keeps the raw CSV
// columns so the import stays lossless; the derived fields mirror what
// content.ItemList computes from them (Prices/Grades/Requirements/BaseEffects…).
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

	// Volatile is EF_VOLATILE's value (item action class); nil = not volatile.
	Volatile *int16
	// AttackRange is EF_RANGE; nil = the item grants no reach of its own.
	AttackRange *int16

	Effects []ItemCatalogEffect
	Fields  []string
	// SHA256 hashes the canonical CSV row, so a reimport can report drift
	// between the file and the database instead of silently overwriting.
	SHA256 string
}

// ItemCatalogEffect is one resolved EF_* effect of an item's base stats.
type ItemCatalogEffect struct {
	Eff uint8 `json:"eff"`
	Val int16 `json:"val"`
}

// SkillCatalogEntry is one row of skill_catalog (Release/Common/SkillData.csv).
// AffectTime is stored already divided by the legacy loader's divisor, matching
// content.Spell — the database holds the value the runtime uses.
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

// NPCTemplateEntry is one STRUCT_MOB template (Release/TMsrv/run/npc/<file>) as
// persisted in npc_template. Blob is always the canonical 816-byte layout;
// SourceSize records the on-disk size (756/816/920) for audit.
type NPCTemplateEntry struct {
	Name       string
	FileName   string
	SourceSize int16
	SHA256     string
	Blob       []byte
}

// WorldItemSeedEntry is one static world object from InitItem.csv.
type WorldItemSeedEntry struct {
	Index  int16
	X      int16
	Y      int16
	Rotate int16
}
