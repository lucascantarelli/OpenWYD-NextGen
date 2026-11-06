package crossover

import (
	"strings"
	"testing"
	"time"
)

// TestBuildVerifyRoundTrip trava o contrato do token: um token assinado para uma
// conta/canal só verifica para aquela conta, começa com o campo legado
// "*<canal> " que o cliente usa para escolher o servidor, e cabe no Zero[52].
func TestBuildVerifyRoundTrip(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	tok := Build("key", "Test", 2, 1, now)

	if len(tok) > EncSize {
		t.Fatalf("token %q has %d bytes, must fit Enc[%d]", tok, len(tok), EncSize)
	}
	if !strings.HasPrefix(tok, "*1 ") {
		t.Fatalf("token %q does not start with the legacy wire prefix %q", tok, "*1 ")
	}
	if !IsToken(tok) {
		t.Fatalf("IsToken rejected a freshly built token %q", tok)
	}
	claims, ok := Verify("key", "test", tok, now)
	if !ok {
		t.Fatalf("Verify rejected a freshly built token %q", tok)
	}
	if claims.Target != 1 || claims.Slot != 2 {
		t.Errorf("claims = %+v, want target 1 slot 2", claims)
	}
}

// TestVerifyRejects cobre todo jeito de um token falhar: outra conta, outra
// chave, campo adulterado, token expirado, alvo do Enc divergente e lixo.
func TestVerifyRejects(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	good := Build("key", "test", 0, 2, now)

	cases := []struct {
		name    string
		key     string
		account string
		token   string
		now     time.Time
	}{
		{"other account", "key", "outra", good, now},
		{"other key", "outra-chave", "test", good, now},
		{"expired", "key", "test", good, now.Add(TTL + time.Second)},
		{"tampered target", "key", "test", strings.Replace(good, ":2:", ":3:", 1), now},
		{"wire target disagrees", "key", "test", strings.Replace(good, "*2 ", "*1 ", 1), now},
		{"wire without space", "key", "test", strings.Replace(good, "*2 ", "*2", 1), now},
		{"malformed wire target", "key", "test", "*x ~x1:2:0:1:deadbeefdeadbeef", now},
		{"garbage", "key", "test", "~x1:1:2", now},
		{"empty", "key", "test", "", now},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, ok := Verify(c.key, c.account, c.token, c.now); ok {
				t.Fatalf("Verify accepted %q (key %q, account %q)", c.token, c.key, c.account)
			}
		})
	}
}

// TestRealPasswordIsNotAToken garante que uma senha normal nunca é lida como
// token, e que chave vazia desliga o caminho em vez de aceitar qualquer coisa.
func TestRealPasswordIsNotAToken(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	if IsToken("test123") {
		t.Fatal("a plain password was classified as a crossover token")
	}
	if _, ok := Verify("key", "test", "test123", now); ok {
		t.Fatal("a plain password verified as a crossover token")
	}
	tok := Build("", "test", 0, 2, now)
	if _, ok := Verify("", "test", tok, now); ok {
		t.Fatal("an empty key verified a token; the path must stay closed")
	}
}
