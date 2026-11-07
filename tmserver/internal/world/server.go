package world

import (
	"context"
	"errors"
	"net"
)

// Serve roda o mundo: inicia o loop de aceite em ln e então roda o loop do jogo
// (bloqueante) até ctx ser cancelado. No cancelamento ele para de aceitar,
// drena/salva as sessões e devolve ctx.Err().
func (w *World) Serve(ctx context.Context, ln net.Listener) error {
	go w.acceptLoop(ctx, ln)
	return w.Run(ctx)
}

// acceptLoop aceita conexões e entrega cada uma ao loop como connectEvent.
// Roda na própria goroutine; fechar o listener no cancelamento de ctx faz
// Accept retornar para o loop poder sair.
func (w *World) acceptLoop(ctx context.Context, ln net.Listener) {
	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()
	for {
		c, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return
			}
			w.log.Warn("accept failed", "err", err)
			return
		}
		// Classificar sonda de status HTTP vs conexão de jogo CPSock fora da
		// goroutine de aceite para uma sonda lenta nunca bloquear o aceite de outros clientes.
		go w.handleConn(c)
	}
}
