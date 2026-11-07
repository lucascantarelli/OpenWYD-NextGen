//go:build integration

package store

import (
	"testing"

	"openwyd-nextgen/internal/domain"
)

// Migration 0027: a base sem equipamento faz ida e volta por SaveCharacter /
// LoadCharacter; uma linha salva antes dela carrega com HasBase falso (o tmServer
// deriva uma vez); e um save que não carrega a base (um tmServer anterior a
// 0027, por exemplo após um rollback) a limpa, então uma base obsoleta nunca é confiada.
func TestBaseScoreColumns(t *testing.T) {
	s, ctx := freshStore(t)
	accID, err := s.SaveAccount(ctx, domain.Account{Name: "basescore", PassHash: "$argon2id$hash",
		Characters: []domain.Character{{Slot: 0, Name: "Archer", Class: 3, Level: 399,
			Str: 112, Int: 112, Dex: 2612, Con: 960, Hp: 2000, MaxHp: 2000, Mp: 500, MaxMp: 500, ClassMaster: 2}}})
	if err != nil {
		t.Fatalf("SaveAccount: %v", err)
	}
	ch, err := s.LoadCharacter(ctx, accID, 0)
	if err != nil {
		t.Fatalf("LoadCharacter: %v", err)
	}
	if ch.HasBase {
		t.Fatalf("a row without base columns loaded HasBase true: %+v", ch)
	}

	ch.HasBase = true
	ch.BaseStr, ch.BaseInt, ch.BaseDex, ch.BaseCon, ch.BaseMaxHp, ch.BaseMaxMp = 12, 12, 2512, 860, 1800, 450
	ch.Str = 212
	if err := s.SaveCharacter(ctx, accID, ch); err != nil {
		t.Fatalf("SaveCharacter: %v", err)
	}
	got, err := s.LoadCharacter(ctx, accID, 0)
	if err != nil {
		t.Fatalf("LoadCharacter: %v", err)
	}
	if !got.HasBase || got.BaseStr != 12 || got.BaseInt != 12 || got.BaseDex != 2512 || got.BaseCon != 860 ||
		got.BaseMaxHp != 1800 || got.BaseMaxMp != 450 || got.Str != 212 {
		t.Fatalf("base round trip: %+v", got)
	}

	// Um save sem a base (um escritor mais antigo) invalida a guardada: o
	// CurrentScore que ele escreveu pode carregar alocações que a base antiga não tem.
	got.HasBase = false
	got.Str = 213
	if err := s.SaveCharacter(ctx, accID, got); err != nil {
		t.Fatalf("SaveCharacter without base: %v", err)
	}
	again, err := s.LoadCharacter(ctx, accID, 0)
	if err != nil {
		t.Fatalf("LoadCharacter: %v", err)
	}
	if again.HasBase || again.Str != 213 {
		t.Fatalf("a save without the base kept a stale base: %+v", again)
	}
}
