package dbclient

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	dbv1 "openwyd-nextgen/api/db/v1"
	"openwyd-nextgen/tmserver/internal/mobstat"
)

// MobStatSource busca as sobreposições de stat de template de mob/NPC editadas
// pelo moderador (mob-template-editing-plan.md) no NpcConfigService do dbServer.
// É de propósito independente de NpcConfig/npccfg: um override de stat vale para
// QUALQUER arquivo npc/<template_name>, não só o subconjunto de mercadores que
// NpcConfig materializa, e é buscado uma vez no boot (sem poll de versão — sem
// hot-reload para este recurso, igual ao comportamento de reiniciar do EDITAPPMOB).
type MobStatSource struct {
	api dbv1.NpcConfigServiceClient
}

// NewMobStatSource embrulha uma conexão gRPC como MobStatSource.
func NewMobStatSource(conn grpc.ClientConnInterface) *MobStatSource {
	return &MobStatSource{api: dbv1.NewNpcConfigServiceClient(conn)}
}

// Fetch devolve todo override de stat de template, com chave template_name, pronto
// para ser aplicado sobre os bytes crus do template via mobstat.Apply.
func (c *MobStatSource) Fetch(ctx context.Context) (map[string]mobstat.Override, error) {
	resp, err := c.api.ListMobTemplateStats(ctx, &dbv1.ListMobTemplateStatsRequest{})
	if err != nil {
		return nil, fmt.Errorf("dbclient: list mob template stats: %w", err)
	}
	out := make(map[string]mobstat.Override, len(resp.GetOverrides()))
	for _, st := range resp.GetOverrides() {
		ov := mobstat.Override{
			DisplayName: st.GetDisplayName(),
			Clan:        uint8(st.GetClan()), Merchant: uint8(st.GetMerchant()), Class: uint8(st.GetClass()),
			Coin: st.GetCoin(), Exp: st.GetExp(), SPX: int16(st.GetSpx()), SPY: int16(st.GetSpy()),
			Level: st.GetLevel(), AC: st.GetAc(), Damage: st.GetDamage(), ChaosRate: uint8(st.GetChaosRate()),
			AttackRun: uint8(st.GetAttackRun()), Direction: uint8(st.GetDirection()),
			Str: int16(st.GetStr()), Int: int16(st.GetIntel()), Dex: int16(st.GetDex()), Con: int16(st.GetCon()),
			Special: [4]int16{int16(st.GetSpecial1()), int16(st.GetSpecial2()), int16(st.GetSpecial3()), int16(st.GetSpecial4())},
			MaxHp:   st.GetMaxHp(), Hp: st.GetHp(), MaxMp: st.GetMaxMp(), Mp: st.GetMp(),
			LearnedSkill: st.GetLearnedSkill(), ScoreBonus: uint16(st.GetScoreBonus()),
			SkillBar: [4]uint8{uint8(st.GetSkillBar1()), uint8(st.GetSkillBar2()), uint8(st.GetSkillBar3()), uint8(st.GetSkillBar4())},
			RegenHP:  uint16(st.GetRegenHp()), RegenMP: uint16(st.GetRegenMp()),
			Resist: [4]int8{int8(st.GetResist1()), int8(st.GetResist2()), int8(st.GetResist3()), int8(st.GetResist4())},
		}
		for _, it := range st.GetEquip() {
			ov.Equip = append(ov.Equip, mobstat.EquipItem{
				Slot: int(it.GetSlot()), Index: uint16(it.GetItemIndex()),
				Eff: [3][2]uint8{
					{uint8(it.GetEff1()), uint8(it.GetEffv1())},
					{uint8(it.GetEff2()), uint8(it.GetEffv2())},
					{uint8(it.GetEff3()), uint8(it.GetEffv3())},
				},
			})
		}
		out[st.GetTemplateName()] = ov
	}
	return out, nil
}
