//go:build e2e

// Validação ponta a ponta dos EVENTOS DE MUNDO (issue #116) contra uma stack VIVA,
// conduzida por um cliente CPSock headless que fala exatamente o que o WYD.exe fala.
//
// Por que isto existe: os eventos de mundo são movidos a timer e seu único
// contrato observável é o frame S→C que o cliente recebe — um servidor que calcula o
// estado certo mas envia o pacote errado é indistinguível de um quebrado
// no nível de teste unitário (SESSION-PRIMER §5.1, "servidor-correto ≠ cliente-correto").
// Estes testes verificam os bytes no wire, de um socket real, contra uma cadeia real
// tmServer → dbServer → PostgreSQL.
//
// O cliente aqui é deliberadamente burro: ele faz o handshake INITCODE, um
// MSG_AccountLogin real, MSG_CreateCharacter e MSG_CharacterLogin, e então lê
// frames. Ele não renderiza nada, então valida a metade do SERVIDOR de "o que o WYD.exe
// veria" — a metade de renderização ainda precisa de um humano no cliente real, e
// os eventos que não têm frame S→C algum estão listados em
// docs/migration/worldevents-e2e-report.md em vez de silenciosamente pulados.
//
// Rodar contra uma stack subida por scripts/run-local.sh:
//
//	W2PP_E2E_ADDR=127.0.0.1:8281 W2PP_E2E_ACCOUNT=test W2PP_E2E_PASSWORD=test123 \
//	  W2PP_E2E_VERSION=12000 go test -tags=e2e -run TestE2EWorldEvent ./tmserver/internal/world/ -v
//
// Os casos movidos a GM precisam da conta promovida a papel de moderador/admin:
//
//	docker compose exec db psql -U postgres -c "UPDATE account SET role='admin' WHERE name='test'"
package world

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"openwyd-nextgen/tmserver/internal/protocol"
)

// clientKeyWord semeia a ofuscação C→S. Qualquer valor serve (o servidor o lê
// do byte 2); 7 é o que o teste de fumaça usa.
const clientKeyWord = 7

// cnfCharLoginWeatherOffset é onde o CurrentWeather viaja no snapshot de login:
// corpo do MSG_CNFCharacterLogin @1032, um unsigned short (protocol/mob.go
// EncodeCNFCharacterLoginRaw, sm.Weather legado em ProcessDBMessage.cpp:834).
// Weather NÃO é enviado como pacote próprio no login — se este offset estiver errado o
// cliente mostra o céu errado até a próxima transmissão.
const cnfCharLoginWeatherOffset = 1032

// liveClient é um substituto headless do WYD.exe: uma conexão CPSock mais um
// backlog de frames já retirados do socket.
//
// O backlog importa. Entrar no mundo dispara uma rajada de frames não relacionados
// (CreateMob para cada entidade visível, affects, slots de item), e um frame de
// evento de mundo chega intercalado com eles. Toda espera portanto varre adiante e
// guarda o que não queria, para uma espera posterior ainda poder achá-lo.
type liveClient struct {
	t       *testing.T
	conn    net.Conn
	account string
	backlog []protocol.Frame
	connID  uint16
	// snapshot é o MSG_CNFCharacterLogin com que este cliente entrou no mundo.
	// Weather viaja nele, então os testes o leem em vez de rederivar o estado.
	snapshot protocol.Frame
}

// dialLive abre uma conexão e completa o handshake INITCODE.
func dialLive(t *testing.T, addr, account string) *liveClient {
	t.Helper()
	return &liveClient{t: t, conn: liveDial(t, addr), account: account}
}

func (c *liveClient) close() {
	if c.conn != nil {
		_ = c.conn.Close()
	}
}

// send codifica e escreve um frame C→S.
func (c *liveClient) send(typ protocol.Type, id uint16, body []byte) {
	c.t.Helper()
	wire, err := protocol.Encode(protocol.Header{Type: typ, ID: id}, body, clientKeyWord)
	if err != nil {
		c.t.Fatalf("encode %#x: %v", typ, err)
	}
	if _, err := c.conn.Write(wire); err != nil {
		c.t.Fatalf("write %#x: %v", typ, err)
	}
}

