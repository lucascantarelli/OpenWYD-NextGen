package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// RecordInstanceFlags stores one boot snapshot of a service's process flags
// (issue 36). History is kept on purpose: comparing the current snapshot with the
// previous one is how an operator notices that a restart changed something.
func (s *Store) RecordInstanceFlags(ctx context.Context, server string, flags []byte) error {
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO server_instance_flags (server, flags) VALUES ($1, $2::jsonb)`,
		server, string(flags)); err != nil {
		return fmt.Errorf("store: record instance flags for %q: %w", server, err)
	}
	return nil
}

// LatestInstanceFlags returns the most recent snapshot for a service, plus when it
// was captured. ok=false means the service never recorded one.
func (s *Store) LatestInstanceFlags(ctx context.Context, server string) ([]byte, time.Time, bool, error) {
	var raw []byte
	var captured time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT flags, captured_at FROM server_instance_flags
		 WHERE server = $1
		 ORDER BY captured_at DESC, id DESC
		 LIMIT 1`, server).Scan(&raw, &captured)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, time.Time{}, false, nil
	}
	if err != nil {
		return nil, time.Time{}, false, fmt.Errorf("store: latest instance flags for %q: %w", server, err)
	}
	return raw, captured, true, nil
}
