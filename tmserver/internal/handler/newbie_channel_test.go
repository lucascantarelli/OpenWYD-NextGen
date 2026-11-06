package handler

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"openwyd-nextgen/tmserver/internal/world"
)

// TestIsNoviceChannel locks the fixed-channel rule: the novice channel is the
// one whose 1-based number matches -novice-channel, on any day.
func TestIsNoviceChannel(t *testing.T) {
	cases := []struct {
		noviceChannel int
		serverIndex   int
		want          bool
	}{
		{1, 0, true},   // canal 1 da config, instância 0
		{1, 1, false},  // canal 2 não herda o papel
		{2, 1, true},   // canal 2 da config, instância 1
		{2, 0, false},  //
		{0, 0, false},  // canal verde desligado
		{0, 3, false},  //
		{3, 2, true},   // canal 3
		{-1, 0, false}, // valor inválido não marca ninguém
	}
	for _, c := range cases {
		d := &Dispatcher{noviceChannel: c.noviceChannel, serverIndex: c.serverIndex}
		if got := d.isNoviceChannel(); got != c.want {
			t.Errorf("noviceChannel=%d serverIndex=%d: isNoviceChannel = %v, quer %v",
				c.noviceChannel, c.serverIndex, got, c.want)
		}
	}
}

// TestApplyNoviceChannel covers the three configurations: no novice channel
// (event on everywhere, as the portal flag says), a fixed novice channel (event
// only there) and the portal flag off (event off everywhere), plus the PvP gate
// that reads the same predicate.
func TestApplyNoviceChannel(t *testing.T) {
	newWorld := func() *world.World {
		return world.New(world.Config{}, slog.New(slog.DiscardHandler), nil, nil)
	}

	t.Run("sem canal verde, evento vale em todo canal", func(t *testing.T) {
		w := newWorld()
		d := &Dispatcher{log: slog.New(slog.DiscardHandler), newbieCfg: true, serverIndex: 1}
		d.applyNoviceChannel(w)
		if !d.expEvents.NewbieEvent || !w.NewbieEvent() {
			t.Fatal("evento deveria ligar em qualquer canal quando -novice-channel é 0")
		}
		if d.isNoviceChannel() {
			t.Fatal("sem canal configurado, nenhum canal é o verde (o PvP não é recusado)")
		}
	})

	t.Run("canal verde fixo no 1: instância 0 liga, instância 1 não", func(t *testing.T) {
		novice := &Dispatcher{log: slog.New(slog.DiscardHandler), newbieCfg: true, noviceChannel: 1, serverIndex: 0}
		other := &Dispatcher{log: slog.New(slog.DiscardHandler), newbieCfg: true, noviceChannel: 1, serverIndex: 1}
		w1, w2 := newWorld(), newWorld()

		novice.applyNoviceChannel(w1)
		other.applyNoviceChannel(w2)

		if !novice.expEvents.NewbieEvent || !w1.NewbieEvent() {
			t.Fatal("canal 1 deveria rodar o evento newbie")
		}
		if !novice.isNoviceChannel() {
			t.Fatal("canal 1 deveria recusar PvP (isNoviceChannel)")
		}
		if other.expEvents.NewbieEvent || w2.NewbieEvent() {
			t.Fatal("canal 2 não deveria rodar o evento newbie")
		}
		if other.isNoviceChannel() {
			t.Fatal("canal 2 não deveria recusar PvP")
		}
	})

	t.Run("config desligada vence o canal verde", func(t *testing.T) {
		w := newWorld()
		d := &Dispatcher{log: slog.New(slog.DiscardHandler), newbieCfg: false, noviceChannel: 1, serverIndex: 0}
		d.applyNoviceChannel(w)
		if d.expEvents.NewbieEvent || w.NewbieEvent() {
			t.Fatal("config desligada no portal não pode ligar o evento")
		}
	})
}

// TestApplyNoviceChannelLogsRoleOnce: o papel (novice/regular) sai no log de boot
// uma única vez, para dar prova positiva nos dois canais — sem isso o canal verde
// fica invisível, porque nele o gate concorda com o portal e nunca há transição.
func TestApplyNoviceChannelLogsRoleOnce(t *testing.T) {
	cases := []struct {
		serverIndex int
		wantRole    string
	}{
		{0, "role=novice"},
		{1, "role=regular"},
	}
	for _, c := range cases {
		var buf bytes.Buffer
		w := world.New(world.Config{}, slog.New(slog.DiscardHandler), nil, nil)
		d := &Dispatcher{log: slog.New(slog.NewTextHandler(&buf, nil)), newbieCfg: true,
			noviceChannel: 1, serverIndex: c.serverIndex}
		d.applyNoviceChannel(w)
		d.applyNoviceChannel(w)
		d.applyNoviceChannel(w)
		if got := strings.Count(buf.String(), "novice channel role"); got != 1 {
			t.Fatalf("index %d: linha de papel apareceu %d vezes, quer 1: %s", c.serverIndex, got, buf.String())
		}
		if !strings.Contains(buf.String(), c.wantRole) {
			t.Fatalf("index %d: log sem %q: %s", c.serverIndex, c.wantRole, buf.String())
		}
	}
}

// TestApplyNoviceChannelIsIdempotent: o tick do mundo chama isto o tempo todo;
// sem mudança de estado não pode logar nem alternar nada.
func TestApplyNoviceChannelIsIdempotent(t *testing.T) {
	w := world.New(world.Config{}, slog.New(slog.DiscardHandler), nil, nil)
	d := &Dispatcher{log: slog.New(slog.DiscardHandler), newbieCfg: true, noviceChannel: 1, serverIndex: 0}
	d.applyNoviceChannel(w)
	d.applyNoviceChannel(w)
	d.applyNoviceChannel(w)
	if !w.NewbieEvent() {
		t.Fatal("evento deveria continuar ligado")
	}
}
