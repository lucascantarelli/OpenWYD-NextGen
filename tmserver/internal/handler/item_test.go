package handler

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"

	"openwyd-nextgen/tmserver/internal/content"
	"openwyd-nextgen/tmserver/internal/level"
	"openwyd-nextgen/tmserver/internal/protocol"
	"openwyd-nextgen/tmserver/internal/world"
)

func TestMeetsEquipReq(t *testing.T) {
	d := New(Config{ItemReqs: map[int]content.ItemReq{
		900: {Lvl: 100, Str: 50}, // uma espada pesada
	}})
	sword := world.Item{Index: 900}
	if d.meetsEquipReq(&world.Entity{ClassMaster: classMasterMortal, Level: 99, Str: 60}, sword) {
		t.Error("equip allowed below the level requirement")
	}
	if d.meetsEquipReq(&world.Entity{ClassMaster: classMasterMortal, Level: 100, Str: 49}, sword) {
		t.Error("equip allowed below the str requirement")
	}
	if !d.meetsEquipReq(&world.Entity{ClassMaster: classMasterMortal, Level: 100, Str: 50}, sword) {
		t.Error("equip rejected when requirements are met")
	}
	if !d.meetsEquipReq(&world.Entity{ClassMaster: classMasterMortal}, world.Item{Index: 1}) {
		t.Error("an item with no catalog requirement must always pass")
	}
	// Basedef.cpp:5024-5028 (BASE_CanEquip): o requisito de nível/atributo só
	// condiciona MORTAIS — personagens Arch e Celestial o ignoram por completo, mesmo muito
	// abaixo do requisito do item (issue #209).
	if !d.meetsEquipReq(&world.Entity{ClassMaster: classMasterArch, Level: 1, Str: 1}, sword) {
		t.Error("arch character blocked by an equip requirement it should bypass")
	}
	if !d.meetsEquipReq(&world.Entity{ClassMaster: classMasterCelestial, Level: 1, Str: 1}, sword) {
		t.Error("celestial character blocked by an equip requirement it should bypass")
	}
}

// TestMeetsEquipReqClasseD fixa o gate contra o equipamento de classe D reescalado da
// issue #278: Tunica_de_Mytril(A) (ItemList.csv:1346) passou de nível 254 para 174,
// então um mortal de nível 174 com os atributos do conjunto precisa poder vesti-la.
func TestMeetsEquipReqClasseD(t *testing.T) {
	d := New(Config{ItemReqs: map[int]content.ItemReq{
		1346: {Lvl: 174, Str: 117, Int: 205},
	}})
	tunic := world.Item{Index: 1346}
	foema := func(lvl int32) *world.Entity {
		return &world.Entity{ClassMaster: classMasterMortal, Level: lvl, Str: 117, Int: 205}
	}
	if d.meetsEquipReq(foema(173), tunic) {
		t.Error("Tunica_de_Mytril(A) equipped one level below its requirement")
	}
	if !d.meetsEquipReq(foema(174), tunic) {
		t.Error("Tunica_de_Mytril(A) rejected at level 174, the rescaled requirement")
	}
}

func itemDB(carry0 int16) *fakeDB {
	db := newDB()
	st := world.CharacterState{Slot: 0, Name: "Hero", X: 5, Y: 5, HP: 1000, MaxHP: 1000}
	st.Carry[0] = world.Item{Index: carry0}
	db.loadResult = st
	return db
}

func dropFrame(t *testing.T, c net.Conn, sourPos int, gx, gy uint16) {
	t.Helper()
	body := protocol.MsgDropItemBody{SourType: world.ItemPlaceCarry, SourPos: int32(sourPos), GridX: gx, GridY: gy}
	send(t, c, protocol.MsgDropItem, body.Encode())
}

func getFrame(t *testing.T, c net.Conn, itemID int32, destPos int) {
	t.Helper()
	body := protocol.MsgGetItemBody{ItemID: itemID, DestType: world.ItemPlaceCarry, DestPos: int32(destPos)}
	send(t, c, protocol.MsgGetItem, body.Encode())
}

func TestDropAndGet(t *testing.T) {
	addr, stop, _ := startServerClock(t, itemDB(1100))
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	dropFrame(t, c, 0, 5, 5)
	if ty, _, ok := readMaybe(t, c); !ok || ty != protocol.MsgCNFDropItem {
		t.Fatalf("drop got %#x ok=%v, want CNFDropItem", ty, ok)
	}

	// O primeiro item de chão recebe id 1 ⇒ ItemID 10001 no wire.
	getFrame(t, c, world.GroundItemIDOffset+1, 0)
	if ty, _, ok := readMaybe(t, c); !ok || ty != protocol.MsgCNFGetItem {
		t.Errorf("get got %#x ok=%v, want CNFGetItem", ty, ok)
	}
}

func TestDropBlacklisted(t *testing.T) {
	addr, stop, _ := startServerClock(t, itemDB(508)) // 508 é não largável
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	dropFrame(t, c, 0, 5, 5)
	if ty, _, ok := readMaybe(t, c); ok {
		t.Errorf("blacklisted drop produced %#x; should be blocked", ty)
	}
}

func TestGetDecayed(t *testing.T) {
	addr, stop, _ := startServerClock(t, itemDB(1100))
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	// Nada largado ainda ⇒ id de chão vazio ⇒ DecayItem.
	getFrame(t, c, world.GroundItemIDOffset+1, 0)
	if ty, _, ok := readMaybe(t, c); !ok || ty != protocol.MsgDecayItem {
		t.Errorf("get got %#x ok=%v, want DecayItem", ty, ok)
	}
}

func TestGetTooFar(t *testing.T) {
	addr, stop, _ := startServerClock(t, itemDB(1100))
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	dropFrame(t, c, 0, 15, 15) // o jogador está em (5,5); o item está a >3 células
	if ty, _, ok := readMaybe(t, c); !ok || ty != protocol.MsgCNFDropItem {
		t.Fatalf("drop got %#x ok=%v", ty, ok)
	}
	getFrame(t, c, world.GroundItemIDOffset+1, 0)
	if ty, _, ok := readMaybe(t, c); ok {
		t.Errorf("far get produced %#x; should be rejected", ty)
	}
}

// TestDupRace prova a reivindicação atômica: duas coletas do mesmo item de chão ⇒ exatamente
// um CNFGetItem, a outra DecayItem (o loop as serializa).
func TestDupRace(t *testing.T) {
	addr, stop, _ := startServerClock(t, itemDB(1100))
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	dropFrame(t, c, 0, 5, 5)
	if ty, _, ok := readMaybe(t, c); !ok || ty != protocol.MsgCNFDropItem {
		t.Fatalf("drop got %#x ok=%v", ty, ok)
	}

	getFrame(t, c, world.GroundItemIDOffset+1, 0)
	getFrame(t, c, world.GroundItemIDOffset+1, 1)

	ty1, _, _ := readMaybe(t, c)
	ty2, _, _ := readMaybe(t, c)
	got := map[protocol.Type]int{ty1: 1}
	got[ty2]++
	if got[protocol.MsgCNFGetItem] != 1 || got[protocol.MsgDecayItem] != 1 {
		t.Errorf("dup race responses = %#x, %#x; want one CNFGetItem + one DecayItem", ty1, ty2)
	}
}

// equipDB semeia o personagem de teste com um item opcional no carry 0 e um
// item opcional no slot de equipamento 1 (para exercitar equipar e desequipar).
func equipDB(carry0, equip1 int16) *fakeDB {
	db := newDB()
	st := world.CharacterState{Slot: 0, Name: "Hero", X: 5, Y: 5, HP: 1000, MaxHP: 1000}
	if carry0 != 0 {
		st.Carry[0] = world.Item{Index: carry0}
	}
	if equip1 != 0 {
		st.Equip[1] = world.Item{Index: equip1}
	}
	db.loadResult = st
	return db
}

func equipItemDB(carry0 world.Item) *fakeDB {
	db := newDB()
	st := world.CharacterState{Slot: 0, Name: "Hero", X: 5, Y: 5, HP: 1000, MaxHP: 1000}
	st.Carry[0] = carry0
	db.loadResult = st
	return db
}

// TestTradingItemEquip arrasta um item do inventário para um slot de equipamento (0x0376) e
// verifica que o equipamento renderizado é atualizado via _MSG_UpdateEquip com o código do item.
func TestTradingItemEquip(t *testing.T) {
	addr, stop, _ := startServerClock(t, equipDB(1100, 0))
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	tradeItemFrame(t, c, world.ItemPlaceCarry, 0, world.ItemPlaceEquip, 1, 0)
	expect(t, c, protocol.MsgTradingItem)
	expect(t, c, protocol.MsgSendItem)
	expect(t, c, protocol.MsgSendItem)
	ue := expect(t, c, protocol.MsgUpdateEquip)
	if got := le16(ue[2:4]); got != 1100 { // código visual de Equip[1] @body2
		t.Errorf("equip visual[1] = %d, want 1100", got)
	}
}

func TestTradingItemEquipSendsRefineGlow(t *testing.T) {
	item := world.Item{Index: 1100, Effects: [3]world.Effect{{Effect: efSanc, Value: 9}}}
	addr, stop, _ := startServerClock(t, equipItemDB(item))
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	tradeItemFrame(t, c, world.ItemPlaceCarry, 0, world.ItemPlaceEquip, 1, 0)
	expect(t, c, protocol.MsgTradingItem)
	expect(t, c, protocol.MsgSendItem)
	expect(t, c, protocol.MsgSendItem)
	ue := expect(t, c, protocol.MsgUpdateEquip)
	if want, got := uint16(1100|9*0x1000), le16(ue[2:4]); got != want {
		t.Errorf("equip visual[1] = %d, want %d (+9 overlay)", got, want)
	}
	if got := ue[33]; got != efSanc { // AnctCode[1] @body32+1
		t.Errorf("equip anct[1] = %#x, want EF_SANC", got)
	}
}

// TestTradingItemUnequip prova que o equipamento é carregado do banco (para o slot
// ter algo a remover) e desequipar limpa o equipamento renderizado.
func TestTradingItemUnequip(t *testing.T) {
	addr, stop, _ := startServerClock(t, equipDB(0, 2200))
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	tradeItemFrame(t, c, world.ItemPlaceEquip, 1, world.ItemPlaceCarry, 5, 0)
	expect(t, c, protocol.MsgTradingItem)
	expect(t, c, protocol.MsgSendItem) // slot de equipamento 1 (agora vazio)
	expect(t, c, protocol.MsgSendItem) // slot de carry 5 (agora contém o item)
	ue := expect(t, c, protocol.MsgUpdateEquip)
	if got := le16(ue[2:4]); got != 0 { // Equip[1] agora vazio
		t.Errorf("equip visual[1] = %d, want 0 after unequip", got)
	}
}

// weaponAutoShiftDB semeia o personagem de teste com um único item no
// slot de carry 0, para exercitar a troca automática de mão no equipar rápido.
func weaponAutoShiftDB(item int16) *fakeDB {
	db := newDB()
	st := world.CharacterState{Slot: 0, Name: "Hero", X: 5, Y: 5, HP: 1000, MaxHP: 1000}
	st.Carry[0] = world.Item{Index: item}
	db.loadResult = st
	return db
}

// startServerClockItemPos é o startServerClock com um catálogo ItemPos injetado,
// necessário para exercitar o gate de slot de equipamento (canEquipSlot / shiftWeaponToRightHand).
func startServerClockItemPos(t *testing.T, persist world.Persistence, itemPos map[int]int) (string, func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	d := New(Config{Log: log, ItemPos: itemPos})
	w := world.New(world.Config{GridDim: 16}, log, persist, d.Handle)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { _ = w.Serve(ctx, ln); close(done) }()
	return ln.Addr().String(), func() {
		cancel()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Error("server did not stop")
		}
	}
}

// TestTradingItemWeaponAutoShift prova a correção da issue #65: equipar rápido uma
// arma de uma mão (nPos 192, cabe em qualquer mão) no slot da mão secundária (7)
// com a mão principal (6) vazia faz ela ser movida automaticamente para o slot 6, espelhando
// _MSG_TradingItem.cpp:394-414.
func TestTradingItemWeaponAutoShift(t *testing.T) {
	addr, stop := startServerClockItemPos(t, weaponAutoShiftDB(1100), map[int]int{1100: 192})
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	tradeItemFrame(t, c, world.ItemPlaceCarry, 0, world.ItemPlaceEquip, 7, 0)
	expect(t, c, protocol.MsgTradingItem) // eco da troca primária
	expect(t, c, protocol.MsgSendItem)    // slot de carry 0, agora vazio
	expect(t, c, protocol.MsgSendItem)    // slot de equipamento 7, agora contém 1100

	expect(t, c, protocol.MsgTradingItem) // eco da troca automática: 7 -> 6
	expect(t, c, protocol.MsgSendItem)    // slot de equipamento 6, agora contém 1100
	expect(t, c, protocol.MsgSendItem)    // slot de equipamento 7, agora vazio de novo

	ue := expect(t, c, protocol.MsgUpdateEquip)
	if got := le16(ue[12:14]); got != 1100 { // código visual de Equip[6]
		t.Errorf("equip visual[6] = %d, want 1100 after auto-shift", got)
	}
	if got := le16(ue[14:16]); got != 0 { // código visual de Equip[7]
		t.Errorf("equip visual[7] = %d, want 0 after auto-shift", got)
	}
}

// TestTradingItemShieldNotShifted prova que um escudo de verdade (nPos 128, só mão
// secundária) fica isento da troca automática: precisa continuar no slot 7 mesmo com o slot 6
// vazio (guarda hab != 128 de _MSG_TradingItem.cpp:400).
func TestTradingItemShieldNotShifted(t *testing.T) {
	addr, stop := startServerClockItemPos(t, weaponAutoShiftDB(2200), map[int]int{2200: 128})
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	tradeItemFrame(t, c, world.ItemPlaceCarry, 0, world.ItemPlaceEquip, 7, 0)
	expect(t, c, protocol.MsgTradingItem) // eco da troca primária
	expect(t, c, protocol.MsgSendItem)    // slot de carry 0, agora vazio
	expect(t, c, protocol.MsgSendItem)    // slot de equipamento 7, agora contém 2200

	ue := expect(t, c, protocol.MsgUpdateEquip)
	if got := le16(ue[14:16]); got != 2200 { // código visual de Equip[7]
		t.Errorf("equip visual[7] = %d, want 2200 (shield stays put)", got)
	}
	if got := le16(ue[12:14]); got != 0 { // código visual de Equip[6]
		t.Errorf("equip visual[6] = %d, want 0 (nothing shifted)", got)
	}
}

func TestUseItemEquip(t *testing.T) {
	addr, stop, _ := startServerClock(t, itemDB(1100))
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	body := protocol.MsgUseItemBody{
		SourType: world.ItemPlaceCarry, SourPos: 0,
		DestType: world.ItemPlaceEquip, DestPos: 0,
	}
	send(t, c, protocol.MsgUseItem, body.Encode())
	if ty, _, ok := readMaybe(t, c); !ok || ty != protocol.MsgUseItem {
		t.Errorf("equip got %#x ok=%v, want UseItem echo", ty, ok)
	}
}

func territorioDB(item int16, classMaster uint8) *fakeDB {
	db := newDB()
	st := world.CharacterState{Slot: 0, Name: "Hero", X: 5, Y: 5, HP: 1000, MaxHP: 1000, ClassMaster: classMaster}
	st.Carry[0] = world.Item{Index: item}
	db.loadResult = st
	return db
}

// TestUseEntradaTerritorio cobre os tickets Vol-188 "Entrada do Território" (LAN)
// (issue #203, _MSG_UseItem.cpp:4202-4236): o tier de classe certo teleporta
// para as coordenadas Lan_N/M/A correspondentes (centro ± [-3,+1] por eixo) e consome um;
// o tier errado é recusado com ressincronização de slot e sem teleporte.
func TestUseEntradaTerritorio(t *testing.T) {
	tests := []struct {
		name         string
		item         int16
		classMaster  uint8
		wantTeleport bool
		wantX, wantY int16
	}{
		{"normal_mortal", 4111, classMasterMortal, true, 3639, 3639},
		{"medio_mortal", 4112, classMasterMortal, true, 3782, 3527},
		{"avancado_celestial", 4113, classMasterCelestial, true, 3911, 3655},
		{"avancado_mortal_rejected", 4113, classMasterMortal, false, 0, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db := territorioDB(tc.item, tc.classMaster)
			addr, stop := startServerClockVol(t, db, map[int]int{int(tc.item): volEntradaTerritorio})
			defer stop()
			c := enterWorld(t, addr)
			defer c.Close()

			body := protocol.MsgUseItemBody{SourType: world.ItemPlaceCarry, SourPos: 0}
			send(t, c, protocol.MsgUseItem, body.Encode())

			var sawTeleport, sawConsume, sawKeep bool
			var gotX, gotY int16
			for {
				ty, payload, ok := readMaybe(t, c)
				if !ok {
					break
				}
				switch ty {
				case protocol.MsgAction:
					var ab protocol.MsgActionBody
					if err := ab.Decode(payload); err != nil {
						t.Fatalf("decode action: %v", err)
					}
					if ab.Effect == 1 {
						sawTeleport = true
						gotX, gotY = ab.TargetX, ab.TargetY
					}
				case protocol.MsgSendItem:
					if int(le16(payload[0:2])) != world.ItemPlaceCarry || int(le16(payload[2:4])) != 0 {
						continue
					}
					switch int16(le16(payload[4:6])) {
					case 0:
						sawConsume = true
					case tc.item:
						sawKeep = true
					}
				}
			}

			if tc.wantTeleport {
				if !sawTeleport {
					t.Fatal("expected teleport jump, got none")
				}
				if gotX < tc.wantX-3 || gotX > tc.wantX+1 || gotY < tc.wantY-3 || gotY > tc.wantY+1 {
					t.Errorf("teleport to (%d,%d), want ~(%d,%d) ±[-3,+1]", gotX, gotY, tc.wantX, tc.wantY)
				}
				if !sawConsume {
					t.Error("ticket not consumed")
				}
			} else {
				if sawTeleport {
					t.Error("rejected ticket teleported the player")
				}
				if !sawKeep {
					t.Error("rejected ticket: expected slot resync keeping the item")
				}
			}
		})
	}
}

