package main

import (
	"context"
	"io"
	"log/slog"
	"net"
	"strings"
	"testing"
)

// freePort devolve uma porta TCP livre no loopback; nada escuta nela depois do
// close, então ela serve para provar que o newApp não abriu listener.
func freePort(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reservar porta: %v", err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatalf("liberar porta: %v", err)
	}
	return addr
}

// TestNewAppBuildsWorldWithoutListening monta o mundo com a configuração mínima
// (-content vazio, sem dbServer) e prova que a porta continua livre: o wiring
// não escuta nada até `run`.
func TestNewAppBuildsWorldWithoutListening(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	addr := freePort(t)
	a, err := newApp(context.Background(), config{addr: addr, contentDir: "", metricsAddr: "", statusAddr: ""}, logger)
	if err != nil {
		t.Fatalf("newApp: %v", err)
	}
	if a.w == nil {
		t.Fatal("newApp devolveu app sem mundo")
	}
	if a.addr != addr {
		t.Fatalf("addr = %q, want %q", a.addr, addr)
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("newApp já escutava em %s: %v", addr, err)
	}
	_ = ln.Close()
}

// TestNewAppRejectsContentTreeWithoutRates prova o fail-fast do conteúdo: uma
// árvore sem Common/Settings é erro de newApp, não um mundo com taxas zeradas.
func TestNewAppRejectsContentTreeWithoutRates(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	_, err := newApp(context.Background(), config{contentDir: t.TempDir(), metricsAddr: ""}, logger)
	if err == nil {
		t.Fatal("newApp aceitou árvore de conteúdo sem CompRate/SancRate/QuestsRate")
	}
	if !strings.Contains(err.Error(), "CompRate") {
		t.Fatalf("erro = %q, want menção ao arquivo de taxa ausente", err)
	}
}
