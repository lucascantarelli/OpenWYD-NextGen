package grpcsrv

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	webv1 "openwyd-nextgen/api/web/v1"
	"openwyd-nextgen/internal/domain"
	"openwyd-nextgen/internal/store"
	"openwyd-nextgen/webserver/internal/characteradmin"
)

// CharacterAdmin is the character-inspection surface the server depends on
// (satisfied by *characteradmin.Service). Kept as an interface so the server is
// unit-testable.
type CharacterAdmin interface {
	Search(ctx context.Context, moderatorID int64, query string) (characteradmin.Result, []store.CharacterSearchHit, error)
	Get(ctx context.Context, moderatorID, characterID int64) (characteradmin.Result, characteradmin.Detail, error)
	Grant(ctx context.Context, moderatorID, characterID int64, item domain.Item, quantity int, reason string) (characteradmin.Result, []int64, error)
	ListGrants(ctx context.Context, moderatorID, characterID int64) (characteradmin.Result, []store.CharacterItemGrant, error)
}

// CharacterAdminServer implements webv1.CharacterAdminServiceServer.
type CharacterAdminServer struct {
	webv1.UnimplementedCharacterAdminServiceServer
	admin CharacterAdmin
}

// NewCharacterAdmin builds the CharacterAdminService over the given logic.
func NewCharacterAdmin(a CharacterAdmin) *CharacterAdminServer {
	return &CharacterAdminServer{admin: a}
}

// SearchCharacters finds characters by name or account name.
func (s *CharacterAdminServer) SearchCharacters(ctx context.Context, req *webv1.SearchCharactersRequest) (*webv1.SearchCharactersResponse, error) {
	res, hits, err := s.admin.Search(ctx, req.GetModeratorId(), req.GetQuery())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "search characters: %v", err)
	}
	out := make([]*webv1.AdminCharacterSummary, 0, len(hits))
	for _, h := range hits {
		out = append(out, &webv1.AdminCharacterSummary{
			Id: h.ID, AccountId: h.AccountID, AccountName: h.AccountName,
			Name: h.Name, Slot: int32(h.Slot), Class: int32(h.Class),
			Level: h.Level, Exp: h.Exp, Coin: h.Coin,
		})
	}
	return &webv1.SearchCharactersResponse{Result: characterResultToProto(res), Characters: out}, nil
}

// GetCharacter returns one character's status + inventory.
func (s *CharacterAdminServer) GetCharacter(ctx context.Context, req *webv1.GetCharacterAdminRequest) (*webv1.GetCharacterAdminResponse, error) {
	res, detail, err := s.admin.Get(ctx, req.GetModeratorId(), req.GetCharacterId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get character: %v", err)
	}
	resp := &webv1.GetCharacterAdminResponse{Result: characterResultToProto(res)}
	if res == characteradmin.OK {
		resp.Character = adminCharacterDetailToProto(detail)
		resp.SkillNames = detail.SkillNames
		resp.AddLabels = detail.Adds
	}
	return resp, nil
}

// GrantItem enqueues the item units in delivery_queue.
func (s *CharacterAdminServer) GrantItem(ctx context.Context, req *webv1.GrantCharacterItemRequest) (*webv1.GrantCharacterItemResponse, error) {
	item := domain.Item{
		Index: int16(req.GetItemIndex()),
		Eff1:  uint8(req.GetEff1()), EffV1: uint8(req.GetEffv1()),
		Eff2: uint8(req.GetEff2()), EffV2: uint8(req.GetEffv2()),
		Eff3: uint8(req.GetEff3()), EffV3: uint8(req.GetEffv3()),
	}
	if !characteradmin.ValidateAdds(item) {
		// Código fora de ItemEffect.h: o cliente não saberia o que fazer com o add.
		return &webv1.GrantCharacterItemResponse{Result: webv1.AdminResult_ADMIN_RESULT_INVALID}, nil
	}
	res, ids, err := s.admin.Grant(ctx, req.GetModeratorId(), req.GetCharacterId(), item,
		int(req.GetQuantity()), req.GetReason())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "grant item: %v", err)
	}
	return &webv1.GrantCharacterItemResponse{Result: characterResultToProto(res), DeliveryIds: ids}, nil
}