// idealStoneDB monta um fakeDB para um personagem com uma Pedra_Ideal (item 5338,
// EF_VOLATILE 211) no carry 0, no tier/nível dado (issue #407).
func idealStoneDB(classMaster uint8, lvl int) *fakeDB {
	db := newDB()
	st := world.CharacterState{Slot: 0, Name: "Hero", X: 5, Y: 5, HP: 1000, MaxHP: 1000, ClassMaster: classMaster, Level: lvl, MortalLevel: 99, ArchCrystalStage: 4}
	st.Carry[0] = world.Item{Index: pedraIdealItem}
	db.loadResult = st
	return db
}

// startIdealStoneServer inicia um servidor cujo catálogo conhece o
// EF_VOLATILE 211 do 5338, a única coisa que o roteia para useIdealStone.
func startIdealStoneServer(t *testing.T, db world.Persistence) (string, func()) {
	t.Helper()
	return startServerClockVol(t, db, map[int]int{pedraIdealItem: volIdealStone})
}

// TestIdealStoneTriggerIsThe5338 fixa a issue #407: a criação legada de Celestial
// é o bloco EF_VOLATILE 211 (_MSG_UseItem.cpp:3003), que só a
// Pedra_Ideal 5338 carrega. A 1742 (Pedra_da_Imortalidade, sem EF_VOLATILE)
// passa por todos os blocos e não faz nada, mesmo para um Arch pronto; a #222 tinha
// ligado a transformação a ela. Um não-Arch e não-Celestial recebe a
// ressincronização silenciosa de :3005-3009, e um Celestial vira Sub-Celestial só com um
// Sephirot em Equip[11] (:3026-3085, issue #410), guardando a pedra em silêncio
// no caso contrário.
func TestIdealStoneTriggerIsThe5338(t *testing.T) {
	tests := []struct {
		name        string
		classMaster uint8
		item        int16
		sephirot    int16
		wantTier    uint8
		wantNotice  bool
	}{
		{"arch 5338 becomes celestial", classMasterArch, pedraIdealItem, 0, classMasterCelestial, false},
		{"arch 1742 does nothing", classMasterArch, immortalityStoneItem, 0, classMasterArch, false},
		{"mortal 5338 silent resync", classMasterMortal, pedraIdealItem, 0, classMasterMortal, false},
		{"celestial 5338 without sephirot keeps the stone", classMasterCelestial, pedraIdealItem, 0, classMasterCelestial, false},
		{"celestial 5338 with sephirot becomes sub", classMasterCelestial, pedraIdealItem, 1762, classMasterCelestialCS, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := idealStoneDB(tt.classMaster, 399)
			db.loadResult.Carry[0] = world.Item{Index: tt.item}
			db.loadResult.Equip[sephirotEquipSlot] = world.Item{Index: tt.sephirot}
			addr, stop := startIdealStoneServer(t, db)
			defer stop()
			c := enterWorld(t, addr)
			defer c.Close()
			send(t, c, protocol.MsgUseItem, (&protocol.MsgUseItemBody{SourType: world.ItemPlaceCarry, SourPos: 0}).Encode())
			sawLogout, sawNotice := false, false
			for {
				ty, _, ok := readMaybe(t, c)
				if !ok {
					break
				}
				sawNotice = sawNotice || ty == protocol.MsgMessageBoxOk
				if ty == protocol.MsgCNFCharacterLogout {
					sawLogout = true
					break
				}
			}
			if sawNotice != tt.wantNotice {
				t.Errorf("notice = %v, want %v", sawNotice, tt.wantNotice)
			}
			if !sawLogout {
				send(t, c, protocol.MsgCharacterLogout, nil)
				expect(t, c, protocol.MsgCNFCharacterLogout)
			}
			save, n := db.lastSavedChar()
			if n == 0 || save.ClassMaster != tt.wantTier {
				t.Fatalf("saved tier = %d (saves %d), want %d", save.ClassMaster, n, tt.wantTier)
			}
			if tt.wantTier == classMasterCelestialCS && (save.SaveCelestial[1].Class != int32(tt.sephirot-archSephirotMin) || hasItem(save.Equip, tt.sephirot)) {
				t.Errorf("sub sheet class = %d, sephirot kept = %v", save.SaveCelestial[1].Class, hasItem(save.Equip, tt.sephirot))
			}
			// A 1742 segue o caminho comum de equipar vol 0, então agora pode ser vestida.
			consumed := !hasItem(save.Carry, tt.item) && !hasItem(save.Equip, tt.item)
			if consumed != (tt.wantTier != tt.classMaster) {
				t.Errorf("stone %d consumed = %v, want %v", tt.item, consumed, !consumed)
			}
		})
	}
}

// TestUseIdealStoneCreatesCelestial cobre a transformação Arch→Celestial
// (issues #222, #407): um Arch no teto do tier (level.MaxLevel) clicando com o direito na Pedra
// Ideal (5338) vira um Celestial de nível 1, com o item consumido e os gates de
// quest celestial reiniciados, e a mudança persiste.
func TestUseIdealStoneCreatesCelestial(t *testing.T) {
	db := idealStoneDB(classMasterArch, int(level.MaxLevel))
	addr, stop := startIdealStoneServer(t, db)
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	body := protocol.MsgUseItemBody{SourType: world.ItemPlaceCarry, SourPos: 0}
	send(t, c, protocol.MsgUseItem, body.Encode())

	seenLogout := false
	for !seenLogout {
		ty, _, ok := readMaybe(t, c)
		if !ok {
			t.Fatal("connection ended before Celestial logout confirmation")
		}
		seenLogout = ty == protocol.MsgCNFCharacterLogout
	}
	char, n := db.lastSavedChar()
	if n == 0 {
		t.Fatal("character was never saved")
	}
	if char.ClassMaster != classMasterCelestial {
		t.Errorf("saved ClassMaster = %d, want %d (celestial)", char.ClassMaster, classMasterCelestial)
	}
	if char.Level != 0 {
		t.Errorf("saved Level = %d, want 0 (legacy zero-based reset)", char.Level)
	}
	if char.Exp != 0 {
		t.Errorf("saved Exp = %d, want 0 (reset)", char.Exp)
	}
	if char.CelLv40 != 0 || char.CelLv90 != 0 || char.CelCircle != 0 {
		t.Errorf("saved celestial gates = %d/%d/%d, want all 0 (fresh)", char.CelLv40, char.CelLv90, char.CelCircle)
	}
	if hasItem(char.Carry, pedraIdealItem) {
		t.Error("saved carry still has the Pedra Ideal; want consumed")
	}
	if char.CelestialArchLevel != 5 {
		t.Errorf("saved Arch band = %d, want 5", char.CelestialArchLevel)
	}
	if !hasItem(char.Equip, 3502) || !hasItem(char.Equip, 3199) {
		t.Errorf("saved Celestial equipment = %+v, want body 3502 and neutral cape 3199", char.Equip)
	}
	if char.SpecialBonus != 855 || char.LearnedSkill != 1<<30 {
		t.Errorf("saved skills = bonus %d mask %d", char.SpecialBonus, char.LearnedSkill)
	}
}

// TestUseIdealStoneRequiresMaxArchLevel verifica que um Arch abaixo do teto do tier é
// recusado: sem troca de tier, o item fica onde está, e o cliente recebe uma ressincronização de slot
// em vez dos sinais de desbloqueio.
func TestUseIdealStoneRequiresMaxArchLevel(t *testing.T) {
	db := idealStoneDB(classMasterArch, 354)
	addr, stop := startIdealStoneServer(t, db)
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	body := protocol.MsgUseItemBody{SourType: world.ItemPlaceCarry, SourPos: 0}
	send(t, c, protocol.MsgUseItem, body.Encode())

	sawResync := false
	for {
		ty, payload, ok := readMaybe(t, c)
		if !ok {
			break
		}
		switch ty {
		case protocol.MsgCombineComplete, protocol.MsgMotion:
			t.Fatalf("got %#x for a below-level Arch; want no unlock signal", ty)
		case protocol.MsgMessageBoxOk, protocol.MsgMessageChat, protocol.MsgMessagePanel:
			// :3011-3015 só ressincroniza o slot (issue #413).
			t.Fatalf("got %#x for a below-level Arch; the legacy refusal is silent", ty)
		}
		if ty == protocol.MsgSendItem && int16(le16(payload[4:6])) == pedraIdealItem {
			sawResync = true
		}
	}
	if !sawResync {
		t.Error("expected a slot resync keeping the Pedra Ideal")
	}

	send(t, c, protocol.MsgCharacterLogout, nil)
	expect(t, c, protocol.MsgCNFCharacterLogout)
	char, n := db.lastSavedChar()
	if n == 0 {
		t.Fatal("character was never saved")
	}
	if char.ClassMaster != classMasterArch {
		t.Errorf("saved ClassMaster = %d, want %d (unchanged, still arch)", char.ClassMaster, classMasterArch)
	}
}

func TestCelestialArchBandsAndEquipment(t *testing.T) {
	tests := []struct {
		level int32
		band  uint8
		body  int16
	}{
		{355, 1, 3500}, {369, 1, 3500}, {370, 2, 3500}, {379, 2, 3500},
		{380, 3, 3501}, {397, 3, 3501}, {398, 4, 3501}, {399, 5, 3502}, {450, 5, 3502},
	}
	d := New(Config{})
	for _, tc := range tests {
		e := world.Entity{Class: 0, ClassMaster: classMasterArch, Level: tc.level, MortalLevel: 99, ArchCrystalStage: 4, Clan: clanHekalotia}
		e.Equip[0] = world.Item{Index: 21}
		e.Carry[0] = world.Item{Index: pedraIdealItem}
		d.buildCelestialSnapshot(&e, 0)
		if e.CelestialArchLevel != tc.band || e.Equip[1].Index != tc.body {
			t.Errorf("level %d => band/body %d/%d, want %d/%d", tc.level, e.CelestialArchLevel, e.Equip[1].Index, tc.band, tc.body)
		}
		if e.Level != 0 || e.Exp != 0 || e.BaseAC != 230 || e.BaseDamage != 0 {
			t.Errorf("level %d reset incomplete: %+v", tc.level, e)
		}
		if e.BaseStr != 8 || e.BaseInt != 4 || e.BaseDex != 7 || e.BaseCon != 6 || e.BaseMaxHP != 80 || e.BaseMaxMP != 45 {
			t.Errorf("level %d class base mismatch", tc.level)
		}
		if e.SpecialBonus != 855 || e.LearnedSkill != 1<<30 || e.Equip[capeEquipSlot].Index != 3197 {
			t.Errorf("level %d celestial state mismatch", tc.level)
		}
		if !e.Carry[0].Empty() {
			t.Errorf("level %d Ideal Stone not consumed", tc.level)
		}
		if e.Equip[0].Effects[1] != (world.Effect{Effect: 98, Value: 3}) || e.Equip[0].Effects[2] != (world.Effect{Effect: 106, Value: 21}) {
			t.Errorf("level %d body effects mismatch: %+v", tc.level, e.Equip[0])
		}
	}
}

func TestCelestialClassBases(t *testing.T) {
	want := [4][6]int32{{8, 4, 7, 6, 80, 45}, {5, 8, 5, 5, 60, 65}, {6, 6, 9, 5, 70, 55}, {8, 9, 13, 6, 75, 60}}
	d := New(Config{})
	for class, base := range want {
		e := world.Entity{Class: uint8(class), ClassMaster: classMasterArch, Level: 399, MortalLevel: 99}
		e.Carry[0] = world.Item{Index: pedraIdealItem, Effects: [3]world.Effect{{Effect: 61, Value: 3}}}
		d.buildCelestialSnapshot(&e, 0)
		got := [6]int32{int32(e.BaseStr), int32(e.BaseInt), int32(e.BaseDex), int32(e.BaseCon), e.BaseMaxHP, e.BaseMaxMP}
		if got != base {
			t.Errorf("class %d base = %v, want %v", class, got, base)
		}
	}
}

func TestUseIdealStonePersistenceFailurePreservesArch(t *testing.T) {
	db := idealStoneDB(classMasterArch, 399)
	db.saveErr = errors.New("injected save failure")
	addr, stop := startIdealStoneServer(t, db)
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()
	body := protocol.MsgUseItemBody{SourType: world.ItemPlaceCarry, SourPos: 0}
	send(t, c, protocol.MsgUseItem, body.Encode())
	seenStone := false
	for {
		ty, payload, ok := readMaybe(t, c)
		if !ok {
			break
		}
		if ty == protocol.MsgCNFCharacterLogout {
			t.Fatal("failed persistence returned player to selection")
		}
		if ty == protocol.MsgSendItem && int16(le16(payload[4:6])) == pedraIdealItem {
			seenStone = true
			break
		}
	}
	if !seenStone {
		t.Fatal("failed persistence did not restore the visible Ideal Stone")
	}
	send(t, c, protocol.MsgCharacterLogout, nil)
	expect(t, c, protocol.MsgCNFCharacterLogout)
	save, n := db.lastSavedChar()
	if n == 0 || save.ClassMaster != classMasterArch || !hasItem(save.Carry, pedraIdealItem) {
		t.Fatalf("post-failure save = %+v count=%d, want unchanged Arch with stone", save, n)
	}
}

// TestImmortalityStoneMortalStillEquips é uma guarda de regressão: um Mortal carregando
// a Pedra_da_Imortalidade (1742) ainda precisa seguir o caminho normal de equipar,
// já que o fluxo Mortal→Arch do Rei precisa dela equipada no slot 10.
func TestImmortalityStoneMortalStillEquips(t *testing.T) {
	db := idealStoneDB(classMasterMortal, 100)
	db.loadResult.Carry[0] = world.Item{Index: immortalityStoneItem}
	addr, stop, _ := startServerClock(t, db)
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	body := protocol.MsgUseItemBody{
		SourType: world.ItemPlaceCarry, SourPos: 0,
		DestType: world.ItemPlaceEquip, DestPos: immortalityStoneEquipSlot,
	}
	send(t, c, protocol.MsgUseItem, body.Encode())
	if ty, _, ok := readMaybe(t, c); !ok || ty != protocol.MsgUseItem {
		t.Errorf("mortal equip got %#x ok=%v, want UseItem echo (equip path)", ty, ok)
	}
}

const (
	itemMagicBeanBlue    = 3407
	itemMagicBeanLight   = 3416
	itemMagicBeanRemover = 3417
)

func magicBeanDB(bean, equip world.Item) *fakeDB {
	return magicBeanDBAt(bean, equip, 1, "")
}

func magicBeanDBAt(bean, equip world.Item, equipSlot int, role string) *fakeDB {
	db := newDB()
	db.accounts["tester"].role = role
	st := world.CharacterState{Slot: 0, Name: "Hero", X: 5, Y: 5, HP: 1000, MaxHP: 1000}
	st.Carry[0] = bean
	st.Equip[equipSlot] = equip
	db.loadResult = st
	return db
}

func useMagicBeanFrame(t *testing.T, c net.Conn, dstSlot int) {
	t.Helper()
	body := protocol.MsgUseItemBody{
		SourType: world.ItemPlaceCarry, SourPos: 0,
		DestType: world.ItemPlaceEquip, DestPos: int32(dstSlot),
	}
	send(t, c, protocol.MsgUseItem, body.Encode())
}

func magicBeanVols(bean int16) map[int]int {
	return map[int]int{int(bean): volMagicBean}
}

func TestUseMagicBeanPaintsEquippedSet(t *testing.T) {
	armor := world.Item{Index: itemArmor, Effects: [3]world.Effect{{Effect: efSanc, Value: 9}}}
	addr, stop := startServerClockVol(t, magicBeanDB(world.Item{Index: itemMagicBeanBlue}, armor), magicBeanVols(itemMagicBeanBlue))
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	useMagicBeanFrame(t, c, 1)

	if got := readPanel(t, c); got != noticeText(NoticeRefineSuccess) {
		t.Fatalf("notice = %q, quer %q", got, noticeText(NoticeRefineSuccess))
	}
	expect(t, c, protocol.MsgUpdateScore)
	item := expect(t, c, protocol.MsgSendItem)
	if got := le16(item[0:2]); got != world.ItemPlaceEquip {
		t.Fatalf("send item place = %d, want equip", got)
	}
	if got := le16(item[2:4]); got != 1 {
		t.Fatalf("send item slot = %d, want 1", got)
	}
	if got := le16(item[4:6]); got != itemArmor {
		t.Fatalf("painted item index = %d, want armor", got)
	}
	if item[6] != magicBeanPaintLo || item[7] != 9 {
		t.Fatalf("effect0 = %d.%d, want paint %d preserving sanc value 9", item[6], item[7], magicBeanPaintLo)
	}
}

func TestUseMagicBeanRejectsWeaponForPlayers(t *testing.T) {
	weapon := world.Item{Index: 900, Effects: [3]world.Effect{{Effect: efSanc, Value: 9}}}
	db := magicBeanDBAt(world.Item{Index: itemMagicBeanBlue}, weapon, weaponSlotR, "")
	addr, stop := startServerClockVol(t, db, magicBeanVols(itemMagicBeanBlue))
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	useMagicBeanFrame(t, c, weaponSlotR)

	if code := noticeCode(t, expect(t, c, protocol.MsgMessageBoxOk)); code != NoticeCantUseHere {
		t.Fatalf("notice = %d, want CantUseHere", code)
	}
	item := expect(t, c, protocol.MsgSendItem)
	if got := le16(item[0:2]); got != world.ItemPlaceCarry {
		t.Fatalf("reject send item place = %d, want carry", got)
	}
	if got := le16(item[4:6]); got != itemMagicBeanBlue {
		t.Fatalf("reject source item = %d, want magic bean", got)
	}
	if ty, _, ok := readMaybe(t, c); ok {
		t.Fatalf("player weapon magic bean use produced extra frame %#x", ty)
	}

	send(t, c, protocol.MsgCharacterLogout, nil)
	expect(t, c, protocol.MsgCNFCharacterLogout)
	save, n := db.lastSavedChar()
	if n == 0 {
		t.Fatal("character was not saved on logout")
	}
	carry0, ok := savedItemAt(save.Carry, 0)
	if !ok || carry0.Index != itemMagicBeanBlue {
		t.Fatalf("saved carry0 = %+v ok=%v, want unconsumed magic bean", carry0, ok)
	}
	equip6, ok := savedItemAt(save.Equip, weaponSlotR)
	if !ok || equip6.Index != 900 || equip6.Eff1 != efSanc || equip6.EffV1 != 9 {
		t.Fatalf("saved weapon = %+v ok=%v, want unpainted +9 weapon", equip6, ok)
	}
}

