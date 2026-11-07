package content

import (
	"fmt"
	"os"
)

// Dimensões dos mapas (data-formats.md §2). HeightMap.dat é 4096×4096 (16 MiB, 1
// byte/célula); AttributeMap.dat é 1024×1024 (1 MiB) — um atributo por bloco 4×4
// de células de altura.
const (
	HeightMapDim    = 4096
	AttributeMapDim = 1024
)

// Grid é um mapa de bytes quadrado e denso (HeightMap/AttributeMap), row-major: índice
// y*Dim + x (pHeightGrid[y][x]).
type Grid struct {
	Dim  int
	Data []byte
}

// At devolve o valor da célula em (x,y), ou 0 se fora dos limites.
func (g *Grid) At(x, y int) byte {
	if x < 0 || y < 0 || x >= g.Dim || y >= g.Dim {
		return 0
	}
	return g.Data[y*g.Dim+x]
}

// LoadGrid lê um mapa de bytes denso dim×dim e verifica seu tamanho exato (o arquivo
// é um asset, mantido binário — data-formats.md §2).
func LoadGrid(path string, dim int) (*Grid, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("content: read map %q: %w", path, err)
	}
	if want := dim * dim; len(data) != want {
		return nil, fmt.Errorf("content: map %q size %d != %d (dim %d)", path, len(data), want, dim)
	}
	return &Grid{Dim: dim, Data: data}, nil
}

// LoadHeightMap lê HeightMap.dat (4096²). AttributeMap.dat (1024²) é carregado
// com LoadGrid(path, AttributeMapDim); a semântica de bits por byte é UNVERIFIED
// (data-formats.md §2).
func LoadHeightMap(path string) (*Grid, error) { return LoadGrid(path, HeightMapDim) }
