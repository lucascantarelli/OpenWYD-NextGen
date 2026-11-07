package handler

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"openwyd-nextgen/tmserver/internal/protocol"
	"openwyd-nextgen/tmserver/internal/world"
)

func combatDB() *fakeDB {
	db := newDB()
	db.loadResult = world.CharacterState{
		Slot: 0, Name: "Hero", X: 5, Y: 5,
		HP: 1000, MaxHP: 1000, Damage: 200, AC: 40,
	}
	return db
}

func attackFrame(t *testing.T, c net.Conn, tick uint32, targetID, skill int) {
	t.Helper()
	body := protocol.MsgAttackBody{
		SkillIndex: int16(skill),
		Dam:        []protocol.DamEntry{{TargetID: int32(targetID), Damage: 0}},
	}
	wire, err := protocol.Encode(protocol.Header{Type: protocol.MsgAttack, ClientTick: tick}, body.Encode(), 9)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Write(wire); err != nil {
		t.Fatal(err)
	}
}

// TestAttackHitExact é o caso dourado exato ponta a ponta: com um LCG limpo
// (semente 1) e um atacante/alvo conhecidos, o dano server-authoritative é
// determinístico. BASE_GetDoubleCritical consome o primeiro rand() mesmo quando nenhum
// crítico entra, então BASE_GetDamage vê o segundo rand(): 140 * 110 / 100 = 154.
func TestAttackHitExact(t *testing.T) {
	addr, stop, _ := startServerClock(t, combatDB())
	defer stop()

	attacker := enterWorld(t, addr) // conn 1 (atacante)
	defer attacker.Close()
	target := enterWorld(t, addr) // conn 2 (alvo), em view
	defer target.Close()

	send(t, attacker, protocol.MsgPKMode, protocol.EncodeStandardParm(1)) // PvP exige PK mode
	attackFrame(t, attacker, serverTime, 2, 0)

	ty, payload, ok := readMaybe(t, target)
	if !ok || ty != protocol.MsgAttack {
		t.Fatalf("target got %#x ok=%v, want MsgAttack broadcast", ty, ok)
	}
	var got protocol.MsgAttackBody
	if err := got.Decode(payload); err != nil {
		t.Fatal(err)
	}
	if len(got.Dam) != 1 || got.Dam[0].TargetID != 2 {
		t.Fatalf("Dam = %+v", got.Dam)
	}
	if got.Dam[0].Damage != 7 {
		t.Errorf("server damage = %d, want 7 (exact LCG golden)", got.Dam[0].Damage)
	}
}

func TestAttackSyncsVictimHpMp(t *testing.T) {
	addr, stop, _ := startServerClock(t, combatDB())
	defer stop()

	attacker := enterWorld(t, addr) // conn 1 (atacante)
	defer attacker.Close()
	target := enterWorld(t, addr) // conn 2 (alvo), em view
	defer target.Close()

	send(t, attacker, protocol.MsgPKMode, protocol.EncodeStandardParm(1)) // PvP exige PK mode
	attackFrame(t, attacker, serverTime, 2, 0)

	var damage, hp, reqHp int32
	sawAttack, sawHpSync := false, false
	for i := 0; i < 6 && (!sawAttack || !sawHpSync); i++ {
		ty, payload, ok := readMaybe(t, target)
		if !ok {
			break
		}
		switch ty {
		case protocol.MsgAttack:
			var got protocol.MsgAttackBody
			if err := got.Decode(payload); err != nil {
				t.Fatal(err)
			}
			if len(got.Dam) != 1 || got.Dam[0].TargetID != 2 {
				t.Fatalf("Dam = %+v", got.Dam)
			}
			damage = got.Dam[0].Damage
			sawAttack = true
		case protocol.MsgSetHpMp:
			if len(payload) < 16 {
				t.Fatalf("SetHpMp body = %d bytes, want at least 16", len(payload))
			}
			hp = int32(binary.LittleEndian.Uint32(payload[0:4]))
			reqHp = int32(binary.LittleEndian.Uint32(payload[8:12]))
			sawHpSync = true
		default:
			t.Fatalf("target got %#x, want MsgAttack/MsgSetHpMp", ty)
		}
	}
	if !sawAttack {
		t.Fatal("target did not receive MsgAttack")
	}
	if !sawHpSync {
		t.Fatal("target did not receive MsgSetHpMp after taking PvP damage")
	}
	if damage <= 0 {
		t.Fatalf("damage = %d, want > 0", damage)
	}
	wantHP := int32(1000) - damage
	if hp != wantHP || reqHp != wantHP {
		t.Errorf("SetHpMp Hp/ReqHp = %d/%d, want %d/%d", hp, reqHp, wantHP, wantHP)
	}
}

