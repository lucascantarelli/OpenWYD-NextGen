// Package mobstat aplica sobreposições de stats editadas por moderador em STRUCT_MOB
// bytes do template de mob/NPC (mob-template-editing-plan.md) — o equivalente
// posterior da ferramenta legada EDITAPPMOB. É irmão de npccfg: aquele
// pacote carrega sobreposições de posicionamento/loja de NPC materializadas por
// handler/npcconfig.go; este carrega sobreposições de stats de combate/atributo
// aplicadas a QUALQUER template npc/<template_name> no carregamento (só no boot —
// sem hot-reload para este recurso, igual ao comportamento de reiniciar-para-aplicar
// do EDITAPPMOB). Override é tipo próprio, não domain.MobTemplateStat, para que só
// tmserver/internal/dbclient dependa do formato de wire — npccfg.Definition
// segue o mesmo desacoplamento para posicionamento/loja de NPC.
package mobstat

import (
	"strings"

	"openwyd-nextgen/internal/savefmt"
)

// EquipItem é a sobreposição de um slot de Equip[] (0..15, MAX_EQUIP).
type EquipItem struct {
	Slot  int
	Index uint16
	Eff   [3][2]uint8
}

// Override é uma sobreposição de stats editada por moderador para um template
// STRUCT_MOB cru, resolvida do snapshot de ListMobTemplateStats do dbServer. Espelha o
// superfície de campos editável no EDITAPPMOB, menos Carry[] (trabalho do npccfg.ShopItem)
// e a posição de spawn gerenciada pelo banco (trabalho de npccfg.Definition).
type Override struct {
	DisplayName                     string // "" mantém o nome do próprio arquivo do template
	Clan, Merchant, Class           uint8
	Coin                            int32
	Exp                             int64
	SPX, SPY                        int16
	Level, AC, Damage               int32
	ChaosRate, AttackRun, Direction uint8
	Str, Int, Dex, Con              int16
	Special                         [4]int16
	MaxHp, Hp, MaxMp, Mp            int32
	LearnedSkill                    int32
	ScoreBonus                      uint16
	SkillBar                        [4]uint8
	RegenHP, RegenMP                uint16
	Resist                          [4]int8
	Equip                           []EquipItem
}

// Apply devolve uma cópia de template com os campos de ov aplicados. O score
// editado é espelhado em BaseScore E CurrentScore, replicando o
// `mob->CurrentScore = mob->BaseScore` do próprio EDITAPPMOB ao salvar. Equip é
// autoritativo como a sobreposição de loja do npccfg: slots não listados em ov.Equip são
// limpos, não deixados nos valores originais do template — um moderador esvaziando
// o editor de equipamento significa "sem equipamento", não "mantém o que estava lá".
// O template normalmente já é um blob canônico MobSize (npctemplate.Load
// alarga os layouts legados no carregamento do conteúdo), mas as formas legadas de
// 756/920 bytes são aceitas aqui também para uma sobreposição nunca degradar em
// silêncio para no-op; o resultado é sempre MobSize. Um template corrompido volta inalterado.
func Apply(template []byte, ov Override) []byte {
	mob, _, err := savefmt.DecodeMobAny(template)
	if err != nil {
		return template
	}

	if strings.TrimSpace(ov.DisplayName) != "" {
		mob.Name = [16]byte{}
		copy(mob.Name[:], ov.DisplayName)
	}
	mob.Clan = ov.Clan
	mob.Merchant = ov.Merchant
	mob.Class = ov.Class
	mob.Coin = ov.Coin
	mob.Exp = ov.Exp
	mob.SPX = ov.SPX
	mob.SPY = ov.SPY

	score := savefmt.Score{
		Level: ov.Level, AC: ov.AC, Damage: ov.Damage,
		Merchant: ov.Merchant, AttackRun: ov.AttackRun, Direction: ov.Direction, ChaosRate: ov.ChaosRate,
		MaxHp: ov.MaxHp, MaxMp: ov.MaxMp, Hp: ov.Hp, Mp: ov.Mp,
		Str: ov.Str, Int: ov.Int, Dex: ov.Dex, Con: ov.Con,
		Special: ov.Special,
	}
	mob.BaseScore = score
	mob.CurrentScore = score

	mob.Equip = [savefmt.MaxEquip]savefmt.Item{}
	for _, it := range ov.Equip {
		if it.Slot < 0 || it.Slot >= savefmt.MaxEquip {
			continue
		}
		mob.Equip[it.Slot] = savefmt.Item{
			Index: int16(it.Index),
			Effects: [3]savefmt.Effect{
				{Effect: it.Eff[0][0], Value: it.Eff[0][1]},
				{Effect: it.Eff[1][0], Value: it.Eff[1][1]},
				{Effect: it.Eff[2][0], Value: it.Eff[2][1]},
			},
		}
	}

	mob.LearnedSkill = ov.LearnedSkill
	mob.ScoreBonus = ov.ScoreBonus
	mob.SkillBar = ov.SkillBar
	mob.RegenHP = ov.RegenHP
	mob.RegenMP = ov.RegenMP
	mob.Resist = ov.Resist

	return savefmt.EncodeMob(mob)
}

// ApplyOverride aplica overrides[name] a template se presente, devolvendo
// o template inalterado caso contrário. Compartilhado pelos dois lugares que resolvem
// um template por nome no boot (spawnNPCs e o TemplateLoader do
// dbclient.NewNpcConfig) para o passo de buscar-e-aplicar não se repetir em cada ponto.
func ApplyOverride(template []byte, name string, overrides map[string]Override) []byte {
	if ov, ok := overrides[name]; ok {
		return Apply(template, ov)
	}
	return template
}
