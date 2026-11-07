package handler

import (
	"encoding/binary"
	"net"
	"strings"
	"testing"

	"openwyd-nextgen/tmserver/internal/protocol"
	"openwyd-nextgen/tmserver/internal/world"
)

// celestialDB carrega um único personagem Celestial (o tier que os comandos de desbloqueio
// exigem). O nível fica acima dos gates, então os comandos são a única variável.
func celestialDB(classMaster uint8) *fakeDB {
	db := newDB()
	db.loadResult = world.CharacterState{
		Slot: 0, Name: "Celeste", Class: 0, ClassMaster: classMaster,
		X: 5, Y: 5, HP: 1000, MaxHP: 1000, Level: 41,
	}
	return db
}

func chatDB() *fakeDB {
	db := newDB()
	db.loads = map[int64]world.CharacterState{
		7:  {Slot: 0, Name: "Hero", X: 5, Y: 5, HP: 1000, MaxHP: 1000},
		11: {Slot: 0, Name: "HeroB", X: 5, Y: 5, HP: 1000, MaxHP: 1000},
	}
	return db
}

func chatFrame(t *testing.T, c net.Conn, text string) {
	t.Helper()
	send(t, c, protocol.MsgMessageChat, []byte(text))
}

func whisperFrame(t *testing.T, c net.Conn, target, text string) {
	t.Helper()
	var body protocol.MsgWhisperBody
	copy(body.MobName[:], target)
	body.String = []byte(text)
	send(t, c, protocol.MsgMessageWhisper, body.Encode())
}

// TestCommandTeleport verifica que um comando /city (entregue como sussurro ao
// nome da cidade) teletransporta o jogador (MSG_Action jump) em vez de virar sussurro.
func TestCommandTeleport(t *testing.T) {
	addr, stop, _ := startServerClock(t, chatDB())
	defer stop()
	a := enterWorldAs(t, addr, "tester")
	defer a.Close()

	whisperFrame(t, a, "armia", "")
	if ty, _, ok := readMaybe(t, a); !ok || ty != protocol.MsgAction {
		t.Errorf("got %#x ok=%v, want MsgAction (teleport)", ty, ok)
	}
}

// TestCommandNoatunTeleport verifica que /noatun teletransporta para Noatun com o mesmo
// espalhamento rand%3 usado pelos outros comandos de teleporte grátis.
func TestCommandNoatunTeleport(t *testing.T) {
	addr, stop, _ := startServerClock(t, chatDB())
	defer stop()
	a := enterWorldAs(t, addr, "tester")
	defer a.Close()

	whisperFrame(t, a, "noatun", "")
	ty, payload, ok := readMaybe(t, a)
	if !ok || ty != protocol.MsgAction {
		t.Fatalf("got %#x ok=%v, want MsgAction (teleport)", ty, ok)
	}
	var body protocol.MsgActionBody
	if err := body.Decode(payload); err != nil {
		t.Fatalf("decode MsgAction: %v", err)
	}
	if body.TargetX < 1052 || body.TargetX > 1054 || body.TargetY < 1726 || body.TargetY > 1728 {
		t.Errorf("/noatun target = %d,%d, want within 1052..1054,1726..1728", body.TargetX, body.TargetY)
	}
}

// TestCommandKingdomAliases verifica que os comandos de comércio em português e inglês
// roteiam pela capa equipada para os destinos legados fixos da issue #318.
func TestCommandKingdomAliases(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		cape int16
		want [2]int16
	}{
		{"reino no cape", "reino", 0, kingdomDestNeutral},
		{"kingdom white cape", "kingdom", capaBrancaDoMonstroIndex, kingdomDestNeutral},
		{"reino green cape", "reino", greenCapeItem, kingdomDestNeutral},
		{"kingdom other neutral cape", "kingdom", 100, kingdomDestNeutral},
		{"reino hekalotia base", "reino", 545, kingdomDestHekalotia},
		{"kingdom hekalotia elite", "kingdom", 3191, kingdomDestHekalotia},
		{"reino hekalotia advanced", "reino", 3197, kingdomDestHekalotia},
		{"kingdom akelonia base", "kingdom", 546, kingdomDestAkelonia},
		{"reino akelonia elite", "reino", 3192, kingdomDestAkelonia},
		{"kingdom akelonia advanced", "kingdom", 3198, kingdomDestAkelonia},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db := newDB()
			db.loads = map[int64]world.CharacterState{
				7: {Slot: 0, Name: "Hero", X: 5, Y: 5, HP: 1000, MaxHP: 1000,
					Equip: [world.MaxEquip]world.Item{reinoCapeSlot: {Index: tc.cape}}},
			}
			addr, stop, _ := startServerClock(t, db)
			defer stop()
			a := enterWorldAs(t, addr, "tester")
			defer a.Close()

			whisperFrame(t, a, tc.cmd, "")
			ty, payload, ok := readMaybe(t, a)
			if !ok || ty != protocol.MsgAction {
				t.Fatalf("got %#x ok=%v, want MsgAction (teleport)", ty, ok)
			}
			var body protocol.MsgActionBody
			if err := body.Decode(payload); err != nil {
				t.Fatalf("decode MsgAction: %v", err)
			}
			if body.TargetX != tc.want[0] || body.TargetY != tc.want[1] {
				t.Errorf("/%s target = %d,%d, want %d,%d",
					tc.cmd, body.TargetX, body.TargetY, tc.want[0], tc.want[1])
			}
		})
	}
}