func TestUseMagicBeanAllowsWeaponForModerators(t *testing.T) {
	weapon := world.Item{Index: 900, Effects: [3]world.Effect{{Effect: efSanc, Value: 9}}}
	addr, stop := startServerClockVol(t, magicBeanDBAt(world.Item{Index: itemMagicBeanBlue}, weapon, weaponSlotR, "moderator"), magicBeanVols(itemMagicBeanBlue))
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	useMagicBeanFrame(t, c, weaponSlotR)

	if got := readPanel(t, c); got != noticeText(NoticeRefineSuccess) {
		t.Fatalf("notice = %q, quer %q", got, noticeText(NoticeRefineSuccess))
	}
	expect(t, c, protocol.MsgUpdateScore)
	item := expect(t, c, protocol.MsgSendItem)
	if got := le16(item[0:2]); got != world.ItemPlaceEquip {
		t.Fatalf("send item place = %d, want equip", got)
	}
	if got := le16(item[2:4]); got != weaponSlotR {
		t.Fatalf("send item slot = %d, want weapon slot", got)
	}
	if got := le16(item[4:6]); got != 900 {
		t.Fatalf("painted item index = %d, want weapon", got)
	}
	if item[6] != magicBeanPaintLo || item[7] != 9 {
		t.Fatalf("effect0 = %d.%d, want paint %d preserving sanc value 9", item[6], item[7], magicBeanPaintLo)
	}
}

func TestUseMagicBeanHighestColorAndRemover(t *testing.T) {
	tests := []struct {
		name string
		bean int16
		dst  world.Item
		want uint8
	}{
		{
			name: "light blue writes 125",
			bean: itemMagicBeanLight,
			dst:  world.Item{Index: itemArmor, Effects: [3]world.Effect{{Effect: efSanc, Value: 8}}},
			want: magicBeanPaintHi,
		},
		{
			name: "remover writes EF_SANC",
			bean: itemMagicBeanRemover,
			dst:  world.Item{Index: itemArmor, Effects: [3]world.Effect{{Effect: magicBeanPaintLo + 4, Value: 8}}},
			want: efSanc,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, stop := startServerClockVol(t, magicBeanDB(world.Item{Index: tt.bean}, tt.dst), magicBeanVols(tt.bean))
			defer stop()
			c := enterWorld(t, addr)
			defer c.Close()

			useMagicBeanFrame(t, c, 1)

			expect(t, c, protocol.MsgMessageBoxOk)
			expect(t, c, protocol.MsgUpdateScore)
			item := expect(t, c, protocol.MsgSendItem)
			if item[6] != tt.want || item[7] != 8 {
				t.Fatalf("effect0 = %d.%d, want %d.8", item[6], item[7], tt.want)
			}
		})
	}
}

// TestUseMagicBeanBroadcastsVisual prova a correção da #157: aplicar ou remover tinta
// atualiza o EquipVisual/EquipAnct em cache e empurra MSG_UpdateEquip, para a cor
// vestida sobreviver a um teleporte posterior (enterWorldView/createMobFrom retransmitem daquele
// cache). Antes da correção useMagicBean não enviava MSG_UpdateEquip, então este fluxo ficava mudo.
func TestUseMagicBeanBroadcastsVisual(t *testing.T) {
	tests := []struct {
		name     string
		bean     int16
		dst      world.Item
		wantAnct uint8
	}{
		{
			name:     "paint sets the color overlay",
			bean:     itemMagicBeanBlue,
			dst:      world.Item{Index: itemArmor, Effects: [3]world.Effect{{Effect: efSanc, Value: 9}}},
			wantAnct: magicBeanPaintLo - 3, // VisualAnctCode devolve ef-3 para um efeito de tinta
		},
		{
			name:     "remover clears the overlay back to sanc",
			bean:     itemMagicBeanRemover,
			dst:      world.Item{Index: itemArmor, Effects: [3]world.Effect{{Effect: magicBeanPaintLo + 4, Value: 8}}},
			wantAnct: efSanc,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, stop := startServerClockVol(t, magicBeanDB(world.Item{Index: tt.bean}, tt.dst), magicBeanVols(tt.bean))
			defer stop()
			c := enterWorld(t, addr)
			defer c.Close()

			useMagicBeanFrame(t, c, 1)

			ue := expect(t, c, protocol.MsgUpdateEquip)
			if got := ue[33]; got != tt.wantAnct { // AnctCode[1] @ body32+1
				t.Fatalf("equip anct[1] = %#x, want %#x", got, tt.wantAnct)
			}
		})
	}
}

func TestUseMagicBeanRejectsMissingSanc(t *testing.T) {
	tests := []struct {
		name  string
		equip world.Item
	}{
		{name: "empty equip slot"},
		{name: "unrefined item", equip: world.Item{Index: itemArmor}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, stop := startServerClockVol(t, magicBeanDB(world.Item{Index: itemMagicBeanBlue}, tt.equip), magicBeanVols(itemMagicBeanBlue))
			defer stop()
			c := enterWorld(t, addr)
			defer c.Close()

			useMagicBeanFrame(t, c, 1)

			item := expect(t, c, protocol.MsgSendItem)
			if got := le16(item[0:2]); got != world.ItemPlaceCarry {
				t.Fatalf("reject send item place = %d, want carry", got)
			}
			if got := le16(item[4:6]); got != itemMagicBeanBlue {
				t.Fatalf("reject source item = %d, want magic bean", got)
			}
			if ty, _, ok := readMaybe(t, c); ok {
				t.Fatalf("missing-sanc magic bean use produced extra frame %#x", ty)
			}
		})
	}
}

func TestUseMagicBeanRejectsNonSetSlot(t *testing.T) {
	armor := world.Item{Index: itemArmor, Effects: [3]world.Effect{{Effect: efSanc, Value: 9}}}
	addr, stop := startServerClockVol(t, magicBeanDB(world.Item{Index: itemMagicBeanBlue}, armor), magicBeanVols(itemMagicBeanBlue))
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	useMagicBeanFrame(t, c, 0)

	if code := noticeCode(t, expect(t, c, protocol.MsgMessageBoxOk)); code != NoticeOnlyToEquips {
		t.Fatalf("notice = %d, want OnlyToEquips", code)
	}
	item := expect(t, c, protocol.MsgSendItem)
	if got := le16(item[0:2]); got != world.ItemPlaceCarry {
		t.Fatalf("reject send item place = %d, want carry", got)
	}
	if got := le16(item[4:6]); got != itemMagicBeanBlue {
		t.Fatalf("reject source item = %d, want magic bean", got)
	}
}

func TestUseMagicBeanStackPersistsOneConsumed(t *testing.T) {
	armor := world.Item{Index: itemArmor, Effects: [3]world.Effect{{Effect: efSanc, Value: 9}}}
	bean := world.Item{Index: itemMagicBeanBlue, Effects: [3]world.Effect{{Effect: efAmount, Value: 3}}}
	db := magicBeanDB(bean, armor)
	addr, stop := startServerClockVol(t, db, magicBeanVols(itemMagicBeanBlue))
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	useMagicBeanFrame(t, c, 1)
	expect(t, c, protocol.MsgSendItem)

	send(t, c, protocol.MsgCharacterLogout, nil)
	expect(t, c, protocol.MsgCNFCharacterLogout)

	save, n := db.lastSavedChar()
	if n == 0 {
		t.Fatal("character was not saved on logout")
	}
	carry0, ok := savedItemAt(save.Carry, 0)
	if !ok {
		t.Fatal("saved carry slot 0 is empty; want stacked magic bean")
	}
	if carry0.Index != itemMagicBeanBlue || carry0.Eff1 != efAmount || carry0.EffV1 != 2 {
		t.Fatalf("saved carry0 = %+v, want magic bean amount 2", carry0)
	}
	equip1, ok := savedItemAt(save.Equip, 1)
	if !ok {
		t.Fatal("saved equip slot 1 is empty; want painted armor")
	}
	if equip1.Eff1 != magicBeanPaintLo || equip1.EffV1 != 9 {
		t.Fatalf("saved equip1 effects = %+v, want paint %d value 9", equip1, magicBeanPaintLo)
	}
}

func TestMagicBeanEffectSlotScan(t *testing.T) {
	it := world.Item{Effects: [3]world.Effect{
		{Effect: efDamage, Value: 1},
		{Effect: efAc, Value: 2},
		{Effect: efHp, Value: 3},
	}}
	if got := magicBeanEffectSlot(it, false); got != -1 {
		t.Fatalf("paint slot = %d, want -1 for three real effects", got)
	}
	it.Effects[1] = world.Effect{Effect: efSanc, Value: 9}
	if got := magicBeanEffectSlot(it, false); got != 1 {
		t.Fatalf("paint slot = %d, want EF_SANC slot 1", got)
	}
	if got := magicBeanEffectSlot(it, true); got != -1 {
		t.Fatalf("remover slot = %d, want -1 when only EF_SANC is available", got)
	}
	it.Effects[2] = world.Effect{Effect: magicBeanPaintLo + 2, Value: 9}
	if got := magicBeanEffectSlot(it, true); got != 2 {
		t.Fatalf("remover slot = %d, want paint slot 2", got)
	}
}

func savedItemAt(items []world.SavedItem, slot int) (world.SavedItem, bool) {
	for _, it := range items {
		if it.Slot == slot {
			return it, true
		}
	}
	return world.SavedItem{}, false
}

// TestTradingItemCarryMove é o caso mais básico que o usuário encontrou: arrastar um item de
// um slot de inventário para um vazio via _MSG_TradingItem (0x0376). O item se move
// e os dois slots são atualizados (a origem vazia, o destino agora preenchido).
func TestTradingItemCarryMove(t *testing.T) {
	addr, stop, _ := startServerClock(t, itemDB(1100)) // item 1100 no slot de carry 0
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	tradeItemFrame(t, c, world.ItemPlaceCarry, 0, world.ItemPlaceCarry, 3, 0)
	if ty, _, ok := readMaybe(t, c); !ok || ty != protocol.MsgTradingItem {
		t.Fatalf("move echo = %#x ok=%v, want TradingItem", ty, ok)
	}
	src := expect(t, c, protocol.MsgSendItem) // slot 0, agora vazio
	if le16(src[2:4]) != 0 || le16(src[4:6]) != 0 {
		t.Errorf("source slot = %d item %d, want slot 0 empty", le16(src[2:4]), le16(src[4:6]))
	}
	dst := expect(t, c, protocol.MsgSendItem) // slot 3, agora contém o item
	if le16(dst[2:4]) != 3 || le16(dst[4:6]) != 1100 {
		t.Errorf("dest slot = %d item %d, want slot 3 item 1100", le16(dst[2:4]), le16(dst[4:6]))
	}
}

// TestEquipBonusDivines verifica que os tipos de efeito divino além do conjunto SIDC básico
// entram no score: EF_SPECIAL1-4, EF_DAMAGEADD e EF_ACADD como efeitos de instância.
func TestEquipBonusDivines(t *testing.T) {
	// O item 700 é uma joia de dano (nUnique 45 ∈ [41,50]) então seu EF_DAMAGEADD conta.
	d := New(Config{ItemUnique: map[int]int{700: 45}})
	e := &world.Entity{}
	// Um anel com um especial divino, um dano fixo divino e um AC fixo divino.
	e.Equip[0] = world.Item{Index: 700, Effects: [3]world.Effect{
		{Effect: efSpecial1, Value: 10},
		{Effect: efDamageAdd, Value: 25},
		{Effect: efAcAdd, Value: 7},
	}}
	b := d.equipBonus(e)
	if b.special[0] != 10 {
		t.Errorf("special[0] = %d, want 10", b.special[0])
	}
	if b.damage != 25 {
		t.Errorf("damage = %d, want 25 (EF_DAMAGEADD on a jewel)", b.damage)
	}
	if b.ac != 7 {
		t.Errorf("ac = %d, want 7 (EF_ACADD)", b.ac)
	}
}

func TestEquipBonusSpecialAll(t *testing.T) {
	d := New(Config{})
	e := &world.Entity{}
	e.Equip[0] = world.Item{Index: 700, Effects: [3]world.Effect{
		{Effect: efSpecialAll, Value: 5},
	}}

	b := d.equipBonus(e)
	want := [4]int16{0, 5, 5, 5}
	if b.special != want {
		t.Errorf("special = %v, want %v", b.special, want)
	}
}

// TestEquipBonusDamageAddGate confirma que EF_DAMAGEADD é ignorado num item NÃO-joia
// (nUnique fora de [41,50]) — só joias de dano o contribuem (captura §B/E).
func TestEquipBonusDamageAddGate(t *testing.T) {
	d := New(Config{}) // sem nUnique → não é joia
	e := &world.Entity{}
	e.Equip[0] = world.Item{Index: 701, Effects: [3]world.Effect{{Effect: efDamageAdd, Value: 25}}}
	if got := d.equipBonus(e).damage; got != 0 {
		t.Errorf("damage = %d, want 0 (EF_DAMAGEADD only counts on jewels)", got)
	}
}

// TestEquipBonusHpAddPercent confirma que EF_HPADD é PERCENTUAL (não fixo): ele se acumula
// em hpAddPct e effectiveMaxHP multiplica o MaxHP base por ele (captura §E).
func TestEquipBonusHpAddPercent(t *testing.T) {
	d := New(Config{})
	e := &world.Entity{BaseMaxHP: 1000}
	e.Equip[0] = world.Item{Index: 702, Effects: [3]world.Effect{{Effect: efHpAdd, Value: 10}}}
	d.refreshScore(e) // HPADD é percentual → em cache em HpAddPct, aplicado na leitura
	if e.HpAddPct != 10 {
		t.Fatalf("HpAddPct = %d, want 10", e.HpAddPct)
	}
	if got := effectiveMaxHP(e); got != 1100 {
		t.Errorf("effectiveMaxHP = %d, want 1100 (1000 +10%%)", got)
	}
}

// TestCanEquipSlot verifica o gate da máscara nPos: um item cabe num slot se nPos tem
// o bit daquele slot; consumíveis (nPos 0) não cabem em lugar nenhum; itens desconhecidos são permitidos.
func TestCanEquipSlot(t *testing.T) {
	d := New(Config{ItemPos: map[int]int{
		3381: 0,     // Poção Divina: não cabe em lugar nenhum
		11:   1,     // item de corpo: slot 0 (1<<0)
		861:  192,   // arma dupla: slots 6,7
		342:  16384, // montaria: slot 14
	}})
	cases := []struct {
		idx  int16
		slot int
		want bool
	}{
		{3381, 0, false}, {11, 0, true}, {11, 1, false},
		{861, 6, true}, {861, 7, true}, {861, 0, false},
		{342, 14, true}, {342, 7, false},
		{0, 0, true}, {9999, 0, true}, // vazio + desconhecido são permitidos
	}
	for _, c := range cases {
		if got := d.canEquipSlot(c.idx, c.slot); got != c.want {
			t.Errorf("canEquipSlot(%d, %d) = %v, want %v", c.idx, c.slot, got, c.want)
		}
	}
}

// TestRepairEquip confirma que um consumível mal equipado (poção no slot do corpo) é
// devolvido ao inventário e o equipamento válido fica no lugar.
func TestRepairEquip(t *testing.T) {
	d := New(Config{ItemPos: map[int]int{3381: 0, 1406: 4}}) // 1406 nPos 4 = slot 2
	st := world.CharacterState{Class: 1}
	st.Equip[0] = world.Item{Index: 3381} // poção indevidamente no slot do corpo
	st.Equip[2] = world.Item{Index: 1406} // armadura corretamente no slot 2
	d.repairEquip(&st)
	if st.Equip[0].Index == 3381 {
		t.Error("potion still in the body slot after repair")
	}
	if st.Equip[2].Index != 1406 {
		t.Error("valid armor was wrongly relocated")
	}
	found := false
	for _, it := range st.Carry {
		if it.Index == 3381 {
			found = true
		}
	}
	if !found {
		t.Error("displaced potion was not preserved in the inventory")
	}
}

// TestDivineAffectBonus verifica que o buff da Poção Divina (Affect 34) soma +20% ao
// MaxHp/MaxMp/Dano efetivos na leitura, e é identidade quando ausente (captura §C).
func TestDivineAffectBonus(t *testing.T) {
	d := New(Config{})
	e := &world.Entity{ID: world.MaxUser + 1, BaseMaxHP: 1000, BaseMaxMP: 500, BaseDamage: 200}
	d.refreshScore(e)
	if got := effectiveMaxHP(e); got != 1000 {
		t.Fatalf("no-buff effectiveMaxHP = %d, want 1000", got)
	}
	if got := d.effectiveDamage(e); got != 200 {
		t.Fatalf("no-buff effectiveDamage = %d, want 200", got)
	}
	e.Affect[0] = world.Affect{Type: world.AffectDivine, Level: 1}
	if got := effectiveMaxHP(e); got != 1200 {
		t.Errorf("divine effectiveMaxHP = %d, want 1200 (+20%%)", got)
	}
	if got := effectiveMaxMP(e); got != 600 {
		t.Errorf("divine effectiveMaxMP = %d, want 600 (+20%%)", got)
	}
	if got := d.effectiveDamage(e); got != 240 {
		t.Errorf("divine effectiveDamage = %d, want 240 (+20%%)", got)
	}
}

