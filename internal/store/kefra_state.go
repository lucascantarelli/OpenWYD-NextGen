package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"openwyd-nextgen/internal/domain"
)

// LoadKefraState devolve revisão zero quando o evento nunca foi salvo.
func (s *Store) LoadKefraState(ctx context.Context) (domain.KefraState, error) {
	var st domain.KefraState
	err := s.pool.QueryRow(ctx, `SELECT defeated, next_spawn_unix, last_spawn_unix, revision FROM kefra_state WHERE id = 1`).Scan(&st.Defeated, &st.NextSpawnUnix, &st.LastSpawnUnix, &st.Revision)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.KefraState{}, nil
	}
	if err != nil {
		return domain.KefraState{}, fmt.Errorf("store: load kefra state: %w", err)
	}
	return st, nil
}

// SaveKefraState ignora revisões mais antigas ou repetidas, tornando as
// retentativas seguras mesmo quando uma RPC que estourou o prazo confirma depois da
// atualização seguinte ou do save de shutdown.
func (s *Store) SaveKefraState(ctx context.Context, st domain.KefraState) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO kefra_state (id, defeated, next_spawn_unix, last_spawn_unix, revision)
		VALUES (1, $1, $2, $3, $4) ON CONFLICT (id) DO UPDATE SET
		defeated = EXCLUDED.defeated, next_spawn_unix = EXCLUDED.next_spawn_unix,
		last_spawn_unix = EXCLUDED.last_spawn_unix, revision = EXCLUDED.revision
		WHERE kefra_state.revision < EXCLUDED.revision`, st.Defeated, st.NextSpawnUnix, st.LastSpawnUnix, st.Revision)
	if err != nil {
		return fmt.Errorf("store: save kefra state: %w", err)
	}
	return nil
}
