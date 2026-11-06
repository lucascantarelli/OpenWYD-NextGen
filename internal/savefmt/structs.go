package savefmt

import "encoding/binary"

// Effect é um dos três pares efeito/valor de um STRUCT_ITEM.
type Effect struct {
	Effect uint8
	Value  uint8
}

// Item é STRUCT_ITEM (8 bytes). sIndex==0 significa slot vazio.
type Item struct {
	Index   int16
	Effects [3]Effect
}

// Empty informa se o slot de item está vazio (sIndex==0).
func (it Item) Empty() bool { return it.Index == 0 }

// Score é STRUCT_SCORE (48 bytes) — atributos base ou atuais.
type Score struct {
	Level     int32
	AC        int32
	Damage    int32
	Merchant  uint8
	AttackRun uint8
	Direction uint8
	ChaosRate uint8
	MaxHp     int32
	MaxMp     int32
	Hp        int32
	Mp        int32
	Str       int16
	Int       int16
	Dex       int16
	Con       int16
	Special   [4]int16
}

// Affect é STRUCT_AFFECT (8 bytes) — um buff/debuff persistido e temporizado.
type Affect struct {
	Type  uint8
	Value uint8
	Level uint16
	Time  uint32
}

// Mob é STRUCT_MOB (816 bytes, alinhamento natural) — um personagem persistido.
// Jogadores e mobs compartilham esta struct em memória; aqui ela é o personagem
// em disco.
type Mob struct {
	Name         [16]byte
	Clan         uint8
	Merchant     uint8
	Guild        uint16
	Class        uint8
	Rsv          uint16
	Quest        uint8
	Coin         int32
	Exp          int64
	SPX          int16
	SPY          int16
	BaseScore    Score
	CurrentScore Score
	Equip        [MaxEquip]Item
	Carry        [MaxCarry]Item
	LearnedSkill int32 // long (4 bytes no Win32 x86)
	Magic        uint32
	ScoreBonus   uint16
	SpecialBonus uint16
	SkillBonus   uint16
	Critical     uint8
	SaveMana     uint8
	SkillBar     [4]uint8
	GuildLevel   uint8
	RegenHP      uint16
	RegenMP      uint16
	Resist       [4]int8
}

// AccountInfo é STRUCT_ACCOUNTINFO (216 bytes) — dados de nível de conta.
//
// AccountPass e NumericToken estão em TEXTO PLANO no disco — dívida de segurança
// crítica que o conversor resolve com hash na importação; a stack nova nunca os
// persiste em claro.
type AccountInfo struct {
	AccountName  [16]byte
	AccountPass  [12]byte
	RealName     [24]byte
	SSN1         int32
	SSN2         int32
	Email        [48]byte
	Telephone    [16]byte
	Address      [78]byte
	NumericToken [6]byte
	Year         int32
	YearDay      int32
}

// Quest é STRUCT_QUEST (56 bytes, alinhamento 8) — a quest diária ativa.
type Quest struct {
	IndexQuest int16
	Nivel      int16
	IDMob1     int16
	QtdMob1    int16
	IDMob2     int16
	QtdMob2    int16
	IDMob3     int16
	QtdMob3    int16
	ExpReward  int32 // long
	GoldReward int32
	Item       [2]Item
	LastTime   int64 // time_t (8 bytes; largura não verificada)
	MobCount   [3]int16
}

// MobExtra é STRUCT_MOBEXTRA (552 bytes, alinhamento 8) — dados extras por
// personagem (cidadania, fama, alma, progresso de quest, donate).
//
// O bloco escalar inicial segue o alinhamento natural do MSVC x86:
// ClassMaster@0, Citizen@2, SecLearnedSkill@4, Fame@8, Soul@12 e
// MortalFace@14. O resto do bloco de 552 bytes fica cru até que as subestruturas
// de quest e donate/NT sejam verificadas por completo.
type MobExtra struct {
	Raw [MobExtraSize]byte
}

// ClassMaster devolve o byte baixo de ClassMaster (offset 0).
func (m MobExtra) ClassMaster() uint8 { return m.Raw[0] }

// Citizen devolve o byte Citizen (offset 2).
func (m MobExtra) Citizen() uint8 { return m.Raw[2] }

// SecLearnedSkill devolve a máscara de skills secundárias aprendidas (offset 4).
func (m MobExtra) SecLearnedSkill() int32 {
	return int32(binary.LittleEndian.Uint32(m.Raw[4:8]))
}

// Fame devolve o campo Fame do MobExtra (offset 8).
func (m MobExtra) Fame() int32 { return int32(binary.LittleEndian.Uint32(m.Raw[8:12])) }

// KefraTicket devolve as entradas restantes no Salão (offset 452 do MOBEXTRA).
func (m MobExtra) KefraTicket() int32 {
	return int32(binary.LittleEndian.Uint32(m.Raw[452:456]))
}

// Soul devolve o byte de atributo alma (offset 12).
func (m MobExtra) Soul() uint8 { return m.Raw[12] }

// MortalFace devolve o campo MortalFace (offset 14).
func (m MobExtra) MortalFace() int16 {
	return int16(binary.LittleEndian.Uint16(m.Raw[14:16]))
}

// AccountFile é STRUCT_ACCOUNTFILE (7952 bytes) — o blob inteiro da conta em
// disco. Modela só o formato ATUAL; os layouts legados de 4294 / 7500–7600 não
// estão verificados (version.go).
type AccountFile struct {
	Info         AccountInfo
	Char         [MobPerAccount]Mob
	Cargo        [MaxCargo]Item
	Coin         int32
	ShortSkill   [ShortSkillBars][ShortSkillSlot]uint8
	Affect       [MobPerAccount][MaxAffect]Affect
	MobExtra     [MobPerAccount]MobExtra
	Donate       int32
	TempKey      [52]byte
	ReceivedItem bool
	QuestDaily   Quest
	BlockPass    [16]byte
	IsBlocked    bool
}

// NightmareEntries devolve NT (offset 448 do MOBEXTRA).
func (m MobExtra) NightmareEntries() int32 {
	return int32(binary.LittleEndian.Uint32(m.Raw[448:452]))
}

// LastNightmareUse devolve LastNT, um time_t do MSVC (offset 440, 8 bytes).
func (m MobExtra) LastNightmareUse() int64 {
	return int64(binary.LittleEndian.Uint64(m.Raw[440:448]))
}
