package grpcsrv

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	dbv1 "openwyd-nextgen/api/db/v1"
	"openwyd-nextgen/internal/domain"
)

// LoadKefraState devolve o ciclo durável, ou revisão zero no primeiro boot.
func (s *Server) LoadKefraState(ctx context.Context, _ *dbv1.LoadKefraStateRequest) (*dbv1.LoadKefraStateResponse, error) {
	st, err := s.store.LoadKefraState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load kefra state: %v", err)
	}
	return &dbv1.LoadKefraStateResponse{State: &dbv1.KefraState{Defeated: st.Defeated, NextSpawnUnix: st.NextSpawnUnix, LastSpawnUnix: st.LastSpawnUnix, Revision: st.Revision}}, nil
}

// SaveKefraState valida o ciclo antes de gravar sua revisão monotônica.
func (s *Server) SaveKefraState(ctx context.Context, req *dbv1.SaveKefraStateRequest) (*dbv1.SaveKefraStateResponse, error) {
	p := req.GetState()
	if p.GetRevision() <= 0 || p.GetNextSpawnUnix() <= 0 || p.GetLastSpawnUnix() < 0 || p.GetLastSpawnUnix() >= p.GetNextSpawnUnix() {
		return nil, status.Error(codes.InvalidArgument, "invalid kefra state")
	}
	st := domain.KefraState{Defeated: p.GetDefeated(), NextSpawnUnix: p.GetNextSpawnUnix(), LastSpawnUnix: p.GetLastSpawnUnix(), Revision: p.GetRevision()}
	if err := s.store.SaveKefraState(ctx, st); err != nil {
		return nil, status.Errorf(codes.Internal, "save kefra state: %v", err)
	}
	return &dbv1.SaveKefraStateResponse{Ok: true}, nil
}
