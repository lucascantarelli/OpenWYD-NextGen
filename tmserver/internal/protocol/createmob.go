package protocol

// Pacotes de entidades em vista (byte a byte contra Basedef.h, verificado pelo compilador).
//
//   MSG_CreateMob (0x0364, 232B): gera um jogador/NPC na visão de um cliente. O HEADER.ID
//     é ESCENE_FIELD (30000) — o id da entidade vai no campo MobID (@16).
//   MSG_RemoveMob (0x0165, 16B): despawn. HEADER.ID É o id da entidade; RemoveType
//     (@12) = 0 fora de vista / 1 morte / 2 logout.

const (
	createMobSize = 232
	removeMobSize = 16
)

// CreateMobData é o subconjunto de MSG_CreateMob necessário para renderizar uma
// entidade em vista. Equip guarda códigos VISUAIS de item (u16[16]); Affect guarda os
// slots de ícone/visual de GetAffect; AnctCode guarda os bytes correspondentes de
// overlay bytes. 0 = empty/no overlay.
type CreateMobData struct {
	MobID                int
	Name                 string
	PosX, PosY           int16
	Guild                uint16
	GuildMemberType      uint8
	Level, Ac, Damage    int32
	MaxHp, MaxMp, Hp, Mp int32
	Str, Int, Dex, Con   int16
	Merchant             uint8 // NPC type (shop/bank/…); makes the name always-visible
	AttackRun            uint8 // speed byte (run<<4 | move): the client animates this entity's walks with it — 0 = crawling avatar that rubber-bands
	Direction            uint8
	CreateType           uint16 // 0 normal, 2 "just entered"
	Equip                [16]uint16
	Affect               [MaxAffect]uint16
	AnctCode             [16]uint8
	// IsPlayer escolhe a codificação do MobName: jogadores (id < MAX_USER) empacotam
	// dados de PK em MobName[12..15]; mobs/NPCs enviam o nome cru de 16 bytes (seus nomes
	// podem ter 16 caracteres, por exemplo "Ciclope_Arqueiro", e não têm coloração de PK).
	IsPlayer bool
	// PKPoint é o nível de caos/PK empacotado em MobName[12] (só jogadores). NEUTRAL
	// é 75 (nick branco); 0 = vermelho piscando (caos/culpado); <75 = caos. Ver
	// PackMobName (GetFunc.cpp:1082-1101).
	PKPoint uint8
	CurKill uint8  // MobName[13]
	TotKill uint16 // MobName[14..15]
}

func writeCreateMobScore(b []byte, d CreateMobData) {
	le.PutUint32(b[0:], uint32(d.Level))
	le.PutUint32(b[4:], uint32(d.Ac))
	le.PutUint32(b[8:], uint32(d.Damage))
	b[12] = d.Merchant  // STRUCT_SCORE.Merchant — NPC type
	b[13] = d.AttackRun // STRUCT_SCORE.AttackRun — speed
	b[14] = d.Direction
	// b[15] STRUCT_SCORE.ChaosRate não é usado pelo servidor (o NPTool o edita como
	// campo de regen; ele NÃO dirige a cor do nick — isso é MobName[12]). Deixe 0.
	le.PutUint32(b[16:], uint32(d.MaxHp))
	le.PutUint32(b[20:], uint32(d.MaxMp))
	le.PutUint32(b[24:], uint32(d.Hp))
	le.PutUint32(b[28:], uint32(d.Mp))
	le.PutUint16(b[32:], uint16(d.Str))
	le.PutUint16(b[34:], uint16(d.Int))
	le.PutUint16(b[36:], uint16(d.Dex))
	le.PutUint16(b[38:], uint16(d.Con))
}

// PackMobName escreve o campo MobName de 16 bytes para um JOGADOR: só os primeiros
// 12 bytes guardam o nome; os bytes [12..15] carregam os dados de PK/caos que o
// cliente lê para colorir o nick (GetFunc.cpp:1082-1101). MobName[12] = PKPoint (75 =
// white, 0 = red blinking), [13] = curkill, [14..15] = totkill (little-endian).
// NPCs/mobs (id ≥ MAX_USER) passam pkPoint=0 curkill=0 totkill=0 — o legado só
// empacota isso para id < MAX_USER, e o nome de um mob não tem coloração de PK.
func PackMobName(dst []byte, name string, pkPoint, curKill uint8, totKill uint16) {
	for i := 0; i < 16; i++ {
		dst[i] = 0
	}
	nb := []byte(name)
	if len(nb) > 12 {
		nb = nb[:12]
	}
	copy(dst[0:12], nb)
	dst[12] = pkPoint
	dst[13] = curKill
	dst[14] = byte(totKill)
	dst[15] = byte(totKill >> 8)
}

// EncodeCreateMobBody monta o corpo (depois do cabeçalho de 12 bytes) de
// MSG_CreateMob (0x0364). Envie com HEADER.ID = IDScene (30000); o id da entidade é MobID.
func EncodeCreateMobBody(d CreateMobData) []byte {
	b := make([]byte, createMobSize-HeaderSize) // 220
	le.PutUint16(b[0:], uint16(d.PosX))         // PosX @abs12 → body0
	le.PutUint16(b[2:], uint16(d.PosY))         // PosY @abs14 → body2
	le.PutUint16(b[4:], uint16(d.MobID))        // MobID @abs16 → body4
	// MobName @abs18 → corpo6. Jogadores: nome em [0..11] + dados de PK em [12..15] (cores
	// o nick). Mobs/NPCs: nome cru completo de 16 bytes (sem empacotamento de PK).
	if d.IsPlayer {
		PackMobName(b[6:6+16], d.Name, d.PKPoint, d.CurKill, d.TotKill)
	} else {
		copy(b[6:6+16], d.Name)
	}
	for i := 0; i < 16; i++ { // Equip[16] @abs34 → body22
		le.PutUint16(b[22+i*2:], d.Equip[i])
	}
	for i := 0; i < MaxAffect; i++ { // Affect[32] @abs66 → body54
		le.PutUint16(b[54+i*2:], d.Affect[i])
	}
	le.PutUint16(b[118:], d.Guild)      // Guild @abs130 → body118
	b[120] = d.GuildMemberType          // GuildMemberType @abs132 → body120
	writeCreateMobScore(b[124:], d)     // Score @abs136 → body124
	le.PutUint16(b[172:], d.CreateType) // CreateType @abs184 → body172
	copy(b[174:190], d.AnctCode[:])     // AnctCode[16] @abs186 → body174
	return b
}

// EncodeRemoveMobBody monta o corpo de MSG_RemoveMob (0x0165). Envie com
// HEADER.ID = o id da entidade. removeType: 0 fora de vista, 1 morte, 2 logout.
func EncodeRemoveMobBody(removeType int32) []byte {
	b := make([]byte, removeMobSize-HeaderSize) // 4
	le.PutUint32(b[0:], uint32(removeType))
	return b
}
