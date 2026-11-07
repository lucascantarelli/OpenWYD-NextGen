// Package grpcsrv implementa o AccountService gRPC do dbServer (api/db/v1) sobre
// o store PostgreSQL. É a fronteira de persistência com que o tmServer fala via
// gRPC+mTLS (migration-plan.md §3.5); o CPSock legado nunca chega aqui.
package grpcsrv

import (
	"context"
	"errors"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	dbv1 "openwyd-nextgen/api/db/v1"
	"openwyd-nextgen/internal/crossover"
	"openwyd-nextgen/internal/domain"
	"openwyd-nextgen/internal/playcode"
	"openwyd-nextgen/internal/secret"
	"openwyd-nextgen/internal/store"
)

// Store é a superfície de persistência de que o serviço depende (satisfeita
// por *store.Store). Fica como interface para o serviço ser testável em unidade
// sem banco vivo.
type Store interface {
	LoadKefraState(ctx context.Context) (domain.KefraState, error)
	SaveKefraState(ctx context.Context, state domain.KefraState) error
	AccountByName(ctx context.Context, name string) (store.AccountAuth, error)
	AccountAuthByID(ctx context.Context, id int64) (store.AccountAuth, error)
	ConsumePlayCode(ctx context.Context, accountID int64, codeHash []byte) (bool, error)
	ListCharacters(ctx context.Context, accountID int64) ([]domain.Character, error)
	LoadCharacter(ctx context.Context, accountID int64, slot int) (domain.Character, error)
	CreateCharacter(ctx context.Context, accountID int64, ch domain.Character) (int64, error)
	CreateArchCharacter(ctx context.Context, accountID int64, ch domain.Character) (int64, int, error)
	DeleteCharacter(ctx context.Context, accountID int64, slot int) error
	PinHashByID(ctx context.Context, id int64) (string, error)
	SetPinHash(ctx context.Context, id int64, hash string) error
	SaveCharacter(ctx context.Context, accountID int64, ch domain.Character) error
	QuoteKingdomCape(ctx context.Context) (domain.KingdomCapeQuote, error)
	PurchaseKingdomCape(ctx context.Context, accountID, expectedRevision int64, kingdom uint8, ch domain.Character) (domain.KingdomCapeQuote, bool, error)
	LoadCargo(ctx context.Context, accountID int64) (int32, []domain.Item, error)
	SaveCargo(ctx context.Context, accountID int64, coin int32, items []domain.Item) error
	PendingItemDeliveries(ctx context.Context, accountID int64) ([]domain.Delivery, error)
	SaveCargoWithDeliveries(ctx context.Context, accountID int64, coin int32, items []domain.Item, deliveredIDs, lostIDs []int64) error
	SetBlockedByName(ctx context.Context, name string, blocked bool) error
	RecordDuelResult(ctx context.Context, winnerName, loserName string) error
	CreateGuild(ctx context.Context, accountID int64, slot int, characterName, guildName string, clan, citizen uint8, serverIndex int, cost int32) (domain.Guild, error)
	SetGuildMember(ctx context.Context, accountID int64, slot int, characterName string, guildID uint16, guildLevel uint8) error
	LeaveGuild(ctx context.Context, accountID int64, slot int) error
	PromoteGuildMember(ctx context.Context, guildID uint16, leaderAccountID int64, leaderSlot int, accountID int64, slot int, cost int32) (uint8, error)
	TransferGuildLeader(ctx context.Context, guildID uint16, oldAccountID int64, oldSlot int, newAccountID int64, newSlot int) error
	SetGuildRelation(ctx context.Context, guildID, targetGuildID uint16, kind domain.GuildRelationKind) error
	ListGuilds(ctx context.Context) ([]domain.Guild, error)
	ListGuildRelations(ctx context.Context) ([]domain.GuildRelation, error)
	LoadGuildZones(ctx context.Context) ([]domain.GuildZone, error)
	SaveGuildZone(ctx context.Context, zone domain.GuildZone) error
	LoadGuildTowerState(ctx context.Context) (domain.GuildTowerState, error)
	SaveGuildTowerState(ctx context.Context, state domain.GuildTowerState) error
	LoadCastleQuestState(ctx context.Context) (domain.CastleQuestState, error)
	SaveCastleQuestState(ctx context.Context, state domain.CastleQuestState) error
}

