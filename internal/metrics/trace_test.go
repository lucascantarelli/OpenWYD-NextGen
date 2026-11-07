package metrics

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestNewTraceIDFormat(t *testing.T) {
	seen := make(map[string]struct{}, 64)
	for i := 0; i < 64; i++ {
		id := NewTraceID()
		if !ValidTraceID(id) {
			t.Fatalf("NewTraceID() = %q, fora do formato de 32 hex", id)
		}
		if strings.ToLower(id) != id {
			t.Fatalf("NewTraceID() = %q, deveria ser minúsculo", id)
		}
		if _, dup := seen[id]; dup {
			t.Fatalf("NewTraceID() repetiu %q", id)
		}
		seen[id] = struct{}{}
	}
}

func TestValidTraceID(t *testing.T) {
	cases := map[string]bool{
		"":                                   false,
		"0123456789abcdef0123456789abcdef":   true,
		"0123456789ABCDEF0123456789ABCDEF":   true,
		"0123456789abcdef0123456789abcde":    false, // 31
		"0123456789abcdef0123456789abcdef0":  false, // 33
		"0123456789abcdef0123456789abcdeg":   false, // não-hex
		"0123456789abcdef 123456789abcdef":   false, // espaço
		"0123456789abcdef0123456789abcdef\n": false, // newline
	}
	for id, want := range cases {
		if got := ValidTraceID(id); got != want {
			t.Errorf("ValidTraceID(%q) = %v, quer %v", id, got, want)
		}
	}
}

// echoService é um serviço gRPC mínimo registrado à mão: o teste exercita os
// interceptores reais sobre um transporte real (bufconn), não um mock deles.
const echoFullMethod = "/test.Echo/Call"

type echoServer struct{}

func (echoServer) call(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

var echoDesc = grpc.ServiceDesc{
	ServiceName: "test.Echo",
	HandlerType: (*any)(nil),
	Methods: []grpc.MethodDesc{{
		MethodName: "Call",
		Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
			in := new(emptypb.Empty)
			if err := dec(in); err != nil {
				return nil, err
			}
			if interceptor == nil {
				return srv.(echoServer).call(ctx, in)
			}
			info := &grpc.UnaryServerInfo{Server: srv, FullMethod: echoFullMethod}
			handler := func(ctx context.Context, req any) (any, error) {
				return srv.(echoServer).call(ctx, req.(*emptypb.Empty))
			}
			return interceptor(ctx, in, info, handler)
		},
	}},
}

// traceRig sobe servidor e cliente reais (bufconn) com os interceptores de
// trace, e captura as linhas de log do servidor.
type traceRig struct {
	client *grpc.ClientConn
	logs   *bytes.Buffer
	reg    *Registry
}

func newTraceRig(t *testing.T) *traceRig {
	t.Helper()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	reg := New("test")

	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(UnaryServerInterceptor("test", reg, logger)))
	srv.RegisterService(&echoDesc, echoServer{})

	lis := bufconn.Listen(1 << 20)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	cc, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.DialContext(ctx) }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(UnaryClientInterceptor()))
	if err != nil {
		t.Fatalf("grpc.NewClient: %v", err)
	}
	t.Cleanup(func() { _ = cc.Close() })

	return &traceRig{client: cc, logs: &buf, reg: reg}
}

// call executa o RPC e devolve o trace_id que o servidor registrou no header,
// que é o id que ele usou no log.
func (r *traceRig) call(ctx context.Context, t *testing.T) string {
	t.Helper()
	var md metadata.MD
	if err := r.client.Invoke(ctx, echoFullMethod, &emptypb.Empty{}, &emptypb.Empty{}, grpc.Header(&md)); err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	vals := md.Get(TraceHeader)
	if len(vals) == 0 {
		t.Fatalf("servidor não devolveu %s no header (md=%v)", TraceHeader, md)
	}
	return vals[0]
}

