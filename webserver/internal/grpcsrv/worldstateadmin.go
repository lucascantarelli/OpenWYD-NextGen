package grpcsrv

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	webv1 "github.com/jeanluca/w2pp-openwyd/api/web/v1"
	"github.com/jeanluca/w2pp-openwyd/webserver/internal/worldstateadmin"
)

// WorldStateAdmin is the world-state surface the server depends on (satisfied by
// *worldstateadmin.Service). Kept as an interface so the server is unit-testable.
type WorldStateAdmin interface {
	Get(ctx context.Context, moderatorID int64) (worldstateadmin.Result, worldstateadmin.Snapshot, error)
	SetCityTax(ctx context.Context, moderatorID int64, zone, tax int) (worldstateadmin.Result, error)
	SetKefraDefeated(ctx context.Context, moderatorID int64, defeated bool) (worldstateadmin.Result, error)
	SetGuildMaster(ctx context.Context, moderatorID int64, guildID int32, characterID int64, reason string) (worldstateadmin.Result, string, error)
	RemoveGuildMember(ctx context.Context, moderatorID int64, guildID int32, characterID int64, reason string) (worldstateadmin.Result, string, error)
	SetGuildRelation(ctx context.Context, moderatorID int64, guildID, targetGuildID int32, kind int, reason string) (worldstateadmin.Result, string, error)
	SetZoneOwner(ctx context.Context, moderatorID int64, zone, guildID int32, reason string) (worldstateadmin.Result, string, error)
	SetTowerOwner(ctx context.Context, moderatorID int64, guildID int32, reason string) (worldstateadmin.Result, string, error)
}

// WorldStateAdminServer implements webv1.WorldStateAdminServiceServer.
type WorldStateAdminServer struct {
	webv1.UnimplementedWorldStateAdminServiceServer
	admin WorldStateAdmin
}

// NewWorldStateAdmin builds the WorldStateAdminService over the given logic.
func NewWorldStateAdmin(a WorldStateAdmin) *WorldStateAdminServer {
	return &WorldStateAdminServer{admin: a}
}

// GetWorldState returns guilds, relations, kingdom zones, tower, castle and Kefra
// in one snapshot.
func (s *WorldStateAdminServer) GetWorldState(ctx context.Context, req *webv1.GetWorldStateRequest) (*webv1.GetWorldStateResponse, error) {
	res, snap, err := s.admin.Get(ctx, req.GetModeratorId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get world state: %v", err)
	}

	guilds := make([]*webv1.AdminGuild, 0, len(snap.Guilds))
	for _, g := range snap.Guilds {
		members := make([]*webv1.AdminGuildMember, 0, len(g.Members))
		for _, m := range g.Members {
			members = append(members, &webv1.AdminGuildMember{
				CharacterId: m.CharacterID, AccountId: m.AccountID,
				Slot: int32(m.Slot), Name: m.Name, GuildLevel: int32(m.GuildLevel),
			})
		}
		guilds = append(guilds, &webv1.AdminGuild{
			Id: int32(g.ID), Name: g.Name, Clan: int32(g.Clan), Fame: g.Fame,
			Citizen: int32(g.Citizen), MasterName: g.MasterName,
			Members: members, CityZones: g.CityZones, TowerOwner: g.TowerOwner,
		})
	}

	relations := make([]*webv1.AdminGuildRelation, 0, len(snap.Relations))
	for _, r := range snap.Relations {
		relations = append(relations, &webv1.AdminGuildRelation{
			GuildId: r.GuildID, GuildName: r.GuildName,
			TargetGuildId: r.TargetGuildID, TargetName: r.TargetName, Kind: r.Kind,
		})
	}

	zones := make([]*webv1.AdminKingdomZone, 0, len(snap.Zones))
	for _, z := range snap.Zones {
		zones = append(zones, &webv1.AdminKingdomZone{
			Zone: int32(z.Zone), ChargeGuild: z.ChargeGuild, ChargeGuildName: z.ChargeGuildName,
			ChallengeGuild: z.ChallengeGuild, ChallengeGuildName: z.ChallengeGuildName,
			Clan: int32(z.Clan), Victory: int32(z.Victory), CityTax: int32(z.CityTax),
			ChallengeMoney: z.ChallengeMoney, TaxVault: z.TaxVault,
		})
	}

	return &webv1.GetWorldStateResponse{
		Result:           worldStateResultToProto(res),
		Guilds:           guilds,
		Relations:        relations,
		Zones:            zones,
		TowerOwnerGuild:  snap.TowerOwnerGuild,
		TowerOwnerName:   snap.TowerOwnerName,
		TowerUpdatedUnix: snap.TowerUpdatedUnix,
		Castle: &webv1.AdminCastleQuest{
			Level: snap.Castle.Level, TimeLeft: snap.Castle.TimeLeft,
			Clear: snap.Castle.Clear, LeaderName: snap.Castle.LeaderName,
		},
		Kefra: &webv1.AdminKefra{
			Initialized: snap.Kefra.Initialized, Defeated: snap.Kefra.Defeated,
			NextSpawnUnix: snap.Kefra.NextSpawnUnix, LastSpawnUnix: snap.Kefra.LastSpawnUnix,
			Revision: snap.Kefra.Revision,
		},
	}, nil
}

// SetCityTax rewrites one kingdom zone with the new tax.
func (s *WorldStateAdminServer) SetCityTax(ctx context.Context, req *webv1.SetCityTaxRequest) (*webv1.SetWorldStateResponse, error) {
	res, err := s.admin.SetCityTax(ctx, req.GetModeratorId(), int(req.GetZone()), int(req.GetCityTax()))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "set city tax: %v", err)
	}
	out := &webv1.SetWorldStateResponse{Result: worldStateResultToProto(res)}
	switch res {
	case worldstateadmin.Invalid:
		out.Message = "taxa fora da faixa aceita (0..30) ou zona inexistente"
	case worldstateadmin.NotFound:
		out.Message = "esta zona ainda não existe em guild_zone"
	case worldstateadmin.Forbidden:
		out.Message = "sua conta não pode alterar o estado do mundo"
	}
	return out, nil
}

// SetKefraDefeated flips the Kefra flag that gates the Desert portal.
func (s *WorldStateAdminServer) SetKefraDefeated(ctx context.Context, req *webv1.SetKefraDefeatedRequest) (*webv1.SetWorldStateResponse, error) {
	res, err := s.admin.SetKefraDefeated(ctx, req.GetModeratorId(), req.GetDefeated())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "set kefra defeated: %v", err)
	}
	out := &webv1.SetWorldStateResponse{Result: worldStateResultToProto(res)}
	switch res {
	case worldstateadmin.NotFound:
		out.Message = "o tmServer ainda não gravou um ciclo do Kefra; sem a linha, defeat/next_spawn não têm valor válido"
	case worldstateadmin.Forbidden:
		out.Message = "sua conta não pode alterar o estado do mundo"
	}
	return out, nil
}

func worldStateResultToProto(r worldstateadmin.Result) webv1.AdminResult {
	switch r {
	case worldstateadmin.OK:
		return webv1.AdminResult_ADMIN_RESULT_OK
	case worldstateadmin.Forbidden:
		return webv1.AdminResult_ADMIN_RESULT_FORBIDDEN
	case worldstateadmin.NotFound:
		return webv1.AdminResult_ADMIN_RESULT_NOT_FOUND
	default:
		return webv1.AdminResult_ADMIN_RESULT_INVALID
	}
}