// readFrame retira exatamente um frame do socket, respeitando o deadline.
func (c *liveClient) readFrame(deadline time.Time) (protocol.Frame, error) {
	if err := c.conn.SetReadDeadline(deadline); err != nil {
		return protocol.Frame{}, err
	}
	var sz [2]byte
	if _, err := io.ReadFull(c.conn, sz[:]); err != nil {
		return protocol.Frame{}, err
	}
	size := int(binary.LittleEndian.Uint16(sz[:]))
	if size < protocol.HeaderSize {
		return protocol.Frame{}, fmt.Errorf("frame size %d below header size", size)
	}
	buf := make([]byte, size)
	copy(buf, sz[:])
	if _, err := io.ReadFull(c.conn, buf[2:]); err != nil {
		return protocol.Frame{}, err
	}
	h, payload, _, err := protocol.Decode(buf)
	if err != nil {
		return protocol.Frame{}, err
	}
	return protocol.Frame{Header: h, Payload: payload}, nil
}

// waitFor varre o backlog e depois o socket procurando o primeiro frame do tipo typ.
// Frames de outros tipos são retidos no backlog, nunca descartados.
func (c *liveClient) waitFor(typ protocol.Type, within time.Duration) (protocol.Frame, bool) {
	c.t.Helper()
	for i, f := range c.backlog {
		if f.Header.Type == typ {
			c.backlog = append(c.backlog[:i], c.backlog[i+1:]...)
			return f, true
		}
	}
	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		f, err := c.readFrame(deadline)
		if err != nil {
			return protocol.Frame{}, false
		}
		if f.Header.Type == typ {
			return f, true
		}
		c.backlog = append(c.backlog, f)
	}
	return protocol.Frame{}, false
}

// waitForAny é o waitFor sobre um conjunto de tipos, para os ramos em que o servidor
// legitimamente responde um de vários frames. Esperar por eles um a um
// gastaria um timeout inteiro no ramo que não aconteceu.
func (c *liveClient) waitForAny(types []protocol.Type, within time.Duration) (protocol.Frame, bool) {
	c.t.Helper()
	match := func(t protocol.Type) bool {
		for _, want := range types {
			if t == want {
				return true
			}
		}
		return false
	}
	for i, f := range c.backlog {
		if match(f.Header.Type) {
			c.backlog = append(c.backlog[:i], c.backlog[i+1:]...)
			return f, true
		}
	}
	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		f, err := c.readFrame(deadline)
		if err != nil {
			return protocol.Frame{}, false
		}
		if match(f.Header.Type) {
			return f, true
		}
		c.backlog = append(c.backlog, f)
	}
	return protocol.Frame{}, false
}

// waitForChat varre um MSG_MessageChat cujo texto contém substr. Anúncios de
// evento de mundo (torre, castelo, drops de evento) todos viajam neste frame.
func (c *liveClient) waitForChat(substr string, within time.Duration) (string, bool) {
	c.t.Helper()
	deadline := time.Now().Add(within)
	// Backlog primeiro, depois o socket, com a mesma regra de casamento.
	for i := 0; i < len(c.backlog); i++ {
		if text, ok := chatText(c.backlog[i]); ok && strings.Contains(text, substr) {
			c.backlog = append(c.backlog[:i], c.backlog[i+1:]...)
			return text, true
		}
	}
	for time.Now().Before(deadline) {
		f, err := c.readFrame(deadline)
		if err != nil {
			return "", false
		}
		if text, ok := chatText(f); ok && strings.Contains(text, substr) {
			return text, true
		}
		c.backlog = append(c.backlog, f)
	}
	return "", false
}

// chatText extrai o texto terminado em NUL de um frame MSG_MessageChat.
func chatText(f protocol.Frame) (string, bool) {
	if f.Header.Type != protocol.MsgMessageChat {
		return "", false
	}
	return cstring(f.Payload), true
}

// cstring lê uma string terminada em NUL de b.
func cstring(b []byte) string {
	if i := bytes.IndexByte(b, 0); i >= 0 {
		b = b[:i]
	}
	return string(b)
}

// quiet drena tudo que o servidor empurra por d, para um waitFor posterior não casar
// um frame que o passo ANTERIOR causou. Não retém nada: chamadores o usam como barreira.
func (c *liveClient) quiet(d time.Duration) {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if _, err := c.readFrame(deadline); err != nil {
			return
		}
	}
}