func TestCommandKingAliases(t *testing.T) {
	tests := []struct {
		name     string
		cmd      string
		cape     int16
		want     [2]int16
		teleport bool
	}{
		{"rei hekalotia base", "rei", 545, kingDestHekalotia, true},
		{"king hekalotia advanced", "king", 3197, kingDestHekalotia, true},
		{"rei akelonia base", "rei", 546, kingDestAkelonia, true},
		{"king akelonia advanced", "king", 3198, kingDestAkelonia, true},
		{"rei no cape", "rei", 0, [2]int16{}, false},
		{"king white cape", "king", capaBrancaDoMonstroIndex, [2]int16{}, false},
		{"rei green cape", "rei", greenCapeItem, [2]int16{}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db := newDB()
			db.loads = map[int64]world.CharacterState{
				7: {Slot: 0, Name: "Hero", X: 5, Y: 5, HP: 1000, MaxHP: 1000,
					Equip: [world.MaxEquip]world.Item{reinoCapeSlot: {Index: tc.cape}}},
			}
			addr, stop, _ := startServerClock(t, db)
			defer stop()
			a := enterWorldAs(t, addr, "tester")
			defer a.Close()

			whisperFrame(t, a, tc.cmd, "")
			ty, payload, ok := readMaybe(t, a)
			if !tc.teleport {
				if ok {
					t.Fatalf("neutral /%s produced %#x; want handled no-op", tc.cmd, ty)
				}
				return
			}
			if !ok || ty != protocol.MsgAction {
				t.Fatalf("got %#x ok=%v, want MsgAction (teleport)", ty, ok)
			}
			var body protocol.MsgActionBody
			if err := body.Decode(payload); err != nil {
				t.Fatalf("decode MsgAction: %v", err)
			}
			if body.TargetX != tc.want[0] || body.TargetY != tc.want[1] {
				t.Errorf("/%s target = %d,%d, want %d,%d",
					tc.cmd, body.TargetX, body.TargetY, tc.want[0], tc.want[1])
			}
		})
	}
}

func TestCommandColorTeleportsRetired(t *testing.T) {
	for _, cmd := range []string{"red", "blue"} {
		t.Run(cmd, func(t *testing.T) {
			addr, stop, _ := startServerClock(t, chatDB())
			defer stop()
			a := enterWorldAs(t, addr, "tester")
			defer a.Close()

			whisperFrame(t, a, cmd, "")
			// O aviso agora sai como texto (MSG_MessagePanel, ID 0): a caixa
			// numérica antiga abria vazia no cliente 7662.
			if got := readPanel(t, a); got != partyTargetGone {
				t.Fatalf("/%s produced %q; want the not-connected panel", cmd, got)
			}
		})
	}
}

// TestCommandBuffs verifica que /buffs limpa os affects e empurra um score novo.
func TestCommandBuffs(t *testing.T) {
	addr, stop, _ := startServerClock(t, chatDB())
	defer stop()
	a := enterWorldAs(t, addr, "tester")
	defer a.Close()

	whisperFrame(t, a, "buffs", "")
	if ty, _, ok := readMaybe(t, a); !ok || ty != protocol.MsgUpdateScore {
		t.Errorf("got %#x ok=%v, want MsgUpdateScore after /buffs", ty, ok)
	}
}

// TestCommandSair verifica que /sair é tratado como comando (não entregue como sussurro
// a um jogador inexistente): um jogador de guilda saindo sem ninguém em view não produz pacote.
func TestCommandSair(t *testing.T) {
	db := newDB()
	db.loads = map[int64]world.CharacterState{
		7: {Slot: 0, Name: "Hero", X: 5, Y: 5, HP: 1000, MaxHP: 1000, GuildID: 5},
	}
	addr, stop, _ := startServerClock(t, db)
	defer stop()
	a := enterWorldAs(t, addr, "tester")
	defer a.Close()

	whisperFrame(t, a, "sair", "")
	// Tratado como comando → nada de NoticeNotConnected (que um sussurro real para "sair"
	// dispararia). Sem jogadores em view, a transmissão da tag de guilda não alcança ninguém.
	if ty, _, ok := readMaybe(t, a); ok {
		t.Errorf("/sair produced %#x; want a silent handled command", ty)
	}
}

