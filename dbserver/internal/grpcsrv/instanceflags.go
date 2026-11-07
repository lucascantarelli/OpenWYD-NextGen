package grpcsrv

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	dbv1 "openwyd-nextgen/api/db/v1"
)

// InstanceFlagsStore é a superfície de escrita de que o snapshot de flags precisa
// (satisfeita por *store.Store).
type InstanceFlagsStore interface {
	RecordInstanceFlags(ctx context.Context, server string, flags []byte) error
}

// InstanceFlagsServer implementa dbv1.InstanceFlagsServiceServer: o tmServer grava
// as flags com que subiu para o portal mostrá-las em leitura (issue 36).
type InstanceFlagsServer struct {
	dbv1.UnimplementedInstanceFlagsServiceServer
	store InstanceFlagsStore
}

// NewInstanceFlags monta o serviço de flags sobre o store recebido.
func NewInstanceFlags(s InstanceFlagsStore) *InstanceFlagsServer {
	return &InstanceFlagsServer{store: s}
}

// RecordInstanceFlags acrescenta um snapshot de boot.
func (s *InstanceFlagsServer) RecordInstanceFlags(ctx context.Context, req *dbv1.RecordInstanceFlagsRequest) (*dbv1.RecordInstanceFlagsResponse, error) {
	server := req.GetServer()
	if server == "" {
		return nil, status.Error(codes.InvalidArgument, "record instance flags: server is required")
	}
	if req.GetFlagsJson() == "" {
		return nil, status.Error(codes.InvalidArgument, "record instance flags: flags_json is required")
	}
	if err := s.store.RecordInstanceFlags(ctx, server, []byte(req.GetFlagsJson())); err != nil {
		return nil, status.Errorf(codes.Internal, "record instance flags: %v", err)
	}
	return &dbv1.RecordInstanceFlagsResponse{}, nil
}