// Server implementa dbv1.AccountServiceServer.
type Server struct {
	dbv1.UnimplementedAccountServiceServer
	store Store
	// crossoverKey valida o token de troca de canal que o cliente carrega no campo
	// de senha depois de escolher outro canal (canais.md §8.1.2). Vazio
	// desativa o caminho, então uma chave ruim nunca vira bypass de login.
	crossoverKey string
}

// WithCrossoverKey liga o caminho de token de troca de servidor (W2PP_CROSSOVER_KEY).
func (s *Server) WithCrossoverKey(key string) *Server { s.crossoverKey = key; return s }

const (
	classMasterArch   = 1
	classMasterMortal = 2
	maxClass          = 4
	// pkPointNeutral é o valor limpo/neutro do byte PKPoint legado
	// (GetFunc.cpp, 75 = neutro; issue #210). Um personagem recém-criado precisa
	// começar aqui explicitamente: a lista de colunas do CreateCharacter é explícita,
	// então o zero do Go (0 = maximamente caótico) venceria o DEFAULT da coluna
	// no banco.
	pkPointNeutral = 75
)

// New monta um AccountService sobre o store recebido.
func New(s Store) *Server { return &Server{store: s} }

// AccountLogin autentica nome de conta + senha contra o hash argon2id
// guardado. Texto plano nunca é comparado (secret.VerifySecret).
func (s *Server) AccountLogin(ctx context.Context, req *dbv1.AccountLoginRequest) (*dbv1.AccountLoginResponse, error) {
	auth, err := s.store.AccountByName(ctx, req.GetAccountName())
	if errors.Is(err, store.ErrNotFound) {
		return &dbv1.AccountLoginResponse{Result: dbv1.LoginResult_LOGIN_RESULT_NO_ACCOUNT}, nil
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "account lookup: %v", err)
	}
	if auth.IsBlocked {
		return &dbv1.AccountLoginResponse{Result: dbv1.LoginResult_LOGIN_RESULT_BLOCKED}, nil
	}
	// Troca de canal (canais.md §8.1.2): o cliente que veio de outro canal manda um
	// token assinado no campo de senha em vez da senha. A assinatura cobre o nome
	// da conta, então um token não pode ser reusado por outra conta; um inválido ou
	// expirado é apenas uma senha errada.
	if tok := req.GetPassword(); s.crossoverKey != "" && crossover.IsToken(tok) {
		if _, ok := crossover.Verify(s.crossoverKey, req.GetAccountName(), tok, time.Now()); !ok {
			return &dbv1.AccountLoginResponse{Result: dbv1.LoginResult_LOGIN_RESULT_BAD_PASSWORD}, nil
		}
		return &dbv1.AccountLoginResponse{
			Result:    dbv1.LoginResult_LOGIN_RESULT_OK,
			AccountId: auth.ID,
			Role:      auth.Role,
		}, nil
	}
	ok, err := secret.VerifySecret(req.GetPassword(), auth.PassHash)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "verify password: %v", err)
	}
	// O cliente web loga com um código de uso único vindo do portal no campo de
	// senha (ADR 017 do cliente web). A senha é tentada primeiro, então uma conta
	// cuja senha real pareça um código continua funcionando.
	if !ok && playcode.Valid(req.GetPassword()) {
		ok, err = s.store.ConsumePlayCode(ctx, auth.ID, playcode.Hash(req.GetPassword()))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "consume play code: %v", err)
		}
	}
	if !ok {
		return &dbv1.AccountLoginResponse{Result: dbv1.LoginResult_LOGIN_RESULT_BAD_PASSWORD}, nil
	}
	return &dbv1.AccountLoginResponse{
		Result:    dbv1.LoginResult_LOGIN_RESULT_OK,
		AccountId: auth.ID,
		Role:      auth.Role, // levado ao tmServer para autorização de GM in-game (issue #122)
	}, nil
}

