package content

import (
	"path/filepath"
	"testing"
)

func TestLoadBaseSummons(t *testing.T) {
	dir := filepath.Join("..", "..", "..", "tmserver", "content")
	templates, missing, err := LoadBaseSummons(dir)
	if err != nil {
		t.Skipf("motivo: conteúdo ausente — tmserver/content/TMsrv/run/BaseSummon não está nesta árvore: %v", err)
	}
	if len(templates) != 40 {
		t.Fatalf("templates = %d slots, want 40 (CNPCSummon::Initialize order)", len(templates))
	}
	// As 8 criaturas de evocação do BM são obrigatórias e têm o tamanho em bytes de
	// qualquer template STRUCT_MOB.
	for i := 0; i < bmSummonCount; i++ {
		if templates[i] == nil {
			t.Errorf("BM summon %d (%s) missing", i, baseSummonFiles[i])
			continue
		}
		if len(templates[i]) != BaseMobSize {
			t.Errorf("summon %d (%s) = %d bytes, want %d", i, baseSummonFiles[i], len(templates[i]), BaseMobSize)
		}
	}
	// O template de montaria com caixa corrigida precisa carregar no Linux (o C++ diz
	// "Dragao_menor"; o arquivo é Dragao_Menor).
	if templates[13] == nil {
		t.Errorf("summon 13 (Dragao_Menor) missing — case fix regressed?")
	}
	for _, name := range missing {
		t.Logf("optional summon missing: %s", name)
	}
}