// TestAttackPlayerWithoutPKModeBlocked: dano PvP nunca entra sem o
// atacante ter optado por PK mode antes (tecla K, _MSG_PKMode) — sequência
// da issue #59 (apertar K sozinho não pode piscar o apelido; só um acerto PvP
// aplicado pisca, e um acerto não entra sem PK mode).
func TestAttackPlayerWithoutPKModeBlocked(t *testing.T) {
	addr, stop, _ := startServerClock(t, combatDB())
	defer stop()

	attacker := enterWorld(t, addr) // conn 1, PK mode nunca alternado
	defer attacker.Close()
	target := enterWorld(t, addr) // conn 2, em view
	defer target.Close()

	attackFrame(t, attacker, serverTime, 2, 0)

	ty, payload, ok := readMaybe(t, target)
	if !ok || ty != protocol.MsgAttack {
		t.Fatalf("target got %#x ok=%v, want MsgAttack broadcast", ty, ok)
	}
	var got protocol.MsgAttackBody
	if err := got.Decode(payload); err != nil {
		t.Fatal(err)
	}
	if len(got.Dam) != 1 || got.Dam[0].Damage != 0 {
		t.Errorf("Dam = %+v, want damage 0 (blocked, no PK mode)", got.Dam)
	}
}

// TestAttackPlayerWithPKModeSetsGuilty: alternar o PK mode sozinho não pode piscar
// o apelido (nenhum PKInfo(1) no ack); aplicar um acerto PvP depois precisa.
func TestAttackPlayerWithPKModeSetsGuilty(t *testing.T) {
	addr, stop, _ := startServerClock(t, combatDB())
	defer stop()

	attacker := enterWorld(t, addr) // conn 1
	defer attacker.Close()
	target := enterWorld(t, addr) // conn 2, em view
	defer target.Close()
	drainRaw(t, attacker)
	drainRaw(t, target)

	send(t, attacker, protocol.MsgPKMode, protocol.EncodeStandardParm(1))
	// Apertar K só envia um PKInfo (flag de atacável) — NADA de CreateMob, então a cor
	// do apelido (MobName[12]) fica intocada. Só drena o ack.
	if ty, _, ok := readMaybeRaw(t, attacker); !ok || ty != protocol.MsgPKInfo {
		t.Fatalf("attacker K-ack = %#x ok=%v, want PKInfo", ty, ok)
	}
	drainRaw(t, target) // o toggle também alcança o alvo como espectador

	attackFrame(t, attacker, serverTime, 2, 0)

	// Aplicar o acerto recolore o apelido do atacante de vermelho: o alvo (em view) recebe um
	// CreateMob novo do atacante (id 1) com MobName[12] = 0 (caos/vermelho),
	// antes do frame MsgAttack. Procure por ele, depois pelo dano.
	sawRedNick, sawAttack := false, false
	for i := 0; i < 20 && !sawAttack; i++ {
		ty, payload, ok := readMaybeRaw(t, target)
		if !ok {
			break
		}
		switch ty {
		case protocol.MsgCreateMob:
			if _, _, id := createMobFields(t, payload); id == 1 {
				if payload[6+12] != 0 {
					t.Errorf("attacker self-CreateMob MobName[12] = %d, want 0 (red/chaos after PvP hit)", payload[6+12])
				}
				sawRedNick = true
			}
		case protocol.MsgAttack:
			var got protocol.MsgAttackBody
			if err := got.Decode(payload); err != nil {
				t.Fatal(err)
			}
			if len(got.Dam) != 1 || got.Dam[0].Damage <= 0 {
				t.Errorf("Dam = %+v, want a landed hit (PK mode on)", got.Dam)
			}
			sawAttack = true
		}
	}
	if !sawRedNick {
		t.Error("attacker's nick was not recolored red (no CreateMob with MobName[12]=0) after landing a PvP hit")
	}
	if !sawAttack {
		t.Error("no MsgAttack broadcast to target")
	}
}

