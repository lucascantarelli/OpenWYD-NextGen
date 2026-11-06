// Package npctemplate resolve os arquivos crus de template STRUCT_MOB da árvore
// de conteúdo em tmserver/content/TMsrv/run/npc.
package npctemplate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"openwyd-nextgen/internal/savefmt"
)

const (
	// Size é sizeof(STRUCT_MOB), o formato cru de template de NPC/mob.
	Size = savefmt.MobSize
)

// Resolution identifica o arquivo concreto escolhido para o template pedido.
type Resolution struct {
	// Name é o nome real do arquivo em tmserver/content/TMsrv/run/npc.
	Name string
	// Path é o caminho absoluto ou relativo usado para ler o arquivo.
	Path string
	// Version é o layout em disco que Load encontrou, para o chamador informar
	// quanto do catálogo precisou de conversão. Zero quando a resolução falha.
	Version savefmt.MobVersion
}

// Dir devolve o diretório de templates de NPC para uma raiz de conteúdo.
func Dir(contentDir string) string {
	return filepath.Join(contentDir, "TMsrv", "run", "npc")
}

// Resolve mapeia um nome de template do conteúdo/config para o arquivo real em
// tmserver/content/TMsrv/run/npc. Primeiro honra o nome exato do Linux; depois cai
// na comparação parecida com a do Windows, que os dados originais exigem: sem
// distinção de caixa e ignorando pontos finais.
func Resolve(contentDir, name string) (Resolution, error) {
	if name == "" {
		return Resolution{}, fmt.Errorf("npctemplate: empty template name")
	}
	if filepath.Base(name) != name || strings.ContainsAny(name, `/\`) {
		return Resolution{}, fmt.Errorf("npctemplate: invalid template name %q", name)
	}
	dir := Dir(contentDir)
	exactPath := filepath.Join(dir, name)
	if st, err := os.Stat(exactPath); err == nil {
		if st.IsDir() {
			return Resolution{}, fmt.Errorf("npctemplate: %s is a directory", exactPath)
		}
		return Resolution{Name: name, Path: exactPath}, nil
	} else if !os.IsNotExist(err) {
		return Resolution{}, fmt.Errorf("npctemplate: stat %s: %w", exactPath, err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return Resolution{}, fmt.Errorf("npctemplate: read %s: %w", dir, err)
	}
	want := canonicalName(name)
	var matches []Resolution
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		entryName := entry.Name()
		if canonicalName(entryName) == want {
			matches = append(matches, Resolution{Name: entryName, Path: filepath.Join(dir, entryName)})
		}
	}
	switch len(matches) {
	case 0:
		return Resolution{}, fmt.Errorf("npctemplate: template %q not found in %s", name, dir)
	case 1:
		return matches[0], nil
	default:
		names := make([]string, len(matches))
		for i, match := range matches {
			names[i] = match.Name
		}
		return Resolution{}, fmt.Errorf("npctemplate: template %q is ambiguous in %s: %s", name, dir, strings.Join(names, ", "))
	}
}

// Load lê um template resolvido por Resolve e devolve um blob STRUCT_MOB
// canônico de Size bytes. Os layouts legados de 756/920 bytes que o conteúdo
// embarca no mesmo diretório são normalizados aqui (savefmt.NormalizeMob), para
// que tudo a jusante — que indexa o layout de 816 bytes direto — veja sempre a
// forma canônica.
func Load(contentDir, name string) ([]byte, Resolution, error) {
	res, err := Resolve(contentDir, name)
	if err != nil {
		return nil, Resolution{}, err
	}
	b, err := os.ReadFile(res.Path)
	if err != nil {
		return nil, Resolution{}, fmt.Errorf("npctemplate: read %s: %w", res.Path, err)
	}
	canonical, version, err := savefmt.NormalizeMob(b)
	if err != nil {
		return nil, Resolution{}, fmt.Errorf("npctemplate: npc %s = %d bytes: %w", res.Name, len(b), err)
	}
	res.Version = version
	return canonical, res, nil
}

func canonicalName(name string) string {
	return strings.ToLower(strings.TrimRight(name, "."))
}