func TestGuildCreateCommand(t *testing.T) {
	db := newDB()
	db.loads = map[int64]world.CharacterState{
		7: {Slot: 0, Name: "Hero", X: 5, Y: 5, HP: 1000, MaxHP: 1000, Clan: 7, Citizen: 1, Coin: 100_000_000},
	}
	addr, stop, _ := startServerClock(t, db)
	defer stop()
	a := enterWorldAs(t, addr, "tester")
	defer a.Close()
	drainRaw(t, a)

	whisperFrame(t, a, "create", "Knights")
	if ty, _, ok := readMaybe(t, a); !ok || ty != protocol.MsgUpdateEtc {
		t.Fatalf("/create got %#x ok=%v, want UpdateEtc", ty, ok)
	}
	if ty, _, ok := readMaybeRaw(t, a); !ok || ty != protocol.MsgCreateMob {
		t.Fatalf("/create got %#x ok=%v, want CreateMob tag refresh", ty, ok)
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	if len(db.createdGuilds) != 1 || db.createdGuilds[0].Name != "Knights" {
		t.Fatalf("created guilds = %+v, want Knights", db.createdGuilds)
	}
}

// TestGuildCreateRegistersName verifica que /create registra o nome da guilda no
// mundo na hora, então /nick o mostra sem esperar por um restart (#417).
func TestGuildCreateRegistersName(t *testing.T) {
	db := newDB()
	db.loads = map[int64]world.CharacterState{
		7: {Slot: 0, Name: "Hero", X: 5, Y: 5, HP: 1000, MaxHP: 1000, Clan: 7, Citizen: 1, Coin: 100_000_000},
	}
	addr, stop, _ := startServerClock(t, db)
	defer stop()
	a := enterWorldAs(t, addr, "tester")
	defer a.Close()
	drainRaw(t, a)

	whisperFrame(t, a, "create", "Knights")
	expect(t, a, protocol.MsgUpdateEtc)
	drainRaw(t, a)

	whisperFrame(t, a, "nick", "Hero")
	p := expect(t, a, protocol.MsgMessageChat)
	want := "Hero [Knights (Fama guilda: 0)] Cidadania: 1 / Fama: 0"
	if got := cstr(p); got != want {
		t.Errorf("/nick text = %q, want %q", got, want)
	}
}

func TestGuildSubcreateCommand(t *testing.T) {
	db := newDB()
	db.loads = map[int64]world.CharacterState{
		7:  {Slot: 0, Name: "Hero", X: 5, Y: 5, HP: 1000, MaxHP: 1000, Clan: 1, GuildID: 5, GuildLevel: 9, Coin: 100_000_000},
		11: {Slot: 0, Name: "HeroB", X: 5, Y: 5, HP: 1000, MaxHP: 1000, Clan: 1, GuildID: 5},
	}
	addr, stop, _ := startServerClock(t, db)
	defer stop()
	a := enterWorldAs(t, addr, "tester")
	defer a.Close()
	b := enterWorldAs(t, addr, "tradeb")
	defer b.Close()
	drainRaw(t, a)
	drainRaw(t, b)

	whisperFrame(t, a, "subcreate", "HeroB Sub")
	if ty, _, ok := readMaybe(t, a); !ok || ty != protocol.MsgUpdateEtc {
		t.Fatalf("/subcreate got %#x ok=%v, want UpdateEtc", ty, ok)
	}
	if ty, _, ok := readMaybeRaw(t, a); !ok || ty != protocol.MsgCreateMob {
		t.Fatalf("/subcreate leader got %#x ok=%v, want CreateMob tag refresh", ty, ok)
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	if len(db.promoted) != 1 || db.promoted[0] != 6 {
		t.Fatalf("promoted = %v, want [6]", db.promoted)
	}
}

func TestGuildHandoverCommand(t *testing.T) {
	db := newDB()
	db.loads = map[int64]world.CharacterState{
		7:  {Slot: 0, Name: "Hero", X: 5, Y: 5, HP: 1000, MaxHP: 1000, Clan: 1, GuildID: 5, GuildLevel: 9},
		11: {Slot: 0, Name: "HeroB", X: 5, Y: 5, HP: 1000, MaxHP: 1000, Clan: 1, GuildID: 5},
	}
	addr, stop, _ := startServerClock(t, db)
	defer stop()
	a := enterWorldAs(t, addr, "tester")
	defer a.Close()
	b := enterWorldAs(t, addr, "tradeb")
	defer b.Close()
	drainRaw(t, a)
	drainRaw(t, b)

	whisperFrame(t, a, "handover", "HeroB")
	if ty, _, ok := readMaybeRaw(t, a); !ok || ty != protocol.MsgCreateMob {
		t.Fatalf("/handover leader got %#x ok=%v, want CreateMob tag refresh", ty, ok)
	}
	if ty, _, ok := readMaybeRaw(t, b); !ok || ty != protocol.MsgCreateMob {
		t.Fatalf("/handover target got %#x ok=%v, want CreateMob tag refresh", ty, ok)
	}
}

// TestCommandChaosPoints verifica que /cp reporta os pontos de caos de quem chamou como
// uma linha de MessageChat. Um jogador recém-logado nunca é culpado, então o valor é 0.
func TestCommandChaosPoints(t *testing.T) {
	addr, stop, _ := startServerClock(t, chatDB())
	defer stop()
	a := enterWorldAs(t, addr, "tester")
	defer a.Close()

	whisperFrame(t, a, "cp", "")
	ty, payload, ok := readMaybe(t, a)
	if !ok || ty != protocol.MsgMessageChat {
		t.Fatalf("got %#x ok=%v, want MessageChat", ty, ok)
	}
	want := "Pontos Caos atual: 0"
	if string(payload[:len(want)]) != want {
		t.Errorf("chat text = %q, want %q", payload, want)
	}
}

// trumpetDB é chatDB com "Hero" (conta 7) carregando uma pilha de Trombetas
// Mágicas no slot 0. amount == 0 significa nenhuma trombeta.
func trumpetDB(amount uint8) *fakeDB {
	db := chatDB()
	hero := db.loads[7]
	if amount > 0 {
		hero.Carry[0] = amountItem(itemMagicTrumpet, amount)
	}
	db.loads[7] = hero
	return db
}

// TestCommandGritar verifica que /gritar queima uma Trombeta Mágica e grita para
// todo jogador online, o gritador incluído (_MSG_MessageWhisper.cpp:114).
func TestCommandGritar(t *testing.T) {
	addr, stop, _ := startServerClock(t, trumpetDB(3))
	defer stop()
	a := enterWorldAs(t, addr, "tester")
	defer a.Close()
	b := enterWorldAs(t, addr, "tradeb")
	defer b.Close()

	whisperFrame(t, a, "gritar", "ola mundo")

	// O slot do próprio gritador é ressincronizado primeiro: 3 → 2.
	_, slot, index, amount := sendItemSlotAmount(expect(t, a, protocol.MsgSendItem))
	if slot != 0 || index != itemMagicTrumpet || amount != 2 {
		t.Errorf("slot sync = (slot %d, index %d, amount %d), want (0, %d, 2)", slot, index, amount, itemMagicTrumpet)
	}

	want := "[Hero]> ola mundo"
	for _, tc := range []struct {
		name string
		c    net.Conn
	}{{"shouter", a}, {"other", b}} {
		payload := expect(t, tc.c, protocol.MsgMagicTrumpet)
		if len(payload) != protocol.MagicTrumpetBodySize {
			t.Errorf("%s: body = %d bytes, want %d", tc.name, len(payload), protocol.MagicTrumpetBodySize)
			continue
		}
		if got := cstr(payload); got != want {
			t.Errorf("%s: shout = %q, want %q", tc.name, got, want)
		}
		if payload[94] != 1 {
			t.Errorf("%s: String[94] = %d, want 1 (legacy marker byte)", tc.name, payload[94])
		}
	}
}

// TestCommandGritarLastTrumpet verifica que o slot é limpo, não decrementado,
// quando a última unidade é gasta (BASE_ClearItem, _MSG_MessageWhisper.cpp:135).
func TestCommandGritarLastTrumpet(t *testing.T) {
	addr, stop, _ := startServerClock(t, trumpetDB(1))
	defer stop()
	a := enterWorldAs(t, addr, "tester")
	defer a.Close()

	whisperFrame(t, a, "gritar", "adeus")
	_, slot, index, _ := sendItemSlotAmount(expect(t, a, protocol.MsgSendItem))
	if slot != 0 || index != 0 {
		t.Errorf("slot sync = (slot %d, index %d), want (0, 0) — an empty slot", slot, index)
	}
}

// TestCommandGritarNoTrumpet verifica que o comando recusa sem o item: o
// quem chamou é informado do motivo (um desvio deliberado — o legado retornava em silêncio em
// _MSG_MessageWhisper.cpp:139) e ninguém mais ouve nada.
func TestCommandGritarNoTrumpet(t *testing.T) {
	addr, stop, _ := startServerClock(t, trumpetDB(0))
	defer stop()
	a := enterWorldAs(t, addr, "tester")
	defer a.Close()
	b := enterWorldAs(t, addr, "tradeb")
	defer b.Close()

	whisperFrame(t, a, "gritar", "ola mundo")
	ty, payload, ok := readMaybe(t, a)
	if !ok || ty != protocol.MsgMessageChat {
		t.Fatalf("got %#x ok=%v, want MessageChat", ty, ok)
	}
	if got := cstr(payload); got != "Voc\xea precisa de uma Trombeta M\xe1gica para gritar." {
		t.Errorf("refusal = %q", got)
	}
	if ty, _, ok := readMaybe(t, b); ok && ty == protocol.MsgMagicTrumpet {
		t.Error("the shout reached other players without a trumpet being spent")
	}
}

// TestCommandGritarEmpty verifica que um /gritar sozinho é no-op: sem grito e, acima
// de tudo, sem queimar trombeta (o legado gritava "[Nome]> " e cobrava por isso).
func TestCommandGritarEmpty(t *testing.T) {
	addr, stop, _ := startServerClock(t, trumpetDB(3))
	defer stop()
	a := enterWorldAs(t, addr, "tester")
	defer a.Close()

	whisperFrame(t, a, "gritar", "   ")
	if ty, _, ok := readMaybe(t, a); ok {
		t.Errorf("empty shout produced %#x, want silence", ty)
	}
}

// TestCommandSpkAlias verifica que /spk, o alias legado, roteia para o mesmo comando.
func TestCommandSpkAlias(t *testing.T) {
	addr, stop, _ := startServerClock(t, trumpetDB(2))
	defer stop()
	a := enterWorldAs(t, addr, "tester")
	defer a.Close()

	whisperFrame(t, a, "spk", "oi")
	if got := cstr(expect(t, a, protocol.MsgMagicTrumpet)); got != "[Hero]> oi" {
		t.Errorf("shout = %q, want %q", got, "[Hero]> oi")
	}
}

func TestChatPublicBroadcast(t *testing.T) {
	addr, stop, _ := startServerClock(t, chatDB())
	defer stop()
	a := enterWorldAs(t, addr, "tester")
	defer a.Close()
	b := enterWorldAs(t, addr, "tradeb")
	defer b.Close()

	chatFrame(t, a, "hello world")
	ty, p, ok := readMaybe(t, b)
	if !ok || ty != protocol.MsgMessageChat {
		t.Fatalf("got %#x ok=%v, want MessageChat broadcast", ty, ok)
	}
	if string(p[:len("hello world")]) != "hello world" {
		t.Errorf("chat text = %q", p)
	}
}

func TestWhisperDeliver(t *testing.T) {
	addr, stop, _ := startServerClock(t, chatDB())
	defer stop()
	a := enterWorldAs(t, addr, "tester")
	defer a.Close()
	b := enterWorldAs(t, addr, "tradeb") // name "HeroB"
	defer b.Close()

	whisperFrame(t, a, "HeroB", "psst")
	if ty, _, ok := readMaybe(t, b); !ok || ty != protocol.MsgMessageWhisper {
		t.Errorf("got %#x ok=%v, want MessageWhisper delivered", ty, ok)
	}
}

// whisperFrameSized envia um sussurro no layout da struct 7662: MobName[16],
// String[128], Color (frame 156) e o tamanho total de corpo dado (146 do
// dialeto web, 148 do WYD.exe).
func whisperFrameSized(t *testing.T, c net.Conn, target, text string, color uint16, size int) {
	t.Helper()
	body := make([]byte, size)
	copy(body[:15], target)
	copy(body[16:16+127], text)
	binary.LittleEndian.PutUint16(body[16+whisperColorOffset:], color)
	send(t, c, protocol.MsgMessageWhisper, body)
}

func readWhisper(t *testing.T, c net.Conn) (name, text string, color uint16, size int) {
	t.Helper()
	ty, p, ok := readMaybe(t, c)
	if !ok || ty != protocol.MsgMessageWhisper {
		t.Fatalf("got %#x ok=%v, want MessageWhisper", ty, ok)
	}
	var body protocol.MsgWhisperBody
	if err := body.Decode(p); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.String) >= whisperColorOffset+2 {
		color = binary.LittleEndian.Uint16(body.String[whisperColorOffset:])
	}
	return cstr(body.MobName[:]), cstr(body.String), color, len(p)
}

// TestWhisperRewrite: o cliente do receptor mostra MobName como remetente e
// imprime &String[1] no recado privado, então o servidor escreve o nome de quem
// fala e prefixa um espaço (_MSG_MessageWhisper.cpp copia o nome; o
// espaço é nosso, ver privateWhisperText). Marcadores e Color não podem ser forjados.
func TestWhisperRewrite(t *testing.T) {
	long := strings.Repeat("x", 140)
	cases := []struct {
		name, text string
		color      uint16
		size       int
		wantText   string
	}{
		{"windows layout", "psst", 0, 148, " psst"},
		{"web dialect layout", "psst", 0, 146, " psst"},
		{"short frame", "psst", 0, 0, " psst"},
		{"guild marker", "-fake guild", 3, 148, " -fake guild"},
		{"party marker", "=fake party", 0, 148, " =fake party"},
		{"kingdom marker", "@@fake", 0, 148, " @@fake"},
		{"grey color", "grey", 7, 148, " grey"},
		{"cut to fit", long, 0, 148, " " + strings.Repeat("x", 126)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			addr, stop, _ := startServerClock(t, chatDB())
			defer stop()
			a := enterWorldAs(t, addr, "tester") // name "Hero"
			defer a.Close()
			b := enterWorldAs(t, addr, "tradeb") // name "HeroB"
			defer b.Close()

			if tc.size == 0 {
				whisperFrame(t, a, "HeroB", tc.text)
			} else {
				whisperFrameSized(t, a, "HeroB", tc.text, tc.color, tc.size)
			}
			name, text, color, size := readWhisper(t, b)
			if name != "Hero" {
				t.Errorf("sender = %q, want Hero", name)
			}
			if text != tc.wantText {
				t.Errorf("text = %q, want %q", text, tc.wantText)
			}
			if color != 0 {
				t.Errorf("color = %d, want 0", color)
			}
			if tc.size != 0 && size != tc.size {
				t.Errorf("size = %d, want the received %d", size, tc.size)
			}
		})
	}
}

