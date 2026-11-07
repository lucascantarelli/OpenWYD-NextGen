package handler

import (
	"openwyd-nextgen/tmserver/internal/level"
	"openwyd-nextgen/tmserver/internal/world"
)

const (
	classMasterArch        = 1
	classMasterMortal      = 2
	classMasterCelestial   = 3
	classMasterCelestialCS = 4
	classMasterSCelestial  = 5
)

const (
	baseACMortal         int32 = 4
	baseACArch           int32 = 230
	baseDamageChar       int32 = 5
	baseDamageMortalArch       = baseDamageChar
	baseDamageCelestial  int32 = 0
)

// scoreBonusFor é BASE_GetBonusScorePoint por tier (Basedef.cpp:898-1172) sobre
// o BaseScore sem equipamento: os tiers celestiais contam suas concessões de quest,
// incluindo os +200 por reset de Pedra da Fúria (issue #408), e CELESTIALCS/
// SCELESTIAL somam o nível do tier estacionado (issue #410); todo outro tier mantém
// a fórmula MORTAL.
func scoreBonusFor(e *world.Entity) uint16 {
	if isCelestialTier(e.ClassMaster) {
		var parked int32
		switch e.ClassMaster {
		case classMasterCelestialCS:
			parked = int32(e.SubCelestialLevel)
		case classMasterSCelestial:
			parked = int32(e.CelestialLevel)
		}
		return uint16(level.ScoreBonusCelestial(e.Class, e.Level, e.BaseStr, e.BaseInt, e.BaseDex, e.BaseCon,
			e.ArchCrystalStage, e.CelReset, e.CelestialArchLevel, parked))
	}
	return uint16(level.ScoreBonus(e.Class, e.Level, e.BaseStr, e.BaseInt, e.BaseDex, e.BaseCon))
}

// playerBaseDamage reconstrói o BaseScore.Damage sem equipamento omitido por
// api/db/v1.Character. Personagens Mortal e Arch herdam 5 dos quatro
// templates BaseMob (DBSrv/CFileDB.cpp:981-993,1552-1577); a Pedra Ideal
// de conversão o reseta explicitamente para 0 (_MSG_UseItem.cpp:3137), o que também
// vale para os tiers Celestial seguintes.
func playerBaseDamage(e *world.Entity) int32 {
	switch e.ClassMaster {
	case classMasterCelestial, classMasterCelestialCS, classMasterSCelestial:
		return baseDamageCelestial
	default:
		// ClassMaster 0 é o valor de compatibilidade pré-persistência e é tratado
		// como Mortal por completeCharacterLogin.
		return baseDamageMortalArch
	}
}

// playerBaseAC reproduz BaseScore.Ac sem persistir. No legado o
// campo só é escrito na criação (a linha de base por tier acima) e na
// subida de nível, +1 por nível em todo tier (CMob.cpp:1133,1145,1150) — então é uma
// função pura de (tier, nível). Todo tier começa no nível 0 com sua
// linha de base: a criação copia o template BaseMob (BaseScore.Level 0, Ac 4) e
// a virada Celestial define Level = 0 com Ac = 230 (_MSG_UseItem.cpp:3124-3135 +
// handler.useIdealStone), então "linha de base + nível" vale nos três caminhos.
//
// Esta derivação existe porque o contrato do banco não carrega AC: api/db/v1
// Character não tem campo `ac`, então um login sempre lê CurrentScore.Ac == 0 e
// a antiga subtração "base = atual − equipamento" produzia uma base NEGATIVA —
// a defesa lia ~0 com equipamento e ficava negativa ao desequipar (issue #232).
//
// O AC de Cristal é reconstruído do estágio de quest persistido só para Archs:
// a Pedra Ideal reseta o AC e não pode reaplicar bônus históricos de Arch.
func playerBaseAC(e *world.Entity) int32 {
	base := baseACArch
	// ClassMaster == 0 significa "nunca persistido" e é tratado como MORTAL, a mesma
	// convenção que completeCharacterLogin aplica antes disto rodar (character.go).
	if e.ClassMaster == classMasterMortal || e.ClassMaster == 0 {
		base = baseACMortal
	}
	if e.ClassMaster == classMasterArch {
		if e.ArchCrystalStage >= 2 {
			base += 30
		}
		if e.ArchCrystalStage == 4 {
			base += 20
		}
	}
	return base + max(e.Level, 0)
}

