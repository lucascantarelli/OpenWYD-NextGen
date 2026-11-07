package world

import (
	"testing"

	"openwyd-nextgen/internal/teleportdata"
	"openwyd-nextgen/tmserver/internal/rng"
)

// TestTeleportOverlayOverridesAndDisables cobre os critérios de aceite da issue 42
// na fronteira do mundo: uma linha substitui a rota compilada, uma linha desativada impede
// o tile de teleportar, origens intocadas mantêm a rota compilada, e
// limpar o overlay restaura a tabela compilada exatamente.
func TestTeleportOverlayOverridesAndDisables(t *testing.T) {
	defer SetTeleportOverlay(nil)

	// Sobrepor Armia → Noatum: novo destino e um custo de 1000 de ouro.
	if applied := SetTeleportOverlay([]OverlayRoute{{
		SrcX: 2116, SrcY: 2100, DstX: 1200, DstY: 1300, Cost: 1000,
		Requirement: teleportdata.RequirementNone, Enabled: true,
	}}); applied != 1 {
		t.Fatalf("SetTeleportOverlay applied = %d, want 1", applied)
	}
	dx, dy, cost, ok := TeleportDest(2118, 2102) // arredonda para 2116,2100
	if !ok || cost != 1000 {
		t.Fatalf("overlaid Armia route: ok=%v cost=%d, want ok cost=1000", ok, cost)
	}
	if dx < 1200 || dx > 1202 || dy < 1300 || dy > 1302 {
		t.Fatalf("overlaid destination = %d,%d, want (1200..1202,1300..1302)", dx, dy)
	}

	// Uma origem sem linha de overlay ainda resolve pela tabela compilada.
	if _, _, cost, ok := TeleportDest(1044, 1724); !ok || cost != 0 {
		t.Fatalf("compiled Noatum→Armia: ok=%v cost=%d, want ok cost=0", ok, cost)
	}

	// Desativar uma rota remove o tile por completo.
	SetTeleportOverlay([]OverlayRoute{{SrcX: 2480, SrcY: 1716, Enabled: false}})
	if _, _, _, ok := TeleportDest(2480, 1716); ok {
		t.Fatal("disabled Azran route still teleported")
	}

	// Limpar o overlay restaura a rota compilada e seu custo de 700 de ouro.
	SetTeleportOverlay(nil)
	if _, _, cost, ok := TeleportDest(2480, 1716); !ok || cost != 700 {
		t.Fatalf("after clear: ok=%v cost=%d, want ok cost=700", ok, cost)
	}
}

// TestTeleportOverlayKeepsFairyGateAndRNGOrder prova que o overlay não pode enfraquecer um
// gate e que o espalhamento do destino continua consumindo o fluxo MSVC do mundo
// na mesma posição (a issue 42 proíbe mudar a ordem do RNG).
func TestTeleportOverlayKeepsFairyGateAndRNGOrder(t *testing.T) {
	defer SetTeleportOverlay(nil)

	SetTeleportOverlay([]OverlayRoute{{
		SrcX: 2548, SrcY: 1740, DstX: 2281, DstY: 3688, Cost: 0,
		Requirement: teleportdata.RequirementFairy, Enabled: true,
	}})

	// Sem a fada: sem rota e sem consumo de RNG.
	blocked := rng.NewSeeded(7)
	if _, _, _, ok := TeleportDestWithAccess(2548, 1740, TeleportAccess{}, blocked); ok {
		t.Fatal("overlaid Vale gate resolved without the fairy")
	}
	if got, want := blocked.Intn(1000), rng.NewSeeded(7).Intn(1000); got != want {
		t.Fatalf("blocked portal consumed the MSVC stream: got %d want %d", got, want)
	}

	// Com a fada: o destino é base + os dois próximos sorteios, em ordem.
	allowed := rng.NewSeeded(7)
	want := rng.NewSeeded(7)
	dx, dy, _, ok := TeleportDestWithAccess(2548, 1740, TeleportAccess{HasFairy: true}, allowed)
	if !ok {
		t.Fatal("overlaid Vale route did not resolve with the fairy")
	}
	if dx != 2281+int16(want.Intn(3)) || dy != 3688+int16(want.Intn(3)) {
		t.Fatalf("spread order changed: got (%d,%d)", dx, dy)
	}
}