// TestWhisperReply: "/r" (um sussurro para "r") vai para quem sussurrou por último;
// sem ninguém ainda, é o aviso de offline.
func TestWhisperReply(t *testing.T) {
	addr, stop, _ := startServerClock(t, chatDB())
	defer stop()
	a := enterWorldAs(t, addr, "tester") // name "Hero"
	defer a.Close()
	b := enterWorldAs(t, addr, "tradeb") // name "HeroB"
	defer b.Close()

	whisperFrame(t, b, "r", "nobody yet")
	if got := readPanel(t, b); got != partyTargetGone {
		t.Fatalf("got %q, want the not-connected panel for /r without history", got)
	}

	whisperFrame(t, a, "HeroB", "hi")
	readWhisper(t, b)
	whisperFrame(t, b, "r", "hello back")
	name, text, _, _ := readWhisper(t, a)
	if name != "HeroB" || text != " hello back" {
		t.Errorf("reply = %q/%q, want HeroB/\" hello back\"", name, text)
	}
}

func TestDayCommand(t *testing.T) {
	addr, stop, _ := startServerClock(t, chatDB())
	defer stop()
	a := enterWorldAs(t, addr, "tester")
	defer a.Close()

	// O 7662 envia "day" com String vazia (TMHuman.cpp:4759), e só
	// lê um painel de mensagem com HEADER.ID 0 (TMScene.cpp:1371).
	whisperFrame(t, a, "day", "")
	h, p, ok := readMaybeHeader(t, a)
	if !ok || h.Type != protocol.MsgMessagePanel || h.ID != 0 || cstr(p) != "!#11  2" {
		t.Fatalf("got %#x id=%d %q, want the legacy !#11  2 panel with ID 0", h.Type, h.ID, p)
	}
	if ty, _, ok := readMaybe(t, a); ok {
		t.Errorf("extra %#x after the day answer", ty)
	}
}

