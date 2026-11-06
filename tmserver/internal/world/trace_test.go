package world

import (
	"bytes"
	"context"
	"log/slog"
	"net"
	"strings"
	"testing"
	"time"

	"openwyd-nextgen/internal/metrics"
	"openwyd-nextgen/tmserver/internal/protocol"
)

// TestConnectAssignsTraceID proves the trace_id is born at the CPSock accept
// (OBS-02) and reaches both the session the handlers see and the connect log
// line — the id the login RPC later carries to the dbServer.
func TestConnectAssignsTraceID(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	reg := metrics.New("tmserver")

	ids := make(chan string, 1)
	handler := func(_ *World, s *Session, _ protocol.Header, _ []byte) {
		select {
		case ids <- s.TraceID:
		default:
		}
	}

	w := New(Config{GridDim: 16, Metrics: reg}, logger, NopPersistence{}, handler)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { _ = w.Serve(ctx, ln); close(done) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Error("Serve did not stop")
		}
	})

	c := dialClient(t, ln.Addr().String())
	defer c.Close()
	sendFrame(t, c, protocol.Header{Type: protocol.MsgAction}, nil)

	var id string
	select {
	case id = <-ids:
	case <-time.After(2 * time.Second):
		t.Fatal("handler não viu a sessão")
	}
	if !metrics.ValidTraceID(id) {
		t.Fatalf("TraceID da sessão = %q, fora do formato de 32 hex", id)
	}
	// O log do aceite carrega o MESMO id: é por ele que a correlação começa.
	if line := logs.String(); !strings.Contains(line, "trace_id="+id) {
		t.Fatalf("log do aceite sem trace_id=%s:\n%s", id, line)
	}
}