// ListCharacters devolve a projeção da seleção de personagens de uma conta.
func (s *Server) ListCharacters(ctx context.Context, req *dbv1.ListCharactersRequest) (*dbv1.ListCharactersResponse, error) {
	chars, err := s.store.ListCharacters(ctx, req.GetAccountId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list characters: %v", err)
	}
	out := make([]*dbv1.CharacterSummary, 0, len(chars))
	for _, ch := range chars {
		out = append(out, &dbv1.CharacterSummary{
			Slot:    int32(ch.Slot),
			Name:    ch.Name,
			Class:   int32(ch.Class),
			Level:   ch.Level,
			Exp:     ch.Exp,
			GuildId: uint32(ch.GuildID),
			Coin:    ch.Coin,
			MaxHp:   ch.MaxHp,
			Hp:      ch.Hp,
			MaxMp:   ch.MaxMp,
			Mp:      ch.Mp,
			Str:     int32(ch.Str),
			Int:     int32(ch.Int),
			Dex:     int32(ch.Dex),
			Con:     int32(ch.Con),
			Equip:   itemsToProto(ch.Equip),
		})
	}
	return &dbv1.ListCharactersResponse{Characters: out}, nil
}

// LoadCharacter carrega o estado de um personagem pelo slot.
func (s *Server) LoadCharacter(ctx context.Context, req *dbv1.LoadCharacterRequest) (*dbv1.LoadCharacterResponse, error) {
	ch, err := s.store.LoadCharacter(ctx, req.GetAccountId(), int(req.GetSlot()))
	if errors.Is(err, store.ErrNotFound) {
		return nil, status.Error(codes.NotFound, "character slot is empty")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load character: %v", err)
	}
	return &dbv1.LoadCharacterResponse{Character: characterToProto(ch)}, nil
}

// SaveCharacter persiste o estado vivo de um personagem (parcial; ver store.SaveCharacter).
func (s *Server) SaveCharacter(ctx context.Context, req *dbv1.SaveCharacterRequest) (*dbv1.SaveCharacterResponse, error) {
	if stage := req.GetCharacter().GetArchCrystalStage(); stage < 0 || stage > 4 {
		return nil, status.Error(codes.InvalidArgument, "arch crystal stage must be between 0 and 4")
	}
	ch := protoToCharacter(req.GetCharacter())
	err := s.store.SaveCharacter(ctx, req.GetAccountId(), ch)
	if errors.Is(err, store.ErrNotFound) {
		return &dbv1.SaveCharacterResponse{Ok: false}, nil
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "save character: %v", err)
	}
	return &dbv1.SaveCharacterResponse{Ok: true}, nil
}

// QuoteKingdomCape devolve o preço durável e a revisão usados pelos dois Reis.
func (s *Server) QuoteKingdomCape(ctx context.Context, _ *dbv1.QuoteKingdomCapeRequest) (*dbv1.QuoteKingdomCapeResponse, error) {
	q, err := s.store.QuoteKingdomCape(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "quote kingdom cape: %v", err)
	}
	return kingdomCapeQuoteToProto(q), nil
}

// PurchaseKingdomCape confirma o resultado preparado do personagem na revisão cotada.
func (s *Server) PurchaseKingdomCape(ctx context.Context, req *dbv1.PurchaseKingdomCapeRequest) (*dbv1.PurchaseKingdomCapeResponse, error) {
	q, ok, err := s.store.PurchaseKingdomCape(ctx, req.GetAccountId(), req.GetExpectedRevision(), uint8(req.GetKingdom()), protoToCharacter(req.GetCharacter()))
	if errors.Is(err, store.ErrNotFound) {
		return &dbv1.PurchaseKingdomCapeResponse{Quote: kingdomCapeQuoteToProto(q)}, nil
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "purchase kingdom cape: %v", err)
	}
	return &dbv1.PurchaseKingdomCapeResponse{Ok: ok, Quote: kingdomCapeQuoteToProto(q)}, nil
}

