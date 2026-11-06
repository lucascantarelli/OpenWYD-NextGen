// Package crossover implementa o token de troca de servidor ("crossover") que
// deixa o cliente mudar de canal sem redigitar a senha.
//
// # O fluxo
//
// São três saltos entre o comando do cliente e o login no outro canal:
//
//	tmsrv → db   MSG_DBServerChange { NewServerID, Slot }
//	db    → tm   MSG_DBCNFServerChange { AccountName, Enc }
//	             Enc = "*<srv> <8 inteiros> <slot>", gravado em TempKey[52]
//	cliente      reconecta e manda Enc no campo Zero[52] do MSG_AccountLogin
//	db           TempKey == Zero → login aceito sem senha
//
// # A gramática do Enc não é livre
//
// O legado monta o campo com um único sprintf de dez inteiros:
//
//	sprintf(sm.Enc, "*%d %d %d %d %d %d %d %d %d %d", NewServerID, Enc[0..7], Slot);
//
// São **dez inteiros decimais** separados por um espaço. O cliente 7662 lê o
// primeiro deles com o formato "*%d" (literal presente no WYD.exe) para saber
// o canal de destino, e devolve o campo inteiro no login seguinte. Um único
// caractere fora de dígito/espaço no meio do campo quebra o parser do cliente
// durante o handler de troca — ele para de desenhar o mundo sem derrubar a
// sessão de tela, que é o "travou, perdeu as funções" relatado em teste com o
// cliente gráfico. Por isso o token assinado é EMPACOTADO em inteiros, nunca
// escrito como texto livre.
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
	// WirePrefix é o começo do Enc: o primeiro inteiro vem logo depois dele
	// ("*2 ..."). É esse campo que o cliente usa para escolher o canal de
	// destino; sem ele o servidor aceita o token, mas o cliente não sabe para
	// onde reconectar.
	WirePrefix = "*"

	// EncSize é a largura legada de Enc/TempKey (MSG_DBCNFServerChange).
	// O token precisa caber no mesmo campo Zero[52] que o cliente devolve.
	EncSize = 52

	// wireFields é a quantidade de inteiros do sprintf legado: o canal de
	// destino, os oito inteiros de Enc[0..7] e o slot.
	wireFields = 10

	// TTL é por quanto tempo um token vale depois de emitido. O cliente
	// reconecta em segundos; dois minutos cobrem religação lenta, não replay.
	TTL = 2 * time.Minute

	// macHex é quantos caracteres hex do HMAC viajam no token: 64 bits em
	// quatro inteiros de 16 bits, que é a largura que um %d aceita.
	macHex = 16
)

// Wire são os dez inteiros que viajam no Enc, na ordem do sprintf legado
// ("*%d" seguido de Enc[0..7] e do slot):
//
//	Fields[0]    o "%d" depois do "*": canal de destino
//	Fields[1]    Enc[0]: canal de destino de novo (campo reservado ao db)
//	Fields[2]    Enc[1]: slot do personagem
//	Fields[3]    Enc[2]: expiração, 16 bits altos
//	Fields[4]    Enc[3]: expiração, 16 bits baixos
//	Fields[5..8] Enc[4..7]: HMAC em quatro pedaços de 16 bits
//	Fields[9]    campo final do legado: o slot, repetindo Fields[2]
type Wire struct {
	// Target é o canal pedido, começando em 1 (ServerIndex+1).
	Target int
	// Slot é o slot de personagem que deve logar.
	Slot int
	// Fields são os dez inteiros crus, na ordem em que aparecem no frame.
	Fields [wireFields]int
}

// ParseWire lê o Enc exatamente como o cliente e o dbServer legado o leem:
// um "*" seguido de dez inteiros decimais separados por um único espaço.
// Qualquer outra coisa (texto, campo vazio, contagem diferente) é recusada —
// é essa recusa que impede um campo fora da gramática do cliente de circular.
func ParseWire(enc string) (Wire, bool) {
	if len(enc) == 0 || len(enc) >= EncSize {
		return Wire{}, false
	}
	rest, ok := strings.CutPrefix(enc, WirePrefix)
	if !ok {
		return Wire{}, false
	}
	parts := strings.Split(rest, " ")
	if len(parts) != wireFields {
		return Wire{}, false
	}
	var w Wire
	for i, p := range parts {
		if !allDigits(p) {
			return Wire{}, false
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return Wire{}, false
		}
		w.Fields[i] = n
	}
	w.Target = w.Fields[0]
	w.Slot = w.Fields[wireFields-1]
	if w.Target <= 0 || w.Slot < 0 || w.Slot != w.Fields[2] || w.Target != w.Fields[1] {
		return Wire{}, false
	}
	return w, true
}

