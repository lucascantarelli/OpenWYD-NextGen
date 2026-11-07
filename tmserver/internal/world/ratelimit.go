package world

import "time"

// tokenBucket é um limitador de taxa simples por conexão (endurecimento da Fase 7). Ele é
// usado só pela goroutine readLoop de uma conexão, então não precisa de lock.
type tokenBucket struct {
	tokens float64
	max    float64
	refill float64 // tokens adicionados por segundo
	last   time.Time
}

// newTokenBucket monta um balde permitindo refillPerSec mensagens sustentadas com
// profundidade de rajada burst. Um refill não positivo gera um balde desativado (allow
// sempre devolve true).
func newTokenBucket(refillPerSec float64, burst int, now time.Time) *tokenBucket {
	if refillPerSec <= 0 {
		return nil
	}
	b := float64(burst)
	if b <= 0 {
		b = refillPerSec
	}
	return &tokenBucket{tokens: b, max: b, refill: refillPerSec, last: now}
}

// allow informa se uma mensagem pode ser processada no instante now, consumindo um
// token quando pode. Um balde nil fica desativado e sempre permite.
func (b *tokenBucket) allow(now time.Time) bool {
	if b == nil {
		return true
	}
	b.tokens += now.Sub(b.last).Seconds() * b.refill
	if b.tokens > b.max {
		b.tokens = b.max
	}
	b.last = now
	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}
