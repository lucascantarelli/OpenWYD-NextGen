package grpcsrv

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	webv1 "github.com/jeanluca/w2pp-openwyd/api/web/v1"
	"github.com/jeanluca/w2pp-openwyd/webserver/internal/dropadmin"
)

// DropAdmin is the moderator drop-rate surface the server depends on (satisfied
// by *dropadmin.Service). Kept as an interface so the server is unit-testable.
type DropAdmin interface {
	List(ctx context.Context, moderatorID int64) (dropadmin.Result, []dropadmin.Slot, error)
	Upsert(ctx context.Context, moderatorID int64, slot int, value int64, reason string) (dropadmin.Result, error)
	Reset(ctx context.Context, moderatorID int64, slot int, reason string) (dropadmin.Result, error)
	ListCarry(ctx context.Context, moderatorID int64, templateName string) (dropadmin.Result, dropadmin.Carry, error)
	SetCarry(ctx context.Context, moderatorID int64, templateName string, items []dropadmin.CarryInput, reason string) (dropadmin.Result, error)
	ResetCarry(ctx context.Context, moderatorID int64, templateName, reason string) (dropadmin.Result, error)
}

// DropAdminServer implements webv1.DropAdminServiceServer.
type DropAdminServer struct {
	webv1.UnimplementedDropAdminServiceServer
	admin DropAdmin
}

// NewDropAdmin builds the DropAdminService over the given admin logic.
func NewDropAdmin(a DropAdmin) *DropAdminServer { return &DropAdminServer{admin: a} }

// ListDropRates returns all 64 Carry slots with base/current/overridden.
func (s *DropAdminServer) ListDropRates(ctx context.Context, req *webv1.ListDropRatesRequest) (*webv1.ListDropRatesResponse, error) {
	res, slots, err := s.admin.List(ctx, req.GetModeratorId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list drop rates: %v", err)
	}
	out := make([]*webv1.DropRateSlot, 0, len(slots))
	for _, slot := range slots {
		out = append(out, &webv1.DropRateSlot{
			Slot:         int32(slot.Slot),
			BaseValue:    int32(slot.Base),
			CurrentValue: int32(slot.Current),
			Overridden:   slot.Overridden,
		})
	}
	return &webv1.ListDropRatesResponse{Result: dropResultToProto(res), Slots: out}, nil
}

// UpsertDropRate stores one slot divisor.
func (s *DropAdminServer) UpsertDropRate(ctx context.Context, req *webv1.UpsertDropRateRequest) (*webv1.AdminAck, error) {
	res, err := s.admin.Upsert(ctx, req.GetModeratorId(), int(req.GetSlot()), int64(req.GetValue()), req.GetReason())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "upsert drop rate: %v", err)
	}
	return &webv1.AdminAck{Result: dropResultToProto(res)}, nil
}

// ResetDropRate removes the overlay row for a slot.
func (s *DropAdminServer) ResetDropRate(ctx context.Context, req *webv1.ResetDropRateRequest) (*webv1.AdminAck, error) {
	res, err := s.admin.Reset(ctx, req.GetModeratorId(), int(req.GetSlot()), req.GetReason())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "reset drop rate: %v", err)
	}
	return &webv1.AdminAck{Result: dropResultToProto(res)}, nil
}

// ListMobCarry returns one template's Carry[] merged with its overlay.
func (s *DropAdminServer) ListMobCarry(ctx context.Context, req *webv1.ListMobCarryRequest) (*webv1.ListMobCarryResponse, error) {
	res, carry, err := s.admin.ListCarry(ctx, req.GetModeratorId(), req.GetTemplateName())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list mob carry: %v", err)
	}
	out := make([]*webv1.MobCarrySlot, 0, len(carry.Slots))
	for _, slot := range carry.Slots {
		if slot.ItemIndex == 0 {
			continue // vazio: o portal desenha a posição livre a partir do slot
		}
		out = append(out, &webv1.MobCarrySlot{
			Slot: int32(slot.Slot), ItemIndex: int32(slot.ItemIndex), ItemName: slot.ItemName,
			Eff1: int32(slot.Eff[0][0]), Effv1: int32(slot.Eff[0][1]),
			Eff2: int32(slot.Eff[1][0]), Effv2: int32(slot.Eff[1][1]),
			Eff3: int32(slot.Eff[2][0]), Effv3: int32(slot.Eff[2][1]),
			FromOverride: slot.FromOverride,
		})
	}
	return &webv1.ListMobCarryResponse{
		Result:       dropResultToProto(res),
		TemplateName: carry.TemplateName,
		DisplayName:  carry.DisplayName,
		Level:        int32(carry.Level),
		Overridden:   carry.Overridden,
		Slots:        out,
	}, nil
}

// SetMobCarry replaces a template's whole Carry[] table.
func (s *DropAdminServer) SetMobCarry(ctx context.Context, req *webv1.SetMobCarryRequest) (*webv1.AdminAck, error) {
	items := make([]dropadmin.CarryInput, 0, len(req.GetSlots()))
	for _, slot := range req.GetSlots() {
		items = append(items, dropadmin.CarryInput{Slot: int(slot.GetSlot()), ItemIndex: int(slot.GetItemIndex())})
	}
	res, err := s.admin.SetCarry(ctx, req.GetModeratorId(), req.GetTemplateName(), items, req.GetReason())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "set mob carry: %v", err)
	}
	return &webv1.AdminAck{Result: dropResultToProto(res)}, nil
}

// ResetMobCarry drops the template's carry overlay.
func (s *DropAdminServer) ResetMobCarry(ctx context.Context, req *webv1.ResetMobCarryRequest) (*webv1.AdminAck, error) {
	res, err := s.admin.ResetCarry(ctx, req.GetModeratorId(), req.GetTemplateName(), req.GetReason())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "reset mob carry: %v", err)
	}
	return &webv1.AdminAck{Result: dropResultToProto(res)}, nil
}

// dropResultToProto maps dropadmin.Result to the shared AdminResult enum (named
// to avoid colliding with the other resultToProto helpers in this package).
func dropResultToProto(r dropadmin.Result) webv1.AdminResult {
	switch r {
	case dropadmin.OK:
		return webv1.AdminResult_ADMIN_RESULT_OK
	case dropadmin.Forbidden:
		return webv1.AdminResult_ADMIN_RESULT_FORBIDDEN
	case dropadmin.NotFound:
		return webv1.AdminResult_ADMIN_RESULT_NOT_FOUND
	default:
		return webv1.AdminResult_ADMIN_RESULT_INVALID
	}
}
