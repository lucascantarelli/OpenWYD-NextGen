package convert

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"openwyd-nextgen/internal/domain"
	"openwyd-nextgen/internal/savefmt"
)

// FileResult é o resultado da conversão de um arquivo de conta.
type FileResult struct {
	Path    string
	Version savefmt.Version
	Account *domain.Account // só não nulo quando convertido
	Skipped string          // motivo, quando a versão não é modelada
	Err     error           // conversion/IO failure
}

// Report resume uma varredura de diretório.
type Report struct {
	Total     int
	Converted int
	Skipped   int
	Failed    int
	Results   []FileResult
}

// WalkAccounts percorre a árvore legada account/ e converte cada arquivo no
// formato atual (7952 bytes) num domain.Account. Arquivos num layout legado
// UNVERIFIED (4294 / 7500–7600) são registrados como Skipped, não como
// falhas — antes é preciso reverter o layout deles (data-formats.md §1.2). O
// erro devolvido é só para a varredura da árvore em si.
func WalkAccounts(root string) (*Report, error) {
	rep := &Report{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rep.Total++
		res := convertFile(path)
		switch {
		case res.Err != nil:
			rep.Failed++
		case res.Account != nil:
			rep.Converted++
		default:
			rep.Skipped++
		}
		rep.Results = append(rep.Results, res)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("convert: walk %q: %w", root, err)
	}
	sort.Slice(rep.Results, func(i, j int) bool { return rep.Results[i].Path < rep.Results[j].Path })
	return rep, nil
}

func convertFile(path string) FileResult {
	b, err := os.ReadFile(path)
	if err != nil {
		return FileResult{Path: path, Err: fmt.Errorf("read: %w", err)}
	}
	v := savefmt.DetectVersion(len(b))
	res := FileResult{Path: path, Version: v}
	if !v.Modelled() {
		res.Skipped = fmt.Sprintf("UNVERIFIED layout %s (size %d)", v, len(b))
		return res
	}
	af, err := savefmt.Decode(b)
	if err != nil {
		res.Err = fmt.Errorf("decode: %w", err)
		return res
	}
	acc, err := Account(af)
	if err != nil {
		res.Err = err
		return res
	}
	res.Account = &acc
	return res
}
