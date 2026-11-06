// Package crossover implements the server-change token ("crossover") that lets a
// client move between channels without retyping the password.
//
// # O que o legado faz
//
// O cliente troca de canal por um comando (Source/Code/TMSrv/_MSG_MessageWhisper.cpp:75-95),
// e são três saltos (docs/migration/canais.md §8.1.2):
//
//	tmsrv → db   MSG_DBServerChange { NewServerID, Slot }
//	db    → tm   MSG_DBCNFServerChange { AccountName, Enc }
//	             Enc = "*<srv> <Enc[0..7]> <slot>", gravado em TempKey[52]
//	cliente      reconecta e manda Enc no campo Zero[52] do MSG_AccountLogin
//	db           TempKey == Zero → login aceito sem senha (CFileDB.cpp:662-812)
//
// # Por que não copiamos o TempKey
//
// Guardar o token no save custaria uma coluna nova (o formato STRUCT_ACCOUNTFILE
// já tem TempKey em 7812, mas só o caminho de arquivo o usa) e um RPC novo em
// api/db/v1/db.proto para o tmServer pedir ao dbServer. Em vez disso o token é
// ASSINADO: os dois canais e o dbServer compartilham W2PP_CROSSOVER_KEY, então
// quem emite e quem valida chegam ao mesmo resultado sem estado compartilhado.
package crossover

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	// Prefix marks the login password field as a crossover token instead of a
	// real password. A password can never start with it: the login normalizes
	// the name to lowercase and argon2id compares the raw value we sent.
	Prefix = "~x1:"

	// EncSize is the legacy Enc/TempKey width (Basedef.h MSG_DBCNFServerChange).
	// The token must fit the same Zero[52] field the client sends back.
	EncSize = 52

	// TTL is how long a token stays valid after being issued. The client
	// reconnects in seconds; two minutes covers a slow relink, not a replay.
	TTL = 2 * time.Minute

	// macLen is how many hex characters of the HMAC travel in the token: 64 bits
	// is short enough for the 52-byte field and long enough that forging one is
	// not a password bypass.
	macLen = 16
)

// Claims is what a valid token asserts.
type Claims struct {
	// Target is the 1-based channel the client asked for (ServerIndex+1).
	Target int
	// Slot is the character slot that must log in.
	Slot int
	// Expires is the Unix second after which the token is refused.
	Expires int64
}

// Build signs a token for account moving to target (1-based channel) with slot.
// An empty key returns "": an instance without W2PP_CROSSOVER_KEY must never
// mint a token that a peer could accept.
func Build(key, account string, slot, target int, now time.Time) string {
	if key == "" {
		return ""
	}
	exp := now.Add(TTL).Unix()
	return fmt.Sprintf("%s%d:%d:%d:%s", Prefix, target, slot, exp, mac(key, account, slot, target, exp))
}

// Verify checks the token against the account it claims to belong to. It returns
// false for a malformed token, a bad signature, an expired one, or an account
// mismatch — never an error the caller has to distinguish.
func Verify(key, account, token string, now time.Time) (Claims, bool) {
	// An empty key is not a key: without this an unconfigured dbServer would
	// accept a token anyone can forge with the same empty key.
	if key == "" {
		return Claims{}, false
	}
	body, ok := strings.CutPrefix(token, Prefix)
	if !ok {
		return Claims{}, false
	}
	parts := strings.Split(body, ":")
	if len(parts) != 4 {
		return Claims{}, false
	}
	target, err1 := strconv.Atoi(parts[0])
	slot, err2 := strconv.Atoi(parts[1])
	exp, err3 := strconv.ParseInt(parts[2], 10, 64)
	if err1 != nil || err2 != nil || err3 != nil {
		return Claims{}, false
	}
	if len(parts[3]) != macLen {
		return Claims{}, false
	}
	if !hmac.Equal([]byte(parts[3]), []byte(mac(key, account, slot, target, exp))) {
		return Claims{}, false
	}
	if now.Unix() > exp {
		return Claims{}, false
	}
	return Claims{Target: target, Slot: slot, Expires: exp}, true
}

// mac is the truncated hex HMAC over every field the token asserts, including the
// account name, so a token cannot be replayed by another account.
func mac(key, account string, slot, target int, exp int64) string {
	h := hmac.New(sha256.New, []byte(key))
	fmt.Fprintf(h, "%s\x00%d\x00%d\x00%d", strings.ToLower(account), slot, target, exp)
	return hex.EncodeToString(h.Sum(nil))[:macLen]
}
