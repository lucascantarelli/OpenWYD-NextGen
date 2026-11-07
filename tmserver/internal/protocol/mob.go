package protocol

// MSG_CNFCharacterLogin (0x0114) — o snapshot do personagem no mundo enviado quando um
// jogador entra no jogo. Byte a byte contra o Basedef.h original (sonda de
// sizeof/offsetof do compilador): STRUCT_MOB tem 816 bytes, o pacote do cliente 1832.
// Envie com HEADER.ID = IDScene (30000).

const (
	structMobSize         = 816
	cnfCharacterLoginSize = 1832
)

// MobBasics é o subconjunto de um STRUCT_MOB cru necessário para gerar uma entidade do mundo.
type MobBasics struct {
	Name               string
	Clan               uint8 // STRUCT_MOB.Clan @16 — alimenta a checagem de hostilidade da g_pClanTable
	Class              uint8
	Merchant           uint8 // CurrentScore.Merchant — tipo de NPC (loja/banco/…); 0 = monstro
	AttackRun          uint8 // CurrentScore.AttackRun — byte de velocidade, repetido no CreateMob para o cliente animar os passos do mob
	Level, Ac, Damage  int32
	MaxHp, Hp          int32
	Str, Int, Dex, Con int16
	Coin               int32    // STRUCT_MOB.Coin @28; num monstro é a base do ouro dropado (GoldDrop)
	Exp                int64    // STRUCT_MOB.Exp @32; num monstro é a recompensa por abatê-lo
	Resist             [4]uint8 // STRUCT_MOB.Resist @806 — resistências elementais de skill
}

// ParseMobBasics lê os campos relevantes ao spawn de um STRUCT_MOB cru de 816 bytes
// (CurrentScore @92): nome, classe e o score atual.
//
// O blob JÁ PRECISA estar canônico — os layouts legados de template de 756/920 bytes
// têm offsets totalmente diferentes e são alargados para este por
// npctemplate.Load, a única fronteira de carregamento (data-formats.md §1.4.1). Um
// slice mais curto é violação de contrato e entra em panic aqui, de propósito: a
// alternativa é um MobBasics zerado virar em silêncio um mob fantasma sem nome e com
// 0 de HP em SpawnMobAt (world/api.go), e MobEquip indexaria o mesmo
// mesmo blob sem guarda poucas linhas depois de qualquer forma.
func ParseMobBasics(mob816 []byte) MobBasics {
	const cs = 92 // offset do CurrentScore dentro do STRUCT_MOB
	return MobBasics{
		Name:      cstr16(mob816[0:16]),
		Clan:      mob816[16], // Clan @16 (mesmo offset que writeStructMob escreve)
		Class:     mob816[20],
		Coin:      int32(le.Uint32(mob816[28:])), // STRUCT_MOB.Coin @28 (int)
		Exp:       int64(le.Uint64(mob816[32:])), // STRUCT_MOB.Exp @32 (long long)
		Merchant:  mob816[cs+12],                 // CurrentScore.Merchant
		AttackRun: mob816[cs+13],                 // CurrentScore.AttackRun (velocidade)
		Level:     int32(le.Uint32(mob816[cs+0:])),
		Ac:        int32(le.Uint32(mob816[cs+4:])),
		Damage:    int32(le.Uint32(mob816[cs+8:])),
		MaxHp:     int32(le.Uint32(mob816[cs+16:])),
		Hp:        int32(le.Uint32(mob816[cs+24:])),
		Str:       int16(le.Uint16(mob816[cs+32:])),
		Int:       int16(le.Uint16(mob816[cs+34:])),
		Dex:       int16(le.Uint16(mob816[cs+36:])),
		Con:       int16(le.Uint16(mob816[cs+38:])),
		Resist:    [4]uint8(mob816[806:810]), // STRUCT_MOB.Resist @806 (mitigação de skill)
	}
}

// cstr16 corta um campo de nome fixo no primeiro NUL.
func cstr16(b []byte) string {
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}

