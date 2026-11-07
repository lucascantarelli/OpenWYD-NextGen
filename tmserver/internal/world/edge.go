package world

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"
)

// defaultStatusBody é servido quando Config.StatusFile não está definido. Ele espelha o
// formato do tmserver/content/Common/serv00.htm da distribuição (uma pequena lista
// numérica de status que o cliente interpreta para a tela de seleção de canal).
var defaultStatusBody = []byte("4\n-1\n-1\n-1\n-1\n-1\n-1\n-1\n-1\n-1\n")

// handleConn classifica uma conexão recém-aceita antes que ela se torne uma
// sessão de jogo. O cliente WYD sonda o servidor por HTTP ("GET /serv00.htm") para
// ler o status do canal, e só então abre uma conexão de jogo CPSock separada.
// Espiamos os primeiros bytes: um método HTTP é respondido com a página de status e
// fechado; qualquer outra coisa (o INITCODE CPSock) vira sessão normal, com os
// bytes espiados replicados ao enquadrador. Roda na própria goroutine por conexão
// para uma sonda lenta/ociosa nunca travar o loop de aceite.
func (w *World) handleConn(c net.Conn) {
	ip := c.RemoteAddr().String()
	br := bufio.NewReader(c)
	_ = c.SetReadDeadline(time.Now().Add(10 * time.Second))
	pre, err := br.Peek(4)
	if err != nil {
		// Uma conexão que abre mas não envia dados antes de dar timeout/fechar —
		// ex.: uma sonda de porta, OU um cliente que espera o SERVIDOR falar primeiro.
		// Logado (não engolido) para o handshake da borda do cliente poder ser mapeado.
		w.log.Info("connection sent no data", "ip", ip, "err", err.Error())
		_ = c.Close()
		return
	}
	_ = c.SetReadDeadline(time.Time{}) // limpa o deadline para a sessão (possivelmente longeva)

	if isHTTPMethod(pre) {
		w.serveStatus(c, br)
		return
	}

	// CPSock: entregar ao loop, replicando os bytes espiados pelo leitor com buffer.
	w.log.Info("cpsock connection", "ip", ip, "first4", fmt.Sprintf("% x", pre))
	if !w.emit(connectEvent{conn: &prefixConn{Conn: c, r: br}, ip: ip}) {
		_ = c.Close()
	}
}

// isHTTPMethod informa se os primeiros bytes parecem uma requisição HTTP. O
// fluxo CPSock sempre começa com INITCODE (0x1F11F311 = bytes 11 F3 11 1F), então
// não há colisão com esses prefixos de método ASCII.
func isHTTPMethod(b []byte) bool {
	switch string(b) {
	case "GET ", "POST", "HEAD":
		return true
	}
	return false
}

// serveStatus responde à sonda HTTP de status do cliente com a página de status
// página, e então fecha. Lê e loga a linha de requisição (o caminho que o cliente
// pede em cada passo do fluxo de conexão) para o comportamento da borda poder ser mapeado.
func (w *World) serveStatus(c net.Conn, br *bufio.Reader) {
	defer func() { _ = c.Close() }()
	_ = c.SetReadDeadline(time.Now().Add(5 * time.Second))
	reqLine, _ := br.ReadString('\n')
	reqLine = strings.TrimRight(reqLine, "\r\n")

	body := w.statusBody()
	_ = c.SetWriteDeadline(time.Now().Add(5 * time.Second))
	// Espelhar a resposta de um servidor web padrão (combina com o servidor WYD snalmir
	// que funciona: cabeçalhos Date + Server), caso o parser HTTP do cliente seja exigente.
	resp := fmt.Sprintf(
		"HTTP/1.1 200 OK\r\nDate: %s\r\nServer: w2pp\r\nContent-Length: %d\r\nConnection: close\r\n\r\n",
		time.Now().UTC().Format(time.RFC1123), len(body))
	if _, err := c.Write(append([]byte(resp), body...)); err != nil {
		w.log.Warn("status probe write failed", "ip", c.RemoteAddr().String(), "err", err)
		return
	}
	w.log.Info("served status probe", "ip", c.RemoteAddr().String(), "req", reqLine, "bytes", len(body))
}

// statusBody devolve os bytes da página de status: o arquivo configurado quando definido e
// legível, senão o padrão embutido. Lido a cada requisição para o arquivo poder ser
// editado e retestado sem reiniciar o servidor.
func (w *World) statusBody() []byte {
	if w.cfg.StatusFile != "" {
		if b, err := os.ReadFile(w.cfg.StatusFile); err == nil {
			return b
		}
	}
	return defaultStatusBody
}

// prefixConn é um net.Conn cujas leituras vêm de um leitor com buffer (para bytes
// espiados na detecção de protocolo serem replicados ao enquadrador) enquanto escritas e
// fechamento vão direto à conexão subjacente.
type prefixConn struct {
	net.Conn
	r io.Reader
}

func (p *prefixConn) Read(b []byte) (int, error) { return p.r.Read(b) }
