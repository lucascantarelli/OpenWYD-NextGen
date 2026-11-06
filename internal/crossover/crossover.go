// Package crossover implementa o token de troca de servidor ("crossover") que
// deixa o cliente mudar de canal sem redigitar a senha.
//
// # O fluxo
//
// São três saltos entre o comando do cliente e o login no outro canal:
//
//	tmsrv → db   MSG_DBServerChange { NewServerID, Slot }
//	db    → tm   MSG_DBCNFServerChange { AccountName, Enc }
//	             Enc = "*<srv> <Enc[0..7]> <slot>", gravado em TempKey[52]
//	cliente      reconecta e manda Enc no campo Zero[52] do MSG_AccountLogin
//	db           TempKey == Zero → login aceito sem senha
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
	// Prefix marca o campo de senha do login como token de crossover em vez de
	// senha real. Uma senha nunca começa com ele: o login normaliza o nome para
	// minúsculas e o argon2id compara o valor cru que enviamos.
	Prefix = "~x1:"

	// WirePrefix é o começo do Enc que o cliente legado entende: o primeiro
	// campo é o número do canal de destino ("*2 ..."). O corpo assinado vem
	// depois. Sem esse campo o servidor aceita o token, mas o cliente não sabe
	// para onde reconectar e o /srv parece não fazer nada.
	WirePrefix = "*"

	// EncSize é a largura legada de Enc/TempKey (MSG_DBCNFServerChange).
	// O token precisa caber no mesmo campo Zero[52] que o cliente devolve.
	EncSize = 52

	// TTL é por quanto tempo um token vale depois de emitido. O cliente
	// reconecta em segundos; dois minutos cobrem religação lenta, não replay.
	TTL = 2 * time.Minute

	// macLen é quantos caracteres hex do HMAC viajam no token: 64 bits cabem no
	// campo de 52 bytes e são longos o bastante para que forjar um não seja um
	// bypass de senha.
	macLen = 16
)

// Claims é o que um token válido afirma.
type Claims struct {
	// Target é o canal pedido pelo cliente, começando em 1 (ServerIndex+1).
	Target int
	// Slot é o slot de personagem que deve logar.
	Slot int
	// Expires é o segundo Unix a partir do qual o token é recusado.
	Expires int64
}

// Build assina o token da conta que vai para target (canal começando em 1) no
// slot. Chave vazia devolve "": uma instância sem W2PP_CROSSOVER_KEY nunca pode
// emitir um token que o par aceite.
func Build(key, account string, slot, target int, now time.Time) string {
	if key == "" {
		return ""
	}
	exp := now.Add(TTL).Unix()
	signed := fmt.Sprintf("%s%d:%d:%d:%s", Prefix, target, slot, exp, mac(key, account, slot, target, exp))
	return fmt.Sprintf("%s%d %s", WirePrefix, target, signed)
}

// IsToken reports whether a login credential is one of our crossover tokens.
// It accepts the legacy-compatible wire form ("*<canal> ~x1:...") and the
// pre-fix form ("~x1:...") while old tokens are still within their TTL.
func IsToken(token string) bool {
	if strings.HasPrefix(token, Prefix) {
		return true
	}
	if !strings.HasPrefix(token, WirePrefix) {
		return false
	}
	rest := strings.TrimPrefix(token, WirePrefix)
	sp := strings.IndexByte(rest, ' ')
	if sp <= 0 {
		return false
	}
	if _, err := strconv.Atoi(rest[:sp]); err != nil {
		return false
	}
	return strings.HasPrefix(rest[sp+1:], Prefix)
}

// Verify confere o token contra a conta a que ele diz pertencer. Devolve false
// para token malformado, assinatura errada, expirado ou conta divergente — nunca
// um erro que o chamador precise distinguir.
func Verify(key, account, token string, now time.Time) (Claims, bool) {
	// Chave vazia não é chave: sem isso um dbServer não configurado aceitaria um
	// token que qualquer um forja com a mesma chave vazia.
	if key == "" {
		return Claims{}, false
	}
	// O cliente devolve o Enc inteiro ("*<canal> ~x1:..."); o primeiro campo
	// tem de concordar com o alvo assinado, senão um token para o canal 2
	// poderia ser reapresentado como se fosse para o canal 1.
	body := token
	wireTarget := 0
	if strings.HasPrefix(token, WirePrefix) {
		rest := strings.TrimPrefix(token, WirePrefix)
		sp := strings.IndexByte(rest, ' ')
		if sp <= 0 {
			return Claims{}, false
		}
		n, err := strconv.Atoi(rest[:sp])
		if err != nil || n <= 0 {
			return Claims{}, false
		}
		wireTarget = n
		body = rest[sp+1:]
	}
	signed, ok := strings.CutPrefix(body, Prefix)
	if !ok {
		return Claims{}, false
	}
	parts := strings.Split(signed, ":")
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
	if wireTarget != 0 && target != wireTarget {
		return Claims{}, false
	}
	return Claims{Target: target, Slot: slot, Expires: exp}, true
}

// mac é o HMAC hex truncado sobre todo campo que o token afirma, incluindo o
// nome da conta, então outra conta não reaproveita o token.
func mac(key, account string, slot, target int, exp int64) string {
	h := hmac.New(sha256.New, []byte(key))
	fmt.Fprintf(h, "%s\x00%d\x00%d\x00%d", strings.ToLower(account), slot, target, exp)
	return hex.EncodeToString(h.Sum(nil))[:macLen]
}
