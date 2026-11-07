package content

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// InitItem é um objeto estático do mundo vindo de InitItem.csv (STRUCT_INITITEM): um
// portão/porta/decoração que o servidor gera no boot. As colunas são Index,PosX,PosY,Rotate
// (BASE_ReadInitItem, Basedef.cpp:6641); uma 5ª coluna final é ignorada, e uma
// linha cujo Index é -1 é pulada, espelhando o laço sscanf legado.
type InitItem struct {
	Index  int16
	PosX   int16
	PosY   int16
	Rotate int16
}

// LoadInitItems lê TMsrv/run/InitItem.csv para a lista estática de objetos do mundo que o
// mundo semeia no boot (portões/portas). As linhas são separadas por vírgula
// "Index,PosX,PosY,Rotate[,ignorado]"; linhas em branco e linhas com Index==-1 são puladas.
// Um campo numérico malformado é erro fatal (o arquivo é pequeno e escrito à mão).
func LoadInitItems(path string) ([]InitItem, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("content: open InitItem: %w", err)
	}
	defer f.Close()

	var out []InitItem
	sc := bufio.NewScanner(f)
	for line := 1; sc.Scan(); line++ {
		text := strings.TrimSpace(sc.Text())
		if text == "" || strings.HasPrefix(text, "//") {
			continue
		}
		fields := strings.Split(text, ",")
		if len(fields) < 4 {
			return nil, fmt.Errorf("content: InitItem line %d: want >=4 fields, got %d", line, len(fields))
		}
		vals := make([]int, 4)
		for i := 0; i < 4; i++ {
			v, perr := strconv.Atoi(strings.TrimSpace(fields[i]))
			if perr != nil {
				return nil, fmt.Errorf("content: InitItem line %d field %d: %w", line, i+1, perr)
			}
			vals[i] = v
		}
		if vals[0] == -1 { // legacy sentinel: skip
			continue
		}
		out = append(out, InitItem{
			Index:  int16(vals[0]),
			PosX:   int16(vals[1]),
			PosY:   int16(vals[2]),
			Rotate: int16(vals[3]),
		})
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("content: read InitItem: %w", err)
	}
	return out, nil
}