func TestWhisperOffline(t *testing.T) {
	addr, stop, _ := startServerClock(t, chatDB())
	defer stop()
	a := enterWorldAs(t, addr, "tester")
	defer a.Close()

	whisperFrame(t, a, "Ghost", "anyone?")
	if got := readPanel(t, a); got != partyTargetGone {
		t.Errorf("got %q, want the not-connected panel", got)
	}
}

func TestWhisperBlocked(t *testing.T) {
	addr, stop, _ := startServerClock(t, chatDB())
	defer stop()
	a := enterWorldAs(t, addr, "tester")
	defer a.Close()
	b := enterWorldAs(t, addr, "tradeb")
	defer b.Close()

	// B bloqueia sussurros, então conversa em público para a leitura de A confirmar que o toggle rodou.
	chatFrame(t, b, "whisper")
	chatFrame(t, b, "ping")
	if ty, _, ok := readMaybe(t, a); !ok || ty != protocol.MsgMessageChat {
		t.Fatalf("expected B's public ping, got %#x ok=%v", ty, ok)
	}

	whisperFrame(t, a, "HeroB", "hi")
	if got := readPanel(t, a); got != noticeText(NoticeDenyWhisper) {
		t.Errorf("got %q, want deny-whisper text %q", got, noticeText(NoticeDenyWhisper))
	}
}

