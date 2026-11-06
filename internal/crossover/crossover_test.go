package crossover

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

// legacyEnc é o formato exato do sprintf de CFileDB.cpp:2009: "*" seguido de
// dez inteiros decimais separados por um espaço. O cliente 7662 lê o primeiro
// com "*%d" (literal no WYD.exe), então nada fora disso pode sair do servidor.
var legacyEnc = regexp.MustCompile(`^\*\d+( \d+){9}$`)

// TestBuildUsesLegacyEncGrammar prova que o frame emitido está na gramática que
// o cliente consome, campo a campo — não só "chegou algo".
func TestBuildUsesLegacyEncGrammar(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	tok := Build("key", "test", 2, 1, now)

	if !legacyEnc.MatchString(tok) {
		t.Fatalf("token %q não casa com o sprintf legado %q", tok, legacyEnc.String())
	}
	if len(tok) >= EncSize {
		t.Fatalf("token %q tem %d bytes, precisa caber em Enc[%d] com o NUL", tok, len(tok), EncSize)
	}
	w, ok := ParseWire(tok)
	if !ok {
		t.Fatalf("ParseWire recusou o token recém-emitido %q", tok)
	}
	if w.Target != 1 {
		t.Errorf("primeiro campo = %d, want 1 (o canal de destino)", w.Fields[0])
	}
	if w.Slot != 2 || w.Fields[2] != 2 || w.Fields[9] != 2 {
		t.Errorf("slot = %d/%d/%d, want 2 nos três campos", w.Slot, w.Fields[2], w.Fields[9])
	}
	if w.Fields[1] != 1 {
		t.Errorf("Enc[0] = %d, want 1 (canal de destino, repetido no campo reservado)", w.Fields[1])
	}
}

// TestBuildVerifyRoundTrip trava o contrato: um token assinado para uma conta e
// um canal só verifica para eles, e o Enc volta pelo mesmo parser do cliente.
func TestBuildVerifyRoundTrip(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	tok := Build("key", "Test", 2, 1, now)

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
	if want := now.Add(TTL).Unix(); claims.Expires != want {
		t.Errorf("expires = %d, want %d", claims.Expires, want)
	}
}

// TestTokenFitsWidestChannel confere o pior caso do campo: canal de dois
// dígitos, slot e expiração no máximo, para o Enc nunca estourar o Zero[52].
func TestTokenFitsWidestChannel(t *testing.T) {
	now := time.Unix(4_102_444_800, 0) // 2100: expiração de dez dígitos
	tok := Build("key", "aaaaaaaaaaaaaaaa", 3, 10, now)
	if len(tok) >= EncSize {
		t.Fatalf("worst-case token %q has %d bytes, must fit Enc[%d]", tok, len(tok), EncSize)
	}
	if _, ok := ParseWire(tok); !ok {
		t.Fatalf("ParseWire rejected the widest token %q", tok)
	}
}

// TestParseWireRejectsForeignLayout fixa a regressão: o formato anterior
// ("*2 ~x1:2:0:<exp>:<hmac>") está fora da gramática do cliente e precisa ser
// recusado, tanto na entrada quanto como token.
func TestParseWireRejectsForeignLayout(t *testing.T) {
	for _, tok := range []string{
		"*2 ~x1:2:0:1800000000:deadbeefdeadbeef",
		"~x1:2:0:1800000000:deadbeefdeadbeef",
		"*2 2 0 1 2 3",
		"*2 2 0 1 2 3 4 5 6 7 8",
		"*2  2 0 1 2 3 4 5 6 7",
		"*2 2 0 1 2 3 4 5 6 -1",
		"*x 2 0 1 2 3 4 5 6 7",
		"",
	} {
		if _, ok := ParseWire(tok); ok {
			t.Errorf("ParseWire accepted %q, which is outside the client grammar", tok)
		}
		if IsToken(tok) {
			t.Errorf("IsToken accepted %q, which is outside the client grammar", tok)
		}
	}
}

// TestVerifyRejects cobre todo jeito de um token falhar: outra conta, outra
// chave, campo adulterado, token expirado e lixo.
func TestVerifyRejects(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	good := Build("key", "test", 0, 2, now)

	fields := strings.Split(strings.TrimPrefix(good, "*"), " ")
	withField := func(i int, v string) string {
		cp := append([]string(nil), fields...)
		cp[i] = v
		return "*" + strings.Join(cp, " ")
	}

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
		{"tampered target", "key", "test", withField(0, "1"), now},
		{"tampered slot", "key", "test", withField(2, "1"), now},
		{"slot disagrees with the last field", "key", "test", withField(9, "3"), now},
		{"tampered mac", "key", "test", withField(5, "0"), now},
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
	if tok != "" {
		t.Fatalf("Build with an empty key returned %q, want empty", tok)
	}
	if _, ok := Verify("", "test", "*2 2 0 1 2 3 4 5 6 7", now); ok {
		t.Fatal("an empty key verified a token; the path must stay closed")
	}
}

// TestBuildProducesTenDecimalFields is the byte-level check the client's parser
// depends on: ten fields, all decimal, first one the channel.
func TestBuildProducesTenDecimalFields(t *testing.T) {
	tok := Build("key", "test", 0, 2, time.Unix(1_800_000_000, 0))
	parts := strings.Split(strings.TrimPrefix(tok, "*"), " ")
	if len(parts) != wireFields {
		t.Fatalf("token %q has %d fields, want %d", tok, len(parts), wireFields)
	}
	for i, p := range parts {
		if _, err := strconv.Atoi(p); err != nil {
			t.Fatalf("field %d of %q is not decimal: %v", i, tok, err)
		}
	}
}
