package handler

import (
	"openwyd-nextgen/tmserver/internal/world"
)

// Canal verde (novato), docs/migration/canais.md §3.
//
// Em produção o canal novato é um canal FIXO: o WYD Global o chama de "canal
// novato" e outros eventos se referenciam a ele por posição ("se o novato for o
// Canal 1, a invasão ocorre no Canal 2"), o que só faz sentido com número fixo.
// A primeira versão daqui rodava o evento em `(dia-1) % canais`, o que não
// corresponde a nenhum comportamento observado em produção e ainda confundia
// duas coisas diferentes: a lista de canais do cliente (estática) e o sistema de
// novato (propriedade do canal).
//
// Aqui o canal verde é escolhido por configuração, em número de canal (1-based):
// `-novice-channel` / `W2PP_NOVICE_CHANNEL`. 0 desliga o canal verde.
//
// No canal verde: o evento de EXP para novatos fica ligado (quando o portal
// também o liga em world_event_config.newbie_event_enabled) e o PvP entre
// jogadores é recusado (combat.go). O evento de EXP continua sendo do servidor,
// não do canal.

// isNoviceChannel reports whether THIS instance is the configured novice channel.
// Loop-only.
func (d *Dispatcher) isNoviceChannel() bool {
	return d.noviceChannel > 0 && d.noviceChannel == d.serverIndex+1
}

// applyNoviceChannel recomputes the effective newbie flag from the portal
// configuration and this instance's role, and runs the 1→0 side effect
// (in-flight trades) when the channel loses the event. With no novice channel
// configured (0) the event applies to every channel, as before.
// Loop-only.
func (d *Dispatcher) applyNoviceChannel(w *world.World) {
	on := d.newbieCfg && (d.noviceChannel == 0 || d.isNoviceChannel())
	// Uma linha por instância, no primeiro tick: sem ela o canal verde nunca
	// aparece no log (o evento já está ligado quando o gate concorda), e a
	// única evidência seria a ausência de linha no outro canal.
	if !d.noviceRoleLogged {
		d.noviceRoleLogged = true
		role := "regular"
		if d.isNoviceChannel() {
			role = "novice"
		}
		d.log.Info("novice channel role", "role", role, "novice_channel", d.noviceChannel,
			"server_index", d.serverIndex, "newbie_event", on)
	}
	was := d.expEvents.NewbieEvent
	if was == on {
		return
	}
	d.expEvents.NewbieEvent = on
	w.SetNewbieEvent(on)
	d.log.Info("newbie event toggled", "on", on, "novice_channel", d.noviceChannel,
		"server_index", d.serverIndex)
	if was && !on {
		d.cancelTradesOnNewbieEnd(w)
	}
}
