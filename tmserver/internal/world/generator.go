package world

// (Re)geração de mob por gerador: a porta de runtime do GenerateMob
// (Server.cpp:3442-3810) e a contabilidade de CurrentNumMob do DeleteMob
// (Server.cpp:7809-7843). Cada bloco do NPCGener.txt vira um Generator; o tick
// da IA dispara o GenerateMob na cadência MinuteGenerate do bloco e a morte
// decrementa a população, então áreas farmadas repovoam em grupos como no mundo
// original. Tudo isto é estado só do loop.

// Generator é o estado de runtime de um bloco do NPCGener.txt (NPCGENLIST,
// CNPCGene.h:29-51): a receita de spawn mais o contador de população viva.
type Generator struct {
	MinuteGenerate int // período de respawn em minutos; <=0 = o timer nunca regenera
	MinGroup       int // quantidade de seguidores sorteada como MinGroup + rand()%(MaxGroup-MinGroup+1)
	MaxGroup       int
	MaxNumMob      int // teto de população (líder e seguidores contam para ele)
	RouteType      uint8
	Formation      int
	SegX, SegY     [5]int16
	SegRange       [5]int16
	SegWait        [5]int16
	LeaderTmpl     []byte // STRUCT_MOB cru de 816 bytes; nil = gerador inutilizável
	FollowerTmpl   []byte // nil = grupos só com líder ("Follower: 0")
	CurrentNumMob  int    // mobs vivos deste bloco (SpawnMobAt ++ / DespawnMob --)
	// TrainingField marca um bloco cujo ponto de partida cai numa célula 0x80 do
	// AttributeMap — o campo de treino que o legado limita por nível
	// (_MSG_Action.cpp:201-215). Seus abates ganham o multiplicador de EXP do
	// campo (issue #383).
	TrainingField bool
	FightAction   [4]string
	DieAction     [4]string
}

// generateWorldCap impede o timer dos geradores de encher todos os slots de
// entidade: a folga acima dele evita que a fila simples de respawn (blocos com
// MinuteGenerate<=0) passe fome por causa do drop-on-full. Divergência
// deliberada — no original o único teto é o próprio MAX_MOB.
const generateWorldCap = 20000

// formationOffsets é g_pFormation[5][12][2] (Basedef.cpp:193), interpretado como
// [formação][slot do seguidor][x/y]. O uso no Server.cpp indexa o array numa
// ordem de decompilado, mas o NPCGener tem blocos com Formation=4, então a
// primeira dimensão da tabela é a única forma que representa o conteúdo.
var formationOffsets = [5][MaxParty]struct{ x, y int16 }{
	{{1, 1}, {-1, 1}, {1, -1}, {-1, -1}, {1, 0}, {-1, 0}, {0, 1}, {0, -1}, {2, 0}, {-2, 0}, {0, 2}, {0, -2}},
	{{1, 0}, {-1, 0}, {2, 0}, {-2, 0}, {3, 0}, {-3, 0}, {4, 0}, {-4, 0}, {5, 0}, {-5, 0}, {0, 6}, {0, -6}},
	{{1, 1}, {-1, 1}, {1, -1}, {-1, -1}, {1, 0}, {-1, 0}, {0, 1}, {0, -1}, {2, 0}, {-2, 0}, {0, 2}, {0, -2}},
	{{1, 0}, {-1, 0}, {2, 0}, {-2, 0}, {3, 0}, {-3, 0}, {4, 0}, {-4, 0}, {5, 0}, {-5, 0}, {0, 6}, {0, -6}},
	{{2, 0}, {0, 2}, {1, 1}, {0, 1}, {1, 0}, {-1, 3}, {3, -1}, {-1, 2}, {2, -1}, {-1, 1}, {1, -1}, {1, 2}},
}

// RegisterGenerators instala a tabela de geradores (índice = bloco NPCGener =
// Entity.GenIndex). Só na montagem, antes do Run.
func (w *World) RegisterGenerators(gens []*Generator) { w.generators = gens }

// GeneratorCount devolve quantos slots de gerador estão registrados (alguns
// podem ser nil — blocos cujos templates falharam ao carregar).
func (w *World) GeneratorCount() int { return len(w.generators) }

// GeneratorAt devolve o gerador em idx, ou nil. Só no loop (o chamador pode ler
// o CurrentNumMob vivo).
func (w *World) GeneratorAt(idx int) *Generator {
	if idx < 0 || idx >= len(w.generators) {
		return nil
	}
	return w.generators[idx]
}

// EventOwnedGenerator diz se o bloco pertence a um sistema de evento (sala
// d'água, Kefra, Nightmare, quest de runa). Esses sistemas são donos do ciclo de
// vida do bloco, então o overlay do moderador não pode materializá-lo: criaria um
// segundo chefe, torre ou grupo de sala em cima do que o código de evento cria.
func EventOwnedGenerator(idx int) bool {
	return IsWaterGenerator(idx) || IsKefraGenerator(idx) || NightmareGenerator(idx) >= 0 || IsRuneQuestGenerator(idx)
}

