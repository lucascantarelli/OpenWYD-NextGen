package grpcsrv

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	webv1 "github.com/jeanluca/w2pp-openwyd/api/web/v1"
	"github.com/jeanluca/w2pp-openwyd/internal/itemeffects"
	"github.com/jeanluca/w2pp-openwyd/webserver/internal/itemeffectadmin"
)

// ItemEffectAdmin is the base-effect editing surface the server depends on
// (satisfied by *itemeffectadmin.Service). Kept as an interface so the server is
// unit-testable.
type ItemEffectAdmin interface {
	List(ctx context.Context, moderatorID int64, itemIndex int32) (itemeffectadmin.Result, []itemeffectadmin.Effect, string, error)
	Upsert(ctx context.Context, moderatorID int64, itemIndex int32, effect, value int32, reason string) (itemeffectadmin.Result, string, error)
	Delete(ctx context.Context, moderatorID int64, itemIndex int32, effect int32, reason string) (itemeffectadmin.Result, string, error)
}

// ItemEffectAdminServer implements webv1.ItemEffectAdminServiceServer.
type ItemEffectAdminServer struct {
	webv1.UnimplementedItemEffectAdminServiceServer
	admin ItemEffectAdmin
}

// NewItemEffectAdmin builds the ItemEffectAdminService over the given admin logic.
func NewItemEffectAdmin(a ItemEffectAdmin) *ItemEffectAdminServer {
	return &ItemEffectAdminServer{admin: a}
}

// ListEffects returns the item's adds merged with the overlay, plus the vocabulary
// the editor's picker needs. The vocabulary rides along so the portal never keeps
// its own copy of ItemEffect.h.
func (s *ItemEffectAdminServer) ListEffects(ctx context.Context, req *webv1.ListItemEffectsRequest) (*webv1.ListItemEffectsResponse, error) {
	res, effects, reason, err := s.admin.List(ctx, req.GetModeratorId(), req.GetItemIndex())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list item effects: %v", err)
	}
	out := make([]*webv1.ItemEffectEntry, 0, len(effects))
	for _, eff := range effects {
		entry := &webv1.ItemEffectEntry{
			Effect:      eff.Effect,
			Value:       eff.Value,
			Name:        itemeffects.Name(eff.Effect),
			Label:       itemeffects.Label(eff.Effect),
			FromContent: eff.FromContent,
			HasOverride: eff.HasOverride,
			UpdatedBy:   eff.UpdatedBy,
		}
		if !eff.UpdatedAt.IsZero() {
			entry.UpdatedAt = eff.UpdatedAt.UTC().Format(time.RFC3339)
		}
		out = append(out, entry)
	}
	types := itemeffects.All()
	effectTypes := make([]*webv1.ItemEffectType, 0, len(types))
	for _, t := range types {
		effectTypes = append(effectTypes, &webv1.ItemEffectType{Code: t.Code, Name: t.Name, Label: t.Label})
	}
	return &webv1.ListItemEffectsResponse{
		Result: itemEffectResultToProto(res), Effects: out, EffectTypes: effectTypes, Reason: reason,
	}, nil
}

// UpsertEffect stores one effect value in the overlay.
func (s *ItemEffectAdminServer) UpsertEffect(ctx context.Context, req *webv1.UpsertItemEffectRequest) (*webv1.ItemEffectWriteResponse, error) {
	res, message, err := s.admin.Upsert(ctx, req.GetModeratorId(), req.GetItemIndex(),
		req.GetEffect(), req.GetValue(), req.GetReason())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "upsert item effect: %v", err)
	}
	return &webv1.ItemEffectWriteResponse{Result: itemEffectResultToProto(res), Message: message}, nil
}

// DeleteEffect drops the overlay row, restoring the catalog value.
func (s *ItemEffectAdminServer) DeleteEffect(ctx context.Context, req *webv1.DeleteItemEffectRequest) (*webv1.ItemEffectWriteResponse, error) {
	res, message, err := s.admin.Delete(ctx, req.GetModeratorId(), req.GetItemIndex(),
		req.GetEffect(), req.GetReason())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "delete item effect: %v", err)
	}
	return &webv1.ItemEffectWriteResponse{Result: itemEffectResultToProto(res), Message: message}, nil
}

func itemEffectResultToProto(r itemeffectadmin.Result) webv1.AdminResult {
	switch r {
	case itemeffectadmin.OK:
		return webv1.AdminResult_ADMIN_RESULT_OK
	case itemeffectadmin.Forbidden:
		return webv1.AdminResult_ADMIN_RESULT_FORBIDDEN
	case itemeffectadmin.NotFound:
		return webv1.AdminResult_ADMIN_RESULT_NOT_FOUND
	default:
		return webv1.AdminResult_ADMIN_RESULT_INVALID
	}
}