func kingdomCapeQuoteToProto(q domain.KingdomCapeQuote) *dbv1.QuoteKingdomCapeResponse {
	return &dbv1.QuoteKingdomCapeResponse{Revision: q.Revision, HekalotiaCost: q.HekalotiaCost, AkeloniaCost: q.AkeloniaCost}
}

// LoadCargo carrega o cargo compartilhado da conta (ouro + itens). Conta
// inexistente devolve NotFound.
func (s *Server) LoadCargo(ctx context.Context, req *dbv1.LoadCargoRequest) (*dbv1.LoadCargoResponse, error) {
	coin, items, err := s.store.LoadCargo(ctx, req.GetAccountId())
	if errors.Is(err, store.ErrNotFound) {
		return nil, status.Error(codes.NotFound, "account not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load cargo: %v", err)
	}
	return &dbv1.LoadCargoResponse{CargoCoin: coin, Items: itemsToProto(items)}, nil
}

// SaveCargo persiste ouro + itens do cargo compartilhado (substitui tudo).
func (s *Server) SaveCargo(ctx context.Context, req *dbv1.SaveCargoRequest) (*dbv1.SaveCargoResponse, error) {
	err := s.store.SaveCargo(ctx, req.GetAccountId(), req.GetCargoCoin(), protoToItems(req.GetItems()))
	if errors.Is(err, store.ErrNotFound) {
		return &dbv1.SaveCargoResponse{Ok: false}, nil
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "save cargo: %v", err)
	}
	return &dbv1.SaveCargoResponse{Ok: true}, nil
}

// ListPendingDeliveries devolve as concessões de item pendentes da conta na
// caixa delivery_queue (loja de donate web, issue #34).
func (s *Server) ListPendingDeliveries(ctx context.Context, req *dbv1.ListPendingDeliveriesRequest) (*dbv1.ListPendingDeliveriesResponse, error) {
	deliveries, err := s.store.PendingItemDeliveries(ctx, req.GetAccountId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list pending deliveries: %v", err)
	}
	out := make([]*dbv1.Delivery, 0, len(deliveries))
	for _, d := range deliveries {
		out = append(out, &dbv1.Delivery{Id: d.ID, Item: itemToProto(d.Item)})
	}
	return &dbv1.ListPendingDeliveriesResponse{Deliveries: out}, nil
}

// SaveCargoWithDeliveries persiste o cargo e marca as linhas drenadas da caixa
// como entregues/perdidas numa transação (a fronteira anti-duplicação da drenagem).
// Conta inexistente devolve ok=false.
func (s *Server) SaveCargoWithDeliveries(ctx context.Context, req *dbv1.SaveCargoWithDeliveriesRequest) (*dbv1.SaveCargoResponse, error) {
	err := s.store.SaveCargoWithDeliveries(ctx, req.GetAccountId(), req.GetCargoCoin(),
		protoToItems(req.GetItems()), req.GetDeliveredIds(), req.GetLostIds())
	if errors.Is(err, store.ErrNotFound) {
		return &dbv1.SaveCargoResponse{Ok: false}, nil
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "save cargo with deliveries: %v", err)
	}
	return &dbv1.SaveCargoResponse{Ok: true}, nil
}

// SetAccountBlocked inverte account.is_blocked por nome — o lado de escrita do
// comando in-game de ban/unban de GM (issue #122). Conta inexistente devolve ok=false.
func (s *Server) SetAccountBlocked(ctx context.Context, req *dbv1.SetAccountBlockedRequest) (*dbv1.SetAccountBlockedResponse, error) {
	err := s.store.SetBlockedByName(ctx, req.GetAccountName(), req.GetBlocked())
	if errors.Is(err, store.ErrNotFound) {
		return &dbv1.SetAccountBlockedResponse{Ok: false}, nil
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "set account blocked: %v", err)
	}
	return &dbv1.SetAccountBlockedResponse{Ok: true}, nil
}

