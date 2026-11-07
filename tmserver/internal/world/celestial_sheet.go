package world

import (
	"encoding/binary"
	"fmt"
)

// CelestialSheet é uma entrada de STRUCT_MOBEXTRA.SaveCelestial[] (Basedef.h:682-707):
// a ficha de personagem estacionada do tier Celestial que não está em uso.
// O índice 0 guarda o Celestial, o índice 1 o Sub-Celestial. A Pedra Ideal
// escreve [1] quando um Celestial vira CELESTIALCS (_MSG_UseItem.cpp:3040-3073),
// o Urnammu reescreve sua classe ali (_MSG_Quest.cpp:1248-1310) e a Pedra
// Misteriosa troca a ficha ativa pela estacionada (_MSG_UseItem.cpp:3188-3260).
//
// Os campos de BaseScore que o tmServer não modela para jogadores (Merchant,
// AttackRun, Direction, ChaosRate) ficam de fora; o legado só os copia
// junto.
type CelestialSheet struct {
	Class           int32
	Exp             int64
	SPX, SPY        int16
	Level           int32 // BaseScore.Level
	AC, Damage      int32 // BaseScore.Ac/Damage
	MaxHP, MaxMP    int32
	HP, MP          int32
	Str, Int        int16
	Dex, Con        int16
	Special         [4]int16
	LearnedSkill    int32
	SecLearnedSkill int32
	ScoreBonus      uint16
	SpecialBonus    uint16
	SkillBonus      uint16
	SkillBar1       [4]uint8  // MOB.SkillBar
	SkillBar2       [16]uint8 // CharShortSkill
	Soul            uint8
}

// celestialSheetVersion prefixa o blob persistido para o layout abaixo poder crescer
// sem adivinhar nas linhas antigas.
const celestialSheetVersion = 1

// celestialSheetSize é o tamanho codificado de uma ficha (veja MarshalCelestialSheets).
const celestialSheetSize = 96

// celestialSheetsBlobSize é o tamanho codificado das duas entradas de SaveCelestial.
const celestialSheetsBlobSize = 1 + 2*celestialSheetSize

// MarshalCelestialSheets codifica SaveCelestial[2] para a coluna character.save_celestial
// (migration 0033). É o nosso próprio layout fixo little-endian, não a
// imagem MSVC de STRUCT_MOBEXTRA: byte de versão, depois por ficha Class@0 Exp@4 SPX@12
// SPY@14 Level@16 AC@20 Damage@24 MaxHP@28 MaxMP@32 HP@36 MP@40 Str@44 Int@46
// Dex@48 Con@50 Special@52 LearnedSkill@60 SecLearnedSkill@64 ScoreBonus@68
// SpecialBonus@70 SkillBonus@72 SkillBar1@74 SkillBar2@78 Soul@94 (pad @95).
// Um personagem que nunca teve Sub-Celestial codifica como nil, então sua linha mantém
// o padrão vazio.
func MarshalCelestialSheets(sheets [2]CelestialSheet) []byte {
	if sheets == [2]CelestialSheet{} {
		return nil
	}
	b := make([]byte, celestialSheetsBlobSize)
	b[0] = celestialSheetVersion
	le := binary.LittleEndian
	for i, c := range sheets {
		p := b[1+i*celestialSheetSize : 1+(i+1)*celestialSheetSize]
		le.PutUint32(p[0:], uint32(c.Class))
		le.PutUint64(p[4:], uint64(c.Exp))
		le.PutUint16(p[12:], uint16(c.SPX))
		le.PutUint16(p[14:], uint16(c.SPY))
		le.PutUint32(p[16:], uint32(c.Level))
		le.PutUint32(p[20:], uint32(c.AC))
		le.PutUint32(p[24:], uint32(c.Damage))
		le.PutUint32(p[28:], uint32(c.MaxHP))
		le.PutUint32(p[32:], uint32(c.MaxMP))
		le.PutUint32(p[36:], uint32(c.HP))
		le.PutUint32(p[40:], uint32(c.MP))
		le.PutUint16(p[44:], uint16(c.Str))
		le.PutUint16(p[46:], uint16(c.Int))
		le.PutUint16(p[48:], uint16(c.Dex))
		le.PutUint16(p[50:], uint16(c.Con))
		for k, v := range c.Special {
			le.PutUint16(p[52+2*k:], uint16(v))
		}
		le.PutUint32(p[60:], uint32(c.LearnedSkill))
		le.PutUint32(p[64:], uint32(c.SecLearnedSkill))
		le.PutUint16(p[68:], c.ScoreBonus)
		le.PutUint16(p[70:], c.SpecialBonus)
		le.PutUint16(p[72:], c.SkillBonus)
		copy(p[74:78], c.SkillBar1[:])
		copy(p[78:94], c.SkillBar2[:])
		p[94] = c.Soul
	}
	return b
}

// UnmarshalCelestialSheets decodifica MarshalCelestialSheets. Um blob vazio é um
// personagem sem fichas estacionadas. Qualquer outro tamanho ou versão é erro: o
// chamador precisa recusar a carga em vez de jogar com fichas zeradas que o
// próximo save sobrescreveria nas reais.
func UnmarshalCelestialSheets(b []byte) ([2]CelestialSheet, error) {
	var sheets [2]CelestialSheet
	if len(b) == 0 {
		return sheets, nil
	}
	if len(b) != celestialSheetsBlobSize || b[0] != celestialSheetVersion {
		return sheets, fmt.Errorf("save_celestial: unsupported blob (len %d, version %d)", len(b), b[0])
	}
	le := binary.LittleEndian
	for i := range sheets {
		p := b[1+i*celestialSheetSize : 1+(i+1)*celestialSheetSize]
		c := &sheets[i]
		c.Class = int32(le.Uint32(p[0:]))
		c.Exp = int64(le.Uint64(p[4:]))
		c.SPX = int16(le.Uint16(p[12:]))
		c.SPY = int16(le.Uint16(p[14:]))
		c.Level = int32(le.Uint32(p[16:]))
		c.AC = int32(le.Uint32(p[20:]))
		c.Damage = int32(le.Uint32(p[24:]))
		c.MaxHP = int32(le.Uint32(p[28:]))
		c.MaxMP = int32(le.Uint32(p[32:]))
		c.HP = int32(le.Uint32(p[36:]))
		c.MP = int32(le.Uint32(p[40:]))
		c.Str = int16(le.Uint16(p[44:]))
		c.Int = int16(le.Uint16(p[46:]))
		c.Dex = int16(le.Uint16(p[48:]))
		c.Con = int16(le.Uint16(p[50:]))
		for k := range c.Special {
			c.Special[k] = int16(le.Uint16(p[52+2*k:]))
		}
		c.LearnedSkill = int32(le.Uint32(p[60:]))
		c.SecLearnedSkill = int32(le.Uint32(p[64:]))
		c.ScoreBonus = le.Uint16(p[68:])
		c.SpecialBonus = le.Uint16(p[70:])
		c.SkillBonus = le.Uint16(p[72:])
		copy(c.SkillBar1[:], p[74:78])
		copy(c.SkillBar2[:], p[78:94])
		c.Soul = p[94]
	}
	return sheets, nil
}