// login executa MSG_AccountLogin e espera a resposta de seleção de personagem.
func (c *liveClient) login(password string) {
	c.t.Helper()
	if _, err := c.conn.Write(loginFrame(c.account, password, "")); err != nil {
		c.t.Fatalf("write login: %v", err)
	}
	f, ok := c.waitFor(protocol.MsgCNFAccountLogin, 10*time.Second)
	if !ok {
		c.t.Fatalf("account %q: no CNFAccountLogin (backlog: %s)", c.account, c.backlogTypes())
	}
	c.connID = f.Header.ID
}

// enterWorld cria o personagem se ele ainda não existe e seleciona o slot 0,
// deixando a sessão em USER_PLAY. Devolve o snapshot CNFCharacterLogin.
//
// A criação é tentada incondicionalmente: numa re-execução o nome está tomado e o
// servidor responde MSG_NewCharacterFail, que é o mesmo estado "slot 0 pronto"
// . Isso mantém o teste idempotente sem ciclo de apagar/recriar.
func (c *liveClient) enterWorld(charName string, class int32) protocol.Frame {
	c.t.Helper()
	create := protocol.MsgCreateCharacterBody{Slot: 0, MobClass: class}
	copy(create.MobName[:], charName)
	c.send(protocol.MsgCreateCharacter, 0, create.Encode())
	// Qualquer resultado serve; ambos significam que o slot 0 está resolvido.
	if _, ok := c.waitForAny([]protocol.Type{protocol.MsgCNFNewCharacter, protocol.MsgNewCharacterFail},
		8*time.Second); !ok {
		c.t.Fatalf("create char %q: neither CNFNewCharacter nor NewCharacterFail", charName)
	}

	login := protocol.MsgCharacterLoginBody{Slot: 0}
	c.send(protocol.MsgCharacterLogin, 0, login.Encode())
	f, ok := c.waitFor(protocol.MsgCNFCharacterLogin, 15*time.Second)
	if !ok {
		c.t.Fatalf("char login: no CNFCharacterLogin (backlog: %s)", c.backlogTypes())
	}
	if len(f.Payload) < cnfCharLoginWeatherOffset+2 {
		c.t.Fatalf("CNFCharacterLogin body too short: %d bytes", len(f.Payload))
	}
	c.snapshot = f
	return f
}

// gm envia um comando "/gm <linha>". O cliente codifica comandos com barra como um
// sussurro cujo NOME DE ALVO é a palavra-chave ("gm") e cujo texto é o resto da
// linha — a peculiaridade legada do _MSG_MessageWhisper (handler/chat.go runCommand).
func (c *liveClient) gm(line string) {
	c.t.Helper()
	body := protocol.MsgWhisperBody{String: append([]byte(line), 0)}
	copy(body.MobName[:], "gm")
	c.send(protocol.MsgMessageWhisper, c.connID, body.Encode())
}

func (c *liveClient) backlogTypes() string {
	var b strings.Builder
	for i, f := range c.backlog {
		if i > 0 {
			b.WriteString(",")
		}
		fmt.Fprintf(&b, "%#x", f.Header.Type)
	}
	if b.Len() == 0 {
		return "(empty)"
	}
	return b.String()
}

// snapshotWeather lê CurrentWeather de um corpo de CNFCharacterLogin.
func snapshotWeather(f protocol.Frame) int32 {
	return int32(binary.LittleEndian.Uint16(f.Payload[cnfCharLoginWeatherOffset:]))
}

// newPlayer é o arranjo comum: conectar, logar, entrar no mundo.
func newPlayer(t *testing.T, account, password, charName string) *liveClient {
	t.Helper()
	c := dialLive(t, env("W2PP_E2E_ADDR", "127.0.0.1:8281"), account)
	t.Cleanup(c.close)
	c.login(password)
	c.enterWorld(charName, 0) // classe 0 = TransKnight
	return c
}

// gmPlayer é newPlayer para a conta privilegiada em teste.
func gmPlayer(t *testing.T) *liveClient {
	t.Helper()
	return newPlayer(t,
		env("W2PP_E2E_ACCOUNT", "test"),
		env("W2PP_E2E_PASSWORD", "test123"),
		env("W2PP_E2E_CHAR", "EvtProbe"))
}

// ---------------------------------------------------------------------------
// Weather (issue #116 fase 1)
// ---------------------------------------------------------------------------