// RecordDuelResult persiste o resultado de um duelo 1v1 (issue #118): as vitórias
// de winner_name e as derrotas de loser_name sobem de um em um. Nome que não
// resolve para um personagem é reportado como ok=false, não como erro.
func (s *Server) RecordDuelResult(ctx context.Context, req *dbv1.RecordDuelResultRequest) (*dbv1.RecordDuelResultResponse, error) {
	err := s.store.RecordDuelResult(ctx, req.GetWinnerName(), req.GetLoserName())
	if errors.Is(err, store.ErrNotFound) {
		return &dbv1.RecordDuelResultResponse{Ok: false}, nil
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "record duel result: %v", err)
	}
	return &dbv1.RecordDuelResultResponse{Ok: true}, nil
}

// CreateGuild cria uma guild e marca o solicitante como líder dela.
func (s *Server) CreateGuild(ctx context.Context, req *dbv1.CreateGuildRequest) (*dbv1.CreateGuildResponse, error) {
	g, err := s.store.CreateGuild(ctx, req.GetAccountId(), int(req.GetSlot()), req.GetCharacterName(),
		req.GetGuildName(), uint8(req.GetClan()), uint8(req.GetCitizen()), int(req.GetServerIndex()), req.GetCost())
	if errors.Is(err, store.ErrNotFound) || errors.Is(err, store.ErrConflict) || errors.Is(err, store.ErrNoFreeSlot) || isUniqueViolation(err) {
		return &dbv1.CreateGuildResponse{Ok: false}, nil
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create guild: %v", err)
	}
	return &dbv1.CreateGuildResponse{Ok: true, Guild: guildToProto(g)}, nil
}

// SetGuildMember define ou limpa a filiação de um personagem a uma guild.
func (s *Server) SetGuildMember(ctx context.Context, req *dbv1.SetGuildMemberRequest) (*dbv1.SetGuildMemberResponse, error) {
	err := s.store.SetGuildMember(ctx, req.GetAccountId(), int(req.GetSlot()), req.GetCharacterName(),
		uint16(req.GetGuildId()), uint8(req.GetGuildLevel()))
	if errors.Is(err, store.ErrNotFound) || errors.Is(err, store.ErrConflict) {
		return &dbv1.SetGuildMemberResponse{Ok: false}, nil
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "set guild member: %v", err)
	}
	return &dbv1.SetGuildMemberResponse{Ok: true}, nil
}

// LeaveGuild remove um personagem da guild dele.
func (s *Server) LeaveGuild(ctx context.Context, req *dbv1.LeaveGuildRequest) (*dbv1.SetGuildMemberResponse, error) {
	err := s.store.LeaveGuild(ctx, req.GetAccountId(), int(req.GetSlot()))
	if errors.Is(err, store.ErrNotFound) {
		return &dbv1.SetGuildMemberResponse{Ok: false}, nil
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "leave guild: %v", err)
	}
	return &dbv1.SetGuildMemberResponse{Ok: true}, nil
}

// PromoteGuildMember promove um membro para o primeiro posto de sub-líder livre.
func (s *Server) PromoteGuildMember(ctx context.Context, req *dbv1.PromoteGuildMemberRequest) (*dbv1.PromoteGuildMemberResponse, error) {
	level, err := s.store.PromoteGuildMember(ctx, uint16(req.GetGuildId()), req.GetLeaderAccountId(), int(req.GetLeaderSlot()),
		req.GetAccountId(), int(req.GetSlot()), req.GetCost())
	if errors.Is(err, store.ErrNotFound) || errors.Is(err, store.ErrConflict) || errors.Is(err, store.ErrNoFreeSlot) {
		return &dbv1.PromoteGuildMemberResponse{Ok: false}, nil
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "promote guild member: %v", err)
	}
	return &dbv1.PromoteGuildMemberResponse{Ok: true, GuildLevel: int32(level)}, nil
}