// TestCommandDestravar40 verifica que /destravar40 num Celestial define o gate Lv40
// (sinalizado por MsgCombineComplete) e a flag persiste através de um save.
func TestCommandDestravar40(t *testing.T) {
	db := celestialDB(classMasterCelestial)
	addr, stop, _ := startServerClock(t, db)
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	whisperFrame(t, c, "destravar40", "")
	ty, p, ok := readMaybe(t, c)
	if !ok || ty != protocol.MsgCombineComplete {
		t.Fatalf("got %#x ok=%v, want MsgCombineComplete", ty, ok)
	}
	if parm := int16(binary.LittleEndian.Uint16(p)); parm != celestialUnlockParm {
		t.Errorf("CombineComplete parm = %d, want %d", parm, celestialUnlockParm)
	}

	// O logout descarrega o save do personagem; a flag do gate precisa ir junto.
	send(t, c, protocol.MsgCharacterLogout, nil)
	expect(t, c, protocol.MsgCNFCharacterLogout)
	char, n := db.lastSavedChar()
	if n == 0 || char.CelLv40 != 1 {
		t.Errorf("saved CelLv40 = %d (n=%d), want 1", char.CelLv40, n)
	}
}

// TestCommandDestravarNonCelestial verifica que o desbloqueio é no-op para um não-Celestial
// (Mortal): o comando continua consumido (não entregue como sussurro) mas nada
// é enviado e nenhuma flag é definida.
func TestCommandDestravarNonCelestial(t *testing.T) {
	db := celestialDB(classMasterMortal)
	addr, stop, _ := startServerClock(t, db)
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	whisperFrame(t, c, "destravar40", "")
	if ty, _, ok := readMaybe(t, c); ok {
		t.Errorf("/destravar40 on a mortal produced %#x; want a silent no-op", ty)
	}
}