// MobSnapshot é o subconjunto de STRUCT_MOB que o snapshot precisa. BaseScore espelha
// CurrentScore aqui (o mundo não os rastreia separadamente nesta fase).
type MobSnapshot struct {
	Name                 string
	Clan                 uint8
	Merchant             uint8
	Guild                uint16
	Class                uint8
	Quest                uint8
	Coin                 int32
	Exp                  int64
	SPX, SPY             int16
	Level                int32
	Ac, Damage           int32
	MaxHp, MaxMp, Hp, Mp int32
	Str, Int, Dex, Con   int16
	Special              [4]int16 // STRUCT_SCORE.Special @40 (maestria de árvore de skill)
	AttackRun            uint8    // STRUCT_SCORE.AttackRun @13 — o cliente lê a própria velocidade daqui e a grava em _MSG_Action.Speed
	Direction            uint8
	Equip                [16]SelItem
	Carry                [64]SelItem
	LearnedSkill         int32
	Magic                uint32
	ScoreBonus           uint16
	SpecialBonus         uint16
	SkillBonus           uint16
	Critical             uint8
	SkillBar             [4]uint8
	GuildLevel           uint8
	RegenHP, RegenMP     uint16
	Resist               [4]uint8
	PKPoint              uint8 // byte de caos do MobName[12]: 75 = nick neutro/branco, 0 = vermelho (só jogadores)
}

// writeMobScore escreve um STRUCT_SCORE de 48 bytes a partir de um MobSnapshot.
func writeMobScore(b []byte, m MobSnapshot) {
	le.PutUint32(b[0:], uint32(m.Level))  // Level @0
	le.PutUint32(b[4:], uint32(m.Ac))     // Ac @4
	le.PutUint32(b[8:], uint32(m.Damage)) // Damage @8
	b[12] = m.Merchant                    // Merchant @12
	b[13] = m.AttackRun                   // AttackRun @13 (speed)
	b[14] = m.Direction                   // Direction @14
	le.PutUint32(b[16:], uint32(m.MaxHp)) // MaxHp @16
	le.PutUint32(b[20:], uint32(m.MaxMp)) // MaxMp @20
	le.PutUint32(b[24:], uint32(m.Hp))    // Hp @24
	le.PutUint32(b[28:], uint32(m.Mp))    // Mp @28
	le.PutUint16(b[32:], uint16(m.Str))   // Str @32
	le.PutUint16(b[34:], uint16(m.Int))   // Int @34
	le.PutUint16(b[36:], uint16(m.Dex))   // Dex @36
	le.PutUint16(b[38:], uint16(m.Con))   // Con @38
	for i, v := range m.Special {         // Special[4] @40
		le.PutUint16(b[40+i*2:], uint16(v))
	}
}

// writeStructMob escreve um STRUCT_MOB de 816 bytes.
func writeStructMob(b []byte, m MobSnapshot) {
	PackMobName(b[0:16], m.Name, m.PKPoint, 0, 0) // MobName @0: name[0..11] + PK data[12..15]
	b[16] = m.Clan                                // Clan @16
	b[17] = m.Merchant                            // Merchant @17
	le.PutUint16(b[18:], m.Guild)                 // Guild @18
	b[20] = m.Class                               // Class @20
	b[24] = m.Quest                               // Quest @24
	le.PutUint32(b[28:], uint32(m.Coin))          // Coin @28
	le.PutUint64(b[32:], uint64(m.Exp))           // Exp @32
	le.PutUint16(b[40:], uint16(m.SPX))           // SPX @40
	le.PutUint16(b[42:], uint16(m.SPY))           // SPY @42
	writeMobScore(b[44:], m)                      // BaseScore @44
	writeMobScore(b[92:], m)                      // CurrentScore @92
	for i := 0; i < 16; i++ {                     // Equip[16] @140
		writeSelItem(b[140+i*8:], m.Equip[i])
	}
	for i := 0; i < 64; i++ { // Carry[64] @268
		writeSelItem(b[268+i*8:], m.Carry[i])
	}
	le.PutUint32(b[780:], uint32(m.LearnedSkill)) // @780
	le.PutUint32(b[784:], m.Magic)                // @784
	le.PutUint16(b[788:], m.ScoreBonus)           // @788
	le.PutUint16(b[790:], m.SpecialBonus)         // @790
	le.PutUint16(b[792:], m.SkillBonus)           // @792
	b[794] = m.Critical                           // @794
	copy(b[796:800], m.SkillBar[:])               // SkillBar[4] @796
	b[800] = m.GuildLevel                         // @800
	le.PutUint16(b[802:], m.RegenHP)              // @802
	le.PutUint16(b[804:], m.RegenMP)              // @804
	copy(b[806:810], m.Resist[:])                 // Resist[4] @806
}

// BaseMobSpawn devolve a posição de spawn (mob.SPX/SPY) guardada num STRUCT_MOB
// cru de template — as coordenadas de início válidas do servidor original.
func BaseMobSpawn(mob816 []byte) (x, y int16) {
	return int16(le.Uint16(mob816[40:42])), int16(le.Uint16(mob816[42:44]))
}

