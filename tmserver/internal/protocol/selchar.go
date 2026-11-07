package protocol

// Structs da tela de personagem S→C. Tamanhos e offsets são byte a byte contra o
// Basedef.h original, verificados por sonda de sizeof/offsetof do compilador no
// código-fonte (alinhamento natural MSVC x86, NÃO pack(1)). Ver as notas de captura da migração.
//
// Framing de wire: Encode prefixa o HEADER de 12 bytes e ofusca [4:Size); os
// construtores abaixo devolvem o CORPO (tudo depois do cabeçalho), então seus
// offsets são o offset absoluto da struct menos HeaderSize.

const (
	structSelCharSize = 840 // STRUCT_SELCHAR

	// cnfAccountLoginSize é MSG_DBCNFAccountLogin (0x010A): SELCHAR + cargo da conta.
	cnfAccountLoginSize = 2008
	// cnfNewCharacterSize é MSG_CNFNewCharacter (0x0110): reenvia a lista SELCHAR.
	cnfNewCharacterSize = 856

	// IDs de Header que o original define para estes pacotes (ESCENE_FIELD = 30000).
	IDSelChar      = 30002 // CNFAccountLogin
	IDNewCharacter = 30001 // CNFNewCharacter
	IDScene        = 30000 // CNFCharacterLogin / AccountSecure ack
)

// SelChar são os dados por slot que a tela de seleção precisa de um
// personagem (uma linha STRUCT_SELCHAR). Campos não definidos ficam zero no wire.
type SelChar struct {
	Slot                 int
	Name                 string
	SPX, SPY             int16
	Level                int32
	MaxHp, Hp, MaxMp, Mp int32
	Str, Int, Dex, Con   int16
	Direction            uint8
	Guild                uint16
	Coin                 int32
	Exp                  int64
	Equip                [16]SelItem
}

// SelItem é um STRUCT_ITEM (8 bytes): sIndex + 3×{effect,value}.
type SelItem struct {
	Index uint16
	Eff   [3][2]uint8
}

func writeSelItem(b []byte, it SelItem) {
	le.PutUint16(b[0:], it.Index)
	for i := 0; i < 3; i++ {
		b[2+i*2] = it.Eff[i][0]
		b[3+i*2] = it.Eff[i][1]
	}
}

// writeStructScore escreve uma prévia de 48 bytes de STRUCT_SCORE (nível/HP/MP/stats).
func writeStructScore(b []byte, c SelChar) {
	le.PutUint32(b[0:], uint32(c.Level))  // Level @0
	b[14] = c.Direction                   // Direction @14
	le.PutUint32(b[16:], uint32(c.MaxHp)) // MaxHp @16
	le.PutUint32(b[20:], uint32(c.MaxMp)) // MaxMp @20
	le.PutUint32(b[24:], uint32(c.Hp))    // Hp @24
	le.PutUint32(b[28:], uint32(c.Mp))    // Mp @28
	le.PutUint16(b[32:], uint16(c.Str))   // Str @32
	le.PutUint16(b[34:], uint16(c.Int))   // Int @34
	le.PutUint16(b[36:], uint16(c.Dex))   // Dex @36
	le.PutUint16(b[38:], uint16(c.Con))   // Con @38
}

// writeSelChar preenche um STRUCT_SELCHAR de 840 bytes (4 slots) a partir de chars (por Slot).
func writeSelChar(b []byte, chars []SelChar) {
	for _, c := range chars {
		s := c.Slot
		if s < 0 || s >= 4 {
			continue
		}
		le.PutUint16(b[0+s*2:], uint16(c.SPX)) // SPX[4] @0
		le.PutUint16(b[8+s*2:], uint16(c.SPY)) // SPY[4] @8
		no := 16 + s*16                        // Name[4][16] @16
		copy(b[no:no+16], c.Name)
		writeStructScore(b[80+s*48:], c) // Score[4] @80
		eq := 272 + s*128                // Equip[4][16] @272 (16 items × 8B)
		for i := range c.Equip {
			writeSelItem(b[eq+i*8:], c.Equip[i])
		}
		le.PutUint16(b[784+s*2:], c.Guild)        // Guild[4] @784
		le.PutUint32(b[792+s*4:], uint32(c.Coin)) // Coin[4] @792
		le.PutUint64(b[808+s*8:], uint64(c.Exp))  // Exp[4] @808
	}
}

// MobEquip lê os 16 slots de equipamento (STRUCT_ITEM[16] @140) de um
// template STRUCT_MOB cru em SelItems, para a tela de seleção pré-visualizar a
// classe de um personagem com seu equipamento inicial (senão o cliente desenha o
// default TransKnight model).
func MobEquip(mob816 []byte) [16]SelItem {
	var eq [16]SelItem
	for i := 0; i < 16; i++ {
		o := 140 + i*8
		eq[i].Index = le.Uint16(mob816[o : o+2])
		eq[i].Eff[0] = [2]uint8{mob816[o+2], mob816[o+3]}
		eq[i].Eff[1] = [2]uint8{mob816[o+4], mob816[o+5]}
		eq[i].Eff[2] = [2]uint8{mob816[o+6], mob816[o+7]}
	}
	return eq
}

const (
	cnfCargoSlots   = 128
	cnfCargoBodyOff = 860  // Cargo[128] @ abs872
	cnfCoinBodyOff  = 1884 // Coin @ abs1896
	cnfNameBodyOff  = 1888 // AccountName @ abs1900
)

// EncodeCNFAccountLoginBody monta o corpo de MSG_CNFAccountLogin (0x010A):
// SELCHAR + account Cargo[128] + Coin + AccountName (Basedef.h MSG_DBCNFAccountLogin,
// repassado ao cliente como está). O cliente cacheia este cargo para a UI do cofre;
// omiti-lo deixa o armazém vazio mesmo quando o servidor tem itens. Envie com
// HEADER.ID = IDSelChar (30002).
func EncodeCNFAccountLoginBody(accountName string, chars []SelChar, coin int32, cargo [cnfCargoSlots]SelItem) []byte {
	b := make([]byte, cnfAccountLoginSize-HeaderSize) // 1996
	le.PutUint32(b[16:], 1)                           // Unknow_28=1 (don't recreate starter potions)
	writeSelChar(b[20:20+structSelCharSize], chars)   // sel @ abs32 → body20
	for i := 0; i < cnfCargoSlots; i++ {
		writeSelItem(b[cnfCargoBodyOff+i*8:], cargo[i])
	}
	le.PutUint32(b[cnfCoinBodyOff:], uint32(coin))
	copy(b[cnfNameBodyOff:cnfNameBodyOff+16], accountName)
	return b
}

// EncodeCNFNewCharacterBody monta o corpo de MSG_CNFNewCharacter (0x0110): ele
// reenvia a lista SELCHAR completa (agora com o personagem novo). sel fica no
// offset absoluto 16 (4 bytes de padding seguem o cabeçalho). Envie com ID = IDNewCharacter.
func EncodeCNFNewCharacterBody(chars []SelChar) []byte {
	b := make([]byte, cnfNewCharacterSize-HeaderSize) // 844
	writeSelChar(b[4:4+structSelCharSize], chars)     // sel @ abs16 → body4
	return b
}