// TestAttackMarksBothSidesGuilty é a mudança de comportamento da issue #210: o legado marca
// Guilty nos DOIS SetGuilty(conn,8) e SetGuilty(idx,8) (_MSG_Attack.cpp "PK -
// War - Miss"), não só no atacante — o modelo Go anterior só tocava no
// timer GuiltyUntil fixo do atacante. O /cp dos dois combatentes precisa ler caótico
// (-75) depois que o acerto entra.
func TestAttackMarksBothSidesGuilty(t *testing.T) {
	addr, stop, _ := startServerClock(t, combatDB())
	defer stop()

	attacker := enterWorld(t, addr) // conn 1
	defer attacker.Close()
	target := enterWorld(t, addr) // conn 2
	defer target.Close()

	send(t, attacker, protocol.MsgPKMode, protocol.EncodeStandardParm(1))
	attackFrame(t, attacker, serverTime, 2, 0)
	if ty, _, ok := readMaybe(t, attacker); !ok || ty != protocol.MsgAttack {
		t.Fatalf("attacker got %#x ok=%v, want MsgAttack echo", ty, ok)
	}
	for i := 0; i < 2; i++ {
		if _, _, ok := readMaybe(t, target); !ok {
			break
		}
	}

	want := "Pontos Caos atual: -75"
	whisperFrame(t, attacker, "cp", "")
	if text, ok := readChaosPointsText(t, attacker); !ok || !strings.HasPrefix(text, want) {
		t.Errorf("attacker /cp = %q ok=%v, want prefix %q", text, ok, want)
	}
	whisperFrame(t, target, "cp", "")
	if text, ok := readChaosPointsText(t, target); !ok || !strings.HasPrefix(text, want) {
		t.Errorf("victim /cp = %q ok=%v, want prefix %q", text, ok, want)
	}
}

// pkGateDB dá duas contas distintas (casando com "tester"=7/"tradeb"=11 do newDB)
// valores iniciais de PKPoint diferentes, então um teste pode pôr um lado abaixo e o
// outro acima do limiar pointPK<=10 de bloqueio de dano.
func pkGateDB(attackerPKPoint, targetPKPoint uint8) *fakeDB {
	db := newDB()
	db.loads = map[int64]world.CharacterState{
		7:  {Slot: 0, Name: "Chaotic", X: 5, Y: 5, HP: 1000, MaxHP: 1000, Damage: 200, AC: 40, PKPoint: attackerPKPoint},
		11: {Slot: 0, Name: "Clean", X: 5, Y: 5, HP: 1000, MaxHP: 1000, Damage: 200, AC: 40, PKPoint: targetPKPoint},
	}
	return db
}

// TestAttackBlockedWhenAttackerTooChaotic é o gate de não-matar "PK - War -
// Miss" do _MSG_Attack.cpp (issue #210): um atacante com PKPoint<=10 (byte cru,
// não o valor -75 de exibição) não pode causar dano a um alvo comparativamente limpo
// (PKPoint>10) de jeito nenhum — o acerto resolve com Dam=0 e um aviso no chat, nenhum
// Guilty é marcado em nenhum dos lados.
func TestAttackBlockedWhenAttackerTooChaotic(t *testing.T) {
	addr, stop, _ := startServerClock(t, pkGateDB(5, 75))
	defer stop()

	attacker := enterWorldAs(t, addr, "tester") // conn 1, PKPoint=5 (muito caótico)
	defer attacker.Close()
	target := enterWorldAs(t, addr, "tradeb") // conn 2, PKPoint=75 (limpo)
	defer target.Close()

	send(t, attacker, protocol.MsgPKMode, protocol.EncodeStandardParm(1))
	attackFrame(t, attacker, serverTime, 2, 0)

	// _DN_CantKillUser carrega o PKPoint cru (issue #422).
	if ty, p, ok := readMaybe(t, attacker); !ok || ty != protocol.MsgMessagePanel || cstr(p) != fmt.Sprintf(cantKillUser, 5) {
		t.Fatalf("attacker got %#x %q ok=%v, want the cant-kill panel", ty, p, ok)
	}
	ty, payload, ok := readMaybe(t, target)
	if !ok || ty != protocol.MsgAttack {
		t.Fatalf("target got %#x ok=%v, want MsgAttack broadcast", ty, ok)
	}
	var got protocol.MsgAttackBody
	if err := got.Decode(payload); err != nil {
		t.Fatal(err)
	}
	if len(got.Dam) != 1 || got.Dam[0].Damage != 0 {
		t.Errorf("Dam = %+v, want damage 0 (blocked: attacker PKPoint<=10 vs clean target)", got.Dam)
	}
}

