package grpcsrv

import (
	"testing"

	"openwyd-nextgen/internal/domain"
)

// As colunas de base da migration 0027 sobrevivem ao mapeamento domínio <-> proto,
// e HasBase viaja junto (falso mantém uma linha pré-0027 no caminho de derivação).
func TestCharacterBaseScoreMapping(t *testing.T) {
	ch := domain.Character{Str: 212, HasBase: true, BaseStr: 12, BaseInt: 13, BaseDex: 2512, BaseCon: 860,
		BaseMaxHp: 2000, BaseMaxMp: 500}
	got := protoToCharacter(characterToProto(ch))
	if got.HasBase != ch.HasBase || got.BaseStr != ch.BaseStr || got.BaseInt != ch.BaseInt ||
		got.BaseDex != ch.BaseDex || got.BaseCon != ch.BaseCon || got.BaseMaxHp != ch.BaseMaxHp ||
		got.BaseMaxMp != ch.BaseMaxMp || got.Str != ch.Str {
		t.Fatalf("round trip %+v, want %+v", got, ch)
	}
	if protoToCharacter(characterToProto(domain.Character{Str: 112})).HasBase {
		t.Fatal("HasBase false must stay false")
	}
}
