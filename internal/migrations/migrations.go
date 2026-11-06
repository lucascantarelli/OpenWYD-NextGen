// Package migrations embute os arquivos SQL versionados para que eles viajem no
// binário do dbserver, sem ferramenta externa de migração em tempo de execução.
package migrations

import "embed"

// FS guarda todos os arquivos *.up.sql / *.down.sql das migrações.
//
//go:embed *.sql
var FS embed.FS
