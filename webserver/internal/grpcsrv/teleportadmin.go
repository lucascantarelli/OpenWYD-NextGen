package grpcsrv

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	webv1 "openwyd-nextgen/api/web/v1"
	"openwyd-nextgen/internal/domain"
	"openwyd-nextgen/internal/teleportdata"
	"openwyd-nextgen/webserver/internal/teleportadmin"
)

// TeleportAdmin is the teleport-editing surface the server depends on
// (satisfied by *teleportadmin.Service). Kept as an interface so the server is
// unit-testable.
type TeleportAdmin interface {
	List(ctx context.Context, moderatorID int64) (teleportadmin.Result, []teleportadmin.RouteView, error)
	Upsert(ctx context.Context, moderatorID int64, in domain.TeleportRoute) (teleportadmin.Result, []string, domain.TeleportRoute, error)
	SetEnabled(ctx context.Context, moderatorID int64, srcX, srcY int, enabled bool) (teleportadmin.Result, error)
	Delete(ctx context.Context, moderatorID int64, srcX, srcY int) (teleportadmin.Result, error)
	Simulate(ctx context.Context, moderatorID int64, srcX, srcY int) (teleportadmin.Result, teleportadmin.Simulation, error)
}

// TeleportAdminServer implements webv1.TeleportAdminServiceServer.
type TeleportAdminServer struct {
	webv1.UnimplementedTeleportAdminServiceServer
	admin TeleportAdmin
}

// NewTeleportAdmin builds the TeleportAdminService over the given admin logic.
func NewTeleportAdmin(a TeleportAdmin) *TeleportAdminServer {
	return &TeleportAdminServer{admin: a}
}

// ListRoutes returns the merged compiled+overlay route list.
func (s *TeleportAdminServer) ListRoutes(ctx context.Context, req *webv1.ListTeleportRoutesRequest) (*webv1.ListTeleportRoutesResponse, error) {
	res, views, err := s.admin.List(ctx, req.GetModeratorId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list teleport routes: %v", err)
	}
	out := make([]*webv1.AdminTeleportRoute, 0, len(views))
	for _, v := range views {
		route := &webv1.AdminTeleportRoute{
			SrcX: int32(v.Route.SrcX), SrcY: int32(v.Route.SrcY),
			DstX: int32(v.Route.DstX), DstY: int32(v.Route.DstY),
			Cost: int32(v.Route.Cost), Requirement: v.Route.Requirement,
			Enabled: v.Route.Enabled, Note: v.Route.Note,
			UpdatedBy:   v.Route.UpdatedBy,
			FromContent: v.FromContent, HasOverride: v.HasOverride,
		}
		if !v.Route.UpdatedAt.IsZero() {
			route.UpdatedAt = v.Route.UpdatedAt.UTC().Format(time.RFC3339)
		}
		out = append(out, route)
	}
	return &webv1.ListTeleportRoutesResponse{Result: teleportResultToProto(res), Routes: out}, nil
}

// UpsertRoute stores a route and echoes the normalized origin plus warnings.
func (s *TeleportAdminServer) UpsertRoute(ctx context.Context, req *webv1.UpsertTeleportRouteRequest) (*webv1.UpsertTeleportRouteResponse, error) {
	in := domain.TeleportRoute{
		SrcX: int(req.GetSrcX()), SrcY: int(req.GetSrcY()),
		DstX: int(req.GetDstX()), DstY: int(req.GetDstY()),
		Cost: int(req.GetCost()), Requirement: req.GetRequirement(),
		Enabled: req.GetEnabled(), Note: req.GetNote(),
	}
	res, warnings, route, err := s.admin.Upsert(ctx, req.GetModeratorId(), in)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "upsert teleport route: %v", err)
	}
	return &webv1.UpsertTeleportRouteResponse{
		Result: teleportResultToProto(res), Warnings: warnings,
		SrcX: int32(route.SrcX), SrcY: int32(route.SrcY),
	}, nil
}

// SetRouteEnabled toggles a route.
func (s *TeleportAdminServer) SetRouteEnabled(ctx context.Context, req *webv1.SetTeleportRouteEnabledRequest) (*webv1.AdminAck, error) {
	res, err := s.admin.SetEnabled(ctx, req.GetModeratorId(), int(req.GetSrcX()), int(req.GetSrcY()), req.GetEnabled())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "set teleport route enabled: %v", err)
	}
	return &webv1.AdminAck{Result: teleportResultToProto(res)}, nil
}

// DeleteRoute drops the overlay row.
func (s *TeleportAdminServer) DeleteRoute(ctx context.Context, req *webv1.DeleteTeleportRouteRequest) (*webv1.AdminAck, error) {
	res, err := s.admin.Delete(ctx, req.GetModeratorId(), int(req.GetSrcX()), int(req.GetSrcY()))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "delete teleport route: %v", err)
	}
	return &webv1.AdminAck{Result: teleportResultToProto(res)}, nil
}

// SimulateRoute resolves an origin without touching the world's RNG.
func (s *TeleportAdminServer) SimulateRoute(ctx context.Context, req *webv1.SimulateTeleportRouteRequest) (*webv1.SimulateTeleportRouteResponse, error) {
	res, sim, err := s.admin.Simulate(ctx, req.GetModeratorId(), int(req.GetSrcX()), int(req.GetSrcY()))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "simulate teleport route: %v", err)
	}
	requirement := sim.Requirement
	if requirement == "" {
		requirement = teleportdata.RequirementNone
	}
	return &webv1.SimulateTeleportRouteResponse{
		Result: teleportResultToProto(res), Ok: sim.OK,
		SrcX: int32(sim.SrcX), SrcY: int32(sim.SrcY),
		DstX: int32(sim.DstX), DstY: int32(sim.DstY),
		Cost: int32(sim.Cost), Requirement: requirement, Warnings: sim.Warnings,
	}, nil
}

func teleportResultToProto(r teleportadmin.Result) webv1.AdminResult {
	switch r {
	case teleportadmin.OK:
		return webv1.AdminResult_ADMIN_RESULT_OK
	case teleportadmin.Forbidden:
		return webv1.AdminResult_ADMIN_RESULT_FORBIDDEN
	case teleportadmin.NotFound:
		return webv1.AdminResult_ADMIN_RESULT_NOT_FOUND
	default:
		return webv1.AdminResult_ADMIN_RESULT_INVALID
	}
}
