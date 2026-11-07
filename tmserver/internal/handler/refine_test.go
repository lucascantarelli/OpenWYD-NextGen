package handler

import (
	"context"
	"encoding/binary"
	"log/slog"
	"net"
	"testing"
	"time"

	"openwyd-nextgen/tmserver/internal/content"
	"openwyd-nextgen/tmserver/internal/protocol"
	"openwyd-nextgen/tmserver/internal/refine"
	"openwyd-nextgen/tmserver/internal/world"
)

// Os itens de pó de refino, como o ItemList.csv os define.
const (
	itemPoeiraOri = 412
	itemPoeiraLac = 413
	itemArmor     = 555 // uma armadura refinável comum
	// itemSephirotTransKnight é uma pedra de quest/troféu (issue #133): equipável,
	// sem EF_VOLATILE, agora marcada EF_NOSANC no ItemList.csv.
	itemSephirotTransKnight = 1760
)

// rateTable é um SancRate injetável. Indexe em level+1, como o real.
type rateTable [3][12]int

func (r rateTable) Rate(anvil, idx int) int {
	if anvil < 0 || anvil >= len(r) || idx < 0 || idx >= len(r[anvil]) {
		return 0
	}
	return r[anvil][idx]
}

// alwaysRate é uma tabela em que toda consulta Ori/Lac devolve rate.
func alwaysRate(rate int) rateTable {
	var t rateTable
	for a := range t {
		for i := range t[a] {
			t[a][i] = rate
		}
	}
	return t
}

// refineFixture monta um dispatcher cujo catálogo espelha a produção: os dois
// pós classificados por EF_VOLATILE, e uma armadura refinável. Espelhar o
// Config real importa — um harness de handler com o catálogo desligado faz silenciosamente
// todo gate EF_ ler 0 (ver a lição B12).
type refineFixture struct {
	d *Dispatcher
	w *world.World
	s *world.Session
	e *world.Entity
}

func newRefineFixture(t *testing.T, tbl refine.RateTable, effects map[int][]content.BaseEffect) *refineFixture {
	t.Helper()
	if effects == nil {
		effects = map[int][]content.BaseEffect{}
	}
	d := New(Config{
		Log:           slog.New(slog.DiscardHandler),
		SancRate:      tbl,
		ItemVolatiles: map[int]int{itemPoeiraOri: volDustOri, itemPoeiraLac: volDustLac, itemLacto100: volDustLac},
		ItemEffects:   effects,
	})
	w := world.New(world.Config{GridDim: 16}, slog.New(slog.DiscardHandler), nil, nil)
	return &refineFixture{
		d: d,
		w: w,
		s: &world.Session{Conn: 1, Mode: world.UserPlay},
		e: &world.Entity{ID: 1, HP: 100},
	}
}

// refine arrasta o pó do slot 0 do carry para o item no slot 1 do carry.
func (f *refineFixture) refine(dust int16) {
	f.e.Carry[0] = world.Item{Index: dust}
	body := protocol.MsgUseItemBody{
		SourType: world.ItemPlaceCarry, SourPos: 0,
		DestType: world.ItemPlaceCarry, DestPos: 1,
	}
	vol := f.d.itemVolatiles[int(dust)]
	f.d.refineItem(f.w, f.s, f.e, body, 0, vol)
}

func (f *refineFixture) target() world.Item { return f.e.Carry[1] }

func TestRefineSuccessRaisesLevel(t *testing.T) {
	f := newRefineFixture(t, alwaysRate(100), nil)
	f.e.Carry[1] = world.Item{Index: itemArmor}

	f.refine(itemPoeiraLac)

	if got := refine.Level(f.target()); got != 1 {
		t.Errorf("level = %d, want 1", got)
	}
	if !f.e.Carry[0].Empty() {
		t.Errorf("dust not consumed: %+v", f.e.Carry[0])
	}
	// O nível precisa cair num par EF_SANC real que o resto do servidor consegue ler.
	if f.target().Effects[0] != (world.Effect{Effect: efSanc, Value: 1}) {
		t.Errorf("Effects[0] = %+v, want {EF_SANC 1}", f.target().Effects[0])
	}
	if got := f.w.SentOfType(f.s, protocol.MsgMessagePanel); got != 1 {
		t.Errorf("message panels = %d, want 1", got)
	}
	if got := f.w.SentOfType(f.s, protocol.MsgMotion); got != 1 {
		t.Errorf("motions = %d, want 1", got)
	}
}