// TestSendAffectDivineTicksNotRawSeconds é a regressão da issue #202: o tempo restante
// exibido da Poção Divina precisa chegar ao cliente em ticks de 8s, como
// todo outro buff (affect1H=450/h, affect1D=10800/dia) — sendAffect enviava
// a diferença crua de segundos DivineEnd-now, inflando a
// duração 8x (e, depois de DivineEnd passar, deixando o sentinela cru 2e9 "infinito"
// vazar sem conversão, mostrando uma duração totalmente deturpada).
func TestSendAffectDivineTicksNotRawSeconds(t *testing.T) {
	db := newDB()
	const remainingSeconds = 10 * 86400 // um buff Divino com 10 dias restantes
	db.loadResult = world.CharacterState{
		Slot: 0, Name: "Divine", X: 5, Y: 5, HP: 1000, MaxHP: 1000,
		DivineEnd: time.Now().Unix() + remainingSeconds,
	}
	addr, stop := startServerClockVol(t, db, nil)
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	affect := expect(t, c, protocol.MsgSendAffect)
	if affect[0] != world.AffectDivine {
		t.Fatalf("slot 0 affect type = %d, want %d (Divine)", affect[0], world.AffectDivine)
	}
	ticks := int64(binary.LittleEndian.Uint32(affect[4:8]))
	if diff := ticks*8 - remainingSeconds; diff < -16 || diff > 16 {
		t.Errorf("divine affect Time = %d ticks (%ds of 8s-ticks), want ~%ds remaining (diff %ds) — "+
			"raw seconds or the infinite sentinel leaked onto the wire", ticks, ticks*8, remainingSeconds, diff)
	}
}

// TestVigorAffectBonus verifica que a Poção de Vigor (Affect 35) soma +10% MaxHp/MaxMp.
func TestVigorAffectBonus(t *testing.T) {
	e := &world.Entity{BaseMaxHP: 1000, BaseMaxMP: 500}
	d := New(Config{})
	d.refreshScore(e)
	e.Affect[0] = world.Affect{Type: world.AffectVigor, Level: 1}
	if got := effectiveMaxHP(e); got != 1100 {
		t.Errorf("vigor effectiveMaxHP = %d, want 1100 (+10%%)", got)
	}
	if got := effectiveMaxMP(e); got != 550 {
		t.Errorf("vigor effectiveMaxMP = %d, want 550 (+10%%)", got)
	}
}

// TestUseVigorConsumesOneFromStack é a regressão da issue #259: useVigor apagava o
// slot de carry inteiro, então beber uma poção de um "Pack Poção de Vigor"
// destruía a pilha toda. O legado decrementa — todo ramo do
// bloco Vol-58 termina com `if (amount > 1) BASE_SetItemAmount(item, amount - 1);
// else memset(item, 0, ...)` (_MSG_UseItem.cpp:2197/2218/2239/2260). A duração
// do buff por sIndex é verificada junto, já que nada a cobria ponta a ponta.
func TestUseVigorConsumesOneFromStack(t *testing.T) {
	cases := []struct {
		name     string
		item     int16
		wantTime uint32
	}{
		{"1h", 3313, affect1H},
		{"7d", 3364, affect1D * 7},
		{"15d", 3365, affect1D * 15},
		{"30d", 3366, affect1D * 30},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := newDB()
			st := world.CharacterState{Slot: 0, Name: "Hero", X: 5, Y: 5, HP: 1000, MaxHP: 1000}
			st.Carry[0] = world.Item{Index: tc.item, Effects: [3]world.Effect{{Effect: efAmount, Value: 5}}}
			db.loadResult = st
			addr, stop := startServerClockVol(t, db, map[int]int{int(tc.item): volVigor})
			defer stop()
			c := enterWorld(t, addr)
			defer c.Close()

			body := protocol.MsgUseItemBody{SourType: world.ItemPlaceCarry, SourPos: 0}
			send(t, c, protocol.MsgUseItem, body.Encode())

			item := expect(t, c, protocol.MsgSendItem)
			if le16(item[4:6]) != uint16(tc.item) {
				t.Fatalf("carry slot 0 item = %d, want %d (whole pack consumed)", le16(item[4:6]), tc.item)
			}
			if item[6] != efAmount || item[7] != 4 {
				t.Errorf("amount effect = (%d,%d), want (%d,4)", item[6], item[7], efAmount)
			}
			expect(t, c, protocol.MsgUpdateScore)
			affect := expect(t, c, protocol.MsgSendAffect)
			if affect[0] != world.AffectVigor {
				t.Fatalf("slot 0 affect type = %d, want %d (Vigor)", affect[0], world.AffectVigor)
			}
			if got := binary.LittleEndian.Uint32(affect[4:8]); got != tc.wantTime {
				t.Errorf("affect time = %d, want %d", got, tc.wantTime)
			}
		})
	}
}

// TestUseVigorLastUnitClearsSlot é a outra metade do idioma legado: uma poção
// sozinha (sem EF_AMOUNT, então itemAmount informa 1) ainda esvazia o slot.
func TestUseVigorLastUnitClearsSlot(t *testing.T) {
	db := newDB()
	st := world.CharacterState{Slot: 0, Name: "Hero", X: 5, Y: 5, HP: 1000, MaxHP: 1000}
	st.Carry[0] = world.Item{Index: 3313}
	db.loadResult = st
	addr, stop := startServerClockVol(t, db, map[int]int{3313: volVigor})
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	body := protocol.MsgUseItemBody{SourType: world.ItemPlaceCarry, SourPos: 0}
	send(t, c, protocol.MsgUseItem, body.Encode())

	item := expect(t, c, protocol.MsgSendItem)
	if le16(item[2:4]) != 0 || le16(item[4:6]) != 0 {
		t.Errorf("slot/item = %d/%d, want slot 0 empty", le16(item[2:4]), le16(item[4:6]))
	}
}

// TestUseVigorThenMoveKeepsReducedAmount é o sintoma que o jogador realmente vê:
// sem um decremento real do lado do servidor o cliente mostra a pilha sumida, e a
// próxima ressincronização de slot (um move, uma compra, um relog) a "revive" — ou, com o
// apagamento antigo, o pacote inteiro realmente sumia. Espelha TestUseThenMoveKeepsReducedAmount.
func TestUseVigorThenMoveKeepsReducedAmount(t *testing.T) {
	db := newDB()
	st := world.CharacterState{Slot: 0, Name: "Hero", X: 5, Y: 5, HP: 1000, MaxHP: 1000}
	st.Carry[0] = world.Item{Index: 3313, Effects: [3]world.Effect{{Effect: efAmount, Value: 10}}}
	db.loadResult = st
	addr, stop := startServerClockVol(t, db, map[int]int{3313: volVigor})
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	body := protocol.MsgUseItemBody{SourType: world.ItemPlaceCarry, SourPos: 0}
	send(t, c, protocol.MsgUseItem, body.Encode())
	afterUse := expect(t, c, protocol.MsgSendItem)
	if le16(afterUse[4:6]) != 3313 || afterUse[6] != efAmount || afterUse[7] != 9 {
		t.Fatalf("after use: item %d amount-effect (%d,%d), want 3313 (%d,9)", le16(afterUse[4:6]), afterUse[6], afterUse[7], efAmount)
	}
	expect(t, c, protocol.MsgUpdateScore)
	expect(t, c, protocol.MsgSendAffect)

	tradeItemFrame(t, c, world.ItemPlaceCarry, 0, world.ItemPlaceCarry, 1, 0)
	expect(t, c, protocol.MsgTradingItem) // eco do move
	expect(t, c, protocol.MsgSendItem)    // slot de origem 0, agora vazio
	moved := expect(t, c, protocol.MsgSendItem)
	if le16(moved[2:4]) != 1 || le16(moved[4:6]) != 3313 {
		t.Fatalf("dest slot = %d item %d, want slot 1 item 3313", le16(moved[2:4]), le16(moved[4:6]))
	}
	if moved[6] != efAmount || moved[7] != 9 {
		t.Errorf("moved item amount-effect = (%d,%d), want (%d,9)", moved[6], moved[7], efAmount)
	}
}

// TestUseDivineConsumesOneFromStack é o irmão da issue #259: useDivine tinha
// o mesmo apagamento de slot inteiro (sua nota "empilhamento ainda não modelado" era anterior a
// consumeOneItem). Legado: _MSG_UseItem.cpp:2165. O item 3360 é a Panqueca (Vol 66).
func TestUseDivineConsumesOneFromStack(t *testing.T) {
	db := newDB()
	st := world.CharacterState{Slot: 0, Name: "Hero", X: 5, Y: 5, HP: 1000, MaxHP: 1000}
	st.Carry[0] = world.Item{Index: 3360, Effects: [3]world.Effect{{Effect: efAmount, Value: 5}}}
	db.loadResult = st
	addr, stop := startServerClockVol(t, db, map[int]int{3360: volDivine30})
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	body := protocol.MsgUseItemBody{SourType: world.ItemPlaceCarry, SourPos: 0}
	send(t, c, protocol.MsgUseItem, body.Encode())

	item := expect(t, c, protocol.MsgSendItem)
	if le16(item[4:6]) != 3360 {
		t.Fatalf("carry slot 0 item = %d, want 3360 (whole pack consumed)", le16(item[4:6]))
	}
	if item[6] != efAmount || item[7] != 4 {
		t.Errorf("amount effect = (%d,%d), want (%d,4)", item[6], item[7], efAmount)
	}
	expect(t, c, protocol.MsgUpdateScore)
	if affect := expect(t, c, protocol.MsgSendAffect); affect[0] != world.AffectDivine {
		t.Errorf("slot 0 affect type = %d, want %d (Divine)", affect[0], world.AffectDivine)
	}
}

// TestEquipBonusRefine verifica as duas regras INDEPENDENTES de refino numa peça de defesa.
// O AC de catálogo é primeiro escalado pelo multiplicador (sanc+10)/10 de BASE_GetItemAbility, e
// só então o LIMIAR +9 soma seus +25 fixos para um nPos de defesa (4/8/128) em cima —
// a captura §E é explícita que o limiar fica acima do valor escalado, não no lugar dele.
func TestEquipBonusRefine(t *testing.T) {
	d := New(Config{
		ItemEffects: map[int][]content.BaseEffect{555: {{Eff: efAc, Val: 100}}}, // armadura, AC base 100
		ItemPos:     map[int]int{555: 4},                                        // nPos 4 = defesa
	})
	armor8 := world.Item{Index: 555, Effects: [3]world.Effect{{Effect: efSanc, Value: 8}}} // +8 (abaixo do limiar)
	armor9 := world.Item{Index: 555, Effects: [3]world.Effect{{Effect: efSanc, Value: 9}}} // +9 (limiar)

	e8 := &world.Entity{}
	e8.Equip[0] = armor8
	e9 := &world.Entity{}
	e9.Equip[0] = armor9

	if ac8 := d.equipBonus(e8).ac; ac8 != 180 {
		t.Errorf("+8 AC = %d, want 180 (100*18/10, no threshold below +9)", ac8)
	}
	if ac9 := d.equipBonus(e9).ac; ac9 != 215 {
		t.Errorf("+9 AC = %d, want 215 (100*19/10 + 25 refine threshold)", ac9)
	}
}

// TestWeaponDamageRefine é a contraparte para mão de arma de TestEquipBonusRefine: o
// EF_DAMAGE de catálogo escala por (sanc+10)/10 e o limiar +9 soma seus +40 fixos em cima
// (nPos 64/192, captura §E).
func TestWeaponDamageRefine(t *testing.T) {
	d := New(Config{
		ItemEffects: map[int][]content.BaseEffect{900: {{Eff: efDamage, Val: 100}}},
		ItemPos:     map[int]int{900: 64}, // mão de arma
	})
	e := &world.Entity{}
	e.Equip[6] = world.Item{Index: 900} // +0 arma única
	if got := d.weaponDamage(e); got != 100 {
		t.Errorf("+0 weaponDamage = %d, want 100", got)
	}
	e.Equip[6] = world.Item{Index: 900, Effects: [3]world.Effect{{Effect: efSanc, Value: 9}}}
	if got := d.weaponDamage(e); got != 230 {
		t.Errorf("+9 weaponDamage = %d, want 230 (100*19/10 + 40 threshold)", got)
	}
}

// TestItemSancUnpacksPity é a regressão da issue #103: o cValue de EF_SANC empacota
// o nível de refino junto com o contador de pena (nível + 10*pena), então ler
// cru informa um nível totalmente errado. Um item +5 que falhou dois refinos
// guarda 25; a leitura crua antiga limitava isso a 15 e dava ao item o
// bônus de limiar +9 que ele nunca ganhou. Qualquer item tocado pelo servidor legado ou
// restaurado de um dump do banco pode carregar um valor empacotado.
func TestItemSancUnpacksPity(t *testing.T) {
	cases := []struct {
		name string
		cVal uint8
		want int
	}{
		{"+5 no pity", 5, 5},
		{"+5 with pity 2", 25, 5},
		{"+0 with pity 1", 10, 0},
		{"+9 with max pity", 209, 9},
		{"+9 clean still reads +9", 9, 9},
		{"+11 packs as 234", 234, 11},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			it := world.Item{Index: 555, Effects: [3]world.Effect{{Effect: efSanc, Value: tt.cVal}}}
			if got := itemSanc(it); got != tt.want {
				t.Errorf("itemSanc(cValue %d) = %d, want %d", tt.cVal, got, tt.want)
			}
		})
	}
}

// As duas regras de refino precisam usar o nível real, não o byte empacotado: um item +5 com pena
// escala como +5 e NÃO pode receber o bônus de limiar +9.
func TestEquipBonusRefineIgnoresPity(t *testing.T) {
	d := New(Config{
		ItemEffects: map[int][]content.BaseEffect{555: {{Eff: efAc, Val: 100}}},
		ItemPos:     map[int]int{555: 4},
	})
	e := &world.Entity{}
	e.Equip[0] = world.Item{Index: 555, Effects: [3]world.Effect{{Effect: efSanc, Value: 25}}} // +5, pena 2
	if ac := d.equipBonus(e).ac; ac != 150 {
		t.Errorf("+5 (pity 2) AC = %d, want 150 (100*15/10) — pity must not unlock the +9 threshold", ac)
	}
}

// TestAttackRunOfBoots verifica que EF_RUNSPEED (botas) eleva o nibble de velocidade
// de movimento (baixo) de AttackRun para o tier da issue #64: a pé=2, botas=4, montaria=6.
func TestAttackRunOfBoots(t *testing.T) {
	d := New(Config{
		ItemEffects: map[int][]content.BaseEffect{
			321: {{Eff: efRunSpeed, Val: 1}},  // botas
			322: {{Eff: efRunSpeed, Val: 99}}, // botas com bônus exagerado ainda usam o tier de botas
		},
	})

	bare := &world.Entity{}
	d.refreshScore(bare)
	if got := attackRunOf(bare); got != baseAttackRun {
		t.Errorf("no boots: attackRunOf = %#x, want base %#x", got, baseAttackRun)
	}

	booted := &world.Entity{}
	booted.Equip[5] = world.Item{Index: 321} // botas ocupam o slot de equipamento 5 (nPos 32)
	d.refreshScore(booted)
	if got := attackRunOf(booted); got != (baseAttackRun&0xF0)|bootMoveSpeed {
		t.Errorf("boots: attackRunOf = %#x, want %#x", got, (baseAttackRun&0xF0)|bootMoveSpeed)
	}

	overboosted := &world.Entity{}
	overboosted.Equip[5] = world.Item{Index: 322}
	d.refreshScore(overboosted)
	if got := attackRunOf(overboosted); got != (baseAttackRun&0xF0)|bootMoveSpeed {
		t.Errorf("overboosted boots: attackRunOf = %#x, want %#x", got, (baseAttackRun&0xF0)|bootMoveSpeed)
	}

	mounted := &world.Entity{}
	mounted.Equip[5] = world.Item{Index: 321}
	mounted.Equip[mountEquipSlot] = world.Item{Index: 342}
	d.refreshScore(mounted)
	if got := attackRunOf(mounted); got != (baseAttackRun&0xF0)|mountedMoveSpeed {
		t.Errorf("mounted with boots: attackRunOf = %#x, want %#x", got, (baseAttackRun&0xF0)|mountedMoveSpeed)
	}
}

// TestRefreshScoreSpecial confirma que refreshScore soma um especial divino na
// entidade viva (e portanto no score enviado ao cliente), e que uma
// ida-e-volta limpa deriveBaseScore→refreshScore reproduz o score carregado (sem contagem dupla).
func TestRefreshScoreSpecial(t *testing.T) {
	d := New(Config{})
	e := &world.Entity{Str: 80, AC: 120, MaxHP: 1000, HP: 1000}
	e.Equip[0] = world.Item{Index: 700, Effects: [3]world.Effect{
		{Effect: efSpecial2, Value: 15},
		{Effect: efStr, Value: 5},
	}}
	d.deriveBaseScore(e) // base = atual carregado − equipamento
	d.refreshScore(e)    // somar o equipamento de novo

	if e.Str != 80 {
		t.Errorf("Str = %d, want 80 (round-trip stable)", e.Str)
	}
	if want := playerBaseAC(e); e.AC != want {
		t.Errorf("AC = %d, want rebuilt base %d", e.AC, want)
	}
	if e.Special[1] != 15 {
		t.Errorf("Special[1] = %d, want 15", e.Special[1])
	}
}

func TestRefreshScoreSpecialAll(t *testing.T) {
	d := New(Config{})
	e := &world.Entity{
		ID:          1,
		BaseSpecial: [4]int16{0, 10, 20, 30},
		Special:     [4]int16{0, 15, 25, 35},
		Str:         80, AC: 120, MaxHP: 1000, HP: 1000,
	}
	e.Equip[0] = world.Item{Index: 700, Effects: [3]world.Effect{
		{Effect: efSpecialAll, Value: 5},
	}}
	d.deriveBaseScore(e)
	d.refreshScore(e)
	d.refreshScore(e)

	wantBase := [4]int16{0, 10, 20, 30}
	want := [4]int16{0, 15, 25, 35}
	if e.BaseSpecial != wantBase {
		t.Errorf("BaseSpecial = %v, want %v", e.BaseSpecial, wantBase)
	}
	if e.Special != want {
		t.Errorf("Special = %v, want %v", e.Special, want)
	}
}

// TestTradingItemEmptyMove recusa uma troca quando os dois slots estão vazios (nada a fazer).
func TestTradingItemEmptyMove(t *testing.T) {
	addr, stop, _ := startServerClock(t, itemDB(1100))
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	tradeItemFrame(t, c, world.ItemPlaceCarry, 10, world.ItemPlaceCarry, 11, 0) // ambos vazios
	if ty, _, ok := readMaybe(t, c); ok {
		t.Errorf("empty→empty move produced %#x; should be a no-op", ty)
	}
}

