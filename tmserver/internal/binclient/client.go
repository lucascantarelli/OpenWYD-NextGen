// Package binclient adapta o BillingService gRPC do binServer (api/bin/v1) à
// porta world.Billing. O gate de login de personagem o chama (fora do loop, via
// World.Go) para decidir se uma conta pode entrar no mundo (migration-plan.md
// §4, Fase 6).
package binclient

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	binv1 "openwyd-nextgen/api/bin/v1"
	"openwyd-nextgen/tmserver/internal/world"
)

// Client é um world.Billing apoiado no binServer.
type Client struct {
	api binv1.BillingServiceClient
}

// New embrulha uma conexão gRPC como gate de Billing.
func New(conn grpc.ClientConnInterface) *Client {
	return &Client{api: binv1.NewBillingServiceClient(conn)}
}

var _ world.Billing = (*Client)(nil)

// Check pergunta ao binServer se a conta pode entrar no mundo.
func (c *Client) Check(ctx context.Context, accountName string) (bool, error) {
	resp, err := c.api.CheckBilling(ctx, &binv1.CheckBillingRequest{AccountName: accountName})
	if err != nil {
		return false, fmt.Errorf("binclient: check billing: %w", err)
	}
	return resp.GetAllowed(), nil
}
