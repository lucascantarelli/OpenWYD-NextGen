// Package combat guarda as fórmulas de dano do WYD como funções puras
// (game-rules.md §4, Basedef.cpp). Elas ficam separadas de toda I/O para poderem
// ser testadas golden exatamente: as chamadas de rand() passam por um Rand injetado, e
// a ORDEM das chamadas é preservada, então alimentar o LCG da MSVC (tmserver/internal/rng)
// reproduz os números originais byte a byte (parity-tests.md §4.0).
package combat

// Rand é o RNG mínimo que estas fórmulas precisam; *rng.MSVC o satisfaz. Intn(n)
// precisa se comportar como o `rand() % n` do C.
type Rand interface {
	Intn(n int) int
}

// ResurrectSkill é o SkillIndex mágico (99) que permite a um jogador morto agir
// (handlers/_MSG_Attack.md).
const ResurrectSkill = 99

// Damage é BASE_GetDamage (corpo a corpo), game-rules.md §4.1 (Basedef.cpp:1265). O AC
// mitiga metade do seu valor; a maestria de arma (combat) estreita a variância e
// eleva o piso. Sempre devolve ao menos 1.
func Damage(r Rand, dam, ac, combat int) int {
	tdam := dam - ac/2

	c := combat / 2
	if c > 7 {
		c = 7
	}
	delta := 12 - c
	rnd := r.Intn(delta) + c + 99 // factor% ∈ [c+99, 110]
	tdam = rnd * tdam / 100

	switch {
	case tdam < -50:
		tdam = 0
	case tdam < 0: // -50 <= tdam < 0
		tdam = (tdam + 50) / 7
	case tdam <= 50: // 0 <= tdam <= 50
		tdam = 5*tdam/4 + 7
	}
	if tdam <= 0 {
		tdam = 1
	}
	return tdam
}

// SkillDamage é BASE_GetSkillDamage, game-rules.md §4.2 (Basedef.cpp:1486).
// Skills usam mais maestria (cap 15) que o corpo a corpo. Sempre devolve ao menos 1.
func SkillDamage(r Rand, dam, ac, combat int) int {
	tdam := dam - ac/2

	c := combat
	if c > 15 {
		c = 15
	}
	delta := 21 - c
	rnd := r.Intn(delta) + c + 90 // factor% ∈ [c+90, 110]
	tdam = tdam * rnd / 100

	switch {
	case tdam < -50:
		tdam = 0
	case tdam > -50 && tdam < 0: // strict on -50 (per §4.2)
		tdam = (tdam + 50) / 10
	case tdam >= 0 && tdam <= 45:
		tdam = 5*tdam/4 + 5
	}
	if tdam <= 0 {
		tdam = 1
	}
	return tdam
}

// HitInput é um golpe atacante→alvo (game-rules.md §4.3-4.5,
// _MSG_Attack.cpp). Autoritativo no servidor: o dano alegado pelo cliente é ignorado.
//
// ParryRate é o resultado de GetParryRate (sua árvore interna é UNVERIFIED,
// game-rules.md §4.4) fornecido por quem chama. MaxDamage limita o resultado (0 =
// sem limite; o valor MAX_DAMAGE do original é UNVERIFIED).
type HitInput struct {
	AttackerDamage int
	TargetAC       int
	TargetIsPlayer bool
	DoubleCritical uint8 // bit 1 = total crit (×2), bit 2 = partial crit
	Master         int   // weapon mastery (combat level)
	UseSkill       bool
	SkillIndex     int
	ParryRate      int  // 0..1000
	TargetRsvBlock bool // target.Rsv & 0x200 → -4 instead of -3 on a low-roll block
	ReflectDamage  int
	ReflectPvP     int
	ForceMobDamage int
	MaxDamage      int
}

// ResolveHit roda o pipeline de golpe por alvo (§4.3): crítico parcial → AC
// (×3 vs players) → BASE_GetDamage/SkillDamage → total critical → parry → reflect
// → forçar/limitar. Um resultado negativo é o código de "erro/bloqueio" (-3, ou -4 num
// roll baixo bloqueado) propagado ao cliente. As chamadas de rand() acontecem na
// ordem original para o fluxo do LCG casar.
func ResolveHit(r Rand, in HitInput) int {
	dam := in.AttackerDamage

	// Partial critical (DoubleCritical & 2): ×1.3–1.4 vs players, ×1.5–1.6 vs mobs.
	if in.DoubleCritical&2 != 0 {
		if in.TargetIsPlayer {
			dam = (r.Intn(2) + 13) * dam / 10
		} else {
			dam = (r.Intn(2) + 15) * dam / 10
		}
	}

	ac := in.TargetAC
	if in.TargetIsPlayer {
		ac *= 3 // players resist 3× in PvP
	}
	if in.UseSkill {
		dam = SkillDamage(r, dam, ac, in.Master)
	} else {
		dam = Damage(r, dam, ac, in.Master)
	}

	// Total critical (DoubleCritical & 1).
	if in.DoubleCritical&1 != 0 {
		dam *= 2
	}

	// Parry/dodge: um roll em 1000 contra ParryRate. Sempre rolado (consome um rand).
	if miss := ResolveParry(r, in.SkillIndex, in.ParryRate, in.TargetRsvBlock); miss != 0 {
		return miss
	}

	// Reflexão/absorção (só PvP), cada passo com piso 1.
	if in.TargetIsPlayer && dam > 0 {
		dam -= in.ReflectDamage
		if dam < 1 {
			dam = 1
		}
		dam -= dam / 100 * in.ReflectPvP
		if dam < 1 {
			dam = 1
		}
	}

	if !in.TargetIsPlayer && dam >= 1 {
		dam += in.ForceMobDamage
	}
	if in.MaxDamage > 0 && dam >= in.MaxDamage {
		dam = in.MaxDamage
	}
	return dam
}