func expChestDB() *fakeDB {
	db := newDB()
	st := world.CharacterState{Slot: 0, Name: "Hero", X: 5, Y: 5, HP: 1000, MaxHP: 1000}
	st.Carry[0] = world.Item{Index: 4140}
	db.loadResult = st
	return db
}

func silverBarDB(itemIdx int16, coin int32) *fakeDB {
	db := newDB()
	st := world.CharacterState{Slot: 0, Name: "Hero", X: 5, Y: 5, HP: 1000, MaxHP: 1000, Coin: coin}
	st.Carry[0] = world.Item{Index: itemIdx}
	db.loadResult = st
	return db
}

func fairyDustDB(levelValue int, item world.Item) *fakeDB {
	db := newDB()
	st := world.CharacterState{
		Slot: 0, Name: "Hero", Class: 1, Level: levelValue,
		X: 5, Y: 5, HP: 1000, MaxHP: 1000, MP: 100, MaxMP: 100,
	}
	st.Carry[0] = item
	db.loadResult = st
	return db
}

func startServerClockVol(t *testing.T, persist world.Persistence, vols map[int]int) (string, func()) {
	t.Helper()
	return startServerClockItems(t, persist, vols, nil)
}

func startServerClockItems(t *testing.T, persist world.Persistence, vols map[int]int, effects map[int][]content.BaseEffect) (string, func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	d := New(Config{Log: log, ItemVolatiles: vols, ItemEffects: effects, ExpEvents: level.ExpEvents{KefraLive: true}})
	w := world.New(world.Config{GridDim: 16}, log, persist, d.Handle)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { _ = w.Serve(ctx, ln); close(done) }()
	return ln.Addr().String(), func() {
		cancel()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Error("server did not stop")
		}
	}
}

// startServerTickItems é o startServerClockItems com o tick de simulação rodando, então
// regenPlayers/applyHp realmente fecham as barras no alvo ReqHp. Testes que
// verificam um HP exato em pleno voo NÃO devem usar isto: o fio passivo
// (naturalRegenTicks) também roda e sobe o HP sozinho.
func startServerTickItems(t *testing.T, persist world.Persistence, vols map[int]int, effects map[int][]content.BaseEffect) (string, func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	d := New(Config{Log: log, ItemVolatiles: vols, ItemEffects: effects, ExpEvents: level.ExpEvents{KefraLive: true}})
	w := world.New(world.Config{GridDim: 16}, log, persist, d.Handle)
	w.SetTickHandler(10*time.Millisecond, d.Tick)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { _ = w.Serve(ctx, ln); close(done) }()
	return ln.Addr().String(), func() {
		cancel()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Error("server did not stop")
		}
	}
}

func healPotionDB(item world.Item, hp, maxHP, mp, maxMP int32) *fakeDB {
	db := newDB()
	st := world.CharacterState{Slot: 0, Name: "Hero", X: 5, Y: 5, HP: hp, MaxHP: maxHP, MP: mp, MaxMP: maxMP}
	st.Carry[0] = item
	db.loadResult = st
	return db
}

func paintDB(bean, target world.Item, targetSlot int) *fakeDB {
	db := newDB()
	st := world.CharacterState{Slot: 0, Name: "Hero", X: 5, Y: 5, HP: 1000, MaxHP: 1000}
	st.Carry[0] = bean
	st.Equip[targetSlot] = target
	db.loadResult = st
	return db
}

func sendPaintUse(t *testing.T, c net.Conn, destSlot int) {
	t.Helper()
	body := protocol.MsgUseItemBody{
		SourType: world.ItemPlaceCarry, SourPos: 0,
		DestType: world.ItemPlaceEquip, DestPos: int32(destSlot),
	}
	send(t, c, protocol.MsgUseItem, body.Encode())
}

func savedSlot(items []world.SavedItem, slot int) (world.SavedItem, bool) {
	for _, it := range items {
		if it.Slot == slot {
			return it, true
		}
	}
	return world.SavedItem{}, false
}

func TestUsePaintBeanAppliesColorOverWire(t *testing.T) {
	bean := world.Item{Index: 3408, Effects: [3]world.Effect{{Effect: efAmount, Value: 3}}}
	target := world.Item{Index: 555, Effects: [3]world.Effect{{Effect: efSanc, Value: 4}}}
	db := paintDB(bean, target, 1)
	addr, stop := startServerClockVol(t, db, map[int]int{3408: volPaint})
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	sendPaintUse(t, c, 1)

	sawNotice, sawEquip, sawScore, sawTarget := false, false, false, false
	for range 8 {
		ty, payload, ok := readMaybe(t, c)
		if !ok {
			break
		}
		switch ty {
		case protocol.MsgMessageBoxOk, protocol.MsgMessagePanel:
			if noticeCode(t, payload) != NoticeRefineSuccess {
				t.Fatalf("notice = %v, want NoticeRefineSuccess", noticeCode(t, payload))
			}
			sawNotice = true
		case protocol.MsgUpdateEquip:
			sawEquip = true
			if got := payload[33]; got != 117-3 { // AnctCode[1] @body32+1
				t.Errorf("paint anct[1] = %#x, want %#x", got, 117-3)
			}
		case protocol.MsgUpdateScore:
			sawScore = true
		case protocol.MsgSendItem:
			place := int(binary.LittleEndian.Uint16(payload[0:2]))
			slot := int(binary.LittleEndian.Uint16(payload[2:4]))
			if place != world.ItemPlaceEquip || slot != 1 {
				continue
			}
			sawTarget = true
			if idx := le16(payload[4:6]); idx != 555 {
				t.Fatalf("paint target index = %d, want 555", idx)
			}
			if payload[6] != 117 || payload[7] != 4 {
				t.Fatalf("paint target effect0 = %d.%d, want 117.4", payload[6], payload[7])
			}
		}
	}
	if !sawNotice || !sawEquip || !sawScore || !sawTarget {
		t.Fatalf("paint responses notice=%v equip=%v score=%v target=%v, want all true", sawNotice, sawEquip, sawScore, sawTarget)
	}

	send(t, c, protocol.MsgCharacterLogout, nil)
	expect(t, c, protocol.MsgCNFCharacterLogout)
	save, n := db.lastSavedChar()
	if n == 0 {
		t.Fatal("character was not saved on logout")
	}
	carry0, ok := savedSlot(save.Carry, 0)
	if !ok || carry0.Index != 3408 || carry0.Eff1 != efAmount || carry0.EffV1 != 2 {
		t.Fatalf("saved carry slot 0 = %+v ok=%v, want stacked paint bean amount 2", carry0, ok)
	}
	equip1, ok := savedSlot(save.Equip, 1)
	if !ok || equip1.Index != 555 || equip1.Eff1 != 117 || equip1.EffV1 != 4 {
		t.Fatalf("saved equip slot 1 = %+v ok=%v, want painted item 117.4", equip1, ok)
	}
}

func TestUsePaintRemoverRestoresSancCarrier(t *testing.T) {
	target := world.Item{Index: 555, Effects: [3]world.Effect{{Effect: 119, Value: 6}}}
	addr, stop := startServerClockVol(t, paintDB(world.Item{Index: 3417}, target, 1), map[int]int{3417: volPaint})
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	sendPaintUse(t, c, 1)

	for range 8 {
		ty, payload, ok := readMaybe(t, c)
		if !ok {
			break
		}
		if ty != protocol.MsgSendItem {
			continue
		}
		place := int(binary.LittleEndian.Uint16(payload[0:2]))
		slot := int(binary.LittleEndian.Uint16(payload[2:4]))
		if place != world.ItemPlaceEquip || slot != 1 {
			continue
		}
		if payload[6] != efSanc || payload[7] != 6 {
			t.Fatalf("removed paint effect0 = %d.%d, want EF_SANC.6", payload[6], payload[7])
		}
		return
	}
	t.Fatal("missing target SendItem after paint remover")
}

func TestUsePaintBeanRejectsUnrefinedTarget(t *testing.T) {
	addr, stop := startServerClockVol(t, paintDB(world.Item{Index: 3408}, world.Item{Index: 555}, 1), map[int]int{3408: volPaint})
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	sendPaintUse(t, c, 1)

	ty, payload, ok := readMaybe(t, c)
	if !ok || ty != protocol.MsgSendItem {
		t.Fatalf("unrefined paint response = %#x ok=%v, want source SendItem", ty, ok)
	}
	if place, slot, idx := le16(payload[0:2]), le16(payload[2:4]), le16(payload[4:6]); place != world.ItemPlaceCarry || slot != 0 || idx != 3408 {
		t.Fatalf("source resend place/slot/index = %d/%d/%d, want carry/0/3408", place, slot, idx)
	}
	if ty, _, ok := readMaybe(t, c); ok {
		t.Fatalf("unrefined paint produced extra frame %#x", ty)
	}
}

func TestUsePaintBeanRejectsForbiddenEquipSlot(t *testing.T) {
	target := world.Item{Index: 555, Effects: [3]world.Effect{{Effect: efSanc, Value: 1}}}
	addr, stop := startServerClockVol(t, paintDB(world.Item{Index: 3408}, target, 8), map[int]int{3408: volPaint})
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	sendPaintUse(t, c, 8)

	if got := readNotice(t, c); got != NoticeOnlyToEquips {
		t.Fatalf("forbidden slot notice = %d, want NoticeOnlyToEquips", got)
	}
	item := expect(t, c, protocol.MsgSendItem)
	if place, slot, idx := le16(item[0:2]), le16(item[2:4]), le16(item[4:6]); place != world.ItemPlaceCarry || slot != 0 || idx != 3408 {
		t.Fatalf("source resend place/slot/index = %d/%d/%d, want carry/0/3408", place, slot, idx)
	}
}

func TestUsePaintRemoverRejectsWithoutEmptyOrPaintSlot(t *testing.T) {
	target := world.Item{Index: 555, Effects: [3]world.Effect{
		{Effect: efSanc, Value: 1},
		{Effect: efSanc, Value: 2},
		{Effect: efSanc, Value: 3},
	}}
	addr, stop := startServerClockVol(t, paintDB(world.Item{Index: 3417}, target, 1), map[int]int{3417: volPaint})
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	sendPaintUse(t, c, 1)

	if got := readNotice(t, c); got != NoticeCantRefineMore {
		t.Fatalf("remover reject notice = %d, want NoticeCantRefineMore", got)
	}
	item := expect(t, c, protocol.MsgSendItem)
	if place, slot, idx := le16(item[0:2]), le16(item[2:4]), le16(item[4:6]); place != world.ItemPlaceCarry || slot != 0 || idx != 3417 {
		t.Fatalf("source resend place/slot/index = %d/%d/%d, want carry/0/3417", place, slot, idx)
	}
}

func TestUseExpChest(t *testing.T) {
	addr, stop := startServerClockVol(t, expChestDB(), map[int]int{4140: volExpChest})
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	body := protocol.MsgUseItemBody{SourType: world.ItemPlaceCarry, SourPos: 0}
	send(t, c, protocol.MsgUseItem, body.Encode())

	sawSendItem, sawAffect, sawScore := false, false, false
	for range 6 {
		ty, payload, ok := readMaybe(t, c)
		if !ok {
			break
		}
		switch ty {
		case protocol.MsgSendItem:
			sawSendItem = true
			if le16(payload[4:6]) != 0 {
				t.Errorf("carry slot 0 item = %d, want empty after use", le16(payload[4:6]))
			}
		case protocol.MsgSendAffect:
			sawAffect = true
			if payload[0] != world.AffectExpChest {
				t.Errorf("affect type = %d, want %d", payload[0], world.AffectExpChest)
			}
			if got := binary.LittleEndian.Uint32(payload[4:8]); got != affectExpChestInc {
				t.Errorf("affect time = %d, want %d", got, affectExpChestInc)
			}
		case protocol.MsgUpdateScore:
			sawScore = true
		}
	}
	if !sawSendItem {
		t.Error("missing MsgSendItem after using exp chest")
	}
	if !sawAffect {
		t.Error("missing MsgSendAffect after using exp chest")
	}
	if !sawScore {
		t.Error("missing MsgUpdateScore after using exp chest")
	}
}

func TestUseExpChestStack(t *testing.T) {
	db := newDB()
	st := world.CharacterState{Slot: 0, Name: "Hero", X: 5, Y: 5, HP: 1000, MaxHP: 1000}
	st.Carry[0] = world.Item{Index: 4140, Effects: [3]world.Effect{{Effect: efAmount, Value: 3}}}
	db.loadResult = st

	addr, stop := startServerClockVol(t, db, map[int]int{4140: volExpChest})
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	body := protocol.MsgUseItemBody{SourType: world.ItemPlaceCarry, SourPos: 0}
	send(t, c, protocol.MsgUseItem, body.Encode())

	payload := expect(t, c, protocol.MsgSendItem)
	if got := le16(payload[4:6]); got != 4140 {
		t.Fatalf("carry slot 0 item = %d, want stacked chest", got)
	}
	if payload[6] != efAmount || payload[7] != 2 {
		t.Errorf("effect0 = %d.%d, want %d.2", payload[6], payload[7], efAmount)
	}
}

// TestUseFrangoAssado é a regressão da issue #134: usar o item não fazia nada
// porque o Vol 63 não tinha case no switch do useItem. Ele deve consumir o item
// e conceder Affect 30 (AffectForceMobDamage) por 4h (_MSG_UseItem.cpp:2308-2341).
func TestUseFrangoAssado(t *testing.T) {
	addr, stop := startServerClockVol(t, expChestDB(), map[int]int{4140: volFrango})
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	body := protocol.MsgUseItemBody{SourType: world.ItemPlaceCarry, SourPos: 0}
	send(t, c, protocol.MsgUseItem, body.Encode())

	sawSendItem, sawAffect, sawScore := false, false, false
	for range 6 {
		ty, payload, ok := readMaybe(t, c)
		if !ok {
			break
		}
		switch ty {
		case protocol.MsgSendItem:
			sawSendItem = true
			if le16(payload[4:6]) != 0 {
				t.Errorf("carry slot 0 item = %d, want empty after use", le16(payload[4:6]))
			}
		case protocol.MsgSendAffect:
			sawAffect = true
			if payload[0] != world.AffectForceMobDamage {
				t.Errorf("affect type = %d, want %d", payload[0], world.AffectForceMobDamage)
			}
			if got := binary.LittleEndian.Uint32(payload[4:8]); got != affect1H*4 {
				t.Errorf("affect time = %d, want %d", got, affect1H*4)
			}
		case protocol.MsgUpdateScore:
			sawScore = true
		}
	}
	if !sawSendItem {
		t.Error("missing MsgSendItem after using frango assado")
	}
	if !sawAffect {
		t.Error("missing MsgSendAffect after using frango assado")
	}
	if !sawScore {
		t.Error("missing MsgUpdateScore after using frango assado")
	}
}

func TestUseLegacyBuffConsumables(t *testing.T) {
	cases := []struct {
		name      string
		item      int16
		vol       int
		wantValue uint8
		wantTime  uint32
	}{
		{"kappa", 787, volBuffKappa, legacyBuffKappa, 80},
		{"kappa 30m", 3310, volBuffKappa30, legacyBuffKappa, affect1H / 2},
		{"kappa 20h", 3319, volBuffKappa20h, legacyBuffKappa, affect1H * 20},
		{"combat", 1764, volBuffCombat, legacyBuffCombat, 80},
		{"combat 60m", 3311, volBuffCombat60, legacyBuffCombat, affect1H},
		{"combat 20h", 3320, volBuffCombat20h, legacyBuffCombat, affect1H * 20},
		{"mental", 1765, volBuffMental, legacyBuffMental, 80},
		{"mental 60m", 3312, volBuffMental60, legacyBuffMental, affect1H},
		{"mental 20h", 3321, volBuffMental20h, legacyBuffMental, affect1H * 20},
		{"sephira 7d", 3361, volBuffMental20h, legacyBuffSephira, affect1H * 168},
		{"sephira 15d", 3362, volBuffMental20h, legacyBuffSephira, affect1H * 360},
		{"sephira 30d", 3363, volBuffMental20h, legacyBuffSephira, affect1H * 720},
		{"antidote", 416, volBuffKappa, legacyBuffAntidote, affect1H / 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := newDB()
			st := world.CharacterState{Slot: 0, Name: "Hero", X: 5, Y: 5, HP: 1000, MaxHP: 1000}
			st.Carry[0] = world.Item{Index: tc.item, Effects: [3]world.Effect{{Effect: efAmount, Value: 5}}}
			db.loadResult = st
			addr, stop := startServerClockVol(t, db, map[int]int{int(tc.item): tc.vol})
			defer stop()
			c := enterWorld(t, addr)
			defer c.Close()

			body := protocol.MsgUseItemBody{SourType: world.ItemPlaceCarry, SourPos: 0}
			send(t, c, protocol.MsgUseItem, body.Encode())

			item := expect(t, c, protocol.MsgSendItem)
			if le16(item[4:6]) != uint16(tc.item) || item[6] != efAmount || item[7] != 4 {
				t.Fatalf("after use item = %d effects (%d,%d), want %d (%d,4)", le16(item[4:6]), item[6], item[7], tc.item, efAmount)
			}
			expect(t, c, protocol.MsgUpdateScore)
			affect := expect(t, c, protocol.MsgSendAffect)
			if affect[0] != affectLegacyConsumableBuff || affect[1] != tc.wantValue {
				t.Errorf("affect = type %d value %d, want type %d value %d", affect[0], affect[1], affectLegacyConsumableBuff, tc.wantValue)
			}
			if got := binary.LittleEndian.Uint32(affect[4:8]); got != tc.wantTime {
				t.Errorf("affect time = %d, want %d", got, tc.wantTime)
			}
		})
	}
}