// Um refino por pó que falha nunca destrói nem rebaixa o item — isso só acontece
// nos ramos selado/celestial/brinco, que estão fora do escopo.
func TestRefineFailKeepsItemAndAccruesPity(t *testing.T) {
	f := newRefineFixture(t, alwaysRate(0), nil)
	f.e.Carry[1] = world.Item{Index: itemArmor}
	before := f.target().Index

	f.refine(itemPoeiraLac)

	if f.target().Index != before {
		t.Errorf("item index changed on failure: %d → %d", before, f.target().Index)
	}
	if got := refine.Level(f.target()); got != 0 {
		t.Errorf("level = %d, want 0 (a failure must not downgrade)", got)
	}
	if !f.e.Carry[0].Empty() {
		t.Error("dust must be consumed on failure too")
	}
	// rand()#2 % 4 = 3 (o stream da semente 1), e 3 > 2, então esta falha específica
	// NÃO acumula pena — o sorteio é de 3 em 4.
	if got := refine.Pity(f.target()); got != 0 {
		t.Errorf("pity = %d, want 0 (rand()#2%%4 = 3 misses the <=2 window)", got)
	}
	if got := f.w.SentOfType(f.s, protocol.MsgMessagePanel); got != 1 {
		t.Errorf("message panels = %d, want 1", got)
	}
	if got := f.w.SentOfType(f.s, protocol.MsgMotion); got != 1 {
		t.Errorf("motions = %d, want 1", got)
	}
}

// A pena acumula ao longo de falhas repetidas e é o que acaba levando um refino
// adiante (g_pSuccessRate).
func TestRefinePityAccrues(t *testing.T) {
	f := newRefineFixture(t, alwaysRate(0), nil)
	f.e.Carry[1] = world.Item{Index: itemArmor}
	for i := 0; i < 6; i++ {
		f.refine(itemPoeiraLac)
	}
	if got := refine.Pity(f.target()); got == 0 {
		t.Errorf("pity = 0 after 6 failures, want > 0")
	}
	if got := refine.Level(f.target()); got != 0 {
		t.Errorf("level = %d, want 0", got)
	}
}

