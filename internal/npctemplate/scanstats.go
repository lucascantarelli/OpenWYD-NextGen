package npctemplate

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"

	"openwyd-nextgen/internal/savefmt"
)

// ScanStats conta uma varredura do diretório de templates por layout em disco.
//
// Existe para os serviços reportarem o catálogo numa única linha de resumo, em
// vez das centenas de WARN por arquivo que a árvore de layouts misturados
// produzia: com aviso por arquivo, um deploy que perdeu 11% do catálogo em
// silêncio ainda parecia saudável no log.
type ScanStats struct {
	Current816      int // savefmt.MobVersionCurrent
	Legacy756       int // savefmt.MobVersionLegacy756
	Legacy756Padded int // savefmt.MobVersionLegacy756Padded
	Unreadable      int // falha em os.ReadFile/os.Stat
	Rejected        int // leu bem, mas não é layout conhecido (arquivo que não é template)
}

// Count registra um arquivo lido com sucesso sob o layout detectado.
func (s *ScanStats) Count(v savefmt.MobVersion) {
	switch v {
	case savefmt.MobVersionCurrent:
		s.Current816++
	case savefmt.MobVersionLegacy756:
		s.Legacy756++
	case savefmt.MobVersionLegacy756Padded:
		s.Legacy756Padded++
	default:
		s.Rejected++
	}
}

// CountUnreadable registra um arquivo que não pôde ser lido.
func (s *ScanStats) CountUnreadable() { s.Unreadable++ }

// Loaded informa quantos arquivos decodificaram como template utilizável.
func (s ScanStats) Loaded() int {
	return s.Current816 + s.Legacy756 + s.Legacy756Padded
}

// Total informa quantos arquivos foram considerados.
func (s ScanStats) Total() int { return s.Loaded() + s.Unreadable + s.Rejected }

// LogValue rende a contagem como um atributo agrupado do slog:
//
//	logger.Info("npc template catalog", "layouts", stats)
func (s ScanStats) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Int("total", s.Total()),
		slog.Int("loaded", s.Loaded()),
		slog.Int("current816", s.Current816),
		slog.Int("legacy756", s.Legacy756),
		slog.Int("legacy756_padded920", s.Legacy756Padded),
		slog.Int("unreadable", s.Unreadable),
		slog.Int("rejected", s.Rejected),
	)
}

// ScanBlobs é o par em memória de ScanDir: decodifica os blobs STRUCT_MOB
// canônicos que o banco guarda (npc_template) e chama visit para cada um que
// decodifica como layout conhecido. Os nomes são visitados em ordem, então os
// catálogos construídos a partir deles são determinísticos sem uma segunda
// ordenação.
func ScanBlobs(blobs map[string][]byte, visit func(name string, mob savefmt.Mob)) ScanStats {
	var stats ScanStats
	names := make([]string, 0, len(blobs))
	for name := range blobs {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		mob, version, err := savefmt.DecodeMobAny(blobs[name])
		stats.Count(version)
		if err != nil {
			continue
		}
		visit(name, mob)
	}
	return stats
}

// ScanDir percorre o diretório de templates de uma raiz de conteúdo e chama
// visit para cada arquivo que decodifica como STRUCT_MOB em qualquer layout que
// savefmt conheça — o canônico de 816 bytes e as formas legadas de 756/920 bytes
// que o conteúdo embarca. Arquivo ilegível, ou que não é template, entra na
// contagem devolvida em vez de derrubar a varredura: é legítimo o diretório ter
// arquivos que não são template.
//
// É a única varredura compartilhada pelos três catálogos do moderador
// (npctemplates, mobtemplates e droptool do webserver), que diferem só no que
// constroem a partir de cada mob decodificado. As entradas chegam na ordem de
// os.ReadDir, ou seja, ordenadas por nome de arquivo.
func ScanDir(contentDir string, logger *slog.Logger, visit func(name string, mob savefmt.Mob)) (ScanStats, error) {
	var stats ScanStats
	dir := Dir(contentDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return stats, fmt.Errorf("npctemplate: read %s: %w", dir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			logger.Warn("npc template read failed", "name", name, "err", err)
			stats.CountUnreadable()
			continue
		}
		mob, version, err := savefmt.DecodeMobAny(b)
		stats.Count(version)
		if err != nil {
			continue
		}
		visit(name, mob)
	}
	return stats, nil
}