func TestUseLegacyBuffConsumableFullAffectsDoesNotConsume(t *testing.T) {
	db := newDB()
	st := world.CharacterState{Slot: 0, Name: "Hero", X: 5, Y: 5, HP: 1000, MaxHP: 1000}
	st.Carry[0] = world.Item{Index: 3310, Effects: [3]world.Effect{{Effect: efAmount, Value: 5}}}
	st.Affects = make([]world.Affect, world.MaxAffect)
	for i := range st.Affects {
		st.Affects[i] = world.Affect{Type: uint8(50 + i), Time: 100}
	}
	db.loadResult = st
	addr, stop := startServerClockVol(t, db, map[int]int{3310: volBuffKappa30})
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()
	for {
		if _, _, ok := readMaybe(t, c); !ok {
			break
		}
	}

	body := protocol.MsgUseItemBody{SourType: world.ItemPlaceCarry, SourPos: 0}
	send(t, c, protocol.MsgUseItem, body.Encode())

	notice := expect(t, c, protocol.MsgMessageBoxOk)
	if noticeCode(t, notice) != NoticeCantEatMore {
		t.Errorf("notice = %d, want NoticeCantEatMore", noticeCode(t, notice))
	}
	item := expect(t, c, protocol.MsgSendItem)
	if le16(item[4:6]) != 3310 || item[6] != efAmount || item[7] != 5 {
		t.Errorf("rejected item = %d effects (%d,%d), want 3310 (%d,5)", le16(item[4:6]), item[6], item[7], efAmount)
	}
}

func TestUseUnknownVolatileRejectedAndResynced(t *testing.T) {
	db := newDB()
	st := world.CharacterState{Slot: 0, Name: "Hero", X: 5, Y: 5, HP: 1000, MaxHP: 1000}
	st.Carry[0] = world.Item{Index: 496, Effects: [3]world.Effect{{Effect: efAmount, Value: 5}}}
	db.loadResult = st
	addr, stop := startServerClockVol(t, db, map[int]int{496: 60})
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	body := protocol.MsgUseItemBody{SourType: world.ItemPlaceCarry, SourPos: 0}
	send(t, c, protocol.MsgUseItem, body.Encode())

	notice := expect(t, c, protocol.MsgMessageBoxOk)
	if noticeCode(t, notice) != NoticeCantUseHere {
		t.Errorf("notice = %d, want NoticeCantUseHere", noticeCode(t, notice))
	}
	item := expect(t, c, protocol.MsgSendItem)
	if le16(item[4:6]) != 496 || item[6] != efAmount || item[7] != 5 {
		t.Errorf("resynced item = %d effects (%d,%d), want 496 (%d,5)", le16(item[4:6]), item[6], item[7], efAmount)
	}
}

// TestUseCoracaoDoce é a regressão da issue #135 para o consumível Vol-205:
// ele concede Velocidade (Affect 2) + Defesa (Affect 11) e realmente decrementa
// a pilha (antes um no-op, já que o Vol 205 não tinha case no useItem).
func TestUseCoracaoDoce(t *testing.T) {
	db := newDB()
	st := world.CharacterState{Slot: 0, Name: "Hero", X: 5, Y: 5, HP: 1000, MaxHP: 1000}
	st.Carry[0] = world.Item{Index: 4145, Effects: [3]world.Effect{{Effect: efAmount, Value: 10}}}
	db.loadResult = st
	addr, stop := startServerClockVol(t, db, map[int]int{4145: volCoracaoDoce})
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	body := protocol.MsgUseItemBody{SourType: world.ItemPlaceCarry, SourPos: 0}
	send(t, c, protocol.MsgUseItem, body.Encode())

	item := expect(t, c, protocol.MsgSendItem)
	if le16(item[4:6]) != 4145 {
		t.Fatalf("carry slot 0 item = %d, want 4145 (stack not emptied)", le16(item[4:6]))
	}
	expect(t, c, protocol.MsgUpdateScore)
	affect := expect(t, c, protocol.MsgSendAffect)
	if affect[0] != 2 || affect[1] != 2 {
		t.Errorf("slot 0 affect = type %d value %d, want type 2 value 2 (Velocidade)", affect[0], affect[1])
	}
	if got := binary.LittleEndian.Uint32(affect[4:8]); got != affect1H/5 {
		t.Errorf("slot 0 affect time = %d, want %d", got, affect1H/5)
	}
	if affect[8] != 11 {
		t.Errorf("slot 1 affect type = %d, want 11 (Defesa)", affect[8])
	}
	if got := binary.LittleEndian.Uint32(affect[12:16]); got != affect1H/5 {
		t.Errorf("slot 1 affect time = %d, want %d", got, affect1H/5)
	}
}

// TestUseChocolateDoAmor é a regressão da issue #135 para o consumível Vol-204:
// Dano (Affect 9) + Skill (Affect 15, Value 55).
func TestUseChocolateDoAmor(t *testing.T) {
	db := newDB()
	st := world.CharacterState{Slot: 0, Name: "Hero", X: 5, Y: 5, HP: 1000, MaxHP: 1000}
	st.Carry[0] = world.Item{Index: 1739, Effects: [3]world.Effect{{Effect: efAmount, Value: 10}}}
	db.loadResult = st
	addr, stop := startServerClockVol(t, db, map[int]int{1739: volChocolate})
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	body := protocol.MsgUseItemBody{SourType: world.ItemPlaceCarry, SourPos: 0}
	send(t, c, protocol.MsgUseItem, body.Encode())

	expect(t, c, protocol.MsgSendItem)
	expect(t, c, protocol.MsgUpdateScore)
	affect := expect(t, c, protocol.MsgSendAffect)
	if affect[0] != 9 {
		t.Errorf("slot 0 affect type = %d, want 9 (Dano)", affect[0])
	}
	if affect[8] != 15 || affect[9] != 55 {
		t.Errorf("slot 1 affect = type %d value %d, want type 15 value 55 (Skill)", affect[8], affect[9])
	}
}

// TestUseThenMoveKeepsReducedAmount é a regressão ponta a ponta da issue #135: usar
// uma pilha parcial precisa deixar o Amount reduzido vivo nos dados de slot do próprio servidor,
// para um move posterior de _MSG_TradingItem (que só ressincroniza o estado real do slot) não
// parecer "reverter" o consumo — o bug que toda esta correção resolve.
func TestUseThenMoveKeepsReducedAmount(t *testing.T) {
	db := newDB()
	st := world.CharacterState{Slot: 0, Name: "Hero", X: 5, Y: 5, HP: 1000, MaxHP: 1000}
	st.Carry[0] = world.Item{Index: 4145, Effects: [3]world.Effect{{Effect: efAmount, Value: 10}}}
	db.loadResult = st
	addr, stop := startServerClockVol(t, db, map[int]int{4145: volCoracaoDoce})
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	body := protocol.MsgUseItemBody{SourType: world.ItemPlaceCarry, SourPos: 0}
	send(t, c, protocol.MsgUseItem, body.Encode())
	afterUse := expect(t, c, protocol.MsgSendItem)
	if le16(afterUse[4:6]) != 4145 || afterUse[6] != efAmount || afterUse[7] != 9 {
		t.Fatalf("after use: item %d amount-effect (%d,%d), want 4145 (61,9)", le16(afterUse[4:6]), afterUse[6], afterUse[7])
	}
	expect(t, c, protocol.MsgUpdateScore)
	expect(t, c, protocol.MsgSendAffect)

	tradeItemFrame(t, c, world.ItemPlaceCarry, 0, world.ItemPlaceCarry, 1, 0)
	expect(t, c, protocol.MsgTradingItem) // eco do move
	expect(t, c, protocol.MsgSendItem)    // slot de origem 0, agora vazio
	moved := expect(t, c, protocol.MsgSendItem)
	if le16(moved[2:4]) != 1 || le16(moved[4:6]) != 4145 {
		t.Fatalf("dest slot = %d item %d, want slot 1 item 4145", le16(moved[2:4]), le16(moved[4:6]))
	}
	if moved[6] != efAmount || moved[7] != 9 {
		t.Errorf("moved item amount-effect = (%d,%d), want (61,9) — quantity reverted on move (issue #135)", moved[6], moved[7])
	}
}

// startServerAdamantita inicia um servidor cujo Dispatcher carrega os mapas de
// catálogo que useAdamantita precisa (ItemUnique/ItemGrades/ItemExtra), que o
// harness simples startServerClockVol não expõe.
func startServerAdamantita(t *testing.T, persist world.Persistence, vols, unique, grades, extra map[int]int) (string, func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	d := New(Config{
		Log: log, ItemVolatiles: vols, ItemUnique: unique, ItemGrades: grades, ItemExtra: extra,
		ExpEvents: level.ExpEvents{KefraLive: true},
	})
	w := world.New(world.Config{GridDim: 16}, log, persist, d.Handle)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { _ = w.Serve(ctx, ln); close(done) }()
	return ln.Addr().String(), func() {
		cancel()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Error("server did not stop")
		}
	}
}

// TestUseAdamantitaWrongTier recusa um item da família Adamantita combinado num
// alvo cujo nUnique não casa com o tier da origem: sem consumo, a recusa
// no estilo do refino de poeira ressincroniza o slot de ORIGEM (refineReject), e o
// alvo fica intocado (nenhuma ressincronização, casando com o comportamento legado de SendItem-só-
// na-origem para esta guarda).
func TestUseAdamantitaWrongTier(t *testing.T) {
	db := newDB()
	st := world.CharacterState{Slot: 0, Name: "Hero", X: 5, Y: 5, HP: 1000, MaxHP: 1000}
	st.Carry[0] = world.Item{Index: 578} // Adamantita, Type = 578-575 = 3
	st.Carry[1] = world.Item{Index: 900} // alvo: nUnique cai no balde Type 0, não 3
	db.loadResult = st
	addr, stop := startServerAdamantita(t, db,
		map[int]int{578: volAdamantita},
		map[int]int{900: 5}, // balde 0
		map[int]int{900: 2},
		nil,
	)
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	body := protocol.MsgUseItemBody{SourType: world.ItemPlaceCarry, SourPos: 0, DestType: world.ItemPlaceCarry, DestPos: 1}
	send(t, c, protocol.MsgUseItem, body.Encode())

	notice := expect(t, c, protocol.MsgMessageBoxOk)
	if noticeCode(t, notice) != NoticeCantRefineMore {
		t.Errorf("notice = %d, want NoticeCantRefineMore", noticeCode(t, notice))
	}
	src := expect(t, c, protocol.MsgSendItem)
	if le16(src[2:4]) != 0 || le16(src[4:6]) != 578 {
		t.Errorf("source resync = slot %d item %d, want slot 0 item 578 (unconsumed)", le16(src[2:4]), le16(src[4:6]))
	}
}

// TestUseAdamantitaMatchingTierConsumes sorteia um combine de Adamantita num alvo
// com tier correspondente e graduável: qualquer que seja o resultado do sorteio, o slot alvo é ressincronizado
// e a Adamantita de origem é gasta (issue #135 — este caminho antes era um
// no-op completo, já que o Vol 9 não tinha case no useItem de modo algum).
func TestUseAdamantitaMatchingTierConsumes(t *testing.T) {
	db := newDB()
	st := world.CharacterState{Slot: 0, Name: "Hero", X: 5, Y: 5, HP: 1000, MaxHP: 1000}
	st.Carry[0] = world.Item{Index: 578} // Adamantita, Type = 3
	st.Carry[1] = world.Item{Index: 900} // alvo: nUnique balde 3, grade 2
	db.loadResult = st
	addr, stop := startServerAdamantita(t, db,
		map[int]int{578: volAdamantita},
		map[int]int{900: 8}, // balde 3
		map[int]int{900: 2},
		map[int]int{900: 950}, // índice do resultado Extra em caso de sucesso
	)
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	body := protocol.MsgUseItemBody{SourType: world.ItemPlaceCarry, SourPos: 0, DestType: world.ItemPlaceCarry, DestPos: 1}
	send(t, c, protocol.MsgUseItem, body.Encode())

	notice := expect(t, c, protocol.MsgMessageBoxOk)
	if code := noticeCode(t, notice); code != NoticeRefineSuccess && code != NoticeFailToRefine {
		t.Errorf("notice = %d, want NoticeRefineSuccess or NoticeFailToRefine", code)
	}
	dst := expect(t, c, protocol.MsgSendItem)
	if le16(dst[2:4]) != 1 {
		t.Fatalf("resynced slot = %d, want 1 (the target)", le16(dst[2:4]))
	}
	if got := le16(dst[4:6]); got != 900 && got != 950 {
		t.Errorf("target item = %d, want 900 (fail, unchanged) or 950 (success, Extra)", got)
	}
	// Nenhum outro MsgSendItem para o slot de origem (o cliente já previu a
	// remoção do arrasto — casa com a convenção de refineSucceed/refineFail).
	if ty, p, ok := readMaybe(t, c); ok && ty == protocol.MsgSendItem {
		t.Errorf("unexpected second MsgSendItem for source slot: %v", p)
	}
}

// TestUseSeloDoGuerreiro é a regressão da issue #135 para o sIndex 4146 (sem
// EF_VOLATILE, então não pode cair no caminho de equipar vol==0): concede um
// amuleto de entrada de Clã 7 a um mortal de nível 354+ sem amuleto equipado, e
// realmente consome o selo.
func TestUseSeloDoGuerreiro(t *testing.T) {
	db := newDB()
	st := world.CharacterState{
		Slot: 0, Name: "Hero", X: 5, Y: 5, HP: 1000, MaxHP: 1000,
		Level: 354, ClassMaster: classMasterMortal, Clan: 7,
	}
	st.Carry[0] = world.Item{Index: itemSeloDoGuerreiro}
	db.loadResult = st
	addr, stop := startServerClockVol(t, db, nil)
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	body := protocol.MsgUseItemBody{SourType: world.ItemPlaceCarry, SourPos: 0}
	send(t, c, protocol.MsgUseItem, body.Encode())

	amulet := expect(t, c, protocol.MsgSendItem)
	if le16(amulet[2:4]) != amuletSlot || le16(amulet[4:6]) != 3191 {
		t.Fatalf("equip slot %d item %d, want slot %d item 3191 (Elite de Hekalotia)", le16(amulet[2:4]), le16(amulet[4:6]), amuletSlot)
	}
	expect(t, c, protocol.MsgUpdateScore)
	src := expect(t, c, protocol.MsgSendItem)
	if le16(src[4:6]) != 0 {
		t.Errorf("carry slot 0 item = %d, want empty after use", le16(src[4:6]))
	}
}

// Pergaminhos da Água usados fora da entrada ou da própria zona mantêm a pilha.
func TestUseWaterScrollOutsideEntranceRejected(t *testing.T) {
	db := newDB()
	st := world.CharacterState{Slot: 0, Name: "Hero", X: 5, Y: 5, HP: 1000, MaxHP: 1000}
	st.Carry[0] = world.Item{Index: 3182, Effects: [3]world.Effect{{Effect: efAmount, Value: 5}}} // Água (A) NV1
	db.loadResult = st
	addr, stop := startServerClockVol(t, db, map[int]int{3182: 161})
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	body := protocol.MsgUseItemBody{SourType: world.ItemPlaceCarry, SourPos: 0}
	send(t, c, protocol.MsgUseItem, body.Encode())

	notice := expect(t, c, protocol.MsgMessageBoxOk)
	if noticeCode(t, notice) != NoticeCantUseHere {
		t.Errorf("notice = %d, want NoticeCantUseHere", noticeCode(t, notice))
	}
	item := expect(t, c, protocol.MsgSendItem)
	if le16(item[4:6]) != 3182 || item[7] != 5 {
		t.Errorf("resynced item = index %d amount %d, want 3182 amount 5 (unconsumed)", le16(item[4:6]), item[7])
	}
}

// setHpMpFields decodifica MSG_SetHpMp (protocol/score.go:110): Hp, Mp, ReqHp, ReqMp.
func setHpMpFields(t *testing.T, p []byte) (hp, mp, reqHp, reqMp int32) {
	t.Helper()
	if len(p) < 16 {
		t.Fatalf("SetHpMp payload too short: %d", len(p))
	}
	return int32(binary.LittleEndian.Uint32(p[0:])), int32(binary.LittleEndian.Uint32(p[4:])),
		int32(binary.LittleEndian.Uint32(p[8:])), int32(binary.LittleEndian.Uint32(p[12:]))
}

// Os testes de poção abaixo rodam SEM handler de tick de propósito: a poção só
// eleva o alvo de pedido ReqHp/ReqMp, que é totalmente determinístico, enquanto a
// barra viva é fechada nele depois pelo tick (applyHp). A rampa em si é coberta
// por TestUseHealPotionRampsOverTicks, onde um tick roda.

// TestUseHealPotion: a poção prepara a cura como um alvo ReqHp e deixa o HP
// intocado — o original nunca escreve CurrentScore.Hp (_MSG_UseItem.cpp:117-126).
func TestUseHealPotion(t *testing.T) {
	const potion = 400
	db := healPotionDB(world.Item{Index: potion}, 500, 1000, 0, 0)
	effects := map[int][]content.BaseEffect{potion: {{Eff: efHp, Val: 50}}}
	addr, stop := startServerClockItems(t, db, map[int]int{potion: volHpMpPotion}, effects)
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	body := protocol.MsgUseItemBody{SourType: world.ItemPlaceCarry, SourPos: 0}
	send(t, c, protocol.MsgUseItem, body.Encode())

	itemPayload := expect(t, c, protocol.MsgSendItem)
	if got := le16(itemPayload[4:6]); got != 0 {
		t.Errorf("carry slot 0 item = %d, want empty after use", got)
	}
	hp, _, reqHp, _ := setHpMpFields(t, expect(t, c, protocol.MsgSetHpMp))
	if reqHp != 550 {
		t.Errorf("ReqHp = %d, want 550 (500 + 50)", reqHp)
	}
	if hp != 500 {
		t.Errorf("Hp = %d, want 500 — the potion stages a target, the tick heals", hp)
	}
}

func TestUseHealPotionClampsToMax(t *testing.T) {
	const potion = 404
	db := healPotionDB(world.Item{Index: potion}, 950, 1000, 0, 0)
	effects := map[int][]content.BaseEffect{potion: {{Eff: efHp, Val: 500}}}
	addr, stop := startServerClockItems(t, db, map[int]int{potion: volHpMpPotion}, effects)
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	body := protocol.MsgUseItemBody{SourType: world.ItemPlaceCarry, SourPos: 0}
	send(t, c, protocol.MsgUseItem, body.Encode())

	expect(t, c, protocol.MsgSendItem)
	if _, _, reqHp, _ := setHpMpFields(t, expect(t, c, protocol.MsgSetHpMp)); reqHp != 1000 {
		t.Errorf("ReqHp = %d, want clamped to 1000", reqHp)
	}
}

