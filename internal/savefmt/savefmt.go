// Package savefmt decodifica e codifica os arquivos de save legados do WYD
// (account/<Primeira>/<nome>), que são dumps crus da struct C STRUCT_ACCOUNTFILE.
//
// CRÍTICO — dois regimes de layout: as structs de save usam alinhamento NATURAL
// (MSVC x86, com padding), NÃO pack(1) como as mensagens de rede. Tudo aqui é
// lido/escrito por OFFSET EXPLÍCITO, little-endian, emulando o layout MSVC x86
// qualquer que seja o SO hospedeiro (long=4, time_t=8). Os valores documentados
// de sizeof/offset são travados por testes golden em savefmt_test.go (o
// equivalente Go do static_assert).
//
// Só o formato ATUAL de 7952 bytes está modelado por inteiro (é o único layout
// verificado campo a campo). Os formatos legados de 4294 e 7500–7600 são
// detectados pelo tamanho, mas seu layout interno não está verificado — ver
// version.go.
package savefmt

import "encoding/binary"

// le é a ordem de bytes dos arquivos de save (little-endian, x86).
var le = binary.LittleEndian

// Tamanhos das structs de save sob alinhamento natural MSVC x86, time_t=8. São
// exatos por contrato e afirmados por teste; não mude sem reconfirmar contra um
// build de referência.
const (
	ItemSize        = 8    // STRUCT_ITEM
	ScoreSize       = 48   // STRUCT_SCORE
	AffectSize      = 8    // STRUCT_AFFECT
	QuestSize       = 56   // STRUCT_QUEST
	MobSize         = 816  // STRUCT_MOB
	MobExtraSize    = 552  // STRUCT_MOBEXTRA
	AccountInfoSize = 216  // STRUCT_ACCOUNTINFO
	AccountFileSize = 7952 // STRUCT_ACCOUNTFILE
)

// Tamanhos das variantes legadas de template STRUCT_MOB avulso encontradas em
// tmserver/content/TMsrv/run/npc/. NÃO são tamanhos alternativos da struct
// modelada — MobSize continua 816 — e sim layouts mais antigos roteados por
// DetectMobVersion e alargados para MobSize por NormalizeMob.
const (
	ScoreSizeLegacy756     = 28  // STRUCT_SCORE compacta do layout de 756 bytes
	MobSizeLegacy756       = 756 // STRUCT_MOB anterior à expansão
	MobSizeLegacy756Padded = 920 // registro de 756 bytes + 164 B de lixo não inicializado
)

// offMobExp fixa o Exp de recompensa de kill dentro do STRUCT_MOB canônico de
// 816 bytes (int64). É o único offset canônico de STRUCT_MOB com nome — os
// demais ficam inline em codec.go — porque PutMobExp precisa escrevê-lo em
// oposição ao offL756Exp legado abaixo, e um 32 solto ao lado de um 28 solto se
// lê como erro de digitação.
const offMobExp = 32

// Offsets de campo do STRUCT_MOB legado de 756 bytes. Reengenheirados dos assets
// que acompanham o projeto e conferidos contra o arquivo de 816 bytes do mesmo
// mob nos 41 pares legado/canônico de tmserver/content/TMsrv/run/npc: Class
// concorda em 100%, ScoreBonus 95%, Critical/SaveMana 98% e Special 90%, e só os
// campos que um rebalance toca (Level, Con, Damage) divergem.
//
// Em relação ao layout atual, este não tem byte de Quest, tem Exp de 32 bits,
// STRUCT_SCORE de 28 bytes, sem Magic e RegenHP/RegenMP de 8 bits. A struct
// inteira alinha em 4 (nenhum membro int64), então fecha em 756 sem padding
// final — 8 (cabeçalho) + 2*20 (scores) + 12 (rabo) = 60 = MobSize −
// MobSizeLegacy756.
const (
	offL756Clan         = 16
	offL756Merchant     = 17
	offL756Guild        = 18
	offL756Class        = 20
	offL756Rsv          = 22
	offL756Coin         = 24
	offL756Exp          = 28 // long (4 bytes) — o layout de 816 alargou para 8
	offL756SPX          = 32
	offL756SPY          = 34
	offL756BaseScore    = 36
	offL756CurrentScore = 64
	offL756Equip        = 92
	offL756Carry        = 220
	offL756LearnedSkill = 732
	offL756ScoreBonus   = 736 // o layout de 816 insere Magic (uint32) antes disto
	offL756SpecialBonus = 738
	offL756SkillBonus   = 740
	offL756Critical     = 742
	offL756SaveMana     = 743
	offL756SkillBar     = 744
	offL756GuildLevel   = 748 // segue 1 byte de padding, como no offset 801 de 816
	offL756RegenHP      = 750 // uchar — o layout de 816 alargou RegenHP/RegenMP para uint16
	offL756RegenMP      = 751
	offL756Resist       = 752 // int8[4]; a struct fecha em 756 sem padding final
)

// Limites de array.
const (
	MobPerAccount  = 4   // MOB_PER_ACCOUNT
	MaxEquip       = 16  // MAX_EQUIP
	MaxCarry       = 64  // MAX_CARRY
	MaxCargo       = 128 // MAX_CARGO
	MaxAffect      = 32  // MAX_AFFECT
	ShortSkillBars = 4
	ShortSkillSlot = 16
)

// Offsets dos campos de topo dentro de STRUCT_ACCOUNTFILE.
// O mapa completo fecha em AccountFileSize=7952, conferido em savefmt_test.go.
const (
	offInfo       = 0
	offChar       = 216  // Char[4]      (4 × MobSize = 3264)
	offCargo      = 3480 // Cargo[128]   (128 × ItemSize = 1024)
	offCoin       = 4504 // int
	offShortSkill = 4508 // uchar[4][16] (64)
	offAffect     = 4572 // affect[4][32] (4·32·8 = 1024)
	offMobExtra   = 5600 // mobExtra[4]  (4 × 552 = 2208); 4 bytes de padding antes (5596→5600, alinhamento 8)
	offDonate     = 7808 // int
	offTempKey    = 7812 // char[52]
	offReceived   = 7864 // bool
	offQuestDaily = 7872 // STRUCT_QUEST (7 bytes de padding depois de ReceivedItem, alinhamento 8)
	offBlockPass  = 7928 // char[16]
	offIsBlocked  = 7944 // bool; struct rounds to 7952 (align 8)
)