// TestCommandDestravar90 verifica que /destravar90 concede a Cythera Mistica (item 3502) ao
// carry, sinaliza o cliente, e persiste tanto o item quanto o gate Lv90.
func TestCommandDestravar90(t *testing.T) {
	db := celestialDB(classMasterCelestial)
	addr, stop, _ := startServerClock(t, db)
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	whisperFrame(t, c, "destravar90", "")
	expect(t, c, protocol.MsgSendItem)        // Cythera Mistica no carry
	expect(t, c, protocol.MsgCombineComplete) // sinal de desbloqueio
	expect(t, c, protocol.MsgMotion)          // emote de desbloqueio

	send(t, c, protocol.MsgCharacterLogout, nil)
	expect(t, c, protocol.MsgCNFCharacterLogout)
	char, n := db.lastSavedChar()
	if n == 0 || char.CelLv90 != 1 {
		t.Errorf("saved CelLv90 = %d (n=%d), want 1", char.CelLv90, n)
	}
	if !hasItem(char.Carry, cytheraMisticaItem) {
		t.Errorf("saved carry missing Cythera Mistica %d: %+v", cytheraMisticaItem, char.Carry)
	}
}

// TestCommandArcana verifica que /arcana coloca a recompensa (item 3507) em Equip[1], define
// a flag de Circle, e persiste as duas.
func TestCommandArcana(t *testing.T) {
	db := celestialDB(classMasterCelestial)
	addr, stop, _ := startServerClock(t, db)
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	whisperFrame(t, c, "arcana", "")
	expect(t, c, protocol.MsgSendItem)        // recompensa em Equip[1]
	expect(t, c, protocol.MsgCombineComplete) // sinal
	expect(t, c, protocol.MsgMotion)          // emote

	send(t, c, protocol.MsgCharacterLogout, nil)
	expect(t, c, protocol.MsgCNFCharacterLogout)
	char, n := db.lastSavedChar()
	if n == 0 || char.CelCircle != 1 {
		t.Errorf("saved CelCircle = %d (n=%d), want 1", char.CelCircle, n)
	}
	if !hasItem(char.Equip, arcanaItemIndex) {
		t.Errorf("saved equip missing arcana item %d: %+v", arcanaItemIndex, char.Equip)
	}
}

// TestCommandNickGuildless verifica que /nick num alvo sem guilda reporta "Sem
// guilda" mais sua cidadania/fama (zeradas).
func TestCommandNickGuildless(t *testing.T) {
	addr, stop, _ := startServerClock(t, chatDB())
	defer stop()
	a := enterWorldAs(t, addr, "tester")
	defer a.Close()
	b := enterWorldAs(t, addr, "tradeb") // name "HeroB"
	defer b.Close()

	whisperFrame(t, a, "nick", "HeroB")
	p := expect(t, a, protocol.MsgMessageChat)
	want := "HeroB [Sem guilda] Cidadania: 0 / Fama: 0"
	if got := cstr(p); got != want {
		t.Errorf("/nick text = %q, want %q", got, want)
	}
}

// TestCommandNickUnregisteredGuild verifica que um alvo com id de guilda não zero
// que nunca foi registrado via /gm guildname|guildfame mostra um placeholder.
func TestCommandNickUnregisteredGuild(t *testing.T) {
	db := newDB()
	db.loads = map[int64]world.CharacterState{
		7:  {Slot: 0, Name: "Hero", X: 5, Y: 5, HP: 1000, MaxHP: 1000},
		11: {Slot: 0, Name: "HeroB", X: 5, Y: 5, HP: 1000, MaxHP: 1000, GuildID: 9, Citizen: 2, Fame: 150},
	}
	addr, stop, _ := startServerClock(t, db)
	defer stop()
	a := enterWorldAs(t, addr, "tester")
	defer a.Close()
	b := enterWorldAs(t, addr, "tradeb")
	defer b.Close()

	whisperFrame(t, a, "nick", "HeroB")
	p := expect(t, a, protocol.MsgMessageChat)
	want := "HeroB [Guild #9 (nao registrada)] Cidadania: 2 / Fama: 150"
	if got := cstr(p); got != want {
		t.Errorf("/nick text = %q, want %q", got, want)
	}
}

