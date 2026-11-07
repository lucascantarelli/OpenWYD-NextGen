package content

import (
	"fmt"
	"os"
	"path/filepath"
)

// baseSummonFiles é a ordem de templates de CNPCSummon::Initialize (CNPCGene.cpp:
// 686-960): mSummon.Mob[i] é carregado de tmserver/content/TMsrv/run/BaseSummon/<name>.
// Os índices 0..7 são as criaturas de evocação do BM (skills 56-63, InstanceValue-1);
// 8+ são invocações de item/montaria/evento. O índice 13 é "Dragao_menor" no C++, mas
// o arquivo de conteúdo é "Dragao_Menor" — o Windows abria sem diferenciar caixa,
// o Linux não, então a caixa corrigida vive aqui.
var baseSummonFiles = [40]string{
	"Condor", "Javali", "Lobo", "Urso", "Tigre", "Gorila", "Dragao_Negro", "Succubus",
	"Porco", "Javali_", "Porco", "Javali", "Lobo", "Dragao_Menor", "Urso", "Dente_de_Sabre",
	"Sem_Sela_N", "Fantasma_N", "Leve_N", "Equip_N", "Andaluz_N",
	"Sem_Sela_B", "Fantasma_B", "Leve_B", "Equip_B", "Andaluz_B",
	"Fenrir", "Dragao", "FenrirSombra", "Tigre_de_Fogo", "Dragao_Vermelho",
	"Unicornio", "Pegasus", "Unisus", "Grifo", "Hipogrifo", "Grifo_Sangrento",
	"Svadilfire", "Sleipnir", "Pantera_Negra",
}

// bmSummonCount é quantos templates iniciais as evocações do BM precisam: esses são
// obrigatórios no boot; a cauda de montaria/evento (8..39) tolera ausência (slot
// nil + aviso ao chamador), já que nada os invoca ainda.
const bmSummonCount = 8

// LoadBaseSummons carrega os templates STRUCT_MOB de invocação (816 bytes cada, mesmo
// formato cru dos templates de classe BaseMob). O slice devolvido é indexado por
// id de invocação (SummonID de GenerateSummon = InstanceValue-1 da spell); templates
// opcionais ausentes deixam um slot nil e seus nomes voltam em missing.
func LoadBaseSummons(dir string) (templates [][]byte, missing []string, err error) {
	templates = make([][]byte, len(baseSummonFiles))
	for i, name := range baseSummonFiles {
		p := filepath.Join(dir, "TMsrv", "run", "BaseSummon", name)
		b, rerr := os.ReadFile(p)
		if rerr != nil {
			if i < bmSummonCount {
				return nil, nil, fmt.Errorf("content: base summon %s: %w", name, rerr)
			}
			missing = append(missing, name)
			continue
		}
		if len(b) != BaseMobSize {
			return nil, nil, fmt.Errorf("content: base summon %s = %d bytes, want %d", name, len(b), BaseMobSize)
		}
		templates[i] = b
	}
	return templates, missing, nil
}
