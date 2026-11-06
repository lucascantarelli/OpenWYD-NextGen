package metrics

import (
	"context"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestExpositionCounterGaugeHistogram(t *testing.T) {
	r := New("test")
	r.Inc("requests_total", "Total de requests.", Labels{"service": "test", "code": "OK"})
	r.Add("requests_total", "Total de requests.", 2, Labels{"service": "test", "code": "OK"})
	r.Set("connections_active", "Conexões ativas.", 7, Labels{"service": "test"})
	r.Observe("tick_seconds", "Duração do tick.", 0.003, Labels{"service": "test"})
	r.GaugeFunc("pool_acquired", "Conexões do pool.", Labels{"service": "test"}, func() float64 { return 3 })

	var b strings.Builder
	if err := r.Write(&b); err != nil {
		t.Fatalf("Write: %v", err)
	}
	out := b.String()
	for _, want := range []string{
		"# TYPE up gauge",
		`up{service="test"} 1`,
		"# TYPE requests_total counter",
		`requests_total{code="OK",service="test"} 3`,
		"# TYPE tick_seconds histogram",
		`tick_seconds_bucket{le="0.005",service="test"} 1`,
		`tick_seconds_bucket{le="+Inf",service="test"} 1`,
		`tick_seconds_count{service="test"} 1`,
		"# TYPE pool_acquired gauge",
		`pool_acquired{service="test"} 3`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("exposição sem %q:\n%s", want, out)
		}
	}
}

func TestHandlerContentTypeAndEscape(t *testing.T) {
	r := New("test")
	r.Set("weird", "Label com aspas.", 1, Labels{"service": "test", "label": "a\"b\nc\\d"})
	rec := httptest.NewRecorder()
	r.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/metrics", nil))
	if rec.Code != 200 {
		t.Fatalf("status = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/plain; version=0.0.4") {
		t.Fatalf("content-type = %q", ct)
	}
	if body := rec.Body.String(); !strings.Contains(body, `label="a\"b\nc\\d"`) {
		t.Fatalf("label não escapado:\n%s", body)
	}
}

func TestConcurrentAdds(t *testing.T) {
	r := New("test")
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.Inc("events_total", "Eventos.", Labels{"service": "test"})
		}()
	}
	wg.Wait()
	var b strings.Builder
	if err := r.Write(&b); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if !strings.Contains(b.String(), `events_total{service="test"} 100`) {
		t.Fatalf("contador concorrente perdido:\n%s", b.String())
	}
}

func TestUnaryInterceptorRecordsMethodAndCode(t *testing.T) {
	r := New("test")
	interceptor := UnaryServerInterceptor("test", r)
	info := &grpc.UnaryServerInfo{FullMethod: "/db.v1.AccountService/AccountLogin"}

	if _, err := interceptor(context.Background(), nil, info, func(context.Context, any) (any, error) {
		return nil, status.Error(codes.Internal, "boom")
	}); err == nil {
		t.Fatal("handler error não propagado")
	}
	if _, err := interceptor(context.Background(), nil, info, func(context.Context, any) (any, error) {
		return "ok", nil
	}); err != nil {
		t.Fatalf("handler OK devolveu erro: %v", err)
	}

	var b strings.Builder
	if err := r.Write(&b); err != nil {
		t.Fatalf("Write: %v", err)
	}
	out := b.String()
	for _, want := range []string{
		`grpc_requests_total{code="OK",method="/db.v1.AccountService/AccountLogin",service="test"} 1`,
		`grpc_requests_total{code="Internal",method="/db.v1.AccountService/AccountLogin",service="test"} 1`,
		`grpc_request_duration_seconds_count{method="/db.v1.AccountService/AccountLogin",service="test"} 2`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("sem %q:\n%s", want, out)
		}
	}
}

func TestServeDisabledWithoutAddr(t *testing.T) {
	// Serve não deve abrir porta nem bloquear quando addr é vazio.
	Serve(context.Background(), "", New("test"), testLogger(t))
}

func testLogger(t *testing.T) *slog.Logger {
	t.Helper()
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