// TestCommandNickRegisteredGuild verifica que /nick mostra o nome/fama da guilda
// registrados (definidos via /gm guildname e /gm guildfame) junto com cidadania
// e fama do personagem.
func TestCommandNickRegisteredGuild(t *testing.T) {
	db := gmDB()
	db.loads[23] = world.CharacterState{ // id de conta 23 = "victim" (ver gmDB)
		Slot: 0, Name: "Victim", Class: 0, Level: 10, X: 5, Y: 5, HP: 1000, MaxHP: 1000,
		GuildID: 9, Citizen: 3, Fame: 777,
	}
	addr, stop, _ := startServerClock(t, db)
	defer stop()
	mod := enterWorldAs(t, addr, "mod")
	defer mod.Close()
	victim := enterWorldAs(t, addr, "victim")
	defer victim.Close()

	gmFrame(t, mod, "guildname 9 Dragoes")
	gmFrame(t, mod, "guildfame 9 4242")
	if ty, _, ok := readMaybe(t, mod); ok {
		t.Errorf("guildname/guildfame produced %#x; want no client-visible packet", ty)
	}

	whisperFrame(t, mod, "nick", "Victim")
	p := expect(t, mod, protocol.MsgMessageChat)
	want := "Victim [Dragoes (Fama guilda: 4242)] Cidadania: 3 / Fama: 777"
	if got := cstr(p); got != want {
		t.Errorf("/nick text = %q, want %q", got, want)
	}
}

// TestCommandNickOffline verifica que /nick num nome desconhecido se comporta como um
// sussurro offline normal (o aviso de não-conectado existente).
func TestCommandNickOffline(t *testing.T) {
	addr, stop, _ := startServerClock(t, chatDB())
	defer stop()
	a := enterWorldAs(t, addr, "tester")
	defer a.Close()

	whisperFrame(t, a, "nick", "Ghost")
	if got := readPanel(t, a); got != partyTargetGone {
		t.Errorf("got %q, want the not-connected panel", got)
	}
}

func TestApplyBonusScore(t *testing.T) {
	db := newDB()
	db.loadResult = world.CharacterState{
		Slot: 0, Name: "Hero", Class: 0, ClassMaster: classMasterMortal, X: 5, Y: 5,
		HP: 1000, MaxHP: 1000, Str: 11, Dex: 10, Level: 10,
		Damage:     50 + 11/2 + 10/3 + 10,
		ScoreBonus: 5,
		Equip:      [world.MaxEquip]world.Item{{Index: 11}},
	}
	addr, stop, _ := startServerClock(t, db)
	defer stop()
	c := enterWorld(t, addr)
	defer c.Close()

	body := protocol.MsgApplyBonusBody{BonusType: protocol.BonusScore, Detail: protocol.DetailStr}
	send(t, c, protocol.MsgApplyBonus, body.Encode())
	if ty, _, ok := readMaybe(t, c); !ok || ty != protocol.MsgUpdateEtc {
		t.Errorf("got %#x ok=%v, want UpdateEtc first", ty, ok)
	}
	ty, payload, ok := readMaybe(t, c)
	if !ok || ty != protocol.MsgUpdateScore {
		t.Errorf("got %#x ok=%v, want UpdateScore second", ty, ok)
	}
	want := baseDamageChar + 12/2 + 10/3 + 10
	if dmg := scoreDamage(payload); dmg != want {
		t.Errorf("UpdateScore Damage = %d, want %d after +1 STR (11→12)", dmg, want)
	}
}

// TestCommandKefra cobre a issue #410: /kefra para no gate do boss enquanto a
// Kefra está viva e abre o continente do Urnammu quando ela é derrotada, com a
// mensagem legada nos dois casos.
func TestCommandKefra(t *testing.T) {
	d, w, s, e := startOnePlayer(t)
	for _, tc := range []struct {
		name     string
		defeated bool
		dest     [2]int16
	}{
		{"kefra alive", false, kefraDestAlive},
		{"kefra defeated", true, kefraDestDefeated},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runInLoop(t, w, func() {
				d.expEvents.KefraLive = tc.defeated
				w.SetEntityPos(s.Conn, 2100, 2100)
				panels := w.SentOfType(s, protocol.MsgMessagePanel)
				if !d.runCommand(w, s, "kefra", nil) {
					t.Fatal("/kefra not handled")
				}
				if e.X < tc.dest[0] || e.X > tc.dest[0]+2 || e.Y < tc.dest[1] || e.Y > tc.dest[1]+2 {
					t.Errorf("teleported to (%d,%d), want (%d..+2,%d..+2)", e.X, e.Y, tc.dest[0], tc.dest[1])
				}
				if w.SentOfType(s, protocol.MsgMessagePanel) == panels {
					t.Error("no message sent")
				}
			})
		})
	}
}
