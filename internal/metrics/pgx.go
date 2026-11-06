package metrics

import (
	"context"
	"math"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RegisterPGXPool expõe o estado do pool no scrape. Nenhuma chamada acontece no
// laço do jogo; o custo é pago só quando o Prometheus coleta.
func RegisterPGXPool(r *Registry, service string, pool *pgxpool.Pool) {
	l := Labels{"service": service}
	r.GaugeFunc("db_pool_acquired_connections", "Conexões adquiridas do pool.", l,
		func() float64 { return float64(pool.Stat().AcquiredConns()) })
	r.GaugeFunc("db_pool_idle_connections", "Conexões ociosas do pool.", l,
		func() float64 { return float64(pool.Stat().IdleConns()) })
	r.GaugeFunc("db_pool_total_connections", "Conexões totais do pool.", l,
		func() float64 { return float64(pool.Stat().TotalConns()) })
	r.GaugeFunc("db_pool_max_connections", "Limite de conexões do pool.", l,
		func() float64 { return float64(pool.Stat().MaxConns()) })
}

// RegisterTableRows expõe a contagem de uma tabela/fila (ex.: delivery_queue) no
// scrape, com timeout curto; erro vira NaN em vez de derrubar o endpoint.
func RegisterTableRows(r *Registry, service, name, help, query string, pool *pgxpool.Pool) {
	r.GaugeFunc(name, help, Labels{"service": service}, func() float64 {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		var n int64
		if err := pool.QueryRow(ctx, query).Scan(&n); err != nil {
			return math.NaN()
		}
		return float64(n)
	})
}