// SkillState é o bloco de skills persistido, sobrescrito sobre o template BaseMob no
// login (STRUCT_MOB @780-799 + STRUCT_SCORE.Special): a máscara de skills aprendidas,
// os pontos livres e a barra de skills. Sem o patch o cliente mostraria
// a janela de skills (vazia) do template, independente do que o personagem aprendeu.
type SkillState struct {
	LearnedSkill int32
	ScoreBonus   uint16
	SpecialBonus uint16
	SkillBonus   uint16
	Special      [4]int16 // CurrentScore.Special (base + equip)
	BaseSpecial  [4]int16 // BaseScore.Special (allocated)
	SkillBar     [4]uint8
}

// EncodeCNFCharacterLoginRaw monta o snapshot a partir de um STRUCT_MOB cru de 816
// bytes (um template BaseMob por classe, que já carrega stats válidos, equipamento
// inicial e skills), sobrescrevendo o estado específico do personagem. loginX/loginY
// são a posição real de entrada no mundo em MSG_CNFCharacterLogin.PosX/PosY; o
// mob.SPX/SPY embutido continua sendo o ponto de save, igual à reescrita de spawn
// tx/ty posterior em ProcessDBMessage.cpp. Quando saveX/saveY são zero, o SPX/SPY
// do template é preservado. pkPoint é empacotado em MobName[12] para a cor do próprio nick.
func EncodeCNFCharacterLoginRaw(mob816 []byte, name string, coin int32, exp int64, equip [16]SelItem, carry [64]SelItem, loginX, loginY, saveX, saveY int16, slot, clientID int, weather uint16, shortSkill [16]uint8, skill SkillState, pkPoint uint8) []byte {
	b := make([]byte, cnfCharacterLoginSize-HeaderSize) // 1820
	copy(b[4:4+structMobSize], mob816)                  // mob @ body4 (raw template)
	// MobName @ corpo4: nome em [0..11] + dados de PK em [12..15]. O cliente colora o
	// próprio nick a partir de MobName[12] (PKPoint: 75 = neutro/branco, 0 = vermelho
	// piscando). Empacotá-lo aqui evita um flash vermelho antes do próprio CreateMob
	// arrives (GetFunc.cpp:1082-1101).
	PackMobName(b[4:4+16], name, pkPoint, 0, 0)
	// Sobrepõe o equipamento persistido no Equip@140 do template, para o
	// equipamento salvo (não o conjunto inicial da classe) aparecer no modelo e na janela de equip.
	for i := 0; i < 16; i++ {
		writeSelItem(b[4+structMobEquip+i*8:], equip[i])
	}
	// Sobrepõe o inventário persistido no Carry@268 (mob) do template, para as
	// compras salvas aparecerem no relog (o template traz um Carry vazio/estoque).
	for i := 0; i < 64; i++ {
		writeSelItem(b[4+structMobCarry+i*8:], carry[i])
	}
	// O template BaseMob é um dump cru de memória com padding 0xCC não inicializado em
	// Quest@24/pad@25-27, que o cliente lê como um campo de ouro de 4 bytes → o
	// -858993664 (0xCCCCCC00) "ouro negativo" (B2). Defina o ouro real nos dois
	// offsets candidatos (24 = o que o cliente exibe, 28 = STRUCT_MOB.Coin).
	le.PutUint32(b[4+24:], uint32(coin))
	le.PutUint32(b[4+28:], uint32(coin))
	// Experiência persistida → STRUCT_MOB.Exp@32, para a barra de exp do cliente refletir o
	// total salvo no login (o template traz Exp=0). O cliente é dono da curva de
	// nível e desenha a barra a partir deste valor cru.
	le.PutUint64(b[4+32:], uint64(exp))
	// O servidor legado mantém mob.SPX/SPY como ponto de save e depois sobrescreve
	// só MSG_CNFCharacterLogin.PosX/PosY com o spawn real tx/ty. Manter
	// esses papéis separados impede cliente e servidor de começarem em
	// tiles diferentes após o login.
	le.PutUint16(b[0:], uint16(loginX)) // PosX @ body0
	le.PutUint16(b[2:], uint16(loginY)) // PosY @ body2
	if saveX != 0 || saveY != 0 {
		le.PutUint16(b[4+40:], uint16(saveX))
		le.PutUint16(b[4+42:], uint16(saveY))
	}
	// Estado de skills persistido sobre o do template: máscara aprendida + pontos livres
	// @780-793, barra de skills @796, e a maestria Special nos dois scores @+40
	// (BaseScore @44, CurrentScore @92). O template traz tudo zero aqui.
	le.PutUint32(b[4+780:], uint32(skill.LearnedSkill))
	le.PutUint16(b[4+788:], skill.ScoreBonus)
	le.PutUint16(b[4+790:], skill.SpecialBonus)
	le.PutUint16(b[4+792:], skill.SkillBonus)
	copy(b[4+796:4+800], skill.SkillBar[:])
	for i := 0; i < 4; i++ {
		le.PutUint16(b[4+44+40+i*2:], uint16(skill.BaseSpecial[i]))
		le.PutUint16(b[4+92+40+i*2:], uint16(skill.Special[i]))
	}
	le.PutUint16(b[1028:], uint16(slot))
	le.PutUint16(b[1030:], uint16(clientID))
	le.PutUint16(b[1032:], weather)
	copy(b[1034:1050], shortSkill[:])
	return b
}

