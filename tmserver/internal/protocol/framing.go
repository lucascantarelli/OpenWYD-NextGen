package protocol

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// ErrBadInitCode é devolvido quando os 4 primeiros bytes de uma conexão não são InitCode.
// O servidor legado descarta essas conexões (protocol-spec.md §1.2).
var ErrBadInitCode = errors.New("protocol: bad INITCODE")

// ErrBadSize é devolvido quando o Size do cabeçalho de um frame está fora da faixa aceita
// [HeaderSize, MaxMessageSize] (protocol-spec.md §1.3, ErrorCode=2).
var ErrBadSize = errors.New("protocol: frame size out of range")

// Framer desmonta o fluxo de bytes CPSock de uma conexão (CPSock::ReadMessage,
// protocol-spec.md §1.3). Consome e valida o INITCODE de 4 bytes uma vez,
// depois entrega frames delimitados por Size. O Size é validado ANTES de qualquer
// alocação por frame (guidelines §18.3). Um Framer não é seguro para uso concorrente;
// uma por conexão (uma goroutine de I/O por conexão, migration-plan.md §3.5).
type Framer struct {
	r        io.Reader
	buf      []byte
	scratch  []byte
	initDone bool
}

// NewFramer devolve um Framer lendo de r.
func NewFramer(r io.Reader) *Framer {
	return &Framer{r: r, scratch: make([]byte, 4096)}
}

// fill lê do reader subjacente até pelo menos n bytes estarem no buffer.
// Devolve io.EOF só quando o fluxo termina exatamente numa fronteira de frame sem
// nada no buffer, e io.ErrUnexpectedEOF quando termina no meio de um frame.
func (f *Framer) fill(n int) error {
	for len(f.buf) < n {
		m, err := f.r.Read(f.scratch)
		if m > 0 {
			f.buf = append(f.buf, f.scratch[:m]...)
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				if len(f.buf) == 0 {
					return io.EOF
				}
				if len(f.buf) < n {
					return io.ErrUnexpectedEOF
				}
				return nil
			}
			return fmt.Errorf("protocol: read: %w", err)
		}
	}
	return nil
}

// ReadFrame devolve o próximo frame de wire ofuscado completo (exatamente Size bytes).
// Na primeira chamada consome o INITCODE. Devolve io.EOF quando o fluxo
// termina limpo entre frames. O slice devolvido pertence a quem chama (não é
// retido pelo Framer); passe-o ao Decode.
func (f *Framer) ReadFrame() ([]byte, error) {
	if !f.initDone {
		if err := f.fill(4); err != nil {
			return nil, err
		}
		if binary.LittleEndian.Uint32(f.buf[0:4]) != InitCode {
			// Mostre os bytes iniciais reais para o handshake de um cliente de verdade ser
			// identificado (diagnóstico; o buffer contém o primeiro segmento TCP).
			n := len(f.buf)
			if n > 64 {
				n = 64
			}
			return nil, fmt.Errorf("%w (first %d bytes: % x)", ErrBadInitCode, n, f.buf[:n])
		}
		f.buf = f.buf[4:]
		f.initDone = true
	}

	if err := f.fill(HeaderSize); err != nil {
		return nil, err
	}
	size := int(binary.LittleEndian.Uint16(f.buf[0:2]))
	if size < HeaderSize || size > MaxMessageSize {
		return nil, fmt.Errorf("%w: %d", ErrBadSize, size)
	}

	if err := f.fill(size); err != nil {
		return nil, err
	}
	frame := make([]byte, size)
	copy(frame, f.buf[:size])
	f.buf = f.buf[size:]
	return frame, nil
}