// TransferGuildLeader passa o posto 9 para outro membro.
func (s *Server) TransferGuildLeader(ctx context.Context, req *dbv1.TransferGuildLeaderRequest) (*dbv1.SetGuildMemberResponse, error) {
	err := s.store.TransferGuildLeader(ctx, uint16(req.GetGuildId()), req.GetOldAccountId(), int(req.GetOldSlot()),
		req.GetNewAccountId(), int(req.GetNewSlot()))
	if errors.Is(err, store.ErrNotFound) || errors.Is(err, store.ErrConflict) {
		return &dbv1.SetGuildMemberResponse{Ok: false}, nil
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "transfer guild leader: %v", err)
	}
	return &dbv1.SetGuildMemberResponse{Ok: true}, nil
}

// SetGuildRelation cria/atualiza ou limpa uma relação direcionada de aliança/guerra.
func (s *Server) SetGuildRelation(ctx context.Context, req *dbv1.SetGuildRelationRequest) (*dbv1.SetGuildRelationResponse, error) {
	err := s.store.SetGuildRelation(ctx, uint16(req.GetGuildId()), uint16(req.GetTargetGuildId()), relationKindFromProto(req.GetKind()))
	if errors.Is(err, store.ErrNotFound) || errors.Is(err, store.ErrConflict) {
		return &dbv1.SetGuildRelationResponse{Ok: false}, nil
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "set guild relation: %v", err)
	}
	return &dbv1.SetGuildRelationResponse{Ok: true}, nil
}

// ListGuilds devolve o snapshot do registro de guilds.
func (s *Server) ListGuilds(ctx context.Context, _ *dbv1.ListGuildsRequest) (*dbv1.ListGuildsResponse, error) {
	guilds, err := s.store.ListGuilds(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list guilds: %v", err)
	}
	return &dbv1.ListGuildsResponse{Guilds: guildsToProto(guilds)}, nil
}

// ListGuildRelations devolve todas as relações direcionadas de aliança/guerra.
func (s *Server) ListGuildRelations(ctx context.Context, _ *dbv1.ListGuildRelationsRequest) (*dbv1.ListGuildRelationsResponse, error) {
	relations, err := s.store.ListGuildRelations(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list guild relations: %v", err)
	}
	return &dbv1.ListGuildRelationsResponse{Relations: relationsToProto(relations)}, nil
}

// LoadGuildZones carrega o estado das zonas de cidade/guild.
func (s *Server) LoadGuildZones(ctx context.Context, _ *dbv1.LoadGuildZonesRequest) (*dbv1.LoadGuildZonesResponse, error) {
	zones, err := s.store.LoadGuildZones(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load guild zones: %v", err)
	}
	return &dbv1.LoadGuildZonesResponse{Zones: zonesToProto(zones)}, nil
}

// SaveGuildZone persiste uma linha de zona de cidade/guild.
func (s *Server) SaveGuildZone(ctx context.Context, req *dbv1.SaveGuildZoneRequest) (*dbv1.SaveGuildZoneResponse, error) {
	err := s.store.SaveGuildZone(ctx, zoneFromProto(req.GetZone()))
	if errors.Is(err, store.ErrNotFound) {
		return &dbv1.SaveGuildZoneResponse{Ok: false}, nil
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "save guild zone: %v", err)
	}
	return &dbv1.SaveGuildZoneResponse{Ok: true}, nil
}

// LoadGuildTowerState carrega a posse atual da GTorre.
func (s *Server) LoadGuildTowerState(ctx context.Context, _ *dbv1.LoadGuildTowerStateRequest) (*dbv1.LoadGuildTowerStateResponse, error) {
	st, err := s.store.LoadGuildTowerState(ctx)
	if errors.Is(err, store.ErrNotFound) {
		return &dbv1.LoadGuildTowerStateResponse{State: &dbv1.GuildTowerState{}}, nil
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load guild tower state: %v", err)
	}
	return &dbv1.LoadGuildTowerStateResponse{State: towerStateToProto(st)}, nil
}

// SaveGuildTowerState persiste a posse atual da GTorre.
func (s *Server) SaveGuildTowerState(ctx context.Context, req *dbv1.SaveGuildTowerStateRequest) (*dbv1.SaveGuildTowerStateResponse, error) {
	if err := s.store.SaveGuildTowerState(ctx, towerStateFromProto(req.GetState())); err != nil {
		return nil, status.Errorf(codes.Internal, "save guild tower state: %v", err)
	}
	return &dbv1.SaveGuildTowerStateResponse{Ok: true}, nil
}

