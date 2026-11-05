package grpcsrv

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	webv1 "openwyd-nextgen/api/web/v1"
)

// SetGuildMaster promotes one guild member to leader (audited).
func (s *WorldStateAdminServer) SetGuildMaster(ctx context.Context, req *webv1.SetGuildMasterRequest) (*webv1.SetWorldStateResponse, error) {
	res, message, err := s.admin.SetGuildMaster(
		ctx, req.GetModeratorId(), req.GetGuildId(), req.GetCharacterId(), req.GetReason())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "set guild master: %v", err)
	}
	return &webv1.SetWorldStateResponse{Result: worldStateResultToProto(res), Message: message}, nil
}

// RemoveGuildMember drops one membership (audited).
func (s *WorldStateAdminServer) RemoveGuildMember(ctx context.Context, req *webv1.RemoveGuildMemberRequest) (*webv1.SetWorldStateResponse, error) {
	res, message, err := s.admin.RemoveGuildMember(
		ctx, req.GetModeratorId(), req.GetGuildId(), req.GetCharacterId(), req.GetReason())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "remove guild member: %v", err)
	}
	return &webv1.SetWorldStateResponse{Result: worldStateResultToProto(res), Message: message}, nil
}

// SetZoneOwner sets or clears the guild in charge of one kingdom zone (audited).
func (s *WorldStateAdminServer) SetZoneOwner(ctx context.Context, req *webv1.SetZoneOwnerRequest) (*webv1.SetWorldStateResponse, error) {
	res, message, err := s.admin.SetZoneOwner(
		ctx, req.GetModeratorId(), req.GetZone(), req.GetGuildId(), req.GetReason())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "set zone owner: %v", err)
	}
	return &webv1.SetWorldStateResponse{Result: worldStateResultToProto(res), Message: message}, nil
}

// SetTowerOwner sets or clears the single GTorre owner (audited).
func (s *WorldStateAdminServer) SetTowerOwner(ctx context.Context, req *webv1.SetTowerOwnerRequest) (*webv1.SetWorldStateResponse, error) {
	res, message, err := s.admin.SetTowerOwner(ctx, req.GetModeratorId(), req.GetGuildId(), req.GetReason())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "set tower owner: %v", err)
	}
	return &webv1.SetWorldStateResponse{Result: worldStateResultToProto(res), Message: message}, nil
}

// SetGuildRelation upserts or removes one directed ally/war relation (audited).
func (s *WorldStateAdminServer) SetGuildRelation(ctx context.Context, req *webv1.SetGuildRelationRequest) (*webv1.SetWorldStateResponse, error) {
	res, message, err := s.admin.SetGuildRelation(
		ctx, req.GetModeratorId(), req.GetGuildId(), req.GetTargetGuildId(), int(req.GetKind()), req.GetReason())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "set guild relation: %v", err)
	}
	return &webv1.SetWorldStateResponse{Result: worldStateResultToProto(res), Message: message}, nil
}