// allDigits recusa "", sinal e qualquer caractere não decimal: o sprintf legado
// só produz dígitos, então nada mais pode entrar no campo.
func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// Build assina o token da conta que vai para target (canal começando em 1) no
// slot. Chave vazia devolve "": uma instância sem W2PP_CROSSOVER_KEY nunca pode
// emitir um token que o par aceite.
//
// O resultado está na gramática do legado, por exemplo:
//
//	*2 2 0 27466 3512 51966 34415 32639 65535 0
func Build(key, account string, slot, target int, now time.Time) string {
	if key == "" {
		return ""
	}
	exp := now.Add(TTL).Unix()
	m := mac(key, account, slot, target, exp)

	words := macWords(m)

	var b strings.Builder
	b.WriteString(WirePrefix)
	fmt.Fprintf(&b, "%d", target)
	// Enc[0..7]: canal, slot, expiração em dois pedaços e o HMAC em quatro.
	for _, v := range []int{
		target,
		slot,
		int(exp >> 16),
		int(exp & 0xffff),
		words[0], words[1], words[2], words[3],
	} {
		fmt.Fprintf(&b, " %d", v)
	}
	// Campo final do legado: o slot que o dbServer grava junto do TempKey.
	fmt.Fprintf(&b, " %d", slot)
	return b.String()
}

// IsToken informa se a credencial de login é um dos nossos tokens de crossover.
// Só a gramática decimal do legado conta: uma senha comum nunca começa com "*"
// e nunca tem dez campos numéricos, então não há ambiguidade com senha real.
func IsToken(token string) bool {
	if !strings.HasPrefix(token, WirePrefix) {
		return false
	}
	_, ok := ParseWire(token)
	return ok
}

// Verify confere o token contra a conta a que ele diz pertencer. Devolve false
// para token fora da gramática, assinatura errada, expirado ou conta divergente
// — nunca um erro que o chamador precise distinguir.
func Verify(key, account, token string, now time.Time) (Claims, bool) {
	// Chave vazia não é chave: sem isso um dbServer não configurado aceitaria um
	// token que qualquer um forja com a mesma chave vazia.
	if key == "" {
		return Claims{}, false
	}
	w, ok := ParseWire(token)
	if !ok {
		return Claims{}, false
	}
	exp := int64(w.Fields[3])<<16 | int64(w.Fields[4])
	if exp <= 0 {
		return Claims{}, false
	}
	got := fmt.Sprintf("%04x%04x%04x%04x", w.Fields[5], w.Fields[6], w.Fields[7], w.Fields[8])
	if !hmac.Equal([]byte(got), []byte(mac(key, account, w.Slot, w.Target, exp))) {
		return Claims{}, false
	}
	if now.Unix() > exp {
		return Claims{}, false
	}
	return Claims{Target: w.Target, Slot: w.Slot, Expires: exp}, true
}

// Claims é o que um token válido afirma.
type Claims struct {
	// Target é o canal pedido pelo cliente, começando em 1 (ServerIndex+1).
	Target int
	// Slot é o slot de personagem que deve logar.
	Slot int
	// Expires é o segundo Unix a partir do qual o token é recusado.
	Expires int64
}

// mac é o HMAC hex truncado sobre todo campo que o token afirma, incluindo o
// nome da conta, então outra conta não reaproveita o token.
func mac(key, account string, slot, target int, exp int64) string {
	h := hmac.New(sha256.New, []byte(key))
	fmt.Fprintf(h, "%s\x00%d\x00%d\x00%d", strings.ToLower(account), slot, target, exp)
	return hex.EncodeToString(h.Sum(nil))[:macHex]
}

// macWords quebra os 16 caracteres hex do HMAC em quatro inteiros de 16 bits,
// que é a única forma de levar os 64 bits em campos %d.
func macWords(m string) [4]int {
	var out [4]int
	for i := range out {
		v, err := strconv.ParseUint(m[i*4:i*4+4], 16, 16)
		if err != nil {
			return out
		}
		out[i] = int(v)
	}
	return out
}
