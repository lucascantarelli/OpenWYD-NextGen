// Package playcode emite e confere os códigos de login de uso único que deixam
// um jogador já autenticado no portal entrar no cliente web sem digitar a senha
// de novo.
//
// O código viaja no campo de senha legado do MSG_AccountLogin (0x020D), então o
// formato de wire e o cliente 7662 ficam intactos. Esse campo tem 12 bytes e o
// cliente o copia com sprintf_s, que exige o terminador — daí o comprimento de
// 10 caracteres.
//
// Os códigos são guardados em SHA-256, não argon2id: argon2id protege senha
// humana de baixa entropia contra adivinhação offline, enquanto um código tem 50
// bits aleatórios, é de uso único e vive dois minutos, então um hash rápido não
// perde nada e permite busca direta no store.
package playcode

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// Length é o comprimento do código; precisa ficar abaixo dos 12 bytes do
	// campo de senha.
	Length = 10
	// Alphabet tem 32 símbolos sem parecidos (sem 0/o, 1/l), então cada byte
	// aleatório vira um símbolo sem viés de módulo.
	Alphabet = "abcdefghijkmnpqrstuvwxyz23456789"
	// TTL é por quanto tempo um código emitido continua válido.
	TTL = 2 * time.Minute

	// AssertionLabel separa por domínio a assinatura da requisição do portal do
	// ticket de jogo do cliente web, que usa o mesmo formato de token.
	AssertionLabel = "wyd-play-code.v1."
	// MinSecretLength é o tamanho mínimo de PLAY_CODE_SECRET.
	MinSecretLength = 32

	maxAssertionBytes = 1024
	maxAssertionLife  = 2 * time.Minute
	clockSkew         = 30 * time.Second
	maxSeen           = 4096
)

// New devolve um código aleatório novo.
func New() (string, error) {
	b := make([]byte, Length)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("playcode: random: %w", err)
	}
	for i := range b {
		b[i] = Alphabet[int(b[i])%len(Alphabet)]
	}
	return string(b), nil
}

// Hash é a forma guardada de um código.
func Hash(code string) []byte {
	h := sha256.Sum256([]byte(code))
	return h[:]
}

// Valid diz se s tem a forma de um código, para que uma senha que não pode ser
// um código nunca custe uma ida ao banco.
func Valid(s string) bool {
	if len(s) != Length {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !strings.ContainsRune(Alphabet, rune(s[i])) {
			return false
		}
	}
	return true
}

// Claims é a requisição assinada do portal: "emita um código para a conta Sub".
// O id da conta é o accountId da sessão do portal (string decimal).
type Claims struct {
	Sub string `json:"sub"`
	Iat int64  `json:"iat"`
	Exp int64  `json:"exp"`
	Jti string `json:"jti"`
}

// Sign codifica claims como base64url(JSON) "." base64url(HMAC-SHA256(segredo,
// AssertionLabel + payload)), o formato do ticket de jogo do cliente web.
func Sign(secret []byte, c Claims) string {
	b, err := json.Marshal(c)
	if err != nil {
		panic(err) // Claims só tem strings e inteiros
	}
	payload := base64.RawURLEncoding.EncodeToString(b)
	return payload + "." + base64.RawURLEncoding.EncodeToString(mac(secret, payload))
}

func mac(secret []byte, payload string) []byte {
	m := hmac.New(sha256.New, secret)
	m.Write([]byte(AssertionLabel + payload))
	return m.Sum(nil)
}

// Verifier confere asserções do portal e recusa um jti já visto. É seguro para
// uso concorrente; vive no webserver, fora do loop do jogo.
type Verifier struct {
	secret []byte
	now    func() time.Time

	mu   sync.Mutex
	seen map[string]time.Time // jti -> validade
}

// NewVerifier devolve um Verifier, ou nil quando o segredo é curto demais (a
// funcionalidade então fica desligada).
func NewVerifier(secret string) *Verifier {
	if len(secret) < MinSecretLength {
		return nil
	}
	return &Verifier{secret: []byte(secret), now: time.Now, seen: map[string]time.Time{}}
}

// Verify valida uma asserção, consome o jti e devolve o id da conta.
// O texto do erro pode ir para o log; o token nunca.
func (v *Verifier) Verify(token string) (int64, error) {
	if token == "" || len(token) > maxAssertionBytes {
		return 0, errors.New("missing or oversized assertion")
	}
	payload, sig, ok := strings.Cut(token, ".")
	if !ok || strings.Contains(sig, ".") {
		return 0, errors.New("malformed assertion")
	}
	got, err := base64.RawURLEncoding.DecodeString(sig)
	if err != nil || !hmac.Equal(got, mac(v.secret, payload)) {
		return 0, errors.New("bad signature")
	}
	raw, err := base64.RawURLEncoding.DecodeString(payload)
	var c Claims
	if err != nil || json.Unmarshal(raw, &c) != nil {
		return 0, errors.New("malformed payload")
	}
	id, err := strconv.ParseInt(c.Sub, 10, 64)
	now := v.now()
	switch {
	case err != nil || id <= 0:
		return 0, errors.New("bad subject")
	case len(c.Jti) < 16 || len(c.Jti) > 64:
		return 0, errors.New("bad jti")
	case c.Exp <= c.Iat || time.Duration(c.Exp-c.Iat)*time.Second > maxAssertionLife:
		return 0, errors.New("bad lifetime")
	case now.Add(clockSkew).Unix() < c.Iat:
		return 0, errors.New("issued in the future")
	case now.Add(-clockSkew).Unix() >= c.Exp:
		return 0, errors.New("expired")
	}

	v.mu.Lock()
	defer v.mu.Unlock()
	if _, used := v.seen[c.Jti]; used {
		return 0, errors.New("replayed")
	}
	if len(v.seen) >= maxSeen {
		for jti, exp := range v.seen {
			if now.Add(-clockSkew).After(exp) {
				delete(v.seen, jti)
			}
		}
		if len(v.seen) >= maxSeen {
			return 0, errors.New("replay cache full")
		}
	}
	v.seen[c.Jti] = time.Unix(c.Exp, 0)
	return id, nil
}
