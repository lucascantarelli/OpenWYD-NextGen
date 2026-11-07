package content

import (
	"fmt"
	"os"
	"path/filepath"
)

// BaseMobSize é sizeof(STRUCT_MOB): o template de personagem novo por classe é um
// dump cru de 816 bytes de STRUCT_MOB (stats, equipamento inicial, skills, spawn).
const BaseMobSize = 816

// baseMobFiles mapeia o índice de classe para o nome do arquivo de template BaseMob
// (tmserver/content/DBsrv/run/BaseMob/{TK,FM,BM,HT}); confirmado pelo byte
// Class @20 de cada template: TK=0, FM=1, BM=2, HT=3.
var baseMobFiles = map[int]string{0: "TK", 1: "FM", 2: "BM", 3: "HT"}

// LoadBaseMobs carrega os templates STRUCT_MOB por classe usados para renderizar e
// gerar um personagem recém-criado. Devolve índice de classe → STRUCT_MOB cru de 816 bytes.
func LoadBaseMobs(dir string) (map[int][]byte, error) {
	out := make(map[int][]byte, len(baseMobFiles))
	for class, name := range baseMobFiles {
		p := filepath.Join(dir, "DBsrv", "run", "BaseMob", name)
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, fmt.Errorf("content: base mob %s: %w", name, err)
		}
		if len(b) != BaseMobSize {
			return nil, fmt.Errorf("content: base mob %s = %d bytes, want %d", name, len(b), BaseMobSize)
		}
		out[class] = b
	}
	return out, nil
}
