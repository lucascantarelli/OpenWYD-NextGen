package protocol

import (
	"bytes"
	"testing"
)

// TestTransformInverse verifica que encodePayload e decodePayload são
// inversos exatos sobre os bytes [4:len), entre sementes e entre os quatro casos
// de deslocamento i&0x3.
// 8-bit wrap (0x00/0xFF bytes).
func TestTransformInverse(t *testing.T) {
	payloads := [][]byte{
		bytes.Repeat([]byte{0x00}, 12),
		bytes.Repeat([]byte{0xFF}, 12),
		{0x10, 0x20, 0x30, 0x40, 0x50, 0x60, 0x70, 0x80},
		func() []byte {
			b := make([]byte, 33)
			for i := range b {
				b[i] = byte(i * 7)
			}
			return b
		}(),
	}
	for _, k := range []uint8{0, 1, 17, 200, 255} {
		for _, body := range payloads {
			// Monta um buffer completo: 4 bytes em claro + (Type/ID/Tick + corpo) sob transform.
			buf := make([]byte, HeaderSize+len(body))
			for i := range buf {
				buf[i] = byte(i*3 + 1)
			}
			copy(buf[HeaderSize:], body)
			orig := append([]byte(nil), buf...)

			encodePayload(buf, k)
			if bytes.Equal(buf[transformStart:], orig[transformStart:]) && len(body) > 0 {
				t.Errorf("encodePayload did not change bytes (k=%d)", k)
			}
			// Os bytes 0..3 precisam ficar intocados pelo transform (§1.4).
			if !bytes.Equal(buf[:transformStart], orig[:transformStart]) {
				t.Errorf("transform touched clear header bytes 0..3")
			}
			decodePayload(buf, k)
			if !bytes.Equal(buf, orig) {
				t.Errorf("decode∘encode != identity (k=%d, len=%d)", k, len(body))
			}
		}
	}
}

// TestChecksumIsTransformOnly documenta que CheckSum = Sum2 - Sum1 depende só
// dos offsets da chave/transform, não dos dados — codificar dois corpos
// diferentes com a mesma chave dá o mesmo checksum.
func TestChecksumIsTransformOnly(t *testing.T) {
	a, _ := Encode(Header{Type: MsgAction}, bytes.Repeat([]byte{0x00}, 16), 50)
	b, _ := Encode(Header{Type: MsgAction}, bytes.Repeat([]byte{0xAB}, 16), 50)
	if a[3] != b[3] {
		t.Errorf("checksum differs for different data (%d vs %d); expected key-only dependence", a[3], b[3])
	}
}