// TestE2EWorldEventWeatherBroadcast é o contrato central do clima: um GM forçando um
// valor precisa produzir UM MSG_UpdateWeather (0x018B) por cliente em jogo, carregando
// esse valor como int32, com HEADER.ID = ESCENE_FIELD (30000).
//
// As duas metades importam ao cliente real: um Type errado é ignorado, e um
// HEADER.ID errado roteia o pacote ao handler de cena errado (SendFunc.cpp:1669-1696).
func TestE2EWorldEventWeatherBroadcast(t *testing.T) {
	c := gmPlayer(t)
	defer c.gm("weather auto") // deixa o servidor em sorteios automáticos

	for _, want := range []int32{1, 2, 0} {
		c.quiet(300 * time.Millisecond)
		c.gm(fmt.Sprintf("weather %d", want))

		f, ok := c.waitFor(protocol.MsgUpdateWeather, 8*time.Second)
		if !ok {
			t.Fatalf("weather %d: no MSG_UpdateWeather (%#x) reached the client",
				want, protocol.MsgUpdateWeather)
		}
		if len(f.Payload) != protocol.MsgUpdateWeatherBodySize {
			t.Errorf("weather %d: body = %d bytes, want %d",
				want, len(f.Payload), protocol.MsgUpdateWeatherBodySize)
		}
		if got := int32(binary.LittleEndian.Uint32(f.Payload)); got != want {
			t.Errorf("weather payload = %d, want %d", got, want)
		}
		if f.Header.ID != protocol.IDScene {
			t.Errorf("weather %d: HEADER.ID = %d, want ESCENE_FIELD (%d)",
				want, f.Header.ID, protocol.IDScene)
		}
		t.Logf("weather %d → MSG_UpdateWeather ok (ID=%d, %d-byte body)",
			want, f.Header.ID, len(f.Payload))
	}
}

// TestE2EWorldEventWeatherIdempotent fixa a guarda
// `ForceWeather != CurrentWeather` do legado: forçar de novo o valor que já está
// vivo NÃO pode pôr um segundo pacote no wire. Um cliente que recebe uma
// mudança de clima redundante reinicia a transição do céu, então a guarda é visível.
func TestE2EWorldEventWeatherIdempotent(t *testing.T) {
	c := gmPlayer(t)
	defer c.gm("weather auto")

	c.gm("weather 1")
	if _, ok := c.waitFor(protocol.MsgUpdateWeather, 8*time.Second); !ok {
		t.Fatalf("setup: first /gm weather 1 produced no MSG_UpdateWeather")
	}
	c.quiet(500 * time.Millisecond)

	c.gm("weather 1") // já é 1 — precisa ser no-op no wire
	if f, ok := c.waitFor(protocol.MsgUpdateWeather, 3*time.Second); ok {
		t.Errorf("re-forcing the live weather re-broadcast it: payload=%d",
			int32(binary.LittleEndian.Uint32(f.Payload)))
	} else {
		t.Log("re-forcing the live weather is silent, as the legacy guard requires")
	}
}

// TestE2EWorldEventWeatherRejectsInvalid cobre a divergência deliberada do
// legado: imple.cpp:1122-1129 enviava QUALQUER int direto ao cliente, e é
// assim que um /weather ruim quebrava o céu. O port valida e responde com uma
// linha de erro em vez de transmitir.
func TestE2EWorldEventWeatherRejectsInvalid(t *testing.T) {
	c := gmPlayer(t)
	defer c.gm("weather auto")

	c.quiet(300 * time.Millisecond)
	c.gm("weather 9")

	if _, ok := c.waitForChat("clima invalido", 5*time.Second); !ok {
		t.Errorf("no rejection message for /gm weather 9")
	}
	if f, ok := c.waitFor(protocol.MsgUpdateWeather, 2*time.Second); ok {
		t.Errorf("invalid weather was BROADCAST: payload=%d",
			int32(binary.LittleEndian.Uint32(f.Payload)))
	}
}