type weaponCoef struct {
	nUnique int
	dexK    float64
	strK    float64
}

var tkWeaponCoefs = []weaponCoef{
	{43, 0.38, 0.42},
	{42, 0.55, 0.60},
	{46, 0.36, 0.40},
	{48, 0.56, 0.60},
	{49, 0.43, 0.47},
	{45, 0.22, 0.27},
	{41, 0.22, 0.27},
}

var fmWeaponCoefs = []weaponCoef{
	{43, 0.50, 0.55},
	{42, 0.75, 0.80},
	{46, 0.40, 0.45},
	{48, 0.50, 0.55},
	{49, 0.60, 0.65},
	{45, 0.55, 0.60},
	{41, 0.50, 0.55},
}

var bmWeaponCoefs = []weaponCoef{
	{43, 0.42, 0.46},
	{42, 0.68, 0.72},
	{46, 0.44, 0.48},
	{48, 0.41, 0.46},
	{49, 0.65, 0.70},
	{45, 0.22, 0.32},
	{41, 0.22, 0.32},
}

var htWeaponCoefs = []weaponCoef{
	{43, 0.49, 0.53},
	{42, 0.68, 0.72},
	{46, 0.53, 0.57},
	{48, 0.51, 0.55},
	{49, 0.47, 0.51},
	{45, 0.27, 0.37},
	{41, 0.27, 0.37},
}

// magicWeaponCoef é o termo mágico de Dex/Int que o legado soma por arma equipada
// (Basedef.cpp:3294-3844). byPos seleciona contra qual campo da linha do catálogo a
// chave nUnique/nPos é casada: o original testa nUnique e nPos em
// ifs SEPARADOS, então um Cajado com nUnique 47 E nPos 64 coleta OS DOIS termos.
type magicWeaponCoef struct {
	key   int
	byPos bool // casa d.itemPos (ArmaPos) em vez de d.itemUnique (ArmaUnique)
	dexK  float64
	intK  float64
}

// Os coeficientes são as constantes literais do .cpp (divididas por 100 no uso) para cada linha
// poder ser comparada direto com o Basedef.cpp. 44 = Lança, 47 = Cajado e Cetro,
// nPos 64 = Cajado. O bloco da Huntress não tem ramo nPos 64 no original.
var tkMagicCoefs = []magicWeaponCoef{
	{key: 44, dexK: 2.0, intK: 2.5},
	{key: 47, dexK: 1.2, intK: 1.5},
	{key: 64, byPos: true, dexK: 2.0, intK: 2.5},
}

var fmMagicCoefs = []magicWeaponCoef{
	{key: 44, dexK: 6.3, intK: 4.2},
	{key: 47, dexK: 6.3, intK: 4.2},
	{key: 64, byPos: true, dexK: 6.3, intK: 4.2},
}

var bmMagicCoefs = []magicWeaponCoef{
	{key: 44, dexK: 7.0, intK: 6.0},
	{key: 47, dexK: 6.8, intK: 5.8},
	{key: 64, byPos: true, dexK: 5.5, intK: 4.5},
}

var htMagicCoefs = []magicWeaponCoef{
	{key: 44, dexK: 40.0, intK: 20.0},
	{key: 47, dexK: 40.0, intK: 20.0},
}

func isPlayerMob(e *world.Entity) bool {
	if !world.IsPlayer(e.ID) {
		return false
	}
	if e.Equip[0].Empty() {
		return e.Class < 4
	}
	return e.Equip[0].Index/10 < 4
}

func dexStrWeaponBonus(str, dex int16, dexK, strK float64) int32 {
	return int32(float64(dex)*dexK + float64(str)*strK)
}

func weaponTableBonus(str, dex int16, nUnique int, table []weaponCoef) int32 {
	for _, w := range table {
		if w.nUnique == nUnique {
			return dexStrWeaponBonus(str, dex, w.dexK, w.strK)
		}
	}
	return 0
}

func classSkillBits(cls uint8) []uint {
	switch cls {
	case 0, 1, 2:
		return []uint{7, 15, 23}
	case 3:
		return []uint{2}
	default:
		return nil
	}
}

func classWeaponTable(cls uint8) []weaponCoef {
	switch cls {
	case 0:
		return tkWeaponCoefs
	case 1:
		return fmWeaponCoefs
	case 2:
		return bmWeaponCoefs
	case 3:
		return htWeaponCoefs
	default:
		return nil
	}
}

