// Trace_id ponta a ponta (OBS-02): um id de correlação por requisição, gerado
// no aceite da conexão CPSock e no BFF, propagado por metadata gRPC
// (`x-trace-id`) e anexado às linhas de log de cada processo.
//
// O id NUNCA é label de métrica: cardenalidade por requisição estoura a série
// temporal. Ele pertence ao contexto e ao log; a série mostra método/código.
package metrics

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// TraceHeader é a chave de metadata que carrega o trace_id. Metadata gRPC é
// case-insensitive e normalizada para minúsculas; o cabeçalho HTTP equivalente
// é `X-Trace-Id`.
const TraceHeader = "x-trace-id"

// traceIDLen é o tamanho do id: 16 bytes em hexadecimal, o mesmo formato de um
// trace id W3C (128 bits).
const traceIDLen = 32

type traceKey struct{}

// NewTraceID gera um id novo (16 bytes aleatórios em hex minúsculo). O gerador
// criptográfico do Go não falha em uso normal; se falhar, o buffer fica zerado,
// que ainda é um id sintaticamente válido — nunca vazio.
func NewTraceID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// ValidTraceID reporta se id tem exatamente 32 dígitos hexadecimais. Entrada
// fora desse formato é descartada e um id novo é gerado, para que um cabeçalho
// forjado não injete texto arbitrário no log.
func ValidTraceID(id string) bool {
	if len(id) != traceIDLen {
		return false
	}
	for i := 0; i < len(id); i++ {
		c := id[i]
		switch {
		case c >= '0' && c <= '9', c >= 'a' && c <= 'f', c >= 'A' && c <= 'F':
		default:
			return false
		}
	}
	return true
}

// WithTraceID devolve um contexto com o id normalizado (minúsculo). Um id
// inválido é ignorado: o contexto segue sem correlação em vez de carregar lixo.
func WithTraceID(ctx context.Context, id string) context.Context {
	if !ValidTraceID(id) {
		return ctx
	}
	return context.WithValue(ctx, traceKey{}, strings.ToLower(id))
}

// TraceID devolve o id do contexto ("" quando não há).
func TraceID(ctx context.Context) string {
	id, _ := ctx.Value(traceKey{}).(string)
	return id
}

// EnsureTraceID devolve o contexto com um id válido e o id. Prefere o que já
// está no contexto; senão aceita o metadata de entrada (`x-trace-id`); senão
// gera um novo. Aceitar o metadata é o que liga o tmServer ao dbServer: o
// cliente injeta, o servidor lê.
func EnsureTraceID(ctx context.Context) (context.Context, string) {
	if id := TraceID(ctx); id != "" {
		return ctx, id
	}
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if v := md.Get(TraceHeader); len(v) > 0 && ValidTraceID(v[0]) {
			id := strings.ToLower(v[0])
			return context.WithValue(ctx, traceKey{}, id), id
		}
	}
	id := NewTraceID()
	return context.WithValue(ctx, traceKey{}, id), id
}

// UnaryClientInterceptor propaga o trace_id do contexto no metadata de saída.
// Sem id no contexto ele gera um e envia assim mesmo: toda RPC de saída leva
// uma correlação, então o servidor nunca registra uma linha órfã.
func UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		id := TraceID(ctx)
		if id == "" {
			id = NewTraceID()
		}
		ctx = metadata.AppendToOutgoingContext(ctx, TraceHeader, id)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