// TestE2EWorldEventWeatherLoginSnapshot prova o segundo caminho de entrega: um
// cliente que loga DEPOIS de uma mudança de clima aprende o valor atual pelo
// snapshot do MSG_CNFCharacterLogin (@1032), não de uma transmissão que nunca viu.
// Sem isso um jogador que reloga renderiza céu limpo durante uma tempestade.
//
// Precisa de uma segunda conta semeada para a primeira sessão poder ficar em jogo:
//
//	docker compose run --rm dbserver seed-account -name test2 -pass test123
func TestE2EWorldEventWeatherLoginSnapshot(t *testing.T) {
	second := os.Getenv("W2PP_E2E_ACCOUNT2")
	if second == "" {
		t.Skip("motivo: ambiente — definir W2PP_E2E_ACCOUNT2 (segunda conta semeada) para o teste de snapshot de login")
	}
	gm := gmPlayer(t)
	defer gm.gm("weather auto")

	const want = int32(2)
	gm.gm(fmt.Sprintf("weather %d", want))
	if _, ok := gm.waitFor(protocol.MsgUpdateWeather, 8*time.Second); !ok {
		t.Fatalf("setup: /gm weather %d produced no broadcast", want)
	}

	late := newPlayer(t, second,
		env("W2PP_E2E_PASSWORD2", "test123"),
		env("W2PP_E2E_CHAR2", "EvtProbe2"))
	if got := snapshotWeather(late.snapshot); got != want {
		t.Errorf("CNFCharacterLogin weather @%d = %d, want %d — a relogging client would render the wrong sky",
			cnfCharLoginWeatherOffset, got, want)
	} else {
		t.Logf("late joiner received weather %d in its login snapshot", got)
	}
}

// ---------------------------------------------------------------------------
// Evento de novato (issue #116 fase 2)
// ---------------------------------------------------------------------------

// createMobScoreOffset é onde STRUCT_SCORE começa dentro de um corpo MSG_CreateMob
// (protocol/createmob.go: Score @abs136 → body124). Level fica em +0, MaxHp em
// +16 e Hp em +24 dentro dele.
const createMobScoreOffset = 124

type mobView struct {
	id        int
	level     int32
	maxHP, hp int32
}

// parseCreateMob lê os campos em que o handicap de novato é visível.
func parseCreateMob(f protocol.Frame) (mobView, bool) {
	const need = createMobScoreOffset + 32
	if f.Header.Type != protocol.MsgCreateMob || len(f.Payload) < need {
		return mobView{}, false
	}
	s := f.Payload[createMobScoreOffset:]
	return mobView{
		id:    int(binary.LittleEndian.Uint16(f.Payload[4:])),
		level: int32(binary.LittleEndian.Uint32(s[0:])),
		maxHP: int32(binary.LittleEndian.Uint32(s[16:])),
		hp:    int32(binary.LittleEndian.Uint32(s[24:])),
	}, true
}

// spawnAndInspect roda `/gm spawn tmpl` e devolve o CreateMob que o servidor
// empurra para a criatura recém-invocada (revealSpawned).
func (c *liveClient) spawnAndInspect(tmpl int, within time.Duration) (mobView, bool) {
	c.t.Helper()
	c.quiet(time.Second) // deixa a rajada de entrada no mundo terminar primeiro
	c.backlog = nil
	c.gm(fmt.Sprintf("spawn %d", tmpl))
	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		f, err := c.readFrame(deadline)
		if err != nil {
			return mobView{}, false
		}
		m, ok := parseCreateMob(f)
		if !ok || m.id < MaxUser { // jogadores compartilham o espaço de ids abaixo de MaxUser
			continue
		}
		return m, true
	}
	return mobView{}, false
}

