package world

import (
	"context"
	"time"
)

// DefaultMobTick é a frequência do tick da IA de criaturas. O loop principal de
// timer do servidor original (que chamava CMob::BattleProcessor/StandingByProcessor por criatura) NÃO
// está no código-fonte disponível, então a cadência exata é UNVERIFIED — SECBATTLE=8 é
// a única pista. 1s é um padrão seguro e ajustável: uma criatura persegue ~1 tile/s e ataca
// no máximo uma vez por janela de cadência.
const DefaultMobTick = time.Second

// SetTickHandler registra o gancho periódico de simulação (IA de criaturas) e seu intervalo.
// O handler roda DENTRO da goroutine do loop — como os handlers de frame, ele pode mutar
// o estado do mundo direto, preservando a invariante de dono único. Chame antes de Run.
func (w *World) SetTickHandler(interval time.Duration, fn func(*World)) {
	w.tickInterval = interval
	w.onTick = fn
}

// tickEvent é o pulso de simulação. Seu apply roda o gancho registrado na
// goroutine do loop, então o gancho nunca corre junto com o resto do estado do mundo.
type tickEvent struct{}

func (tickEvent) apply(w *World) {
	// Timestamp do último tick (OBS-03): o alerta "mundo sem tick" compara
	// time() com este gauge. Gravado ANTES do hook, então um tick que trava
	// (não retorna) também para de atualizar o valor e dispara o alerta.
	w.metricSet("world_tick_timestamp_seconds", "Unix time (s) do último tick aplicado pelo loop; alerta de mundo travado.", float64(time.Now().Unix()))
	if w.onTick != nil {
		w.onTick(w)
	}
}

// runTicker emite um tickEvent a cada tickInterval até o contexto ser cancelado ou
// o mundo parar. Roda na própria goroutine mas só envia eventos para
// o loop (sem acesso direto ao estado), então a regra de dono único se mantém.
func (w *World) runTicker(ctx context.Context) {
	t := time.NewTicker(w.tickInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if !w.emit(tickEvent{}) {
				return // mundo desligando
			}
		}
	}
}

// ForEachMob chama fn para toda entidade viva de criatura/NPC (índice ≥ MaxUser). Só no loop.
func (w *World) ForEachMob(fn func(id int, e *Entity)) {
	for id := MaxUser; id < MaxMob; id++ {
		if e := w.entities[id]; e != nil {
			fn(id, e)
		}
	}
}

// ForEachPlayer chama fn para todo jogador em jogo (sessão + sua entidade). Usado
// pelo tick para manutenção por jogador, como regeneração de HP/MP. Só no loop.
func (w *World) ForEachPlayer(fn func(s *Session, e *Entity)) {
	for _, s := range w.sessions {
		if s == nil || s.Mode != UserPlay {
			continue
		}
		if e := w.entities[s.Conn]; e != nil {
			fn(s, e)
		}
	}
}

// FindEnemyFromView devolve o id da primeira entidade viva ao redor de (x,y) que
// o clã trata como hostil (g_pClanTable), ou 0 se não houver. Porta
// GetEnemyFromView (CMob.cpp:1308-1358) exatamente:
// - janela [x-4, x+5) × [y-4, y+5) (StartX = TargetX-4, Size 9);
// - clãs 7/8 varrem [x-6, x+10) — início −6 com largura HALFGRIDX=16, uma
// caixa intencionalmente assimétrica no original;
// - ordem de varredura y externo/x interno (decide QUAL inimigo é achado primeiro);
// - a célula da própria criatura é pulada, e jogadores ocultos (Rsv & RsvHide) são
// invisíveis para a perseguição (CMob.cpp:1342);
// - um clã fora da faixa aborta a varredura inteira (CMob.cpp:1345-1350).
//
// Invocações ficam fora da aquisição espontânea para o fluxo de auxílio ao dono/retaliação
// continuar sendo o único ponto de entrada de mascote contra criatura.
//
// Só no loop.
func (w *World) FindEnemyFromView(x, y int16, clan uint8) int {
	startX, startY, sizeX, sizeY := int(x)-4, int(y)-4, 9, 9
	if clan == 7 || clan == 8 {
		startX, startY, sizeX, sizeY = int(x)-6, int(y)-6, 16, 16
	}
	for gy := startY; gy < startY+sizeY; gy++ {
		for gx := startX; gx < startX+sizeX; gx++ {
			if gx == int(x) && gy == int(y) {
				continue
			}
			id, ok := w.grid.MobAt(gx, gy)
			if !ok {
				continue // célula vazia/fora dos limites
			}
			e := w.entities[id]
			if e == nil || e.Mode == MobEmpty || e.HP <= 0 {
				continue
			}
			if int(id) < MaxUser && e.Mode != MobUser {
				continue
			}
			if int(id) >= MaxUser && e.NonCombatNPC {
				continue // NPCs de cidade/serviço não são alvos de combate
			}
			if int(id) >= MaxUser && e.Summoner != 0 {
				continue // invocações usam o caminho de auxílio ao dono, não a perseguição ambiente
			}
			if int(id) < MaxUser && e.Rsv&RsvHide != 0 {
				continue // jogadores ocultos não atraem perseguição (CMob.cpp:1342 Rsv & 0x10)
			}
			if clan >= 9 || e.Clan >= 9 {
				// Alguns templates reais de evento/arena trazem Clan 9 (Aberest, Pikeman,
				// Wizard, …), e esta varredura roda a cada tick — Debug, não Warn, senão eles
				// encheriam o log. O original loga e aborta a varredura inteira
				// (CMob.cpp:1345-1350), não só esta célula.
				w.log.Debug("clan out of range in aggro scan", "clan", clan, "target_clan", e.Clan)
				return 0
			}
			if ClanHostile(clan, e.Clan) {
				return int(id)
			}
		}
	}
	return 0
}

// EntityAt informa a entidade que ocupa a célula de grade (x,y), se houver. Só no loop.
func (w *World) EntityAt(x, y int16) (int, bool) {
	id, ok := w.grid.MobAt(int(x), int(y))
	return int(id), ok
}
