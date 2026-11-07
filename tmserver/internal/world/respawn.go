package world

// Respawn de criaturas em execução. Uma criatura morta em combate é removida por DespawnMob, que
// enfileira um respawnEntry; SpawnDueRespawns (chamado a cada tick) recria a criatura em
// seu ponto de origem quando o atraso passa. Isso impede o mundo de se esvaziar
// permanentemente conforme os jogadores farmam. Todas as funções aqui são só no loop (a fila pertence
// à goroutine do loop, como o resto do estado do mundo).

// DefaultRespawnDelay é quanto tempo após a morte uma criatura reaparece, na mesma
// unidade de milissegundos de World.Now().
//
// UNVERIFIED: o RegenTime/RegenMob por gerador do original (Server.cpp) não está
// no fonte disponível e nosso parser de NPCGener não lê campo de regen, então este
// é um substituto ajustável (espelha como DefaultMobTick é tratado em tick.go).
const DefaultRespawnDelay = 15_000 // 15s

// respawnEntry é uma criatura morta aguardando respawn: o MobSpawn completo para recriá-la
// (template, ponto de spawn E sua rota de patrulha da instância), após `due`
// (unidades de World.Now).
type respawnEntry struct {
	spawn MobSpawn
	due   uint32
}

// SpawnDueRespawns recria toda criatura enfileirada cujo atraso passou (due <=
// now) e devolve os novos ids de criatura para o chamador revelá-los aos
// jogadores em visão. Entradas que respawnam (ou falham porque o mundo está cheio) são removidas
// da fila; entradas ainda não vencidas são mantidas. Só no loop.
func (w *World) SpawnDueRespawns(now uint32) []int {
	if len(w.respawnQueue) == 0 {
		return nil
	}
	var ids []int
	kept := w.respawnQueue[:0]
	for _, r := range w.respawnQueue {
		if r.due > now {
			kept = append(kept, r)
			continue
		}
		if id := w.SpawnMobAt(r.spawn); id >= 0 {
			ids = append(ids, id)
		}
		// Em falha de SpawnMob (mundo cheio) a entrada é descartada em vez de tentada
		// para sempre; um mundo cheio não tem slot livre para tentar de novo de qualquer forma.
	}
	w.respawnQueue = kept
	return ids
}

// clearSeenAll remove o id da entidade do conjunto de visão de toda sessão, para um slot reutilizado
// por um spawn posterior ser tratado como entidade nova (um CreateMob novo é enviado).
// Chamado de DespawnMob quando o slot é liberado. Só no loop.
func (w *World) clearSeenAll(id int) {
	for _, s := range w.sessions {
		if s == nil || s.seen == nil {
			continue
		}
		delete(s.seen, id)
	}
}
