// Package secret calcula e confere hash de segredos de conta (senhas, PINs,
// senhas de bloqueio) com argon2id. Credencial em texto plano do legado NUNCA
// entra na pilha nova: este é o único ponto de hashing, compartilhado pelo
// importador do dbServer e pela web-api.
package secret

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"

	"golang.org/x/crypto/argon2"
)

// argonParams são os parâmetros de custo do argon2id usados em todo segredo
// importado.
type argonParams struct {
	memory  uint32
	time    uint32
	threads uint8
	keyLen  uint32
	saltLen uint32
}

// defaultArgon segue a orientação do OWASP para argon2id (64 MiB, t=1, p=4).
var defaultArgon = argonParams{memory: 64 * 1024, time: 1, threads: 4, keyLen: 32, saltLen: 16}

// HashSecret devolve o hash PHC do argon2id para plain, ou "" quando plain é
// vazio (segredo não definido — por exemplo, sem senha de bloqueio). O sal é
// aleatório por chamada, então a mesma entrada gera hashes diferentes.
func HashSecret(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	salt := make([]byte, defaultArgon.saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("secret: generate salt: %w", err)
	}
	h := argon2.IDKey([]byte(plain), salt, defaultArgon.time, defaultArgon.memory, defaultArgon.threads, defaultArgon.keyLen)
	b64 := base64.RawStdEncoding
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, defaultArgon.memory, defaultArgon.time, defaultArgon.threads,
		b64.EncodeToString(salt), b64.EncodeToString(h)), nil
}

// ErrBadHash é devolvido por VerifySecret quando phc não é um hash produzido por
// este pacote (esquema errado, versão errada ou codificação malformada).
var ErrBadHash = errors.New("secret: malformed argon2id hash")

// VerifySecret diz se plain casa com o hash PHC do argon2id produzido por
// HashSecret. A comparação é de tempo constante. Hash guardado vazio significa
// segredo não definido na importação, então ele só casa com plain vazio. O texto
// plano nunca é reconstruído: o hash é derivado de novo a partir de plain e os
// resumos são comparados.
func VerifySecret(plain, phc string) (bool, error) {
	if phc == "" {
		return plain == "", nil
	}
	var (
		version          int
		memory, time     uint32
		threads          uint8
		saltB64, hashB64 string
	)
	// Casa com o formato exato escrito por HashSecret.
	if _, err := fmt.Sscanf(phc, "$argon2id$v=%d$m=%d,t=%d,p=%d$%s",
		&version, &memory, &time, &threads, &saltB64); err != nil {
		return false, fmt.Errorf("%w: %v", ErrBadHash, err)
	}
	if version != argon2.Version {
		return false, fmt.Errorf("%w: version %d", ErrBadHash, version)
	}
	// O %s final do Sscanf capturou "salt$hash"; separa no cifrão.
	sep := -1
	for i := 0; i < len(saltB64); i++ {
		if saltB64[i] == '$' {
			sep = i
			break
		}
	}
	if sep < 0 {
		return false, fmt.Errorf("%w: missing hash segment", ErrBadHash)
	}
	saltB64, hashB64 = saltB64[:sep], saltB64[sep+1:]

	b64 := base64.RawStdEncoding
	salt, err := b64.DecodeString(saltB64)
	if err != nil {
		return false, fmt.Errorf("%w: salt: %v", ErrBadHash, err)
	}
	want, err := b64.DecodeString(hashB64)
	if err != nil {
		return false, fmt.Errorf("%w: hash: %v", ErrBadHash, err)
	}

	got := argon2.IDKey([]byte(plain), salt, time, memory, threads, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}
