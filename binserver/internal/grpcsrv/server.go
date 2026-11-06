// Package grpcsrv implementa o BillingService gRPC do binServer (api/bin/v1)
// sobre a política de cobrança. É a fronteira nova e limpa que o tmServer chama
// no login de personagem; o link legado _AUTH_GAME não é reproduzido.
package grpcsrv

import (
	"context"
	"time"

	binv1 "openwyd-nextgen/api/bin/v1"
	"openwyd-nextgen/binserver/internal/billing"
)

// Clock devolve o instante atual; injetável para testes determinísticos.
type Clock func() time.Time

// Server implementa binv1.BillingServiceServer.
type Server struct {
	binv1.UnimplementedBillingServiceServer
	svc *billing.Service
	now Clock
}

// New monta o BillingService sobre a política dada. Clock nil usa time.Now.
func New(svc *billing.Service, clock Clock) *Server {
	if clock == nil {
		clock = time.Now
	}
	return &Server{svc: svc, now: clock}
}

// CheckBilling decide se a conta pode entrar no mundo.
func (s *Server) CheckBilling(_ context.Context, req *binv1.CheckBillingRequest) (*binv1.CheckBillingResponse, error) {
	d := s.svc.Check(req.GetAccountName(), s.now())
	return &binv1.CheckBillingResponse{
		Allowed: d.Allowed,
		Status:  statusToProto(d.Status),
	}, nil
}

func statusToProto(s billing.Status) binv1.Status {
	switch s {
	case billing.StatusActive:
		return binv1.Status_STATUS_ACTIVE
	case billing.StatusBlocked:
		return binv1.Status_STATUS_BLOCKED
	case billing.StatusExpired:
		return binv1.Status_STATUS_EXPIRED
	default:
		return binv1.Status_STATUS_UNSPECIFIED
	}
}