// ClearGenerator remove toda entidade viva e respawn enfileirado de um slot de
// gerador antes de um snapshot do banco substituir a receita. Só no loop.
func (w *World) ClearGenerator(idx int) {
	if idx < 0 || idx >= len(w.generators) {
		return
	}
	for id := MaxUser; id < MaxMob; id++ {
		if e := w.entities[id]; e != nil && int(e.GenIndex) == idx {
			w.DespawnMob(id, 0)
		}
	}
	kept := w.respawnQueue[:0]
	for _, entry := range w.respawnQueue {
		if int(entry.spawn.GenIndex) != idx {
			kept = append(kept, entry)
		}
	}
	w.respawnQueue = kept
	if g := w.generators[idx]; g != nil {
		g.CurrentNumMob = 0
	}
}

// RemoveGeneratorMob porta um DeleteMob que não é morte (Server.cpp:7809-7843):
// qualquer tipo diferente de zero libera o slot de gerador do mob, enquanto o
// DespawnMob só faz isso em morte (tipo 1, que também alimenta a fila de
// respawn). Sem essa liberação, um bloco cujos mobs saem por tipo 3 fica no teto
// e o timer de minuto nunca o reabastece. Só no loop.
func (w *World) RemoveGeneratorMob(id int, removeType int32) {
	e := w.Entity(id)
	if e == nil || IsPlayer(id) {
		return
	}
	gen := w.GeneratorAt(int(e.GenIndex))
	w.DespawnMob(id, removeType)
	if removeType != 0 && removeType != 1 && gen != nil {
		gen.CurrentNumMob = max(gen.CurrentNumMob-1, 0)
	}
}

// SpawnGeneratorLeader spawna exatamente um líder de gerador no primeiro ponto
// de rota sem consumir RNG. Eventos roteirizados do mundo usam isto em vez do
// GenerateMob para não perturbarem o fluxo de paridade de combate/drop.
func (w *World) SpawnGeneratorLeader(idx int) int {
	g := w.GeneratorAt(idx)
	if g == nil || g.LeaderTmpl == nil {
		return -1
	}
	x, y := g.SegX[0], g.SegY[0]
	if x == 0 {
		for i := range g.SegX {
			if g.SegX[i] != 0 {
				x, y = g.SegX[i], g.SegY[i]
				break
			}
		}
	}
	if x == 0 {
		return -1
	}
	x, y, ok := w.emptyCellNear(x, y)
	if !ok {
		return -1
	}
	sp := MobSpawn{Template: g.LeaderTmpl, X: x, Y: y, RouteType: g.RouteType, GenIndex: int16(idx)}
	sp.SegX, sp.SegY, sp.SegWait = g.SegX, g.SegY, g.SegWait
	return w.SpawnMobAt(sp)
}