// ListCharacterGrants returns the character's mailbox rows.
func (s *CharacterAdminServer) ListCharacterGrants(ctx context.Context, req *webv1.ListCharacterGrantsRequest) (*webv1.ListCharacterGrantsResponse, error) {
	res, grants, err := s.admin.ListGrants(ctx, req.GetModeratorId(), req.GetCharacterId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list character grants: %v", err)
	}
	out := make([]*webv1.AdminCharacterGrant, 0, len(grants))
	for _, g := range grants {
		out = append(out, &webv1.AdminCharacterGrant{
			DeliveryId: g.DeliveryID, ItemIndex: g.ItemIndex,
			Status: g.Status, Source: g.Source, CreatedAt: g.CreatedAt,
		})
	}
	return &webv1.ListCharacterGrantsResponse{Result: characterResultToProto(res), Grants: out}, nil
}

func adminCharacterDetailToProto(d characteradmin.Detail) *webv1.AdminCharacterDetail {
	ch := d.Character
	out := &webv1.AdminCharacterDetail{
		Summary: &webv1.AdminCharacterSummary{
			Id: d.CharacterID, AccountId: d.AccountID, AccountName: d.AccountName,
			Name: ch.Name, Slot: int32(ch.Slot), Class: int32(ch.Class),
			Level: ch.Level, Exp: ch.Exp, Coin: ch.Coin,
		},
		MaxHp: ch.MaxHp, Hp: ch.Hp, MaxMp: ch.MaxMp, Mp: ch.Mp,
		Str: int32(ch.Str), Intel: int32(ch.Int), Dex: int32(ch.Dex), Con: int32(ch.Con),
		LastCity: int32(ch.LastCity), SaveX: int32(ch.SaveX), SaveY: int32(ch.SaveY),
		LearnedSkill: ch.LearnedSkill,
		SkillBar:     []int32{int32(ch.SkillBar[0]), int32(ch.SkillBar[1]), int32(ch.SkillBar[2]), int32(ch.SkillBar[3])},
		Special:      []int32{int32(ch.Special[0]), int32(ch.Special[1]), int32(ch.Special[2]), int32(ch.Special[3])},
	}
	out.Equip = adminItemsToProto(ch.Equip, d.Items)
	out.Carry = adminItemsToProto(ch.Carry, d.Items)
	return out
}

func adminItemsToProto(items []domain.Item, info map[int16]characteradmin.ItemInfo) []*webv1.AdminCharacterItem {
	out := make([]*webv1.AdminCharacterItem, 0, len(items))
	for _, it := range items {
		meta := info[it.Index]
		out = append(out, &webv1.AdminCharacterItem{
			Slot: int32(it.Slot), ItemIndex: int32(it.Index),
			Name: meta.Name, IconUrl: meta.IconURL,
			Eff1: int32(it.Eff1), Effv1: int32(it.EffV1),
			Eff2: int32(it.Eff2), Effv2: int32(it.EffV2),
			Eff3: int32(it.Eff3), Effv3: int32(it.EffV3),
			ExpiresAt: it.ExpiresAt,
		})
	}
	return out
}

func characterResultToProto(r characteradmin.Result) webv1.AdminResult {
	switch r {
	case characteradmin.OK:
		return webv1.AdminResult_ADMIN_RESULT_OK
	case characteradmin.Forbidden:
		return webv1.AdminResult_ADMIN_RESULT_FORBIDDEN
	case characteradmin.NotFound:
		return webv1.AdminResult_ADMIN_RESULT_NOT_FOUND
	default:
		return webv1.AdminResult_ADMIN_RESULT_INVALID
	}
}