// LoadCastleQuestState carrega o estado atual da quest de Castelo/Zakum.
func (s *Server) LoadCastleQuestState(ctx context.Context, _ *dbv1.LoadCastleQuestStateRequest) (*dbv1.LoadCastleQuestStateResponse, error) {
	st, err := s.store.LoadCastleQuestState(ctx)
	if errors.Is(err, store.ErrNotFound) {
		return &dbv1.LoadCastleQuestStateResponse{State: &dbv1.CastleQuestState{}}, nil
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load castle quest state: %v", err)
	}
	return &dbv1.LoadCastleQuestStateResponse{State: castleQuestStateToProto(st)}, nil
}

// SaveCastleQuestState persiste o estado atual da quest de Castelo/Zakum.
func (s *Server) SaveCastleQuestState(ctx context.Context, req *dbv1.SaveCastleQuestStateRequest) (*dbv1.SaveCastleQuestStateResponse, error) {
	if err := s.store.SaveCastleQuestState(ctx, castleQuestStateFromProto(req.GetState())); err != nil {
		return nil, status.Errorf(codes.Internal, "save castle quest state: %v", err)
	}
	return &dbv1.SaveCastleQuestStateResponse{Ok: true}, nil
}

// CreateCharacter cria um personagem num slot livre. Slot/nome ocupado (violação
// de unicidade) devolve ok=false, não erro.
func (s *Server) CreateCharacter(ctx context.Context, req *dbv1.CreateCharacterRequest) (*dbv1.CreateCharacterResponse, error) {
	// Inicializa um personagem jogável de nível 0 ("Nv 1" na tela; os templates
	// BaseMob trazem BaseScore.Level 0 @44). O DBSrv original os semeia a partir dos
	// templates BaseMob por classe (tmserver/content/DBsrv/run/BaseMob/{TK,FM,BM,HT}); até
	// eles estarem ligados, definimos stats base sensatos + HP/MP e uma posição inicial
	// para o personagem entrar no mundo. (UNVERIFIED: atributos base exatos por
	// classe e equipamento inicial / coordenadas de spawn — valores provisórios.)
	ch := domain.Character{
		Slot:        int(req.GetSlot()),
		Name:        req.GetName(),
		Class:       uint8(req.GetClass()),
		ClassMaster: classMasterMortal,
		Level:       0,
		Str:         12, Int: 12, Dex: 12, Con: 12,
		MaxHp: 100, Hp: 100, MaxMp: 100, Mp: 100,
		Coin:  1000000,           // ouro inicial (para a loja ser utilizável)
		SaveX: 2096, SaveY: 2096, // igual ao spawn do template BaseMob
		PKPoint: pkPointNeutral, // Pontos Caos: personagem novo começa limpo (issue #210)
	}
	id, err := s.store.CreateCharacter(ctx, req.GetAccountId(), ch)
	if isUniqueViolation(err) {
		return &dbv1.CreateCharacterResponse{Ok: false}, nil
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create character: %v", err)
	}
	return &dbv1.CreateCharacterResponse{Ok: true, CharacterId: id}, nil
}

