package world

// RuneQuestRooms é o número de salas da Pista de Runas (Pista[7], Server.cpp:759).
// A sala é o sanc do ticket da Pista de Runas, limitada a 6.
const RuneQuestRooms = 7

// RuneQuestParty é um slot de grupo registrado (STRUCT_PISTA.Party, Basedef.h:788).
// LeaderID 0 marca slot livre, como no original.
type RuneQuestParty struct {
	LeaderID   int
	LeaderName string
	Sala       int
	MobCount   int
}

// RuneQuestRoom contém os três slots de grupo de uma sala. A sala 0 usa só dois.
type RuneQuestRoom struct {
	Party [3]RuneQuestParty
}

// RuneQuestState é o estado da pista de runas de posse do loop. PeriodUnix identifica a
// rodada de 20 minutos para ticks atrasados ou repetidos não entrarem nem saírem duas vezes.
type RuneQuestState struct {
	Initialized bool
	PeriodUnix  int64
	Entered     bool
	Exited      bool
	Rooms       [RuneQuestRooms]RuneQuestRoom
}

// RuneQuestState devolve um snapshot por valor. Só no loop.
func (w *World) RuneQuestState() RuneQuestState { return w.runeQuest }

// SetRuneQuestState substitui o estado da pista de runas. Só no loop.
func (w *World) SetRuneQuestState(st RuneQuestState) { w.runeQuest = st }

// runeQuestGenerators são os blocos do NPCGener.txt da pista de runas do
// evento: as faixas RUNEQUEST_* de Basedef.h:431-463 mais os outros
// blocos com MinuteGenerate -1 colocados dentro das salas. O original só os
// invoca a partir do código de evento (nunca sobe blocos -1 no boot), então eles não podem ser invocados
// no boot nem entrar na fila de respawn por morte. Os blocos MinuteGenerate 1 na
// mesma faixa de índice são população permanente das salas e ficam no timer de minuto.
var runeQuestGenerators = [][2]int{
	{5653, 5655}, {5677, 5677}, {5682, 5682}, {5690, 5690},
	{5706, 5765}, {5767, 5767}, {5782, 5782}, {5789, 5789}, {5791, 5791},
	{5815, 5815}, {5817, 5817}, {5832, 5832}, {5841, 5841}, {5849, 5849},
	{5854, 5899}, {5948, 5955}, {5965, 5965}, {5970, 5971},
}

// IsRuneQuestGenerator informa se um bloco gerador pertence ao evento da
// pista de runas.
func IsRuneQuestGenerator(idx int) bool {
	for _, r := range runeQuestGenerators {
		if idx >= r[0] && idx <= r[1] {
			return true
		}
	}
	return false
}

// ForEachRuneQuestGenerator visita todo índice de gerador do evento.
func ForEachRuneQuestGenerator(fn func(idx int)) {
	for _, r := range runeQuestGenerators {
		for idx := r[0]; idx <= r[1]; idx++ {
			fn(idx)
		}
	}
}

// RuneQuestArea informa se uma posição está dentro das salas da pista de runas: a
// caixa que o timer de saída esvazia (ProcessSecMinTimer.cpp:1157,1172).
func RuneQuestArea(x, y int16) bool {
	return x >= 3310 && x <= 3588 && y >= 1005 && y <= 1663
}

// RuneQuestEntryPos é PistaPos[room][group] (Server.cpp:440-477): onde cada
// grupo registrado cai na entrada. A sala 0 não tem terceiro grupo.
var RuneQuestEntryPos = [RuneQuestRooms][3][2]int16{
	{{3350, 1622}, {3431, 1634}, {0, 0}},       // Lich
	{{3362, 1574}, {3385, 1555}, {3414, 1575}}, // Torre
	{{3410, 1453}, {3419, 1426}, {3358, 1436}}, // Amon
	{{3376, 1096}, {3400, 1089}, {3392, 1073}}, // Sulrang
	{{3342, 1394}, {3444, 1394}, {3442, 1293}}, // Labirinto
	{{3421, 1217}, {3426, 1211}, {3424, 1226}}, // Balrog
	{{3404, 1517}, {3392, 1479}, {3383, 1501}}, // Coelho
}

// RuneQuestRunes é PistaRune[room] (Server.cpp:765-800) cortado nas entradas que uma
// recompensa pode sortear. Todo sorteio é rand()%len, e cada linha de origem tem uma
// runa a mais do que o módulo alcança (Lich %4 de 5, Amon %5 de 6, ...), então a última
// entrada nunca caía no original e fica de fora aqui.
var RuneQuestRunes = [RuneQuestRooms][]int16{
	{5110, 5112, 5115, 5113},                         // Lich
	{5114, 5113, 5117, 5111, 5115},                   // Torre
	{5118, 5121, 5122, 5116, 5130},                   // Amon
	{5122, 5126, 5121, 5116, 5119},                   // Sulrang
	{5125, 5126, 5124},                               // Labirinto
	{5120, 5131, 5118, 5119, 5123, 5132},             // Balrog
	{5130, 5131, 5119, 5133, 5120, 5123, 5132, 5129}, // Coelho
}

