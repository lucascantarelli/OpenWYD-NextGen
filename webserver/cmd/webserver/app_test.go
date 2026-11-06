package main

import (
	"context"
	"io"
	"log/slog"
	"net"
	"strings"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	webv1 "openwyd-nextgen/api/web/v1"
	"openwyd-nextgen/internal/domain"
	"openwyd-nextgen/internal/store"
)

func discardLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// TestNewAppWithDepsRejectsEmptyItemCatalog prova o critério de aceite do
// REFACTOR-01: banco sem catálogo é erro explícito de newApp, não WARN de boot.
func TestNewAppWithDepsRejectsEmptyItemCatalog(t *testing.T) {
	_, err := newAppWithDeps(config{}, deps{st: store.New(nil)}, discardLogger())
	if err == nil {
		t.Fatal("newAppWithDeps aceitou item_catalog vazio")
	}
	if !strings.Contains(err.Error(), "item_catalog is empty") {
		t.Fatalf("erro = %q, want menção a item_catalog vazio", err)
	}
}

// TestNewAppWithDepsServesItemCatalogOverBufconn exercita o wiring completo de
// registro sem porta e sem banco: o catálogo vem do `deps` e a RPC o devolve pelo
// transporte em memória.
func TestNewAppWithDepsServesItemCatalogOverBufconn(t *testing.T) {
	d := deps{
		st:    store.New(nil),
		items: []domain.ItemCatalogEntry{{Index: 7, Name: "Pocao_Vermelha", Price: 120}},
	}
	a, err := newAppWithDeps(config{}, d, discardLogger())
	if err != nil {
		t.Fatalf("newAppWithDeps: %v", err)
	}
	lis := bufconn.Listen(1 << 20)
	go func() { _ = a.srv.Serve(lis) }()
	t.Cleanup(a.srv.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.DialContext(ctx) }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial bufconn: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	resp, err := webv1.NewItemCatalogServiceClient(conn).ListItems(context.Background(), &webv1.ListItemsRequest{})
	if err != nil {
		t.Fatalf("ListItems: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("itens = %d, want 1", len(resp.Items))
	}
	if got := resp.Items[0]; got.ItemIndex != 7 || got.Name != "Pocao_Vermelha" {
		t.Fatalf("item = %+v, want índice 7 com o nome do catálogo", got)
	}
}
