package main

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"

	"openwyd-nextgen/internal/metrics"
	"openwyd-nextgen/internal/store"
)

// TestNewAppRequiresDSN prova que a validação de configuração mora no newApp e
// falha antes de qualquer conexão ou listener.
func TestNewAppRequiresDSN(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	if _, err := newApp(context.Background(), serveConfig{}, logger); err == nil {
		t.Fatal("newApp aceitou execução sem DSN")
	} else if !strings.Contains(err.Error(), "dsn") {
		t.Fatalf("erro = %q, want menção a -dsn", err)
	}
}

// TestNewServerRegistersEveryService cobre o passo de wiring sem I/O: com um
// store já resolvido (pool nil, nada é consultado no registro), o gRPC sai com
// os nove serviços do dbServer e sem listener.
func TestNewServerRegistersEveryService(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv, err := newServer(serveConfig{}, store.New(nil), metrics.New("dbserver"), logger)
	if err != nil {
		t.Fatalf("newServer: %v", err)
	}
	defer srv.Stop()

	info := srv.GetServiceInfo()
	want := []string{
		"db.v1.AccountService",
		"db.v1.ContentCatalogService",
		"db.v1.ContentRateConfigService",
		"db.v1.DropConfigService",
		"db.v1.InstanceFlagsService",
		"db.v1.ItemEffectConfigService",
		"db.v1.NpcConfigService",
		"db.v1.TeleportConfigService",
		"db.v1.WorldEventConfigService",
	}
	for _, name := range want {
		if _, ok := info[name]; !ok {
			t.Errorf("serviço %s não registrado", name)
		}
	}
	if len(info) != len(want) {
		t.Errorf("serviços registrados = %d, want %d", len(info), len(want))
	}
}