// TestUseHealPotionRampsOverTicks é o comportamento de cura gradual ponta a ponta: a barra
// fecha no alvo da poção ao longo dos ticks (applyHp, <=2000/chamada) em vez de
// estalar no momento do uso.
func TestUseHealPotionRampsOverTicks(t *testing.T) {
	const potion = 406
	db := healPotionDB(world.Item{Index: potion}, 500, 100000, 0, 0)
	effects := map[int][]content.BaseEffect{potion: {{Eff: efHp, Val: 1000}}}
	addr, stop := startServerTickItems(t, db, map[int]int{potion: volHpMpPotion}, effects)
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	body := protocol.MsgUseItemBody{SourType: world.ItemPlaceCarry, SourPos: 0}
	send(t, c, protocol.MsgUseItem, body.Encode())
	expect(t, c, protocol.MsgSendItem)

	// O sendScore do tick reporta a barra curada. O MaxHP é enorme para o
	// fio passivo (Nível+30 a cada 10 ticks) não alcançar 1500 na janela sozinho.
	p := expect(t, c, protocol.MsgUpdateScore)
	if got := int32(binary.LittleEndian.Uint32(p[24:28])); got < 1500 {
		t.Errorf("Hp = %d, want >= 1500 (500 + the 1000 potion) once the tick applies it", got)
	}
}

// TestUseHealPotionCooldown fixa o gate anti-spam do PotionDelay
// (_MSG_UseItem.cpp:105-115): um segundo uso dentro da janela é recusado, o slot é
// ressincronizado e — crucialmente — a pilha NÃO é consumida duas vezes.
func TestUseHealPotionCooldown(t *testing.T) {
	const potion = 407
	item := world.Item{Index: potion, Effects: [3]world.Effect{{Effect: efAmount, Value: 5}}}
	db := healPotionDB(item, 500, 100000, 0, 0)
	effects := map[int][]content.BaseEffect{potion: {{Eff: efHp, Val: 50}}}
	addr, stop := startServerClockItems(t, db, map[int]int{potion: volHpMpPotion}, effects)
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	body := protocol.MsgUseItemBody{SourType: world.ItemPlaceCarry, SourPos: 0}
	send(t, c, protocol.MsgUseItem, body.Encode())
	if p := expect(t, c, protocol.MsgSendItem); p[7] != 4 {
		t.Fatalf("first use: stack = %d, want 4 (one consumed)", p[7])
	}
	_, _, reqHp, _ := setHpMpFields(t, expect(t, c, protocol.MsgSetHpMp))

	// De novo imediatamente — o relógio injetado não avança, então isto está dentro da
	// janela de 100ms e precisa ser recusado.
	send(t, c, protocol.MsgUseItem, body.Encode())
	p := expect(t, c, protocol.MsgSendItem)
	if p[7] != 4 {
		t.Errorf("second use: stack = %d, want 4 — a refused potion must not be consumed", p[7])
	}
	if ty, _, ok := readMaybe(t, c); ok && ty == protocol.MsgSetHpMp {
		t.Error("refused potion still sent SetHpMp, want the use dropped")
	}
	if reqHp != 550 {
		t.Errorf("ReqHp = %d, want 550 — only the first potion should count", reqHp)
	}
}

func TestUseManaPotion(t *testing.T) {
	const potion = 405
	db := healPotionDB(world.Item{Index: potion}, 1000, 1000, 50, 200)
	effects := map[int][]content.BaseEffect{potion: {{Eff: efMp, Val: 50}}}
	addr, stop := startServerClockItems(t, db, map[int]int{potion: volHpMpPotion}, effects)
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	body := protocol.MsgUseItemBody{SourType: world.ItemPlaceCarry, SourPos: 0}
	send(t, c, protocol.MsgUseItem, body.Encode())

	expect(t, c, protocol.MsgSendItem)
	if _, _, _, reqMp := setHpMpFields(t, expect(t, c, protocol.MsgSetHpMp)); reqMp != 100 {
		t.Errorf("ReqMp = %d, want 100 (50 + 50)", reqMp)
	}
}

func TestUseHealPotionStack(t *testing.T) {
	const potion = 428
	item := world.Item{Index: potion, Effects: [3]world.Effect{{Effect: efAmount, Value: 5}}}
	db := healPotionDB(item, 500, 1000, 0, 0)
	effects := map[int][]content.BaseEffect{potion: {{Eff: efHp, Val: 200}}}
	addr, stop := startServerClockItems(t, db, map[int]int{potion: volHpMpPotion}, effects)
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	body := protocol.MsgUseItemBody{SourType: world.ItemPlaceCarry, SourPos: 0}
	send(t, c, protocol.MsgUseItem, body.Encode())

	itemPayload := expect(t, c, protocol.MsgSendItem)
	if got := le16(itemPayload[4:6]); got != potion {
		t.Fatalf("carry slot 0 item = %d, want stacked potion to remain", got)
	}
	if itemPayload[6] != efAmount || itemPayload[7] != 4 {
		t.Errorf("effect0 = %d.%d, want %d.4", itemPayload[6], itemPayload[7], efAmount)
	}
	if _, _, reqHp, _ := setHpMpFields(t, expect(t, c, protocol.MsgSetHpMp)); reqHp != 700 {
		t.Errorf("ReqHp = %d, want 700 (500 + 200)", reqHp)
	}
}

// TestUseHealPotionBroadcastsToViewers é a regressão da issue #99: quando um jogador
// toma poção, todos em visão precisam saber o novo HP. O cliente acompanha o HP de outra
// entidade SUBTRAINDO os deltas de STRUCT_DAM no MSG_Attack da própria cópia local, então
// uma cura fora de banda é invisível para um atacante a menos que o servidor empurre um
// score explícito — o que o legado faz com GridMulticast de SendScore
// (SendFunc.cpp:1298). O bug era que nosso sendScore só respondia a quem bebeu.
func TestUseHealPotionBroadcastsToViewers(t *testing.T) {
	const potion = 408
	// loadResult é o fallback para QUALQUER conta (handler_test.go:172), então os dois
	// clientes nascem em (5,5) — Chebyshev 0, bem dentro do ViewRange 16.
	db := healPotionDB(world.Item{Index: potion}, 500, 100000, 0, 0)
	effects := map[int][]content.BaseEffect{potion: {{Eff: efHp, Val: 1000}}}
	addr, stop := startServerTickItems(t, db, map[int]int{potion: volHpMpPotion}, effects)
	defer stop()

	drinker := enterWorld(t, addr) // conn 1
	defer drinker.Close()
	observer := enterWorldAs(t, addr, "tradeb") // conn 2, em visão do conn 1
	defer observer.Close()

	body := protocol.MsgUseItemBody{SourType: world.ItemPlaceCarry, SourPos: 0}
	send(t, drinker, protocol.MsgUseItem, body.Encode())

	// O observador precisa receber o score de QUEM BEBEU: HEADER.ID identifica o sujeito,
	// então precisa ser conn 1, não o conn do próprio observador.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		h, p := readFrameHeader(t, observer)
		if h.Type != protocol.MsgUpdateScore || h.ID != 1 {
			continue
		}
		hp := int32(binary.LittleEndian.Uint32(p[24:28]))
		if hp < 1500 {
			continue // o fio passivo pode emitir um score antes da poção chegar
		}
		if curr := int32(binary.LittleEndian.Uint32(p[124:128])); curr != hp {
			t.Errorf("CurrHp = %d, want %d (must agree with Score.Hp)", curr, hp)
		}
		return
	}
	t.Fatal("observer never saw the drinker's healed score — the pot heal is invisible to the attacker (#99)")
}

// TestUseHealPotionNoBroadcastOutOfView fixa o filtro de distância de BroadcastInView: um
// jogador além do ViewRange não pode receber o score de quem bebeu.
func TestUseHealPotionNoBroadcastOutOfView(t *testing.T) {
	const potion = 409
	db := healPotionDB(world.Item{Index: potion}, 500, 100000, 0, 0)
	// A conta 11 nasce longe; a conta 7 mantém o loadResult em (5,5).
	far := db.loadResult
	far.X, far.Y = 500, 500
	db.loads = map[int64]world.CharacterState{11: far}
	effects := map[int][]content.BaseEffect{potion: {{Eff: efHp, Val: 1000}}}
	addr, stop := startServerTickItems(t, db, map[int]int{potion: volHpMpPotion}, effects)
	defer stop()

	drinker := enterWorld(t, addr)
	defer drinker.Close()
	observer := enterWorldAs(t, addr, "tradeb")
	defer observer.Close()

	body := protocol.MsgUseItemBody{SourType: world.ItemPlaceCarry, SourPos: 0}
	send(t, drinker, protocol.MsgUseItem, body.Encode())

	for i := 0; i < 8; i++ {
		h, _, ok := readMaybeHeader(t, observer)
		if !ok {
			return // o fluxo ficou mudo — nada vazou
		}
		if h.Type == protocol.MsgUpdateScore && h.ID == 1 {
			t.Fatal("out-of-view player received the drinker's score, want it filtered by ViewRange")
		}
	}
}

// TestEnterWorldViewScoreStaysUnicast trava a exceção do sendScoreSelf: o score de um
// recém-chegado NÃO pode ser multicast, porque enterWorldView o empurra ANTES de transmitir
// o CreateMob do recém-chegado — clientes em visão receberiam um score de uma entidade que
// ainda não criaram. O legado não tem SendScore nenhum neste caminho
// (ProcessDBMessage.cpp:1017-1037); os observadores aprendem o HP do recém-chegado pelo CreateMob.
func TestEnterWorldViewScoreStaysUnicast(t *testing.T) {
	db := healPotionDB(world.Item{}, 500, 1000, 0, 0)
	addr, stop := startServerClockItems(t, db, nil, nil)
	defer stop()

	first := enterWorld(t, addr) // conn 1, já no mundo
	defer first.Close()
	second := enterWorldAs(t, addr, "tradeb") // conn 2 loga na mesma célula
	defer second.Close()

	// o conn 1 pode ver o CreateMob/PKInfo do conn 2, mas nunca o UpdateScore do conn 2.
	for i := 0; i < 8; i++ {
		h, _, ok := readMaybeHeader(t, first)
		if !ok {
			return
		}
		if h.Type == protocol.MsgUpdateScore && h.ID == 2 {
			t.Fatal("newcomer's score was multicast before its CreateMob — unknown-entity packet (B1)")
		}
	}
}

func TestUseSephiraBookConsumesOneFromStack(t *testing.T) {
	const book = 4200
	db := newDB()
	st := world.CharacterState{Slot: 0, Name: "Hero", X: 5, Y: 5, HP: 1000, MaxHP: 1000}
	st.Carry[0] = world.Item{Index: book, Effects: [3]world.Effect{{Effect: efAmount, Value: 3}}}
	db.loadResult = st

	addr, stop := startServerClockVol(t, db, map[int]int{book: volSephiraLo})
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	body := protocol.MsgUseItemBody{SourType: world.ItemPlaceCarry, SourPos: 0}
	send(t, c, protocol.MsgUseItem, body.Encode())

	item := expect(t, c, protocol.MsgSendItem)
	if got := le16(item[4:6]); got != book {
		t.Fatalf("carry slot 0 item = %d, want stacked Sephira book", got)
	}
	if item[6] != efAmount || item[7] != 2 {
		t.Fatalf("book amount effect = %d.%d, want %d.2", item[6], item[7], efAmount)
	}
	etc := expect(t, c, protocol.MsgUpdateEtc)
	if learn := int64(binary.LittleEndian.Uint64(etc[12:])); learn != 1<<24 {
		t.Fatalf("UpdateEtc.Learn = %#x, want Sephira bit24", learn)
	}
}

func TestUseFairyDust(t *testing.T) {
	const dust = 5001
	addr, stop := startServerClockVol(t, fairyDustDB(0, world.Item{Index: dust}), map[int]int{dust: volFairyDust})
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	body := protocol.MsgUseItemBody{SourType: world.ItemPlaceCarry, SourPos: 0}
	send(t, c, protocol.MsgUseItem, body.Encode())

	sawSendItem, sawScore, sawEtc, sawMotion := false, false, false, false
	for range 6 {
		ty, payload, ok := readMaybe(t, c)
		if !ok {
			break
		}
		switch ty {
		case protocol.MsgSendItem:
			sawSendItem = true
			if le16(payload[4:6]) != 0 {
				t.Errorf("carry slot 0 item = %d, want empty after use", le16(payload[4:6]))
			}
		case protocol.MsgUpdateScore:
			sawScore = true
		case protocol.MsgUpdateEtc:
			sawEtc = true
			if got := int64(binary.LittleEndian.Uint64(payload[4:12])); got != level.NextLevelExp(0) {
				t.Errorf("UpdateEtc.Exp = %d, want %d", got, level.NextLevelExp(0))
			}
		case protocol.MsgMotion:
			sawMotion = true
			if got := le16(payload[0:2]); got != motionLevelUp {
				t.Errorf("motion = %d, want %d", got, motionLevelUp)
			}
		}
	}
	if !sawSendItem {
		t.Error("missing MsgSendItem after using fairy dust")
	}
	if !sawScore {
		t.Error("missing MsgUpdateScore after using fairy dust")
	}
	if !sawEtc {
		t.Error("missing MsgUpdateEtc after using fairy dust")
	}
	if !sawMotion {
		t.Error("missing MsgMotion after using fairy dust")
	}
}

func TestUseFairyDustStack(t *testing.T) {
	const dust = 5001
	item := world.Item{Index: dust, Effects: [3]world.Effect{{Effect: efAmount, Value: 3}}}
	addr, stop := startServerClockVol(t, fairyDustDB(0, item), map[int]int{dust: volFairyDust})
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	body := protocol.MsgUseItemBody{SourType: world.ItemPlaceCarry, SourPos: 0}
	send(t, c, protocol.MsgUseItem, body.Encode())

	payload := expect(t, c, protocol.MsgSendItem)
	if got := le16(payload[4:6]); got != dust {
		t.Fatalf("carry slot 0 item = %d, want stacked dust", got)
	}
	if payload[6] != efAmount || payload[7] != 2 {
		t.Errorf("effect0 = %d.%d, want %d.2", payload[6], payload[7], efAmount)
	}
}

func TestUseFairyDustAtCap(t *testing.T) {
	const dust = 5001
	addr, stop := startServerClockVol(t, fairyDustDB(int(level.MaxLevel), world.Item{Index: dust}), map[int]int{dust: volFairyDust})
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	body := protocol.MsgUseItemBody{SourType: world.ItemPlaceCarry, SourPos: 0}
	send(t, c, protocol.MsgUseItem, body.Encode())

	sawSendItem, sawEtc := false, false
	for range 4 {
		ty, payload, ok := readMaybe(t, c)
		if !ok {
			break
		}
		switch ty {
		case protocol.MsgSendItem:
			sawSendItem = true
			if le16(payload[4:6]) != 0 {
				t.Errorf("carry slot 0 item = %d, want empty after use", le16(payload[4:6]))
			}
		case protocol.MsgUpdateEtc:
			sawEtc = true
			if got := int64(binary.LittleEndian.Uint64(payload[4:12])); got != level.MaxExp {
				t.Errorf("UpdateEtc.Exp = %d, want %d", got, level.MaxExp)
			}
		case protocol.MsgMotion:
			t.Fatalf("got level-up motion at cap")
		}
	}
	if !sawSendItem {
		t.Error("missing MsgSendItem after using fairy dust at cap")
	}
	if !sawEtc {
		t.Error("missing MsgUpdateEtc after using fairy dust at cap")
	}
}

func TestUseSilverBar1Bi(t *testing.T) {
	addr, stop := startServerClockVol(t, silverBarDB(4011, 0), map[int]int{4011: volSilverBar})
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	body := protocol.MsgUseItemBody{SourType: world.ItemPlaceCarry, SourPos: 0}
	send(t, c, protocol.MsgUseItem, body.Encode())

	sawSendItem, sawEtc := false, false
	for range 4 {
		ty, payload, ok := readMaybe(t, c)
		if !ok {
			break
		}
		switch ty {
		case protocol.MsgSendItem:
			sawSendItem = true
			if le16(payload[2:4]) != 0 || le16(payload[4:6]) != 0 {
				t.Errorf("slot/item = %d/%d, want slot 0 empty", le16(payload[2:4]), le16(payload[4:6]))
			}
		case protocol.MsgUpdateEtc:
			sawEtc = true
			if got := int32(le(payload[28:32])); got != 1_000_000_000 {
				t.Errorf("coin = %d, want 1000000000", got)
			}
		}
	}
	if !sawSendItem {
		t.Error("missing MsgSendItem after using silver bar")
	}
	if !sawEtc {
		t.Error("missing MsgUpdateEtc after using silver bar")
	}
}

// TestUseSilverBarConsumesOneFromStack é o irmão da issue #259: useSilverBar também
// apagava o slot, então trocar uma barra de uma pilha queimava o resto. Legado:
// _MSG_UseItem.cpp:4277. Só o ouro de uma barra pode ser creditado por uso.
func TestUseSilverBarConsumesOneFromStack(t *testing.T) {
	db := silverBarDB(4026, 0)
	st := db.loadResult
	st.Carry[0] = world.Item{Index: 4026, Effects: [3]world.Effect{{Effect: efAmount, Value: 5}}}
	db.loadResult = st
	addr, stop := startServerClockVol(t, db, map[int]int{4026: volSilverBar})
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	body := protocol.MsgUseItemBody{SourType: world.ItemPlaceCarry, SourPos: 0}
	send(t, c, protocol.MsgUseItem, body.Encode())

	sawSendItem, sawEtc := false, false
	for range 4 {
		ty, payload, ok := readMaybe(t, c)
		if !ok {
			break
		}
		switch ty {
		case protocol.MsgSendItem:
			sawSendItem = true
			if le16(payload[4:6]) != 4026 {
				t.Fatalf("carry slot 0 item = %d, want 4026 (whole stack consumed)", le16(payload[4:6]))
			}
			if payload[6] != efAmount || payload[7] != 4 {
				t.Errorf("amount effect = (%d,%d), want (%d,4)", payload[6], payload[7], efAmount)
			}
		case protocol.MsgUpdateEtc:
			sawEtc = true
			if got := int32(le(payload[28:32])); got != 1_000_000 {
				t.Errorf("coin = %d, want 1000000 (one bar, not the stack)", got)
			}
		}
	}
	if !sawSendItem {
		t.Error("missing MsgSendItem after using silver bar")
	}
	if !sawEtc {
		t.Error("missing MsgUpdateEtc after using silver bar")
	}
}

