//go:build e2e

// Fumaça ponta a ponta contra uma stack VIVA (docker compose up): um cliente CPSock bruto
// conecta na borda publicada do tmServer, faz o handshake INITCODE e um
// MSG_AccountLogin real, e verifica que o servidor responde MSG_CNFAccountLogin —
// provando a cadeia completa tmServer(CPSock) → dbServer(gRPC/mTLS) → PostgreSQL.
//
// Semear a conta primeiro, depois:
//
//	W2PP_E2E_ADDR=127.0.0.1:8281 W2PP_E2E_ACCOUNT=smoke W2PP_E2E_PASSWORD=smoke123 \
//	  go test -tags=e2e -run TestE2ESmokeLogin ./tmserver/internal/world/ -v
package world

import (
	"encoding/binary"
	"io"
	"net"
	"os"
	"strconv"
	"testing"
	"time"

	"openwyd-nextgen/tmserver/internal/protocol"
)

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// liveDial abre uma conexão TCP com o tmServer vivo e envia INITCODE.
func liveDial(t *testing.T, addr string) net.Conn {
	t.Helper()
	c, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		t.Fatalf("dial %s: %v", addr, err)
	}
	var ic [4]byte
	binary.LittleEndian.PutUint32(ic[:], protocol.InitCode)
	if _, err := c.Write(ic[:]); err != nil {
		t.Fatalf("write INITCODE: %v", err)
	}
	return c
}

// liveRead lê um frame S→C (prefixo de tamanho de 2 bytes, sem INITCODE) e o decodifica.
func liveRead(t *testing.T, c net.Conn) (protocol.Header, []byte) {
	t.Helper()
	_ = c.SetReadDeadline(time.Now().Add(5 * time.Second))
	var sz [2]byte
	if _, err := io.ReadFull(c, sz[:]); err != nil {
		t.Fatalf("read size: %v", err)
	}
	size := int(binary.LittleEndian.Uint16(sz[:]))
	buf := make([]byte, size)
	copy(buf, sz[:])
	if _, err := io.ReadFull(c, buf[2:]); err != nil {
		t.Fatalf("read body: %v", err)
	}
	h, payload, _, err := protocol.Decode(buf)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	return h, payload
}

func loginFrame(account, password, token string) []byte {
	var body protocol.MsgAccountLoginBody
	copy(body.AccountPassword[:], password)
	copy(body.AccountName[:], account)
	copy(body.Zero[:], token)
	body.ClientVersion = protocol.AppVersion
	if v := os.Getenv("W2PP_E2E_VERSION"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			body.ClientVersion = int32(n)
		}
	}
	wire, err := protocol.Encode(protocol.Header{Type: protocol.MsgAccountLogin}, body.Encode(), 7)
	if err != nil {
		panic(err)
	}
	return wire
}

// TestE2ESmokeLogin conduz um login real contra a conta semeada e espera
// a confirmação de seleção de personagem (CNFAccountLogin). Uma lista de personagens vazia
// serve — o ponto é que a autenticação foi e voltou por gRPC+mTLS até o Postgres.
func TestE2ESmokeLogin(t *testing.T) {
	addr := env("W2PP_E2E_ADDR", "127.0.0.1:8281")
	account := env("W2PP_E2E_ACCOUNT", "smoke")
	password := env("W2PP_E2E_PASSWORD", "smoke123")

	// liveLogin retenta enquanto o save da sessão anterior está em andamento, a
	// mesma retentativa que o cliente faz (MsgAlreadyPlaying/StillPlaying).
	c := liveLogin(t, addr, account, password, "")
	defer c.Close()
	t.Log("LOGIN OK — CNFAccountLogin with full character-selection body")
}

// TestE2ESmokeBadPassword confirma que senha errada NÃO é aceita: o servidor
// precisa responder algo diferente de CNFAccountLogin (um painel de aviso).
func TestE2ESmokeBadPassword(t *testing.T) {
	addr := env("W2PP_E2E_ADDR", "127.0.0.1:8281")
	account := env("W2PP_E2E_ACCOUNT", "smoke")

	c := liveDial(t, addr)
	defer c.Close()
	if _, err := c.Write(loginFrame(account, "wrong-pass", "")); err != nil {
		t.Fatalf("write login: %v", err)
	}
	h, _ := liveRead(t, c)
	if h.Type == protocol.MsgCNFAccountLogin {
		t.Fatalf("bad password was ACCEPTED (got CNFAccountLogin) — auth not enforced")
	}
	t.Logf("bad password correctly rejected — response Type=%#x", h.Type)
}