// GenerateMob spawna um grupo (líder + seguidores sorteados) do gerador idx — a
// porta do GenerateMob (Server.cpp:3442-3810). Devolve os ids criados (o líder
// primeiro) para o chamador revelá-los aos jogadores na visão. Só no loop.
//
// A ordem das chamadas de rand() importa para a paridade (o original consome o
// mesmo rand() global): (1) o sorteio do tamanho do grupo acontece ANTES da
// checagem de população (Server.cpp:3489-3491); (2) dois sorteios por ponto de
// rota com faixa, X e depois Y (Server.cpp:3536-3546, deslocamento enviesado
// para −Range como no original); (3) um sorteio por mob criado cujo template tem
// Clan 1 (`rand()%10==1` rebaixa para clan 2, Server.cpp:3624 — curto-circuito:
// sem sorteio para outros clãs).
//
// Esquisitice mantida: o limite de seguidores ignora o líder, então um bloco com
// MaxNumMob=1 ainda spawna líder+1 seguidor e fica em CurrentNumMob=2 até os dois
// morrerem.
// Não portado: o hack de relocação com MinuteGenerate>=500, o desligamento no
// retângulo do Coliseu e os hooks de evento (BrState/GTORRE) — são sistemas de
// evento, fora do escopo.
func (w *World) GenerateMob(idx int) []int {
	g := w.GeneratorAt(idx)
	if g == nil || g.LeaderTmpl == nil {
		return nil
	}
	qmob := g.MaxGroup - g.MinGroup + 1
	if qmob <= 0 {
		qmob = 1 // guarda de "err,zero divide" (Server.cpp:3486)
	}
	n := g.MinGroup + w.rng.Intn(qmob)
	maxNumMob := g.MaxNumMob
	if maxNumMob < 0 {
		// O NPCGener usa -1 pelo menos no Mestre_Grifo. A política antiga de
		// bootstrap populava blocos de mercador estático de cara, então mantém uma
		// instância viva em vez de tratar o teto negativo como já saturado.
		maxNumMob = 1
	}
	if g.CurrentNumMob >= maxNumMob {
		return nil
	}
	if g.CurrentNumMob+n > maxNumMob {
		n = maxNumMob - g.CurrentNumMob
	}
	if w.mobCount >= generateWorldCap {
		return nil
	}

	sp := MobSpawn{Template: g.LeaderTmpl, RouteType: g.RouteType, GenIndex: int16(idx)}
	for i := 0; i < 5; i++ {
		if g.SegX[i] == 0 {
			continue
		}
		sp.SegX[i], sp.SegY[i] = g.SegX[i], g.SegY[i]
		sp.SegWait[i] = g.SegWait[i]
		if r := int(g.SegRange[i]); r > 0 {
			sp.SegX[i] = g.SegX[i] - int16(r) + int16(w.rng.Intn(r+1))
			sp.SegY[i] = g.SegY[i] - int16(r) + int16(w.rng.Intn(r+1))
		}
	}
	baseX, baseY := sp.SegX[0], sp.SegY[0]
	if baseX == 0 { // sem ponto Start: ancora em qualquer um definido
		for i := range sp.SegX {
			if sp.SegX[i] != 0 {
				baseX, baseY = sp.SegX[i], sp.SegY[i]
				break
			}
		}
	}
	if baseX == 0 {
		return nil // gerador sem posição
	}

	x, y, ok := w.emptyCellNear(baseX, baseY)
	if !ok {
		return nil // "err,No empty mobgrid" (Server.cpp:3631)
	}
	sp.X, sp.Y = x, y
	leaderID := w.SpawnMobAt(sp)
	if leaderID < 0 {
		return nil
	}
	le := w.entities[leaderID]
	if le.Clan == 1 && w.rng.Intn(10) == 1 {
		le.Clan = 2
	}
	ids := []int{leaderID}

	if g.FollowerTmpl == nil {
		return ids
	}
	for i := 0; i < n && i < MaxParty; i++ {
		fsp := followerSpawn(sp, g, i)
		fsp.Template = g.FollowerTmpl
		fbaseX, fbaseY := spawnAnchor(fsp, baseX, baseY)
		fx, fy, fok := w.emptyCellNear(fbaseX, fbaseY)
		if !fok {
			break
		}
		fsp.X, fsp.Y = fx, fy
		fid := w.SpawnMobAt(fsp)
		if fid < 0 {
			break
		}
		fe := w.entities[fid]
		fe.Leader = leaderID // vínculo de grupo: seguidor nunca agride sozinho (CMob.cpp:158)
		le.PartyList[i] = fid
		if fe.Clan == 1 && w.rng.Intn(10) == 1 {
			fe.Clan = 2
		}
		ids = append(ids, fid)
	}
	return ids
}

func spawnAnchor(sp MobSpawn, fallbackX, fallbackY int16) (int16, int16) {
	if sp.SegX[0] != 0 {
		return sp.SegX[0], sp.SegY[0]
	}
	for i := range sp.SegX {
		if sp.SegX[i] != 0 {
			return sp.SegX[i], sp.SegY[i]
		}
	}
	return fallbackX, fallbackY
}

func followerSpawn(leader MobSpawn, g *Generator, slot int) MobSpawn {
	sp := leader
	if g == nil || g.Formation < 0 || g.Formation >= len(formationOffsets) || slot < 0 || slot >= MaxParty {
		return sp
	}
	off := formationOffsets[g.Formation][slot]
	for i := 0; i < len(sp.SegX); i++ {
		if sp.SegX[i] == 0 {
			continue
		}
		if g.SegRange[i] != 0 {
			sp.SegX[i] = leader.SegX[i] + off.x
			sp.SegY[i] = leader.SegY[i] + off.y
		} else {
			sp.SegX[i] = g.SegX[i]
			sp.SegY[i] = g.SegY[i]
		}
	}
	return sp
}

// emptyCellNear acha uma célula de grade livre de mob em (x,y) ou em volta,
// varrendo as caixas expansivas que o GetEmptyMobGrid usa (GetFunc.cpp:2027,
// anéis 1..3). O original também recusa células com altura 127; a grade de
// caminhabilidade vive no lado do handler, então essa checagem é pulada aqui —
// as âncoras de spawn vêm do NPCGener curado (UNVERIFIED que nenhum gerador
// ancora em terreno bloqueado).
func (w *World) emptyCellNear(x, y int16) (int16, int16, bool) {
	if _, ok := w.grid.MobAt(int(x), int(y)); !ok && w.grid.inBounds(int(x), int(y)) {
		return x, y, true
	}
	for ring := 1; ring <= 3; ring++ {
		for dy := -ring; dy <= ring; dy++ {
			for dx := -ring; dx <= ring; dx++ {
				cx, cy := int(x)+dx, int(y)+dy
				if !w.grid.inBounds(cx, cy) {
					continue
				}
				if _, ok := w.grid.MobAt(cx, cy); !ok {
					return int16(cx), int16(cy), true
				}
			}
		}
	}
	return 0, 0, false
}
