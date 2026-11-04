package grpcsrv

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	dbv1 "github.com/jeanluca/w2pp-openwyd/api/db/v1"
)

// InstanceFlagsStore is the write surface the flags snapshot needs (satisfied by
// *store.Store).
type InstanceFlagsStore interface {
	RecordInstanceFlags(ctx context.Context, server string, flags []byte) error
}

// InstanceFlagsServer implements dbv1.InstanceFlagsServiceServer: tmServer records
// the flags it booted with so the portal can show them read-only (issue 36).
type InstanceFlagsServer struct {
	dbv1.UnimplementedInstanceFlagsServiceServer
	store InstanceFlagsStore
}

// NewInstanceFlags builds the flags service over the given store.
func NewInstanceFlags(s InstanceFlagsStore) *InstanceFlagsServer {
	return &InstanceFlagsServer{store: s}
}

// RecordInstanceFlags appends one boot snapshot.
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
