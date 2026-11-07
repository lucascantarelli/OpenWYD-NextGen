package store

import (
	"context"
	"fmt"
)

// SetKefraDefeated vira o estado da Kefra a partir da superfície de admin (issue 45).
//
// A revisão sobe no mesmo comando: SaveKefraState recusa uma escrita
// cuja revisão não é maior que a guardada, então o próximo save do mundo em
// execução (que usa sua própria revisão em memória, menor) não pode desfazer
// esta edição em silêncio. O que ele não pode fazer é mudar a luta ao vivo: a
// Kefra vive na goroutine do mundo, então a flag vale a partir do próximo boot.
// applied=false significa que a linha ainda não existe (o tmServer nunca salvou um spawn).
func (s *Store) SetKefraDefeated(ctx context.Context, defeated bool) (bool, error) {
	tag, err := s.pool.Exec(ctx,
		`UPDATE kefra_state SET defeated = $1, revision = revision + 1 WHERE id = 1`,
		defeated,
	)
	if err != nil {
		return false, fmt.Errorf("store: set kefra defeated: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}