func TestRefineGates(t *testing.T) {
	cases := []struct {
		name    string
		dust    int16
		target  world.Item
		effects map[int][]content.BaseEffect
		want    Notice
	}{
		{
			name:   "Ori refuses an item already at +6",
			dust:   itemPoeiraOri,
			target: world.Item{Index: itemArmor, Effects: [3]world.Effect{{Effect: efSanc, Value: 6}}},
			want:   NoticeCantRefineMore,
		},
		{
			name:   "the dust path stops at +9",
			dust:   itemPoeiraLac,
			target: world.Item{Index: itemArmor, Effects: [3]world.Effect{{Effect: efSanc, Value: 9}}},
			want:   NoticeCantRefineMore,
		},
		{
			name:   "+11 and above is refused",
			dust:   itemPoeiraLac,
			target: world.Item{Index: itemArmor, Effects: [3]world.Effect{{Effect: efSanc, Value: 234}}}, // +11
			want:   NoticeCantRefineMore,
		},
		{
			name:   "item 769 is capped at +9",
			dust:   itemPoeiraLac,
			target: world.Item{Index: item769, Effects: [3]world.Effect{{Effect: efSanc, Value: 234}}},
			want:   NoticeCantRefineMore,
		},
		{
			name:   "a dust cannot be refined onto another dust",
			dust:   itemPoeiraLac,
			target: world.Item{Index: itemPoeiraOri},
			want:   NoticeOnlyToEquips,
		},
		{
			name:    "EF_NOSANC items never refine",
			dust:    itemPoeiraLac,
			target:  world.Item{Index: itemArmor},
			effects: map[int][]content.BaseEffect{itemArmor: {{Eff: efNoSanc, Val: 1}}},
			want:    NoticeCantRefineMore,
		},
		{
			// issue #133: Sephirot(TransKnight) é uma pedra de quest/troféu sem
			// EF_VOLATILE, então nada a barrava antes de o ItemList.csv começar a marcá-la
			// como EF_NOSANC.
			name:    "Sephirot(TransKnight) never refines (issue #133)",
			dust:    itemPoeiraLac,
			target:  world.Item{Index: itemSephirotTransKnight},
			effects: map[int][]content.BaseEffect{itemSephirotTransKnight: {{Eff: efNoSanc, Val: 1}}},
			want:    NoticeCantRefineMore,
		},
		{
			name: "an item with all three effect slots taken has nowhere to store a level",
			dust: itemPoeiraLac,
			target: world.Item{Index: itemArmor, Effects: [3]world.Effect{
				{Effect: efAc, Value: 10}, {Effect: efHp, Value: 20}, {Effect: efMp, Value: 30},
			}},
			want: NoticeCantRefineMore,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			f := newRefineFixture(t, alwaysRate(100), tt.effects)
			f.e.Carry[1] = tt.target
			before := f.target()

			f.refine(tt.dust)

			if f.target() != before {
				t.Errorf("a refused refine mutated the item: %+v → %+v", before, f.target())
			}
			if f.e.Carry[0].Empty() {
				t.Error("a refused refine consumed the dust")
			}
			if got := f.w.SentOfType(f.s, protocol.MsgMessagePanel); got != 1 {
				t.Errorf("message panels = %d, want 1", got)
			}
			if got := f.w.SentOfType(f.s, protocol.MsgMotion); got != 0 {
				t.Errorf("motions = %d, want 0 on rejection", got)
			}
		})
	}
}

// Ori para em +6 mesmo quando o sorteio ganha.
func TestRefineOriCapsAtSix(t *testing.T) {
	f := newRefineFixture(t, alwaysRate(100), nil)
	f.e.Carry[1] = world.Item{Index: itemArmor}
	for i := 0; i < 10; i++ {
		f.refine(itemPoeiraOri)
	}
	// 6/9 são os tetos legados, agora os padrões de content.RefinePolicy.
	if got := refine.Level(f.target()); got != 6 {
		t.Errorf("level = %d, want 6 (Ori's cap)", got)
	}
}

// Lac para em +9.
func TestRefineLacCapsAtNine(t *testing.T) {
	f := newRefineFixture(t, alwaysRate(100), nil)
	f.e.Carry[1] = world.Item{Index: itemArmor}
	for i := 0; i < 20; i++ {
		f.refine(itemPoeiraLac)
	}
	if got := refine.Level(f.target()); got != 9 {
		t.Errorf("level = %d, want 9 (Lac's cap)", got)
	}
}

// Lactolerium_100 força a linha Âmago, que é 100% fixo independentemente da
// tabela — aqui a tabela falharia sempre de outro modo.
func TestRefineLacto100AlwaysSucceeds(t *testing.T) {
	f := newRefineFixture(t, alwaysRate(0), nil)
	f.e.Carry[1] = world.Item{Index: itemArmor}

	f.refine(itemLacto100)

	if got := refine.Level(f.target()); got != 1 {
		t.Errorf("level = %d, want 1 (Lactolerium_100 is a guaranteed refine)", got)
	}
}

