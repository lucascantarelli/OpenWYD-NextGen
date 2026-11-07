package world

import "context"

// Billing é a porta que o gate de login de personagem usa para perguntar ao binServer se
// uma conta pode entrar no mundo (api/bin/v1). Esta é a NOVA fronteira de billing
// projetada na Fase 6 (migration-plan.md §4) — o link legado _AUTH_GAME não é
// reproduzido. A implementação real é um adaptador de cliente gRPC; o mundo
// depende só desta interface. Check é chamado FORA do loop via World.Go.
type Billing interface {
	Check(ctx context.Context, accountName string) (allowed bool, err error)
}

// AllowAllBilling é o gate padrão quando nenhum binServer está ligado (bring-up
// de free-to-play e testes): toda conta é permitida.
type AllowAllBilling struct{}

// Check sempre permite.
func (AllowAllBilling) Check(context.Context, string) (bool, error) { return true, nil }