// CreateArchCharacter cria o gêmeo ARCH no primeiro slot livre da conta. O
// personagem novo mantém o nome do Mortal e recebe o item de corpo Arch derivado
// de MortalFace, igual ao DBSrv::_MSG_DBCreateArchCharacter.
func (s *Server) CreateArchCharacter(ctx context.Context, req *dbv1.CreateArchCharacterRequest) (*dbv1.CreateArchCharacterResponse, error) {
	class := int(req.GetClass())
	if class < 0 || class >= maxClass {
		return &dbv1.CreateArchCharacterResponse{Ok: false}, nil
	}
	ch := domain.Character{
		Name:        req.GetName(),
		Class:       uint8(class),
		ClassMaster: classMasterArch,
		MortalLevel: uint16(req.GetMortalLevel()),
		Level:       0, // o gêmeo Arch também parte de g_pBaseSet (CFileDB.cpp:1448)
		Str:         12, Int: 12, Dex: 12, Con: 12,
		MaxHp: 100, Hp: 100, MaxMp: 100, Mp: 100,
		Coin:  1000000,
		SaveX: 2096, SaveY: 2096,
		PKPoint: pkPointNeutral, // Pontos Caos: personagem novo começa limpo (issue #210)
		Equip: []domain.Item{{
			Slot:  0,
			Index: int16(req.GetMortalFace() + 5 + int32(class)),
		}},
	}
	id, slot, err := s.store.CreateArchCharacter(ctx, req.GetAccountId(), ch)
	if errors.Is(err, store.ErrNoFreeSlot) || errors.Is(err, store.ErrNotFound) || isUniqueViolation(err) {
		return &dbv1.CreateArchCharacterResponse{Ok: false}, nil
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create arch character: %v", err)
	}
	return &dbv1.CreateArchCharacterResponse{Ok: true, CharacterId: id, Slot: int32(slot)}, nil
}

// DeleteCharacter remove um personagem depois de confirmar a senha da conta.
func (s *Server) DeleteCharacter(ctx context.Context, req *dbv1.DeleteCharacterRequest) (*dbv1.DeleteCharacterResponse, error) {
	auth, err := s.store.AccountAuthByID(ctx, req.GetAccountId())
	if errors.Is(err, store.ErrNotFound) {
		return &dbv1.DeleteCharacterResponse{Ok: false}, nil
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "account lookup: %v", err)
	}
	ok, err := secret.VerifySecret(req.GetPassword(), auth.PassHash)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "verify password: %v", err)
	}
	if !ok {
		return &dbv1.DeleteCharacterResponse{Ok: false}, nil
	}
	if err := s.store.DeleteCharacter(ctx, req.GetAccountId(), int(req.GetSlot())); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return &dbv1.DeleteCharacterResponse{Ok: false}, nil
		}
		return nil, status.Errorf(codes.Internal, "delete character: %v", err)
	}
	return &dbv1.DeleteCharacterResponse{Ok: true}, nil
}

// SetPin define (ou troca) o PIN numérico da conta, guardado só como hash
// argon2id (caminho de troca legado do _MSG_AccountSecure, mas com hash em vez de texto plano).
func (s *Server) SetPin(ctx context.Context, req *dbv1.SetPinRequest) (*dbv1.SetPinResponse, error) {
	hash, err := secret.HashSecret(req.GetPin())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "hash pin: %v", err)
	}
	if err := s.store.SetPinHash(ctx, req.GetAccountId(), hash); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return &dbv1.SetPinResponse{Ok: false}, nil
		}
		return nil, status.Errorf(codes.Internal, "set pin: %v", err)
	}
	return &dbv1.SetPinResponse{Ok: true}, nil
}

// VerifyPin confere um PIN numérico contra o hash guardado (caminho de verificação
// legado do _MSG_AccountSecure). Conta ainda sem PIN responde NOT_SET para o
// chamador oferecer configuração inicial em vez de tratar como PIN errado.
func (s *Server) VerifyPin(ctx context.Context, req *dbv1.VerifyPinRequest) (*dbv1.VerifyPinResponse, error) {
	hash, err := s.store.PinHashByID(ctx, req.GetAccountId())
	if errors.Is(err, store.ErrNotFound) {
		return &dbv1.VerifyPinResponse{Result: dbv1.PinResult_PIN_RESULT_NO_ACCOUNT}, nil
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "pin lookup: %v", err)
	}
	if hash == "" {
		return &dbv1.VerifyPinResponse{Result: dbv1.PinResult_PIN_RESULT_NOT_SET}, nil
	}
	ok, err := secret.VerifySecret(req.GetPin(), hash)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "verify pin: %v", err)
	}
	if !ok {
		return &dbv1.VerifyPinResponse{Result: dbv1.PinResult_PIN_RESULT_BAD_PIN}, nil
	}
	return &dbv1.VerifyPinResponse{Result: dbv1.PinResult_PIN_RESULT_OK}, nil
}