// TestE2EWorldEventNewbieMobHandicap valida o único efeito VISÍVEL AO CLIENTE da fase 2:
// enquanto NewbieEventServer está ligado, uma criatura abaixo do nível 120 nasce com 3/4 do HP
// (Server.cpp:3326-3327 → world.applyNewbieHandicap). MaxHp é deliberadamente
// deixado em paz, então o cliente renderiza uma criatura cuja barra de vida começa três quartos
// cheia — essa diferença é todo o contrato observável.
//
// O lado de EXP da mesma flag não tem frame S→C próprio e não é
// observável de um cliente; é coberto pelos testes unitários de internal/level.
//
// A flag é movida pela config do PORTAL, não pela flag de boot -newbie-event:
// ApplyWorldEventConfigBoot chama setNewbieEvent com o que
// world_event_config.newbie_event_enabled disser, então o valor do banco ganha da
// flag de CLI no boot. O executor portanto alterna a linha do banco (incrementando
// world_event_meta.version para o polling de 15 ticks recarregar) e roda este teste duas vezes:
//
// W2PP_E2E_NEWBIE_EXPECT=full       # linha false → criaturas nascem com MaxHp
// W2PP_E2E_NEWBIE_EXPECT=handicap   # linha true  → criaturas nascem com 3/4 do MaxHp
func TestE2EWorldEventNewbieMobHandicap(t *testing.T) {
	expect := os.Getenv("W2PP_E2E_NEWBIE_EXPECT")
	if expect != "full" && expect != "handicap" {
		t.Skip("motivo: ambiente — definir W2PP_E2E_NEWBIE_EXPECT como 'full' ou 'handicap' (ver scripts/e2e-worldevents.sh)")
	}
	tmpl := 1
	if v := os.Getenv("W2PP_E2E_SUMMON_TEMPLATE"); v != "" {
		fmt.Sscanf(v, "%d", &tmpl)
	}

	c := gmPlayer(t)
	got, ok := c.spawnAndInspect(tmpl, 15*time.Second)
	if !ok {
		t.Fatalf("no CreateMob after /gm spawn %d", tmpl)
	}
	if got.maxHP <= 0 {
		t.Fatalf("spawned mob has MaxHp %d — template %d is not usable for this check", got.maxHP, tmpl)
	}
	if got.level >= 120 {
		t.Skipf("motivo: ambiente — template de invocação %d é nível %d (>=120) e o handicap só vale abaixo disso; definir W2PP_E2E_SUMMON_TEMPLATE",
			tmpl, got.level)
	}

	want := got.maxHP
	if expect == "handicap" {
		want = 3 * got.maxHP / 4
	}
	if got.hp != want {
		t.Errorf("newbie event %s: level %d mob spawned at HP %d/%d, want %d — the client would draw the wrong health bar",
			expect, got.level, got.hp, got.maxHP, want)
		return
	}
	t.Logf("newbie event %s: level %d mob spawned at %d/%d HP as expected",
		expect, got.level, got.hp, got.maxHP)
}

// ---------------------------------------------------------------------------
// Guerra de captura da torre (issue #116 fase 4)
// ---------------------------------------------------------------------------

// TestE2EWorldEventTowerWar valida a janela da torre condicionada ao calendário ponta a ponta:
// o aviso de anúncio no minuto ≤5 e o aviso de início no minuto ≥6, ambos como
// linhas MSG_MessageChat que todo cliente em jogo recebe.
//
// A janela é um dia de semana na hora 20 com NewbieEventServer ligado (CWarTower.cpp:203,
// handler/towerwar.go). Em vez de esperar uma terça-feira real à noite, o executor
// inicia um SEGUNDO tmserver cuja TZ o coloca dentro da janela — o evento lê
// time.Now(), então o fuso do container é o gate inteiro. Veja
// scripts/e2e-worldevents.sh, que calcula o fuso e exporta o endereço.
func TestE2EWorldEventTowerWar(t *testing.T) {
	addr := os.Getenv("W2PP_E2E_TOWER_ADDR")
	if addr == "" {
		t.Skip("motivo: ambiente — definir W2PP_E2E_TOWER_ADDR para um tmserver dentro da janela da torre (ver scripts/e2e-worldevents.sh)")
	}
	c := dialLive(t, addr, env("W2PP_E2E_ACCOUNT", "test"))
	t.Cleanup(c.close)
	c.login(env("W2PP_E2E_PASSWORD", "test123"))
	c.enterWorld(env("W2PP_E2E_CHAR", "EvtProbe"), 0)

	// tickTowerWar roda uma passada por minuto, então o anúncio cai em ~2
	// minutos do boot quando o servidor iniciou dentro da janela.
	if text, ok := c.waitForChat("[Torre]", 3*time.Minute); ok {
		t.Logf("tower announce reached the client: %q", text)
	} else {
		t.Fatalf("no [Torre] notice within the window — the client would never learn the war started")
	}
	// A transição de início precisa de minuto >= 6, então de um anúncio no minuto 0-5 ela
	// pode estar a seis minutos inteiros. Esperar menos que isso reportaria um
	// evento que funciona como quebrado.
	if text, ok := c.waitForChat("comecou", 8*time.Minute); ok {
		t.Logf("tower start reached the client: %q", text)
	} else {
		t.Errorf("announce arrived but the START notice did not")
	}
}
