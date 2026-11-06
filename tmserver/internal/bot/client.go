// Package bot é o núcleo determinístico do laboratório de bots: cliente CPSock
// headless, máquina de estados de comportamento, perfis, cenários versionados,
// métricas e o executor que liga as peças.
//
// O núcleo não depende de LLM nem do cliente gráfico: fala o mesmo protocolo do
// WYD.exe e reusa o codec de tmserver/internal/protocol, de modo que um bot
// quebra do mesmo jeito que o cliente quebraria.
package bot

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"openwyd-nextgen/tmserver/internal/protocol"
)

// Erros de sessão que o executor trata como casos previstos, não como panes.
var (
	// ErrAlreadyPlaying indica que a conta está em jogo em outra conexão.
	ErrAlreadyPlaying = errors.New("bot: conta já em jogo")
	// ErrStillPlaying indica que o save da sessão anterior ainda está em curso.
	ErrStillPlaying = errors.New("bot: conta ainda salvando")
	// ErrLoginRefused indica recusa explícita do servidor (painel de mensagem).
	ErrLoginRefused = errors.New("bot: login recusado")
	// ErrCharacterRefused indica que o slot escolhido não entrou no mundo.
	ErrCharacterRefused = errors.New("bot: personagem recusado")
)

const (
	// dialTimeout é o prazo de abertura da conexão TCP.
	dialTimeout = 5 * time.Second
	// frameTimeout é o prazo de cada leitura de frame.
	frameTimeout = 5 * time.Second
	// defaultKeyWord é o índice da tabela de obfuscação. O servidor aceita
	// qualquer índice (o campo viaja em claro); fixo para o teste ser
	// reproduzível byte a byte.
	defaultKeyWord uint8 = 7
	// msgPerSecond e msgBurst espelham o limite do servidor (200/s, burst 400):
	// bot rápido demais é desconectado e parece bug do servidor.
	msgPerSecond = 200.0
	msgBurst     = 400.0
	// maxAccountLen é o tamanho útil de AccountName (16 bytes com o NUL).
	maxAccountLen = 15
	// maxPasswordLen é o tamanho útil de AccountPassword (12 bytes com o NUL).
	maxPasswordLen = 11
	// maxFramesPerStep limita a drenagem de frames por passo para o laço não
	// girar indefinidamente num servidor que fala sem parar.
	maxFramesPerStep = 64
	// minAccountListBody é o corpo mínimo aceito de CNFAccountLogin. O servidor
	// manda ~1.9 KB (lista de personagens e cargos); menos que isso é frame
	// truncado e o bot não deve seguir como se tivesse lista.
	minAccountListBody = 1800
)

// Client é uma conexão CPSock headless com o tmServer.
type Client struct {
	conn    net.Conn
	id      uint16
	keyWord uint8
	now     func() time.Time

	mu        sync.Mutex
	tokens    float64
	lastToken time.Time

	// Frames é o contador de frames recebidos desde a conexão.
	Frames int64
	// Bytes é o contador de bytes recebidos desde a conexão.
	Bytes int64
}

// Dial abre a conexão TCP e envia o INITCODE, único handshake exigido antes de
// qualquer frame. O servidor descarta uma conexão que não comece por ele.
func Dial(addr string, timeout time.Duration) (*Client, error) {
	if timeout <= 0 {
		timeout = dialTimeout
	}
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, fmt.Errorf("bot: conectar em %s: %w", addr, err)
	}
	var ic [4]byte
	binary.LittleEndian.PutUint32(ic[:], protocol.InitCode)
	if _, err := conn.Write(ic[:]); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("bot: enviar INITCODE: %w", err)
	}
	return &Client{
		conn:      conn,
		keyWord:   defaultKeyWord,
		now:       time.Now,
		tokens:    msgBurst,
		lastToken: time.Now(),
	}, nil
}

// ID devolve o slot de conexão usado no HEADER.ID dos pacotes enviados.
func (c *Client) ID() uint16 { return c.id }

// Login autentica a conta. token só é usado na troca de canal; version zero usa
// a versão do protocolo do projeto. Erros de conta em jogo voltam tipados para
// o executor decidir se reconecta.
func (c *Client) Login(account, password, token string, version int32) error {
	if account == "" {
		return errors.New("bot: conta vazia")
	}
	if len(account) > maxAccountLen {
		return fmt.Errorf("bot: conta %q excede %d bytes", account, maxAccountLen)
	}
	if len(password) > maxPasswordLen {
		return fmt.Errorf("bot: senha excede %d bytes", maxPasswordLen)
	}
	if version == 0 {
		version = protocol.AppVersion
	}

	var body protocol.MsgAccountLoginBody
	copy(body.AccountPassword[:], password)
	copy(body.AccountName[:], account)
	copy(body.Zero[:], token)
	body.ClientVersion = version

	if err := c.send(protocol.MsgAccountLogin, body.Encode()); err != nil {
		return err
	}
	for i := 0; i < 64; i++ {
		h, payload, err := c.ReadFrame(frameTimeout)
		if err != nil {
			return fmt.Errorf("bot: aguardar login: %w", err)
		}
		switch h.Type {
		case protocol.MsgCNFAccountLogin:
			if len(payload) < minAccountListBody {
				return fmt.Errorf("bot: CNFAccountLogin com %d bytes, esperado >= %d",
					len(payload), minAccountListBody)
			}
			return nil
		case protocol.MsgAlreadyPlaying:
			return ErrAlreadyPlaying
		case protocol.MsgStillPlaying:
			return ErrStillPlaying
		case protocol.MsgMessagePanel:
			return ErrLoginRefused
		}
	}
	return errors.New("bot: servidor não confirmou o login")
}

