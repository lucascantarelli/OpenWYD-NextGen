package grpcsrv

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	dbv1 "openwyd-nextgen/api/db/v1"
	"openwyd-nextgen/internal/domain"
)

// TeleportConfigStore é a superfície de leitura de que o serviço de overlay de
// teleporte precisa (satisfeita por *store.Store). O tmServer só lê por aqui; toda
// escrita vai pelo web-api.
type TeleportConfigStore interface {
	ListTeleportRoutes(ctx context.Context) ([]domain.TeleportRoute, error)
}

// TeleportConfigServer implementa dbv1.TeleportConfigServiceServer: a ponte
// somente leitura que o tmServer consulta para aplicar rotas de teleporte
// editadas pelo moderador (issue 42).
type TeleportConfigServer struct {
	dbv1.UnimplementedTeleportConfigServiceServer
	store TeleportConfigStore
}

// NewTeleportConfig monta o serviço de config de teleporte sobre o store recebido.
func NewTeleportConfig(s TeleportConfigStore) *TeleportConfigServer {
	return &TeleportConfigServer{store: s}
}

// ListTeleportRoutes devolve o snapshot completo do overlay. Não há campo de
// versão de propósito: o payload são algumas dezenas de linhas, então o poll de
// 15 s apenas relê e o mundo troca o snapshot de forma atômica.
func (s *TeleportConfigServer) ListTeleportRoutes(ctx context.Context, _ *dbv1.ListTeleportRoutesRequest) (*dbv1.ListTeleportRoutesResponse, error) {
	routes, err := s.store.ListTeleportRoutes(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list teleport routes: %v", err)
	}
	out := make([]*dbv1.TeleportRoute, 0, len(routes))
	for _, r := range routes {
		out = append(out, &dbv1.TeleportRoute{
			SrcX: int32(r.SrcX), SrcY: int32(r.SrcY),
			DstX: int32(r.DstX), DstY: int32(r.DstY),
			Cost: int32(r.Cost), Requirement: r.Requirement,
			Enabled: r.Enabled, Note: r.Note,
		})
	}
	return &dbv1.ListTeleportRoutesResponse{Routes: out}, nil
}
