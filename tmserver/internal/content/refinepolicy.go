package content

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"openwyd-nextgen/internal/ratetables"
)

// RefinePolicy is the dust-refine ceiling per dust, moved out of the code
// (_MSG_UseItem.cpp hardcoded 6/9/10/11) into Common/Settings/RefineRate.txt and
// the moderator overlay (content_rate_override, file "RefineRate"). The success
// chances stay in SancRate.txt — this table only says how far each dust may push
// an item.
//
// OriMax/PLMax are the highest level each dust can reach, GemLevel is the +10
// special step (a success there goes to +11 and keeps the gem) and HardCap is the
// absolute wall: at or above it the dust path refuses outright.
type RefinePolicy struct {
	OriMax   int
	PLMax    int
	GemLevel int
	HardCap  int
}

// DefaultRefinePolicy is the legacy behaviour (_MSG_UseItem.cpp:203/802/857),
// used when no -content tree is mounted.
func DefaultRefinePolicy() RefinePolicy {
	return RefinePolicy{OriMax: 6, PLMax: 9, GemLevel: 10, HardCap: 11}
}

// RefinePolicyKeys are the editable names, in file order.
var RefinePolicyKeys = []string{"PO_MAX", "PL_MAX", "GEM_LEVEL", "HARD_CAP"}

// RefinePolicyMax bounds every key: REF_11 is 12 and no legacy path refines past
// +12, so a value above 15 could never be reached by an item anyway.
const RefinePolicyMax = 15

// LoadRefinePolicy reads Common/Settings/RefineRate.txt. A missing file is not
// fatal — the legacy defaults stand and the caller logs it — because this table
// is a moderator convenience, not content the world needs to work.
func LoadRefinePolicy(path string) (RefinePolicy, error) {
	f, err := os.Open(path)
	if err != nil {
		return DefaultRefinePolicy(), fmt.Errorf("content: open RefineRate: %w", err)
	}
	defer f.Close()
	return ParseRefinePolicy(f)
}

// ParseRefinePolicy parses "POLICY <KEY> <VALUE>" lines. Malformed lines and
// unknown keys are skipped: the boot must not die because a moderator typed a
// stray word into the file.
func ParseRefinePolicy(r io.Reader) (RefinePolicy, error) {
	p := DefaultRefinePolicy()
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) != 3 || !strings.EqualFold(fields[0], "POLICY") {
			continue
		}
		v, err := strconv.Atoi(fields[2])
		if err != nil || v < 0 || v > RefinePolicyMax {
			continue
		}
		switch strings.ToUpper(fields[1]) {
		case "PO_MAX":
			p.OriMax = v
		case "PL_MAX":
			p.PLMax = v
		case "GEM_LEVEL":
			p.GemLevel = v
		case "HARD_CAP":
			p.HardCap = v
		}
	}
	return p.normalized(), sc.Err()
}

// normalized keeps the table coherent: the dust ceilings cannot exceed the hard
// cap, or a dust would be allowed to reach a level the item can never hold.
func (p RefinePolicy) normalized() RefinePolicy {
	if p.HardCap < 1 {
		p.HardCap = DefaultRefinePolicy().HardCap
	}
	if p.OriMax > p.HardCap {
		p.OriMax = p.HardCap
	}
	if p.PLMax > p.HardCap {
		p.PLMax = p.HardCap
	}
	if p.GemLevel > p.HardCap {
		p.GemLevel = p.HardCap
	}
	return p
}

// ApplyRefineOverrides folds the moderator overlay into the policy, with the same
// (file, family, name) identity every other rate uses.
func ApplyRefineOverrides(p RefinePolicy, overrides []ratetables.Override) (RefinePolicy, int, int) {
	applied, ignored := 0, 0
	for _, o := range overrides {
		if o.File != ratetables.FileRefineRate {
			continue
		}
		if o.Value < 0 || o.Value > RefinePolicyMax {
			ignored++
			continue
		}
		switch strings.ToUpper(o.Name) {
		case "PO_MAX":
			p.OriMax = int(o.Value)
		case "PL_MAX":
			p.PLMax = int(o.Value)
		case "GEM_LEVEL":
			p.GemLevel = int(o.Value)
		case "HARD_CAP":
			p.HardCap = int(o.Value)
		default:
			ignored++
			continue
		}
		applied++
	}
	return p.normalized(), applied, ignored
}
