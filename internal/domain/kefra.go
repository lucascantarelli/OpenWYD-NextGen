package domain

// KefraState é o ciclo semanal durável do chefe. Revision zero significa que
// ainda não há ciclo salvo; os instantes são segundos Unix agendados pelo
// calendário local.
type KefraState struct {
	Defeated      bool
	NextSpawnUnix int64
	LastSpawnUnix int64
	Revision      int64
}
