package main

import (
	"context"
	"io"
	"log/slog"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	binv1 "openwyd-nextgen/api/bin/v1"
)

// TestNewAppServesBillingOverBufconn prova o wiring sem abrir porta nem tocar em
// banco: `newApp` monta o gRPC e o BillingService responde pelo transporte em
// memória. `allow-unknown` é política do serviço, então a resposta também prova
// que o valor do `config` chegou ao billing.
func TestNewAppServesBillingOverBufconn(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	// allowUnknown é o default do parseConfig (free-to-play); montar o config à
	// mão exige dizê-lo, e o teste falha se o valor não chegar ao billing.
	a, err := newApp(context.Background(), config{allowUnknown: true, metricsAddr: ""}, logger)
	if err != nil {
		t.Fatalf("newApp: %v", err)
	}
	if len(a.srv.GetServiceInfo()) == 0 {
		t.Fatal("newApp registrou zero serviços gRPC")
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

	resp, err := binv1.NewBillingServiceClient(conn).CheckBilling(context.Background(), &binv1.CheckBillingRequest{AccountName: "sem-registro"})
	if err != nil {
		t.Fatalf("CheckBilling: %v", err)
	}
	if !resp.Allowed {
		t.Fatalf("CheckBilling allowed = false, want true com allow-unknown (config não chegou ao billing?)")
	}
}