// Uma tintura nunca refina: qualquer pó a converte direto no seu
// Feijão Mágico correspondente (issue #130), ignorando EF_SANC por completo.
func TestRefineTinturaBecomesFeijaoMagico(t *testing.T) {
	cases := []struct {
		name    string
		dust    int16
		tintura int16
	}{
		{"low bound, Ori", itemPoeiraOri, tinturaLo},
		{"middle color, Lac", itemPoeiraLac, tinturaLo + 3},
		{"high bound, Lac", itemPoeiraLac, tinturaHi},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			// alwaysRate(0): todo sorteio de refino comum falharia — prova que a
			// transformação é determinística e nunca toca o sorteio de taxa de sucesso.
			f := newRefineFixture(t, alwaysRate(0), nil)
			f.e.Carry[1] = world.Item{Index: tt.tintura}

			f.refine(tt.dust)

			wantFeijao := tt.tintura + (magicBeanBase - tinturaLo)
			if got := f.target().Index; got != wantFeijao {
				t.Errorf("index = %d, want %d (matching Feijão Mágico)", got, wantFeijao)
			}
			if !f.e.Carry[0].Empty() {
				t.Errorf("dust not consumed: %+v", f.e.Carry[0])
			}
			if got := refine.Level(f.target()); got != 0 {
				t.Errorf("level = %d, want 0 (a tintura must never gain EF_SANC)", got)
			}
		})
	}
}

// TestRefineRNGGolden fixa a ordem das chamadas de rand() contra o stream MSVC da
// semente padrão 1 do CRT, que é onde todo mundo começa:
//
// Rand() #1..#5 = 41, 18467, 6334, 26500, 19169   → %100 = 41, 67, 34, 0, 69
//
// O caminho do pó gasta exatamente UM rand() quando acerta e DOIS quando falha
// (o sorteio, depois o sorteio de pena de 3 em 4). Qualquer sorteio extra ou reordenado desloca todo
// resultado posterior, então este teste é o que de fato trava a paridade.
func TestRefineRNGGolden(t *testing.T) {
	// Taxa 50: sorteios 41 e 34 ganham; 67 e 69 perdem.
	f := newRefineFixture(t, alwaysRate(50), nil)
	f.e.Carry[1] = world.Item{Index: itemArmor}

	steps := []struct {
		roll      int
		wantLevel int
		wantPity  int
		note      string
	}{
		// #1 sorteio 41 <= 50 → sucesso. Um sorteio gasto.
		{41, 1, 0, "roll 41 wins"},
		// #2 sorteio 67 > 50 → falha. #3 = 6334%4 = 2 <= 2 → pena++.
		{67, 1, 1, "roll 67 loses, pity roll 2 hits the <=2 window"},
		// #4 sorteio 26500%100 = 0 <= 50 → sucesso. A pena reseta.
		{0, 2, 0, "roll 0 wins and clears the pity"},
		// #5 sorteio 19169%100 = 69 > 50 → falha. #6 = 15724%4 = 0 <= 2 → pena++.
		{69, 2, 1, "roll 69 loses, pity roll 0 hits the window"},
	}
	for i, st := range steps {
		f.refine(itemPoeiraLac)
		if got := refine.Level(f.target()); got != st.wantLevel {
			t.Errorf("step %d (%s): level = %d, want %d", i+1, st.note, got, st.wantLevel)
		}
		if got := refine.Pity(f.target()); got != st.wantPity {
			t.Errorf("step %d (%s): pity = %d, want %d", i+1, st.note, got, st.wantPity)
		}
	}
}

