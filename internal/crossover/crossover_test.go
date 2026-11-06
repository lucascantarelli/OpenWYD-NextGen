package crossover

import (
	"strings"
	"testing"
	"time"
)

// TestBuildVerifyRoundTrip locks the token contract: a token signed for one
// account/channel verifies only for that account, and the 52-byte budget the
// client's Zero[52] field imposes is respected.
func TestBuildVerifyRoundTrip(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	tok := Build("key", "Test", 2, 1, now)

	if len(tok) > EncSize {
		t.Fatalf("token %q has %d bytes, must fit Enc[%d]", tok, len(tok), EncSize)
	}
	if !strings.HasPrefix(tok, Prefix) {
		t.Fatalf("token %q does not carry the %q prefix", tok, Prefix)
	}
	claims, ok := Verify("key", "test", tok, now)
	if !ok {
		t.Fatalf("Verify rejected a freshly built token %q", tok)
	}
	if claims.Target != 1 || claims.Slot != 2 {
		t.Errorf("claims = %+v, want target 1 slot 2", claims)
	}
}

// TestVerifyRejects covers every way a token must fail: another account, another
// key, a tampered field, an expired one and garbage.
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
		{"no prefix", "key", "test", strings.TrimPrefix(good, Prefix), now},
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

// TestRealPasswordIsNotAToken makes sure a normal password can never be read as a
// token, and that an empty key disables the path instead of accepting anything.
func TestRealPasswordIsNotAToken(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	if _, ok := Verify("key", "test", "test123", now); ok {
		t.Fatal("a plain password verified as a crossover token")
	}
	tok := Build("", "test", 0, 2, now)
	if _, ok := Verify("", "test", tok, now); ok {
		t.Fatal("an empty key verified a token; the path must stay closed")
	}
}
