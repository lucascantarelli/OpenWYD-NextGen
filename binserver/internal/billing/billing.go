// Package billing é a política de cobrança do binServer, DESENHADA DO ZERO: o
// link legado _AUTH_GAME de 196 bytes não foi verificado e NÃO é reproduzido. O
// binServer novo expõe um contrato gRPC limpo (api/bin/v1) com política própria.
// O gate de cobrança no login de personagem do tmServer chama Check; este
// serviço é a fonte da verdade.
package billing

import (
	"sync"
	"time"
)

// Status é a situação de cobrança de uma conta.
type Status int

// Situações de cobrança.
const (
	StatusUnknown Status = iota // sem registro (a política decide liberar/negar)
	StatusActive                // pagante / free-to-play ativo
	StatusBlocked               // bloqueado por decisão administrativa
	StatusExpired               // assinatura vencida
)

// String rende a situação para log e wire.
func (s Status) String() string {
	switch s {
	case StatusActive:
		return "active"
	case StatusBlocked:
		return "blocked"
	case StatusExpired:
		return "expired"
	default:
		return "unknown"
	}
}

// Account é um registro de cobrança. ExpiresAt zero significa "nunca expira".
type Account struct {
	Name      string
	Status    Status
	ExpiresAt time.Time
}

// Decision é o resultado de uma checagem de cobrança.
type Decision struct {
	Allowed bool
	Status  Status
}

// Service guarda os registros de cobrança. O mapa em memória é um substituto
// do armazenamento persistente; a política e a API são estáveis qualquer que
// seja o back-end. Seguro para uso concorrente (o servidor gRPC atende em muitas
// goroutines).
type Service struct {
	mu           sync.RWMutex
	accounts     map[string]Account
	allowUnknown bool // padrão free-to-play para conta sem registro
}

// New cria um Service. allowUnknown define a política para conta sem registro
// (true = free-to-play: libera; false = exige registro).
func New(allowUnknown bool) *Service {
	return &Service{accounts: make(map[string]Account), allowUnknown: allowUnknown}
}

// Set insere ou substitui o registro de uma conta.
func (s *Service) Set(acc Account) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.accounts[acc.Name] = acc
}

// Check decide se a conta pode logar no instante now.
func (s *Service) Check(name string, now time.Time) Decision {
	s.mu.RLock()
	acc, ok := s.accounts[name]
	s.mu.RUnlock()

	if !ok {
		return Decision{Allowed: s.allowUnknown, Status: StatusUnknown}
	}
	if acc.Status == StatusBlocked {
		return Decision{Allowed: false, Status: StatusBlocked}
	}
	if !acc.ExpiresAt.IsZero() && now.After(acc.ExpiresAt) {
		return Decision{Allowed: false, Status: StatusExpired}
	}
	return Decision{Allowed: acc.Status == StatusActive, Status: acc.Status}
}