// TestRefineOverTheWire conduz o caminho real do cliente ponta a ponta: um frame
// _MSG_UseItem carregando um pó em SourPos e o alvo em DestPos precisa chegar ao
// refineItem pelo switch de EF_VOLATILE do useItem, e o item refinado precisa voltar
// num _MSG_SendItem.
//
// Esta é a regressão real da issue #103. Os testes unitários acima chamam refineItem
// direto e todos passariam ainda com o braço do switch removido — que é
// exatamente o bug que foi entregue: o refino caía no `default:` e não fazia nada.
// Ele também confere o pacote, não só o estado do servidor: o nível certo em memória
// ainda renderiza errado se o frame S→C carregar os bytes errados.
func TestRefineOverTheWire(t *testing.T) {
	db := newDB()
	st := world.CharacterState{Slot: 0, Name: "Hero", X: 5, Y: 5, HP: 1000, MaxHP: 1000}
	st.Carry[0] = world.Item{Index: itemPoeiraLac}
	st.Carry[1] = world.Item{Index: itemArmor}
	db.loadResult = st

	addr, stop := startRefineServer(t, db)
	defer stop()
	c := enterWorld(t, addr)

	body := protocol.MsgUseItemBody{
		SourType: world.ItemPlaceCarry, SourPos: 0,
		DestType: world.ItemPlaceCarry, DestPos: 1,
	}
	send(t, c, protocol.MsgUseItem, body.Encode())

	// Ler até o push do item (os frames de score/aviso vêm primeiro).
	for i := 0; ; i++ {
		if i > 8 {
			t.Fatal("no _MSG_SendItem after a refine — the dust fell through useItem's switch")
		}
		ty, payload, ok := readMaybe(t, c)
		if !ok {
			t.Fatal("connection closed before the refined item arrived")
		}
		if ty != protocol.MsgSendItem {
			continue
		}
		place := int(binary.LittleEndian.Uint16(payload[0:]))
		slot := int(binary.LittleEndian.Uint16(payload[2:]))
		index := binary.LittleEndian.Uint16(payload[4:])
		if place != world.ItemPlaceCarry || slot != 1 {
			continue // não o slot alvo
		}
		if index != itemArmor {
			t.Fatalf("SendItem index = %d, want %d", index, itemArmor)
		}
		// Os pares de efeito vão em body+6: o par EF_SANC precisa carregar o novo nível.
		eff0 := world.Effect{Effect: payload[6], Value: payload[7]}
		if eff0 != (world.Effect{Effect: efSanc, Value: 1}) {
			t.Fatalf("SendItem Effects[0] = %+v, want {EF_SANC 1} — the client renders THIS, not the server's copy", eff0)
		}
		return
	}
}

func TestRefineFailureMotionMatchesLegacyFaceRule(t *testing.T) {
	if got := refineFailureMotion(&world.Entity{}); got != refineMotionFailureBase {
		t.Errorf("base-character failure motion = %d, want %d", got, refineMotionFailureBase)
	}
	withFace := &world.Entity{}
	withFace.Equip[0] = world.Item{Index: 10}
	if got := refineFailureMotion(withFace); got != refineMotionFailureFace {
		t.Errorf("equipped-face failure motion = %d, want %d", got, refineMotionFailureFace)
	}
}

// TestRefineFeedbackOverTheWire protege a regressão da issue #317: o cliente 7662
// precisa de um corpo de MessagePanel e um frame de Motion de verdade, não do
// código de aviso MessageBoxOk provisório que o handler emitia antes.
func TestRefineFeedbackOverTheWire(t *testing.T) {
	db := newDB()
	st := world.CharacterState{Slot: 0, Name: "Hero", X: 5, Y: 5, HP: 1000, MaxHP: 1000}
	st.Carry[0] = world.Item{Index: itemPoeiraLac}
	st.Carry[1] = world.Item{Index: itemArmor}
	db.loadResult = st

	addr, stop := startRefineServer(t, db)
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	body := protocol.MsgUseItemBody{
		SourType: world.ItemPlaceCarry, SourPos: 0,
		DestType: world.ItemPlaceCarry, DestPos: 1,
	}
	send(t, c, protocol.MsgUseItem, body.Encode())

	var sawPanel, sawMotion bool
	for i := 0; i < 12 && (!sawPanel || !sawMotion); i++ {
		ty, payload, ok := readMaybe(t, c)
		if !ok {
			t.Fatal("connection closed before refine feedback arrived")
		}
		switch ty {
		case protocol.MsgMessagePanel:
			if got := cstr(payload); got != refineMessageSuccess {
				t.Fatalf("panel text = %q, want %q", got, refineMessageSuccess)
			}
			sawPanel = true
		case protocol.MsgMessageBoxOk:
			t.Fatal("refine feedback used the placeholder MessageBoxOk packet")
		case protocol.MsgMotion:
			motion := binary.LittleEndian.Uint16(payload[0:])
			parm := binary.LittleEndian.Uint16(payload[2:])
			if motion != refineMotionSuccess || parm != refineMotionSuccessParm {
				t.Fatalf("motion = %d/%d, want %d/%d", motion, parm, refineMotionSuccess, refineMotionSuccessParm)
			}
			sawMotion = true
		}
	}
	if !sawPanel || !sawMotion {
		t.Fatalf("incomplete refine feedback: panel=%v motion=%v", sawPanel, sawMotion)
	}
}