// RuneQuestBalrogPos é PistaBalrogPos (Server.cpp:479-486): o ponto de chegada
// de cada sub-sala do Balrog; o índice 4 é a sala do chefe.
var RuneQuestBalrogPos = [5][2]int16{
	{3421, 1217}, {3428, 1180}, {3357, 1186}, {3393, 1255}, {3385, 1204},
}

// RuneQuestBalrogBossRoom é o índice de RuneQuestBalrogPos da sala do chefe.
const RuneQuestBalrogBossRoom = 4

// RuneQuestBalrogPortal é um tile de PistaBalrogPortalPos (Server.cpp:488-529)
// com a sub-sala para onde manda o grupo do líder em cada resultado de rand()%3,
// achatado a partir dos ramos por portal de ProcessSecMinTimer.cpp:271-486
// (os ramos que o sorteio 0..2 não alcança são descartados).
type RuneQuestBalrogPortal struct {
	X, Y int16
	Dest [3]int
}

// RuneQuestBalrogPortals lista todo tile de portal vivo; a tabela de origem preenche
// cada sala com slots {0,0} que nunca batem com uma posição.
var RuneQuestBalrogPortals = []RuneQuestBalrogPortal{
	{3428, 1196, [3]int{1, 1, 0}}, // sala 1 portal 1
	{3380, 1176, [3]int{1, 1, 2}}, // sala 2 portal 1
	{3380, 1164, [3]int{2, 0, 1}}, // sala 2 portal 2
	{3360, 1224, [3]int{2, 2, 3}}, // sala 3 portal 1
	{3352, 1224, [3]int{3, 1, 2}}, // sala 3 portal 2
	{3340, 1224, [3]int{2, 2, 3}}, // sala 3 portal 3
	{3400, 1236, [3]int{3, 3, 4}}, // sala 4 portal 1
	{3392, 1236, [3]int{4, 2, 3}}, // sala 4 portal 2
	{3384, 1236, [3]int{3, 3, 4}}, // sala 4 portal 3
	{3376, 1236, [3]int{3, 3, 4}}, // sala 4 portal 4
}

// RuneQuestTraps são as caixas de armadilha do Labirinto (x1,y1,x2,y2) acionadas por
// ProcessSecMinTimer.cpp:1715-1804, na ordem do fonte.
var RuneQuestTraps = [][4]int16{
	{3436, 1285, 3450, 1299}, {3368, 1285, 3382, 1299}, {3384, 1302, 3398, 1316},
	{3438, 1320, 3452, 1334}, {3369, 1320, 3383, 1334}, {3385, 1336, 3399, 1350},
	{3402, 1353, 3416, 1367}, {3436, 1371, 3450, 1385}, {3368, 1371, 3382, 1385},
	{3402, 1387, 3416, 1401}, {3420, 1286, 3434, 1300}, {3351, 1286, 3365, 1300},
	{3402, 1302, 3416, 1316}, {3420, 1320, 3434, 1334}, {3371, 1337, 3385, 1351},
	{3436, 1337, 3450, 1351}, {3385, 1353, 3399, 1367}, {3352, 1370, 3366, 1384},
	{3419, 1370, 3433, 1384}, {3385, 1387, 3399, 1401}, {3402, 1286, 3416, 1300},
	{3352, 1302, 3366, 1316}, {3420, 1302, 3434, 1316}, {3401, 1320, 3415, 1334},
	{3420, 1336, 3434, 1350}, {3436, 1353, 3450, 1367}, {3368, 1353, 3382, 1367},
	{3401, 1370, 3415, 1384}, {3435, 1387, 3449, 1401}, {3369, 1387, 3383, 1401},
	{3387, 1285, 3401, 1299}, {3368, 1302, 3382, 1316}, {3436, 1302, 3450, 1316},
	{3384, 1319, 3398, 1333}, {3402, 1335, 3416, 1349}, {3419, 1353, 3433, 1367},
	{3352, 1353, 3366, 1367}, {3385, 1370, 3399, 1384}, {3419, 1387, 3433, 1401},
	{3352, 1387, 3366, 1401}, {3335, 1286, 3349, 1300}, {3335, 1302, 3349, 1316},
	{3335, 1354, 3349, 1368}, {3335, 1371, 3349, 1385},
}
