// Package skillcatalog reads the skill lookup the web front-end needs from
// Release/Common/SkillData.csv: index → name (plus the owning class, which is
// just index/24).
//
// The row layout is the same one tmserver/internal/content parses into
// STRUCT_SPELL, but webserver cannot import that package (tmserver internals are
// not a public API), so the tolerance rules are mirrored here instead of the
// whole 20-column struct: index comes from column 0 (indexes are sparse past
// 149), the name is the trailing column, rows shorter than 22 columns are
// skipped, and the file is ISO-8859-1 — accented names ("Possuído") would reach
// gRPC as mojibake without decoding.
//
// Only the display name and class are exposed. Everything else about a skill
// (mana, dano, affect) belongs to the future SkillCatalogService of issue 46,
// which must reuse this parser rather than grow a second one.
package skillcatalog

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"openwyd-nextgen/internal/domain"
)

// Table dimensions, same as content.MaxSkill/MaxSkillIndex (Basedef.h:171/200).
const (
	MaxSkill      = 24
	MaxSkillIndex = 248
)

// Entry is one skill row, trimmed to what the web needs.
type Entry struct {
	Index int32
	Name  string // raw CSV name, e.g. "Toque_Sagrado"
	// DisplayName is Name with underscores turned back into spaces.
	DisplayName string
	// Class is the owning class index (index/MaxSkill): 0 TK, 1 FM, 2 BM, 3 HT.
	Class int32
}

// Catalog is the scanned skill list plus a fingerprint of the file.
type Catalog struct {
	Skills []Entry
	// Version fingerprints SkillData.csv so callers can cache the list and
	// revalidate cheaply; the content tree is read-only at runtime.
	Version string
}

// ByIndex returns the catalog as a lookup map.
func (c Catalog) ByIndex() map[int32]Entry {
	out := make(map[int32]Entry, len(c.Skills))
	for _, e := range c.Skills {
		out[e.Index] = e
	}
	return out
}

// Scan reads <contentDir>/Common/SkillData.csv.
func Scan(contentDir string) (Catalog, error) {
	path := filepath.Join(contentDir, "Common", "SkillData.csv")
	f, err := os.Open(path)
	if err != nil {
		return Catalog{}, fmt.Errorf("skillcatalog: open %s: %w", path, err)
	}
	defer f.Close()

	sum := sha256.New()
	byIndex := make(map[int32]Entry)
	sc := bufio.NewScanner(io.TeeReader(f, sum))
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		fields := strings.Split(line, ",")
		// The legacy sscanf consumes exactly 22 tokens; anything shorter is a
		// header or a broken row, and content.parseSkillData drops it too.
		if len(fields) < 22 {
			continue
		}
		idx, err := strconv.Atoi(strings.TrimSpace(fields[0]))
		if err != nil || idx < 0 || idx >= MaxSkillIndex {
			continue
		}
		name := latin1ToUTF8(strings.TrimSpace(fields[len(fields)-1]))
		if name == "" {
			continue
		}
		byIndex[int32(idx)] = Entry{
			Index:       int32(idx),
			Name:        name,
			DisplayName: strings.TrimSpace(strings.ReplaceAll(name, "_", " ")),
			Class:       int32(idx / MaxSkill),
		}
	}
	if err := sc.Err(); err != nil {
		return Catalog{}, fmt.Errorf("skillcatalog: scan %s: %w", path, err)
	}

	out := make([]Entry, 0, len(byIndex))
	for _, e := range byIndex {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Index < out[j].Index })
	return Catalog{Skills: out, Version: hex.EncodeToString(sum.Sum(nil))[:16]}, nil
}

// latin1ToUTF8 converts a string holding raw ISO-8859-1 bytes (each byte's value
// IS its code point in that encoding) into UTF-8. bufio.Scanner's Text() wraps
// raw bytes without transcoding, so this has to run before the name reaches
// gRPC/JSON.
// FromEntries builds the web-facing skill list from the skill_catalog rows the
// database holds (issue 58). The CSV left the repository in phase 5, so this is
// the only source left; the class stays index/MaxSkill, as the CSV convention.
func FromEntries(rows []domain.SkillCatalogEntry) Catalog {
	out := make([]Entry, 0, len(rows))
	sum := sha256.New()
	for _, row := range rows {
		if row.Index < 0 || row.Index >= MaxSkillIndex || row.Name == "" {
			continue
		}
		out = append(out, Entry{
			Index:       int32(row.Index),
			Name:        row.Name,
			DisplayName: strings.TrimSpace(strings.ReplaceAll(row.Name, "_", " ")),
			Class:       int32(row.Index / MaxSkill),
		})
		fmt.Fprintf(sum, "%d:%s\n", row.Index, row.Name)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Index < out[j].Index })
	return Catalog{Skills: out, Version: hex.EncodeToString(sum.Sum(nil))[:16]}
}

// latin1ToUTF8 converts a string holding raw ISO-8859-1 bytes (each byte's value
// IS its code point in that encoding) into UTF-8.
func latin1ToUTF8(s string) string {
	b := []byte(s)
	runes := make([]rune, len(b))
	for i, c := range b {
		runes[i] = rune(c)
	}
	return string(runes)
}