// startRefineServer é startServerClock com um catálogo que classifica os pós.
func startRefineServer(t *testing.T, persist world.Persistence) (string, func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.DiscardHandler)
	d := New(Config{
		Log:           log,
		SancRate:      alwaysRate(100),
		ItemVolatiles: map[int]int{itemPoeiraOri: volDustOri, itemPoeiraLac: volDustLac},
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

// Um ovo choca na sua montaria no primeiro refino bem-sucedido: o gate de choca
// lê o EF_INCUBATE da INSTANCE, que um ovo novo não carrega.
func TestRefineEggHatches(t *testing.T) {
	const egg = 2304
	f := newRefineFixture(t, alwaysRate(100), map[int][]content.BaseEffect{
		egg: {{Eff: efIncubate, Val: 2}}, // o limiar do catálogo deliberadamente não é lido
	})
	f.e.Carry[1] = world.Item{Index: egg}

	f.refine(itemPoeiraLac)

	if got := f.target().Index; got != egg+eggHatchAdd {
		t.Fatalf("index = %d, want %d (hatched mount)", got, egg+eggHatchAdd)
	}
	// A montaria chocada reinterpreta seus slots como dados de montaria: [0].sValue = HP.
	if got := uint16(f.target().Effects[0].Effect) | uint16(f.target().Effects[0].Value)<<8; got != hatchMountHP {
		t.Errorf("mount HP = %d, want %d", got, hatchMountHP)
	}
	if got := f.target().Effects[1].Effect; got != hatchMountSanc {
		t.Errorf("mount sanc = %d, want %d", got, hatchMountSanc)
	}
	if got := f.target().Effects[2]; got != (world.Effect{Effect: hatchMountFeed, Value: hatchMountKill}) {
		t.Errorf("mount feed/kill = %+v", got)
	}
}

// Um refino de ovo que falha carimba o cooldown de incubação, e o tick de regen
// o conta para baixo só enquanto o ovo está equipado.
func TestEggIncubationCooldown(t *testing.T) {
	const egg = 2304
	f := newRefineFixture(t, alwaysRate(0), nil)
	f.e.Carry[1] = world.Item{Index: egg}

	f.refine(itemPoeiraLac)

	// rand()#3 % 4 = 2 → um cooldown de 2.
	delay := itemInstanceAbility(f.target(), efIncuDelay)
	if delay != 2 {
		t.Fatalf("incubation delay = %d, want 2", delay)
	}

	// Enquanto o cooldown vale, outro refino é recusado.
	f.refine(itemPoeiraLac)
	if got := itemInstanceAbility(f.target(), efIncuDelay); got != delay {
		t.Errorf("a gated refine changed the cooldown: %d → %d", delay, got)
	}
	if f.e.Carry[0].Empty() {
		t.Error("a gated refine consumed the dust")
	}

	// O tick só conta um ovo EQUIPADO.
	f.d.tickIncubation(f.w, f.s, f.e)
	if got := itemInstanceAbility(f.target(), efIncuDelay); got != delay {
		t.Errorf("the cooldown ticked while the egg was in the inventory: %d", got)
	}
	f.e.Equip[mountEquipSlot] = f.target()
	f.d.tickIncubation(f.w, f.s, f.e)
	if got := itemInstanceAbility(f.e.Equip[mountEquipSlot], efIncuDelay); got != delay-1 {
		t.Errorf("equipped cooldown = %d, want %d", got, delay-1)
	}
}
