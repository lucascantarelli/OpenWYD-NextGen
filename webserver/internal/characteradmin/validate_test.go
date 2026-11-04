package characteradmin

import (
	"testing"

	"github.com/jeanluca/w2pp-openwyd/internal/domain"
)

// TestValidateAdds: a grant may carry no add (zero) or any code known to
// ItemEffect.h, and must reject codes the client cannot interpret — 98 is inside
// the gap between EF_DONATE (91) and EF_GRADE0 (100).
func TestValidateAdds(t *testing.T) {
	if !ValidateAdds(domain.Item{}) {
		t.Fatal("item sem adds foi rejeitado")
	}
	if !ValidateAdds(domain.Item{Eff1: 2, EffV1: 70, Eff2: 43, EffV2: 253}) {
		t.Fatal("Dano + Refino foi rejeitado")
	}
	if ValidateAdds(domain.Item{Eff1: 98}) {
		t.Fatal("código 98 (fora do ItemEffect.h) foi aceito")
	}
	if ValidateAdds(domain.Item{Eff1: 2, Eff3: 200}) {
		t.Fatal("código 200 no terceiro par foi aceito")
	}
}