// TestAttackAllowedWhenBothChaotic verifica que o gate de não-matar NÃO dispara
// entre dois jogadores já caóticos (SummonerPointPK<=10): o legado só bloqueia
// quando a VÍTIMA ainda está comparativamente limpa.
func TestAttackAllowedWhenBothChaotic(t *testing.T) {
	addr, stop, _ := startServerClock(t, pkGateDB(5, 5))
	defer stop()

	attacker := enterWorldAs(t, addr, "tester") // conn 1, PKPoint=5
	defer attacker.Close()
	target := enterWorldAs(t, addr, "tradeb") // conn 2, PKPoint=5
	defer target.Close()

	send(t, attacker, protocol.MsgPKMode, protocol.EncodeStandardParm(1))
	attackFrame(t, attacker, serverTime, 2, 0)

	ty, payload, ok := readMaybe(t, target)
	if !ok || ty != protocol.MsgAttack {
		t.Fatalf("target got %#x ok=%v, want MsgAttack broadcast", ty, ok)
	}
	var got protocol.MsgAttackBody
	if err := got.Decode(payload); err != nil {
		t.Fatal(err)
	}
	if len(got.Dam) != 1 || got.Dam[0].Damage <= 0 {
		t.Errorf("Dam = %+v, want a landed hit (victim already chaotic, gate must not fire)", got.Dam)
	}
}

// readFrameHeader lê um frame e devolve seu header completo (não só o tipo),
// pulando o ruído de visibilidade de entidade, então um teste pode afirmar HEADER.ID.
func readFrameHeader(t *testing.T, c net.Conn) (protocol.Header, []byte) {
	t.Helper()
	for {
		_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
		var sz [2]byte
		if _, err := io.ReadFull(c, sz[:]); err != nil {
			t.Fatalf("read size: %v", err)
		}
		buf := make([]byte, binary.LittleEndian.Uint16(sz[:]))
		copy(buf, sz[:])
		if _, err := io.ReadFull(c, buf[2:]); err != nil {
			t.Fatalf("read body: %v", err)
		}
		h, payload, _, err := protocol.Decode(buf)
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if h.Type == protocol.MsgCreateMob || h.Type == protocol.MsgRemoveMob || h.Type == protocol.MsgPKInfo {
			continue
		}
		return h, payload
	}
}

// TestAttackHeaderIsSceneField trava a correção da barra de EXP: a transmissão do ataque precisa
// carregar HEADER.ID = ESCENE_FIELD (protocol.IDScene), exatamente como o original
// (_MSG_Attack.cpp:25). O cliente só aplica CurrentExp/Hp/Mp do próprio atacante
// — ou seja, move a barra de EXP — quando o ataque chega como evento de campo/cena; com
// HEADER.ID = a conn do atacante ele não fazia.
func TestAttackHeaderIsSceneField(t *testing.T) {
	addr, stop, _ := startServerClock(t, combatDB())
	defer stop()

	attacker := enterWorld(t, addr) // conn 1
	defer attacker.Close()
	target := enterWorld(t, addr) // conn 2, em view
	defer target.Close()

	attackFrame(t, attacker, serverTime, 2, 0)

	h, _ := readFrameHeader(t, target)
	if h.Type != protocol.MsgAttack {
		t.Fatalf("target got %#x, want MsgAttack", h.Type)
	}
	if h.ID != protocol.IDScene {
		t.Errorf("attack HEADER.ID = %d, want ESCENE_FIELD %d", h.ID, protocol.IDScene)
	}
}

func TestAttackTooFastDropped(t *testing.T) {
	addr, stop, _ := startServerClock(t, combatDB())
	defer stop()
	attacker := enterWorld(t, addr)
	defer attacker.Close()
	target := enterWorld(t, addr)
	defer target.Close()

	attackFrame(t, attacker, serverTime, 2, 0)
	if ty, _, ok := readMaybe(t, target); !ok || ty != protocol.MsgAttack {
		t.Fatalf("first attack not broadcast (%#x ok=%v)", ty, ok)
	}
	// Segundo ataque só 300ms depois (< cadência de 800ms) ⇒ descartado, sem transmissão.
	attackFrame(t, attacker, serverTime+300, 2, 0)
	if ty, _, ok := readMaybe(t, target); ok {
		t.Errorf("target received %#x; too-fast attack should be dropped", ty)
	}
}

// TestDeadCharacterRevivedOnLogin: um personagem persistido com HP 0 (morto por um mob
// antes de desconectar, agora que mobs podem matar) não pode entrar no mundo morto —
// o login o revive com HP cheio para ele agir normalmente em vez de ficar travado.
func TestDeadCharacterRevivedOnLogin(t *testing.T) {
	db := combatDB()
	db.loadResult.HP = 0 // foi salvo morto
	addr, stop, _ := startServerClock(t, db)
	defer stop()
	attacker := enterWorld(t, addr)
	defer attacker.Close()
	target := enterWorld(t, addr)
	defer target.Close()

	// Revivido → vivo → o ataque resolve e alcança o alvo em view.
	attackFrame(t, attacker, serverTime, 2, 0)
	if ty, _, ok := readMaybe(t, target); !ok || ty != protocol.MsgAttack {
		t.Fatalf("revived attacker could not act: got %#x ok=%v, want MsgAttack", ty, ok)
	}
}
