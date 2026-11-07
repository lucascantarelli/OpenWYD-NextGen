package grpcsrv

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	dbv1 "openwyd-nextgen/api/db/v1"
	"openwyd-nextgen/internal/store"
)

// ItemEffectConfigStore é a superfície de leitura de que o serviço de overlay de
// efeito de item precisa (satisfeita por *store.Store). O tmServer só lê por aqui;
// toda escrita vai pelo web-api, que também é dono da trilha de auditoria.
type ItemEffectConfigStore interface {
	ListItemEffectOverrides(ctx context.Context, itemIndex int32) ([]store.ItemEffectOverride, error)
}

// ItemEffectConfigServer implementa dbv1.ItemEffectConfigServiceServer: a
// ponte somente leitura que o tmServer busca uma vez no boot para aplicar efeitos
// base de item editados pelo moderador (issue 44).
type ItemEffectConfigServer struct {
	dbv1.UnimplementedItemEffectConfigServiceServer
	store ItemEffectConfigStore
}

// NewItemEffectConfig monta o serviço de configuração de efeito de item sobre o store recebido.
func NewItemEffectConfig(s ItemEffectConfigStore) *ItemEffectConfigServer {
	return &ItemEffectConfigServer{store: s}
}

// ListItemEffectOverrides devolve o snapshot completo do overlay. Como na ponte de
// teleporte, não há campo de versão: o payload é uma linha por efeito editado (no
// máximo um punhado), então uma única busca no boot basta.
func (s *ItemEffectConfigServer) ListItemEffectOverrides(ctx context.Context, _ *dbv1.ListItemEffectOverridesRequest) (*dbv1.ListItemEffectOverridesResponse, error) {
	// itemIndex 0 significa "todas as linhas" na consulta do store.
	rows, err := s.store.ListItemEffectOverrides(ctx, 0)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list item effect overrides: %v", err)
	}
	out := make([]*dbv1.ItemEffectOverride, 0, len(rows))
	for _, r := range rows {
		out = append(out, &dbv1.ItemEffectOverride{
			ItemIndex: r.ItemIndex,
			Effect:    int32(r.Effect),
			Value:     int32(r.Value),
		})
	}
	return &dbv1.ListItemEffectOverridesResponse{Rows: out}, nil
}