// SelectCharacter entra no mundo com o personagem do slot.
func (c *Client) SelectCharacter(slot int) error {
	body := protocol.MsgCharacterLoginBody{Slot: int32(slot)}
	if err := c.send(protocol.MsgCharacterLogin, body.Encode()); err != nil {
		return err
	}
	for i := 0; i < 64; i++ {
		h, _, err := c.ReadFrame(frameTimeout)
		if err != nil {
			return fmt.Errorf("bot: aguardar personagem: %w", err)
		}
		switch h.Type {
		case protocol.MsgCNFCharacterLogin:
			// O slot de conexão do mundo vem no header; usá-lo evita que o
			// servidor descarte os pacotes seguintes por ID inválido.
			c.id = h.ID
			return nil
		case protocol.MsgCharacterLoginFail:
			return ErrCharacterRefused
		case protocol.MsgMessagePanel:
			return ErrCharacterRefused
		}
	}
	return errors.New("bot: servidor não confirmou o personagem")
}

// Send envia um pacote do tipo indicado com o corpo informado, respeitando o
// limite de mensagens por segundo do servidor.
func (c *Client) Send(t protocol.Type, payload []byte) error {
	return c.send(t, payload)
}

// ReadFrame lê um frame S→C. O tmServer prefixa o tamanho em 2 bytes, sem
// INITCODE na saída, como o cliente legado espera.
func (c *Client) ReadFrame(timeout time.Duration) (protocol.Header, []byte, error) {
	if timeout <= 0 {
		timeout = frameTimeout
	}
	if err := c.conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return protocol.Header{}, nil, err
	}
	var sz [2]byte
	if _, err := io.ReadFull(c.conn, sz[:]); err != nil {
		return protocol.Header{}, nil, err
	}
	size := int(binary.LittleEndian.Uint16(sz[:]))
	if size < protocol.HeaderSize || size > protocol.MaxMessageSize {
		return protocol.Header{}, nil, fmt.Errorf("bot: frame com tamanho inválido %d", size)
	}
	buf := make([]byte, size)
	copy(buf, sz[:])
	if _, err := io.ReadFull(c.conn, buf[2:]); err != nil {
		return protocol.Header{}, nil, err
	}
	h, payload, mismatch, err := protocol.Decode(buf)
	if err != nil {
		return protocol.Header{}, nil, err
	}
	if mismatch {
		return protocol.Header{}, nil, fmt.Errorf("bot: checksum divergente no tipo %#x", uint16(h.Type))
	}
	c.Frames++
	c.Bytes += int64(size)
	return h, payload, nil
}

// Drain lê os frames pendentes sem bloquear além do prazo, devolvendo quantos
// foram consumidos. Serve para esvaziar o snapshot de entrada no mundo.
func (c *Client) Drain(wait time.Duration) int {
	n := 0
	for n < maxFramesPerStep {
		if _, _, err := c.ReadFrame(wait); err != nil {
			return n
		}
		n++
	}
	return n
}

// Close encerra a conexão.
func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

// send serializa, espera a vez no limitador e escreve o frame.
func (c *Client) send(t protocol.Type, payload []byte) error {
	wire, err := protocol.Encode(protocol.Header{Type: t, ID: c.id}, payload, c.keyWord)
	if err != nil {
		return fmt.Errorf("bot: codificar tipo %#x: %w", uint16(t), err)
	}
	if err := c.take(); err != nil {
		return err
	}
	if _, err := c.conn.Write(wire); err != nil {
		return fmt.Errorf("bot: enviar tipo %#x: %w", uint16(t), err)
	}
	return nil
}

// take aplica o balde de fichas do limite de mensagens por segundo.
func (c *Client) take() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	elapsed := now.Sub(c.lastToken).Seconds()
	c.lastToken = now
	c.tokens += elapsed * msgPerSecond
	if c.tokens > msgBurst {
		c.tokens = msgBurst
	}
	if c.tokens < 1 {
		return errors.New("bot: limite de mensagens por segundo atingido")
	}
	c.tokens--
	return nil
}