func TestUseSilverBarOver2G(t *testing.T) {
	addr, stop := startServerClockVol(t, silverBarDB(4011, 1_500_000_000), map[int]int{4011: volSilverBar})
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	body := protocol.MsgUseItemBody{SourType: world.ItemPlaceCarry, SourPos: 0}
	send(t, c, protocol.MsgUseItem, body.Encode())

	if got := readNotice(t, c); got != NoticeCargoFull {
		t.Fatalf("overflow notice = %d, want NoticeCargoFull", got)
	}
	item := expect(t, c, protocol.MsgSendItem)
	if le16(item[2:4]) != 0 || le16(item[4:6]) != 4011 {
		t.Errorf("slot/item = %d/%d, want slot 0 item 4011 preserved", le16(item[2:4]), le16(item[4:6]))
	}
	if ty, _, ok := readMaybe(t, c); ok && ty == protocol.MsgUpdateEtc {
		t.Error("overflow use sent MsgUpdateEtc; coin should be unchanged")
	}
}

// Gema Estelar (issue #140): o item 700 registra o ponto de save do warp; os itens 776/3430
// (Pergaminho de Portal) teleportam de volta para ele. Os testes abaixo usam os índices reais de
// catálogo por legibilidade, mas o harness classifica itens puramente pelo mapa
// vols, não por uma consulta real ao ItemList.

// gemaEstelarDB semeia um personagem em (x,y) com um ponto de save existente
// (saveX,saveY — 0,0 para "nunca salvo") e os itens carregados dados.
func gemaEstelarDB(x, y, saveX, saveY int16, carry ...world.Item) *fakeDB {
	db := newDB()
	st := world.CharacterState{Slot: 0, Name: "Hero", X: x, Y: y, SaveX: saveX, SaveY: saveY, HP: 1000, MaxHP: 1000}
	for i, it := range carry {
		if i >= len(st.Carry) {
			break
		}
		st.Carry[i] = it
	}
	db.loadResult = st
	return db
}

// startServerClockVolGrid é o startServerClockVol com uma dimensão de grade
// configurável, necessária para testes que colocam o jogador em coordenadas reais de
// cidade/Pesadelo (world.Village e inPesadelo comparam com constantes reais de
// segmento/retângulo legadas, que passam da grade usual de teste 16x16).
func startServerClockVolGrid(t *testing.T, persist world.Persistence, vols map[int]int, gridDim int) (string, func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	d := New(Config{Log: log, ItemVolatiles: vols})
	w := world.New(world.Config{GridDim: gridDim}, log, persist, d.Handle)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { _ = w.Serve(ctx, ln); close(done) }()
	return ln.Addr().String(), func() {
		cancel()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Error("server did not stop")
		}
	}
}

func recallScrollDB(lastCity int16, hp, mp int32, carry ...world.Item) *fakeDB {
	db := newDB()
	st := world.CharacterState{
		Slot: 0, Name: "Hero", X: 5, Y: 5, LastCity: lastCity,
		HP: hp, MaxHP: 1000, MP: mp, MaxMP: 500,
	}
	for i, it := range carry {
		if i >= len(st.Carry) {
			break
		}
		st.Carry[i] = it
	}
	db.loadResult = st
	return db
}

func useItemFrame(t *testing.T, c net.Conn, slot int) {
	t.Helper()
	body := protocol.MsgUseItemBody{SourType: world.ItemPlaceCarry, SourPos: int32(slot)}
	send(t, c, protocol.MsgUseItem, body.Encode())
}

func useItemWarpFrame(t *testing.T, c net.Conn, slot int, warpID uint16) {
	t.Helper()
	body := protocol.MsgUseItemBody{SourType: world.ItemPlaceCarry, SourPos: int32(slot), WarpID: warpID}
	send(t, c, protocol.MsgUseItem, body.Encode())
}

func TestHuntingScrollDestinations(t *testing.T) {
	carry := make([]world.Item, len(huntingScrollDestinations))
	vols := make(map[int]int, len(huntingScrollDestinations))
	for row := range huntingScrollDestinations {
		index := itemHuntingScrollBase + row
		carry[row] = world.Item{Index: int16(index), Effects: [3]world.Effect{{Effect: efAmount, Value: 10}}}
		vols[index] = volHuntingScroll
	}
	db := recallScrollDB(0, 1000, 500, carry...)
	addr, stop := startServerClockVolGrid(t, db, vols, world.DefaultGridDim)
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	for row, destinations := range huntingScrollDestinations {
		for col, want := range destinations {
			warpID := uint16(col + 1)
			useItemWarpFrame(t, c, row, warpID)
			expect(t, c, protocol.MsgSendItem)
			action := expectAction(t, c)
			if action.TargetX != want[0] || action.TargetY != want[1] {
				t.Errorf("item %d WarpID %d target = (%d,%d), want (%d,%d)",
					itemHuntingScrollBase+row, warpID, action.TargetX, action.TargetY, want[0], want[1])
			}
		}
	}
}

func TestHuntingScrollConsumesSingleAndStackedItems(t *testing.T) {
	const stackedAmount = 3
	items := []world.Item{
		{Index: itemHuntingScrollBase},
		{Index: itemHuntingScrollBase + 1, Effects: [3]world.Effect{{Effect: efAmount, Value: stackedAmount}}},
	}
	vols := map[int]int{
		itemHuntingScrollBase:     volHuntingScroll,
		itemHuntingScrollBase + 1: volHuntingScroll,
	}
	db := recallScrollDB(0, 1000, 500, items...)
	addr, stop := startServerClockVolGrid(t, db, vols, world.DefaultGridDim)
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	useItemWarpFrame(t, c, 0, 1)
	single := expect(t, c, protocol.MsgSendItem)
	if got := le16(single[4:6]); got != 0 {
		t.Errorf("single scroll slot item = %d, want empty", got)
	}
	expectAction(t, c)

	useItemWarpFrame(t, c, 1, 1)
	stacked := expect(t, c, protocol.MsgSendItem)
	if got := le16(stacked[4:6]); got != itemHuntingScrollBase+1 {
		t.Fatalf("stacked scroll slot item = %d, want %d", got, itemHuntingScrollBase+1)
	}
	if stacked[6] != efAmount || stacked[7] != stackedAmount-1 {
		t.Errorf("stacked scroll amount effect = %d.%d, want %d.%d", stacked[6], stacked[7], efAmount, stackedAmount-1)
	}
	expectAction(t, c)
}

func TestHuntingScrollRejectsInvalidItemAndWarpID(t *testing.T) {
	tests := []struct {
		name   string
		item   int16
		warpID uint16
	}{
		{name: "item below range", item: itemHuntingScrollBase - 1, warpID: 1},
		{name: "item above range", item: itemHuntingScrollLast + 1, warpID: 1},
		{name: "WarpID zero", item: itemHuntingScrollBase, warpID: 0},
		{name: "WarpID eleven", item: itemHuntingScrollBase, warpID: 11},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := recallScrollDB(0, 1000, 500, world.Item{Index: tt.item})
			addr, stop := startServerClockVolGrid(t, db, map[int]int{int(tt.item): volHuntingScroll}, 16)
			defer stop()
			c := enterWorld(t, addr)
			defer c.Close()

			useItemWarpFrame(t, c, 0, tt.warpID)
			item := expect(t, c, protocol.MsgSendItem)
			if got := int16(le16(item[4:6])); got != tt.item {
				t.Errorf("rejected slot item = %d, want preserved item %d", got, tt.item)
			}
			if ty, _, ok := readMaybe(t, c); ok {
				t.Errorf("rejected hunting scroll produced unexpected frame %#x", ty)
			}
		})
	}
}

// TestUseRecallScrollConsumesAndRecalls porta o bloco "Retorno" de _MSG_UseItem.cpp:
// EF_VOLATILE 11 chama DoRecall e consome um Pergaminho_Retorno.
func TestUseRecallScrollConsumesAndRecalls(t *testing.T) {
	const recall = 410
	db := recallScrollDB(2, 321, 123, world.Item{Index: recall})
	addr, stop := startServerClockVolGrid(t, db, map[int]int{recall: volRecallScroll}, 2600)
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	useItemFrame(t, c, 0)
	item := expect(t, c, protocol.MsgSendItem)
	if got := le16(item[4:6]); got != 0 {
		t.Errorf("recall slot = %d, want empty after use", got)
	}
	action := expectAction(t, c)
	if action.PosX != 5 || action.PosY != 5 || action.Effect != 1 {
		t.Errorf("recall action origin/effect = (%d,%d)/%d, want (5,5)/1", action.PosX, action.PosY, action.Effect)
	}
	if city := world.Village(action.TargetX, action.TargetY); city != 2 {
		t.Fatalf("recall target = (%d,%d) in city %d, want last city 2", action.TargetX, action.TargetY, city)
	}
	if ty, _, ok := readMaybe(t, c); ok && ty == protocol.MsgSetHpMp {
		t.Fatal("recall scroll sent SetHpMp, want no HP/MP bar mutation")
	}

	send(t, c, protocol.MsgCharacterLogout, nil)
	expect(t, c, protocol.MsgCNFCharacterLogout)
	save, n := db.lastSavedChar()
	if n == 0 {
		t.Fatal("character was not saved on logout")
	}
	if save.HP != 321 || save.MP != 123 {
		t.Fatalf("saved HP/MP = %d/%d, want unchanged 321/123", save.HP, save.MP)
	}
	if _, ok := savedItemAt(save.Carry, 0); ok {
		t.Fatal("saved carry slot 0 still has recall scroll, want consumed")
	}
}

// TestUseRecallScrollStack decrementa EF_AMOUNT no pergaminho de retorno de 10 cargas e ainda
// retorna ao spawn da última cidade do personagem.
func TestUseRecallScrollStack(t *testing.T) {
	const recall10x = 411
	db := recallScrollDB(1, 1000, 500, world.Item{Index: recall10x, Effects: [3]world.Effect{{Effect: efAmount, Value: 10}}})
	addr, stop := startServerClockVolGrid(t, db, map[int]int{recall10x: volRecallScroll}, 2700)
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	useItemFrame(t, c, 0)
	item := expect(t, c, protocol.MsgSendItem)
	if got := le16(item[4:6]); got != recall10x {
		t.Fatalf("carry slot 0 item = %d, want stacked recall scroll", got)
	}
	if item[6] != efAmount || item[7] != 9 {
		t.Errorf("effect0 = %d.%d, want %d.9", item[6], item[7], efAmount)
	}
	action := expectAction(t, c)
	if city := world.Village(action.TargetX, action.TargetY); city != 1 {
		t.Fatalf("recall target = (%d,%d) in city %d, want last city 1", action.TargetX, action.TargetY, city)
	}
}

// TestUseGemaEstelarThenPortalRoundTrip: salvar na posição atual e depois
// usar o pergaminho de portal teleporta o jogador de volta exatamente àquele tile —
// a mecânica central que a issue #140 reporta como quebrada (nenhum dos itens fazia nada).
func TestUseGemaEstelarThenPortalRoundTrip(t *testing.T) {
	const gema, portal = 700, 776
	db := gemaEstelarDB(5, 5, 0, 0, world.Item{Index: gema}, world.Item{Index: portal})
	addr, stop := startServerClockVol(t, db, map[int]int{gema: volGemaEstelar, portal: volPortalScroll})
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	useItemFrame(t, c, 0) // Gema Estelar
	saved := expect(t, c, protocol.MsgSendItem)
	if got := le16(saved[4:6]); got != 0 {
		t.Errorf("gema slot = %d, want empty after use", got)
	}
	if got := noticeCode(t, expect(t, c, protocol.MsgMessageBoxOk)); got != NoticeSetWarp {
		t.Errorf("notice = %v, want NoticeSetWarp", got)
	}

	useItemFrame(t, c, 1) // Pergaminho de Portal
	scroll := expect(t, c, protocol.MsgSendItem)
	if got := le16(scroll[4:6]); got != 0 {
		t.Errorf("portal slot = %d, want empty after use", got)
	}
	action := expectAction(t, c)
	if action.TargetX != 5 || action.TargetY != 5 {
		t.Errorf("teleport target = (%d,%d), want (5,5)", action.TargetX, action.TargetY)
	}
}

// TestUsePortalScrollStack: um pergaminho de múltiplas cargas (EF_AMOUNT) decrementa
// em vez de limpar, e ainda teleporta.
func TestUsePortalScrollStack(t *testing.T) {
	const portal = 776
	db := gemaEstelarDB(5, 5, 20, 20, world.Item{Index: portal, Effects: [3]world.Effect{{Effect: efAmount, Value: 3}}})
	addr, stop := startServerClockVol(t, db, map[int]int{portal: volPortalScroll})
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	useItemFrame(t, c, 0)
	payload := expect(t, c, protocol.MsgSendItem)
	if got := le16(payload[4:6]); got != portal {
		t.Fatalf("carry slot 0 item = %d, want stacked scroll", got)
	}
	if payload[6] != efAmount || payload[7] != 2 {
		t.Errorf("effect0 = %d.%d, want %d.2", payload[6], payload[7], efAmount)
	}
	action := expectAction(t, c)
	if action.TargetX != 20 || action.TargetY != 20 {
		t.Errorf("teleport target = (%d,%d), want (20,20)", action.TargetX, action.TargetY)
	}
}

// TestUsePortalScrollNoSavePoint: um personagem que nunca usou uma Gema Estelar
// tem SaveX==SaveY==0; o pergaminho precisa não fazer nada em vez de warpar para a origem do mapa.
func TestUsePortalScrollNoSavePoint(t *testing.T) {
	const portal = 776
	db := gemaEstelarDB(5, 5, 0, 0, world.Item{Index: portal})
	addr, stop := startServerClockVol(t, db, map[int]int{portal: volPortalScroll})
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	useItemFrame(t, c, 0)
	if ty, _, ok := readMaybe(t, c); ok {
		t.Errorf("got %#x, want no response (no save point set)", ty)
	}
}

// TestUseGemaEstelarBlockedInCity: o legado se recusa a registrar um ponto de save
// quando se está dentro dos limites de uma cidade (BASE_GetVillage); o item não é
// consumido e o cliente é informado do porquê. O retângulo CityLimit de Armia é
// (2052,2052)-(2171,2163) (world/city.go), então a grade de teste precisa ser grande
// o bastante para colocar o jogador ali.
func TestUseGemaEstelarBlockedInCity(t *testing.T) {
	const gema = 700
	const armiaX, armiaY = 2100, 2100
	db := gemaEstelarDB(armiaX, armiaY, 0, 0, world.Item{Index: gema})
	addr, stop := startServerClockVolGrid(t, db, map[int]int{gema: volGemaEstelar}, 2200)
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	useItemFrame(t, c, 0)
	if got := readNotice(t, c); got != NoticeCantUseHere {
		t.Fatalf("notice = %d, want NoticeCantUseHere", got)
	}
	item := expect(t, c, protocol.MsgSendItem)
	if got := le16(item[4:6]); got != gema {
		t.Errorf("carry slot 0 item = %d, want %d preserved (not consumed)", got, gema)
	}
}

// TestUseGemaEstelarPesadeloCarveOut: os dois segmentos de dungeon instanciada
// Pesadelo são isentos do gate de cidade/arena ("goto CanSave" legado), então um
// save ainda funciona ali mesmo que o tile normalmente fosse bloqueado.
// (9*128, 1*128) = (1152,128) cai dentro do segmento Pesadelo_A.
func TestUseGemaEstelarPesadeloCarveOut(t *testing.T) {
	const gema = 700
	const pesadeloX, pesadeloY = 1200, 150
	db := gemaEstelarDB(pesadeloX, pesadeloY, 0, 0, world.Item{Index: gema})
	addr, d, w, _ := startKefraServer(t, db)
	c := enterWorld(t, addr)
	defer c.Close()
	// O login agora recorda posições velhas do Pesadelo. Coloque o jogador admitido
	// dentro do mapa para testar a exceção do ponto de save independentemente do login.
	runInLoop(t, w, func() {
		d.itemVolatiles = map[int]int{gema: volGemaEstelar}
		s, _ := w.SessionByName("Hero")
		w.SetEntityPos(s.Conn, pesadeloX, pesadeloY)
	})
	useItemFrame(t, c, 0)
	saved := expect(t, c, protocol.MsgSendItem)
	if got := le16(saved[4:6]); got != 0 {
		t.Errorf("gema slot = %d, want empty after use (Pesadelo carve-out should allow the save)", got)
	}
	if got := noticeCode(t, expect(t, c, protocol.MsgMessageBoxOk)); got != NoticeSetWarp {
		t.Errorf("notice = %v, want NoticeSetWarp", got)
	}
}

// TestUsePortalScrollBlockedFromPesadeloSavePoint: um ponto salvo que por acaso
// cai dentro de um segmento Pesadelo recusa o portal (não dá para warpar para a
// instância de fora), independentemente da posição atual do próprio jogador.
func TestUsePortalScrollBlockedFromPesadeloSavePoint(t *testing.T) {
	const portal = 776
	const pesadeloX, pesadeloY = 1200, 150
	db := gemaEstelarDB(5, 5, pesadeloX, pesadeloY, world.Item{Index: portal})
	addr, stop := startServerClockVol(t, db, map[int]int{portal: volPortalScroll})
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	useItemFrame(t, c, 0)
	if got := readNotice(t, c); got != NoticeCantUseHere {
		t.Fatalf("notice = %d, want NoticeCantUseHere", got)
	}
	item := expect(t, c, protocol.MsgSendItem)
	if got := le16(item[4:6]); got != portal {
		t.Errorf("carry slot 0 item = %d, want %d preserved (not consumed)", got, portal)
	}
}
