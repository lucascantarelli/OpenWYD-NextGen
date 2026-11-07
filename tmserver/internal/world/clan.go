package world

// clanTable é g_pClanTable[9][9] (Basedef.cpp:207-220), a matriz de
// hostilidade clã-contra-clã usada pela perseguição das criaturas: 0 = hostil (um inimigo que a criatura vai
// adquirir), 1 = amistoso (nunca perseguido). Jogadores normalmente carregam Clan 0;
// criaturas carregam o clã guardado no template STRUCT_MOB delas.
var clanTable = [9][9]uint8{
	{1, 0, 1, 1, 1, 0, 1, 1, 1},
	{0, 1, 1, 0, 1, 1, 1, 0, 0},
	{1, 1, 1, 0, 1, 1, 1, 1, 1},
	{1, 0, 0, 1, 1, 0, 1, 1, 1},
	{1, 1, 1, 1, 1, 0, 1, 1, 1},
	{0, 1, 1, 0, 0, 1, 0, 0, 0},
	{1, 1, 1, 1, 1, 0, 1, 1, 1},
	{1, 0, 1, 1, 1, 0, 1, 1, 0},
	{1, 0, 1, 1, 1, 0, 1, 0, 1},
}

// ClanHostile informa se o clã a trata o clã b como inimigo. Clãs fora da
// faixa nunca são hostis, espelhando a guarda do original (CMob.cpp:1345-1350
// loga "err,clan out or range" e aborta a varredura).
func ClanHostile(a, b uint8) bool {
	if a >= 9 || b >= 9 {
		return false
	}
	return clanTable[a][b] == 0
}