func TestServerInterceptorUsesIncomingTraceID(t *testing.T) {
	rig := newTraceRig(t)
	const want = "0123456789abcdef0123456789abcdef"

	got := rig.call(WithTraceID(context.Background(), want), t)
	if got != want {
		t.Fatalf("trace_id no header = %q, quer %q", got, want)
	}
	// A linha do servidor precisa carregar o MESMO id: é isso que faz
	// `docker compose logs | grep <id>` ligar tmServer e dbServer.
	if line := rig.logs.String(); !strings.Contains(line, "trace_id="+want) {
		t.Fatalf("log do servidor sem o trace_id recebido:\n%s", line)
	}
}

func TestServerInterceptorGeneratesWhenMissing(t *testing.T) {
	rig := newTraceRig(t)

	got := rig.call(context.Background(), t)
	if !ValidTraceID(got) {
		t.Fatalf("trace_id gerado = %q, inválido", got)
	}
	if lines := rig.logs.String(); !strings.Contains(lines, "trace_id="+got) {
		t.Fatalf("log do servidor sem o id gerado %q:\n%s", got, lines)
	}
}

func TestServerInterceptorRejectsForgedTraceID(t *testing.T) {
	rig := newTraceRig(t)

	// Metadata de entrada inválida (texto livre) não vira id nem chega ao log.
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs(TraceHeader, "not-a-trace-id"))
	got := rig.call(ctx, t)
	if got == "not-a-trace-id" || !ValidTraceID(got) {
		t.Fatalf("servidor aceitou id forjado: %q", got)
	}
	if strings.Contains(rig.logs.String(), "not-a-trace-id") {
		t.Fatalf("id forjado vazou para o log:\n%s", rig.logs.String())
	}
}

func TestConcurrentCallsDoNotMixTraceIDs(t *testing.T) {
	rig := newTraceRig(t)

	const n = 40
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			want := fmt.Sprintf("%032x", i+1)
			var md metadata.MD
			ctx := WithTraceID(context.Background(), want)
			if err := rig.client.Invoke(ctx, echoFullMethod, &emptypb.Empty{}, &emptypb.Empty{}, grpc.Header(&md)); err != nil {
				errs <- err
				return
			}
			if got := md.Get(TraceHeader); len(got) == 0 || got[0] != want {
				errs <- fmt.Errorf("chamada %d: header %v, quer %s", i, got, want)
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}

	// Todas as linhas existem e cada id aparece, sem id sobrando de outra
	// requisição: o contexto não é compartilhado entre chamadas concorrentes.
	logs := rig.logs.String()
	for i := 0; i < n; i++ {
		want := fmt.Sprintf("trace_id=%032x", i+1)
		if strings.Count(logs, want) != 1 {
			t.Fatalf("log deveria ter exatamente uma linha com %s", want)
		}
	}
}

func TestClientInterceptorGeneratesForOutgoingCall(t *testing.T) {
	rig := newTraceRig(t)

	// Sem id no contexto o cliente ainda manda um, então o servidor registra
	// uma linha correlacionável em vez de uma órfã.
	got := rig.call(context.Background(), t)
	if !ValidTraceID(got) {
		t.Fatalf("id gerado pelo cliente = %q", got)
	}
}

func TestTraceIDIsNeverAMetricLabel(t *testing.T) {
	rig := newTraceRig(t)
	const id = "fedcba9876543210fedcba9876543210"
	rig.call(WithTraceID(context.Background(), id), t)

	var b strings.Builder
	if err := rig.reg.Write(&b); err != nil {
		t.Fatalf("Write: %v", err)
	}
	out := b.String()
	if strings.Contains(out, id) {
		t.Fatalf("trace_id virou label de métrica:\n%s", out)
	}
	if !strings.Contains(out, `grpc_requests_total{code="OK",method="`+echoFullMethod+`",service="test"} 1`) {
		t.Fatalf("contador do interceptor ausente:\n%s", out)
	}
}
