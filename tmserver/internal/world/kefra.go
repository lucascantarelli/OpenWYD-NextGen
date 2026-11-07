package world

import (
	"context"

	"openwyd-nextgen/internal/domain"
)

// KefraState é o ciclo semanal do chefe persistido, independente dos pacotes do cliente.
type KefraState = domain.KefraState

// IsKefraGenerator identifica o chefe e quatro receitas auxiliares do evento.
func IsKefraGenerator(idx int) bool { return idx >= KefraBossGenIndex && idx <= 400 }

// KefraState devolve o ciclo de posse do loop e se sua carga inicial terminou.
func (w *World) KefraState() (KefraState, bool) { return w.kefraState, w.kefraLoaded }

// SetKefraState publica um ciclo carregado ou alterado para portais e shutdown.
// Chame só do loop.
func (w *World) SetKefraState(st KefraState) { w.kefraState, w.kefraLoaded = st, true }

// LoadKefraState devolve um ciclo não inicializado em modo desenvolvimento.
func (NopPersistence) LoadKefraState(context.Context) (KefraState, error) { return KefraState{}, nil }

// SaveKefraState mantém o modo desenvolvimento livre de armazenamento durável.
func (NopPersistence) SaveKefraState(context.Context, KefraState) error { return nil }
