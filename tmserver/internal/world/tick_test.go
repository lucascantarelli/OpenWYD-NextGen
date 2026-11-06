package world

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"openwyd-nextgen/internal/metrics"
)

// TestTickSetsTimestampGauge covers the OBS-03 alert source: a fresh tick
// timestamp lets Prometheus tell "mundo vivo" from "mundo parado".
func TestTickSetsTimestampGauge(t *testing.T) {
	reg := metrics.New("tmserver")
	w := New(Config{GridDim: 16, Metrics: reg}, slogDiscard(), NopPersistence{}, nil)
	w.SetTickHandler(20*time.Millisecond, func(*World) {})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { _ = w.Serve(ctx, ln); close(done) }()
	defer func() {
		cancel()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Error("Serve did not stop")
		}
	}()

	deadline := time.Now().Add(2 * time.Second)
	for {
		var b strings.Builder
		if err := reg.Write(&b); err != nil {
			t.Fatalf("Write: %v", err)
		}
		out := b.String()
		if strings.Contains(out, `world_tick_timestamp_seconds{service="tmserver"}`) {
			// Unix segundos do momento (1.7e9 hoje): o gauge existe e foi
			// gravado pelo loop, não é um zero do registry.
			if !strings.Contains(out, `world_tick_timestamp_seconds{service="tmserver"} 1.`) {
				t.Fatalf("timestamp do tick fora do esperado (Unix s):\n%s", out)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("gauge do tick não apareceu:\n%s", out)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
