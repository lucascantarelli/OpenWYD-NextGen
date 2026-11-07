// Package worldcfg carrega a configuração de mundo gerenciada pelo portal para o
// tmServer. O fluxo é de mão única para edições de moderador (web -> Postgres ->
// dbServer -> tmServer); o tmServer só escreve de volta o progresso de evento.
package worldcfg

import "context"

// EventConfig é o snapshot global de eventos usado pelo laço do tmServer.
type EventConfig struct {
	Enabled            bool
	ItemIndex          int32
	Rate               int32
	StartIndex         int32
	CurrentIndex       int32
	EndIndex           int32
	Indexed            bool
	NoticeEnabled      bool
	DoubleExpEnabled   bool
	NewbieEventEnabled bool
}

// Snapshot é a configuração completa do mundo numa dada versão.
type Snapshot struct {
	Version int64
	Event   EventConfig
}

// Source é o lado de leitura/progresso do tmServer para a configuração de mundo
// gerenciada pelo portal.
type Source interface {
	Version(ctx context.Context) (int64, error)
	Snapshot(ctx context.Context) (Snapshot, error)
	UpdateProgress(ctx context.Context, expectedVersion int64, currentIndex int32) (bool, error)
}