// OverlayLoginScores escreve o BaseScore (@mob+44) e o
// CurrentScore (@mob+92) do personagem num corpo MSG_CNFCharacterLogin montado de
// um template de classe. O legado envia pMob[conn].MOB logo após GetCurrentScore
// (ProcessDBMessage.cpp:812-821), então o cliente começa com o nível, HP/MP e
// atributos reais em vez dos do template até o próximo UpdateScore.
// Merchant@12 e Direction@14 mantêm os bytes do template.
func OverlayLoginScores(body []byte, base, current ScoreData) {
	writeScoreData(body[4+44:4+92], base)
	writeScoreData(body[4+92:4+140], current)
}

// writeScoreData escreve os campos de STRUCT_SCORE que ScoreData carrega (48 bytes).
func writeScoreData(b []byte, s ScoreData) {
	le.PutUint32(b[0:], uint32(s.Level))
	le.PutUint32(b[4:], uint32(s.Ac))
	le.PutUint32(b[8:], uint32(s.Damage))
	b[13] = s.AttackRun
	le.PutUint32(b[16:], uint32(s.MaxHp))
	le.PutUint32(b[20:], uint32(s.MaxMp))
	le.PutUint32(b[24:], uint32(s.Hp))
	le.PutUint32(b[28:], uint32(s.Mp))
	le.PutUint16(b[32:], uint16(s.Str))
	le.PutUint16(b[34:], uint16(s.Int))
	le.PutUint16(b[36:], uint16(s.Dex))
	le.PutUint16(b[38:], uint16(s.Con))
	for i, sp := range s.Special {
		le.PutUint16(b[40+i*2:], uint16(sp))
	}
}

// updateEquipSize é MSG_UpdateEquip (0x036B): HEADER + ushort Equip[16] +
// uchar AnctCode[16] = 12 + 32 + 16 = 60 bytes.
const updateEquipSize = 60

// EncodeUpdateEquip monta _MSG_UpdateEquip (0x006B): os 16 códigos de equipamento
// visíveis que dirigem o equipamento renderizado (SendFunc.cpp:SendEquip). visual
// guarda o código visual por slot (0 = slot vazio); anct guarda os bytes
// correspondentes de brilho ancient/refine. Envie com HEADER.ID = o id da entidade
// para valer no mob certo (o próprio jogador ou um jogador visível).
func EncodeUpdateEquip(visual [16]uint16, anct [16]uint8) []byte {
	b := make([]byte, updateEquipSize-HeaderSize) // 48
	for i := 0; i < 16; i++ {
		le.PutUint16(b[i*2:], visual[i]) // Equip[16] @body0
	}
	copy(b[32:48], anct[:]) // AnctCode[16] @body32
	return b
}

// EncodeCNFCharacterLoginBody monta o corpo de MSG_CNFCharacterLogin (0x0114):
// o snapshot no mundo. loginX/loginY são a posição de entrada do pacote; m.SPX/SPY
// continua como o ponto de save embutido. Envie com HEADER.ID = IDScene (30000).
func EncodeCNFCharacterLoginBody(slot, clientID int, weather uint16, loginX, loginY int16, m MobSnapshot, shortSkill [16]uint8) []byte {
	b := make([]byte, cnfCharacterLoginSize-HeaderSize) // 1820
	le.PutUint16(b[0:], uint16(loginX))                 // PosX @ abs12 → body0
	le.PutUint16(b[2:], uint16(loginY))                 // PosY @ abs14 → body2
	writeStructMob(b[4:4+structMobSize], m)             // mob @ abs16 → body4
	le.PutUint16(b[1028:], uint16(slot))                // Slot @ abs1040 → body1028
	le.PutUint16(b[1030:], uint16(clientID))            // ClientID @ abs1042 → body1030
	le.PutUint16(b[1032:], weather)                     // Weather @ abs1044 → body1032
	copy(b[1034:1050], shortSkill[:])                   // ShortSkill[16] @ abs1046 → body1034
	return b
}
