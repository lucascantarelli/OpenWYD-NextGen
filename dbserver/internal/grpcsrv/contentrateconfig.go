package grpcsrv

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	dbv1 "openwyd-nextgen/api/db/v1"
	"openwyd-nextgen/internal/store"
)

// ContentRateConfigStore é a superfície de leitura de que o serviço de overlay
// de rates precisa (satisfeita por *store.Store). O tmServer só lê por aqui; as
// escritas vão pelo web-api, que é dono da validação e da trilha de auditoria.
type ContentRateConfigStore interface {
	ListContentRateOverrides(ctx context.Context) ([]store.ContentRateOverride, error)
}

// ContentRateConfigServer implementa dbv1.ContentRateConfigServiceServer.
type ContentRateConfigServer struct {
	dbv1.UnimplementedContentRateConfigServiceServer
	store ContentRateConfigStore
}

// NewContentRateConfig monta o serviço de rates sobre o store recebido.
func NewContentRateConfig(s ContentRateConfigStore) *ContentRateConfigServer {
	return &ContentRateConfigServer{store: s}
}

// ListContentRateOverrides devolve o snapshot completo do overlay: uma linha por
// célula editada, que são poucas mesmo num servidor ajustado.
func (s *ContentRateConfigServer) ListContentRateOverrides(ctx context.Context, _ *dbv1.ListContentRateOverridesRequest) (*dbv1.ListContentRateOverridesResponse, error) {
	rows, err := s.store.ListContentRateOverrides(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list content rate overrides: %v", err)
	}
	out := make([]*dbv1.ContentRateOverride, 0, len(rows))
	for _, r := range rows {
		out = append(out, &dbv1.ContentRateOverride{
			File:   r.File,
			Family: r.Family,
			Name:   r.Name,
			Value:  r.Value,
		})
	}
	return &dbv1.ListContentRateOverridesResponse{Rows: out}, nil
}
