package grpcsrv

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	dbv1 "openwyd-nextgen/api/db/v1"
	"openwyd-nextgen/internal/domain"
)

// ContentCatalogStore é a superfície de leitura de que o serviço de catálogo
// precisa (satisfeita por *store.Store). Issue 58, fase 2: são os catálogos que o
// importador semeia e o tmServer busca uma vez no boot.
type ContentCatalogStore interface {
	ListItemCatalog(ctx context.Context) ([]domain.ItemCatalogEntry, error)
	ListSkillCatalog(ctx context.Context) ([]domain.SkillCatalogEntry, error)
	ListNPCTemplates(ctx context.Context) ([]domain.NPCTemplateEntry, error)
	ListWorldItemSeeds(ctx context.Context) ([]domain.WorldItemSeedEntry, error)
}

// ContentCatalogServer implementa dbv1.ContentCatalogServiceServer. É
// estritamente somente leitura: escritas acontecem em tmserver/cmd/importcontent.
type ContentCatalogServer struct {
	dbv1.UnimplementedContentCatalogServiceServer
	store ContentCatalogStore
}

// NewContentCatalog monta o serviço somente leitura de catálogo de conteúdo.
func NewContentCatalog(s ContentCatalogStore) *ContentCatalogServer {
	return &ContentCatalogServer{store: s}
}

// nullIndex mapeia um valor anulável do catálogo para o sentinela do proto (-1).
func nullIndex(v *int16) int32 {
	if v == nil {
		return -1
	}
	return int32(*v)
}

// FetchItemCatalog devolve o catálogo de itens inteiro, ordenado por índice.
func (s *ContentCatalogServer) FetchItemCatalog(ctx context.Context, _ *dbv1.FetchItemCatalogRequest) (*dbv1.FetchItemCatalogResponse, error) {
	items, err := s.store.ListItemCatalog(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list item catalog: %v", err)
	}
	out := make([]*dbv1.CatalogItem, 0, len(items))
	for _, it := range items {
		msg := &dbv1.CatalogItem{
			ItemIndex:   int32(it.Index),
			Name:        it.Name,
			Price:       it.Price,
			Grade:       int32(it.Grade),
			UniqueId:    int32(it.UniqueID),
			Extra:       it.Extra,
			Pos:         int32(it.Pos),
			ReqLevel:    int32(it.ReqLevel),
			ReqStr:      int32(it.ReqStr),
			ReqInt:      int32(it.ReqInt),
			ReqDex:      int32(it.ReqDex),
			ReqCon:      int32(it.ReqCon),
			Volatile:    nullIndex(it.Volatile),
			AttackRange: nullIndex(it.AttackRange),
			Fields:      it.Fields,
			Sha256:      it.SHA256,
		}
		for _, eff := range it.Effects {
			msg.Effects = append(msg.Effects, &dbv1.CatalogEffect{Eff: int32(eff.Eff), Val: int32(eff.Val)})
		}
		out = append(out, msg)
	}
	return &dbv1.FetchItemCatalogResponse{Items: out}, nil
}

// FetchSkillCatalog devolve todas as skills, ordenadas por índice.
func (s *ContentCatalogServer) FetchSkillCatalog(ctx context.Context, _ *dbv1.FetchSkillCatalogRequest) (*dbv1.FetchSkillCatalogResponse, error) {
	skills, err := s.store.ListSkillCatalog(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list skill catalog: %v", err)
	}
	out := make([]*dbv1.CatalogSkill, 0, len(skills))
	for _, sp := range skills {
		out = append(out, &dbv1.CatalogSkill{
			SkillIndex:        int32(sp.Index),
			Name:              sp.Name,
			SkillPoint:        int32(sp.SkillPoint),
			TargetType:        int32(sp.TargetType),
			ManaSpent:         int32(sp.ManaSpent),
			DelayTicks:        int32(sp.Delay),
			RangeTiles:        int32(sp.Range),
			InstanceType:      int32(sp.InstanceType),
			InstanceValue:     int32(sp.InstanceValue),
			TickType:          int32(sp.TickType),
			TickValue:         int32(sp.TickValue),
			AffectType:        int32(sp.AffectType),
			AffectValue:       int32(sp.AffectValue),
			AffectTime:        int32(sp.AffectTime),
			InstanceAttribute: int32(sp.InstanceAttribute),
			TickAttribute:     int32(sp.TickAttribute),
			Aggressive:        int32(sp.Aggressive),
			MaxTarget:         int32(sp.MaxTarget),
			Party:             int32(sp.BParty),
			AffectResist:      int32(sp.AffectResist),
			Passive:           int32(sp.Passive),
		})
	}
	return &dbv1.FetchSkillCatalogResponse{Skills: out}, nil
}

// FetchNPCTemplates devolve todos os blobs de template, ordenados por nome.
func (s *ContentCatalogServer) FetchNPCTemplates(ctx context.Context, _ *dbv1.FetchNPCTemplatesRequest) (*dbv1.FetchNPCTemplatesResponse, error) {
	tpls, err := s.store.ListNPCTemplates(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list npc templates: %v", err)
	}
	out := make([]*dbv1.CatalogTemplate, 0, len(tpls))
	for _, t := range tpls {
		out = append(out, &dbv1.CatalogTemplate{
			TemplateName: t.Name,
			FileName:     t.FileName,
			SourceSize:   int32(t.SourceSize),
			Sha256:       t.SHA256,
			Blob:         t.Blob,
		})
	}
	return &dbv1.FetchNPCTemplatesResponse{Templates: out}, nil
}

// FetchWorldItems devolve os objetos estáticos do mundo na ordem de inserção.
func (s *ContentCatalogServer) FetchWorldItems(ctx context.Context, _ *dbv1.FetchWorldItemsRequest) (*dbv1.FetchWorldItemsResponse, error) {
	seeds, err := s.store.ListWorldItemSeeds(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list world items: %v", err)
	}
	out := make([]*dbv1.CatalogWorldItem, 0, len(seeds))
	for _, it := range seeds {
		out = append(out, &dbv1.CatalogWorldItem{
			ItemIndex: int32(it.Index),
			PosX:      int32(it.X),
			PosY:      int32(it.Y),
			Rotate:    int32(it.Rotate),
		})
	}
	return &dbv1.FetchWorldItemsResponse{Items: out}, nil
}