func (d *Dispatcher) classWeaponDamage(e *world.Entity) int32 {
	if !isPlayerMob(e) {
		return 0
	}
	weapon := e.Equip[weaponSlotR]
	if weapon.Empty() {
		return 0
	}
	nUnique := d.itemUnique[int(weapon.Index)]
	table := classWeaponTable(e.Class)
	if table == nil {
		return 0
	}
	var total int32
	for _, bit := range classSkillBits(e.Class) {
		if e.LearnedSkill&(1<<bit) == 0 {
			continue
		}
		total += weaponTableBonus(e.Str, e.Dex, nUnique, table)
	}
	return total
}

func classMagicTable(cls uint8) []magicWeaponCoef {
	switch cls {
	case 0:
		return tkMagicCoefs
	case 1:
		return fmMagicCoefs
	case 2:
		return bmMagicCoefs
	case 3:
		return htMagicCoefs
	default:
		return nil
	}
}

// classWeaponMagic deriva a magia concedida pela categoria da arma equipada.
// Basedef.cpp:3294-3844 repete este termo em todo bloco de evolução aprendida, o que
// pode triplicar a perda quando um conjurador desequipa uma arma. A issue #280 limita
// esse comportamento legado a uma concessão, assim que qualquer evolução de classe é aprendida.
// Os casamentos separados de nUnique e nPos continuam aditivos, já que descrevem propriedades
// distintas da arma. Dex/Int são lidos pós-equipamento (CurrentScore), e é por isso
// que refreshScore chama isto depois de atribuir e.Dex/e.Int.
//
// O truncamento casa com o original: `magic` é int e o lado direito um
// double, então cada soma trunca. Como `magic` já é inteiro, isso é
// o mesmo que truncar cada termo isoladamente.
func (d *Dispatcher) classWeaponMagic(e *world.Entity) int32 {
	if !isPlayerMob(e) {
		return 0
	}
	weapon := e.Equip[weaponSlotR]
	if weapon.Empty() {
		return 0
	}
	table := classMagicTable(e.Class)
	if table == nil {
		return 0
	}
	nUnique := d.itemUnique[int(weapon.Index)]
	nPos := d.itemPos[int(weapon.Index)]

	var weaponBonus int32
	for _, w := range table {
		key := nUnique
		if w.byPos {
			key = nPos
		}
		if key != w.key {
			continue
		}
		weaponBonus += int32((float64(e.Dex)*w.dexK + float64(e.Int)*w.intK) / 100)
	}
	if weaponBonus == 0 {
		return 0
	}

	for _, bit := range classSkillBits(e.Class) {
		if e.LearnedSkill&(1<<bit) != 0 {
			return weaponBonus
		}
	}
	return 0
}

func skillFlatDamage(e *world.Entity) int32 {
	if e.Class == 3 && e.LearnedSkill&(1<<7) != 0 {
		return 200
	}
	return 0
}

func (d *Dispatcher) derivedDamageBeforeAffects(e *world.Entity) int32 {
	if !isPlayerMob(e) {
		return 0
	}
	return d.classWeaponDamage(e) + skillFlatDamage(e)
}

func attributeDamageLevelTerm(e *world.Entity) int32 {
	if e.ClassMaster == classMasterArch || e.ClassMaster == classMasterMortal {
		return e.Level
	}
	return e.Level + level.MaxLevel
}

func attributeDamageBonus(e *world.Entity, withAffectSpecial bool) int32 {
	if !isPlayerMob(e) {
		return 0
	}
	sp := int32(e.Special[0])
	str, dex := e.Str, e.Dex
	if withAffectSpecial {
		sp = int32(effectiveSpecial(e, 0))
		str, dex = effectiveStr(e), effectiveDex(e)
	}
	return int32(str)/2 + int32(dex)/3 + sp + attributeDamageLevelTerm(e)
}

func skillDerivedACBonus(e *world.Entity, flatAC int32) int32 {
	var bonus int32
	if e.Class == 0 && e.LearnedSkill&(1<<15) != 0 {
		bonus += flatAC / 10
	}
	if e.Class == 3 && e.LearnedSkill&(1<<23) != 0 {
		bonus += int32(e.Special[3])/3 + 10
	}
	return bonus
}
