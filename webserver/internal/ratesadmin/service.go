// Package ratesadmin is the web-api side of the rate editor (issue 36): it serves
// the effective rate tables (file values + overlay) and validates writes.
//
// The parsing comes from internal/ratetables, the same package tmServer will use
// when it applies the overlay at boot — the point is that the UI and the game
// cannot disagree about which cells exist or what the defaults are. The world
// only sees an edit after a restart with `-rates-editing`, which the page says.
package ratesadmin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"openwyd-nextgen/internal/ratetables"
	"openwyd-nextgen/internal/store"
)

// Store is the persistence surface the service needs (satisfied by *store.Store).
type Store interface {
	AccountRole(ctx context.Context, id int64) (string, error)
	ListContentRateOverrides(ctx context.Context) ([]store.ContentRateOverride, error)
	UpsertContentRateOverride(ctx context.Context, o store.ContentRateOverride, moderatorID int64, reason string) error
	DeleteContentRateOverride(ctx context.Context, file, family, name string, moderatorID int64, reason string) (bool, error)
	LatestInstanceFlags(ctx context.Context, server string) ([]byte, time.Time, bool, error)
}

// FlagView is one process flag as recorded by the service on its last boot.
type FlagView struct {
	Name            string `json:"name"`
	Value           string `json:"value"`
	RestartRequired bool   `json:"restart_required"`
	Hint            string `json:"hint"`
}

// Result mirrors web.proto AdminResult. Only infra failures become errors.
type Result int

const (
	// OK means the operation succeeded.
	OK Result = iota
	// Forbidden means the caller is not a moderator/admin.
	Forbidden
	// Invalid means the request failed validation.
	Invalid
	// NotFound means the cell (or overlay row) does not exist.
	NotFound
)

// RowView is one editable cell: the effective value, the file value and whether
// an overlay row is what makes them differ.
type RowView struct {
	Row         ratetables.Row
	Original    int64
	HasOverride bool
}

// Service implements the rate operations.
type Service struct {
	store      Store
	contentDir string
}

// New builds the service. An empty contentDir is allowed: List then returns no
// rows plus a warning, the same graceful degradation the pickers already have.
func New(s Store, contentDir string) *Service {
	return &Service{store: s, contentDir: contentDir}
}

// List returns every cell of the three tables with the overlay applied.
func (s *Service) List(ctx context.Context, moderatorID int64) (Result, []RowView, string, error) {
	if res, err := s.authorize(ctx, moderatorID); res != OK || err != nil {
		return res, nil, "", err
	}
	if s.contentDir == "" {
		return OK, nil, "web-api sem -content/W2PP_CONTENT: as taxas dos arquivos não podem ser lidas", nil
	}
	rows, err := ratetables.Rows(s.contentDir)
	if err != nil {
		return Invalid, nil, "", fmt.Errorf("ratesadmin: read rate files: %w", err)
	}
	overrides, err := s.store.ListContentRateOverrides(ctx)
	if err != nil {
		return Invalid, nil, "", fmt.Errorf("ratesadmin: list overrides: %w", err)
	}

	byKey := make(map[string]store.ContentRateOverride, len(overrides))
	for _, o := range overrides {
		byKey[cellKey(o.File, o.Family, o.Name)] = o
	}

	views := make([]RowView, 0, len(rows))
	for _, row := range rows {
		view := RowView{Row: row, Original: row.Value}
		if o, ok := byKey[cellKey(row.File, row.Family, row.Name)]; ok {
			view.Row.Value = o.Value
			view.HasOverride = true
		}
		views = append(views, view)
	}
	return OK, views, "", nil
}

// Upsert validates the cell against the parsed table (the file decides which
// cells exist and their range) and stores the overlay row.
func (s *Service) Upsert(ctx context.Context, moderatorID int64, file, family, name string, value int64, reason string) (Result, string, error) {
	if res, err := s.authorize(ctx, moderatorID); res != OK || err != nil {
		return res, "", err
	}
	if strings.TrimSpace(reason) == "" {
		return Invalid, "informe o motivo: taxa editada fica na auditoria", nil
	}
	row, res, message, err := s.cell(file, family, name)
	if res != OK || err != nil {
		return res, message, err
	}
	if err := ratetables.Validate(row, value); err != nil {
		return Invalid, err.Error(), nil
	}

	override := store.ContentRateOverride{File: row.File, Family: row.Family, Name: row.Name, Value: value}
	if err := s.store.UpsertContentRateOverride(ctx, override, moderatorID, reason); err != nil {
		return Invalid, "", fmt.Errorf("ratesadmin: upsert %s/%s/%s: %w", file, family, name, err)
	}
	return OK, "", nil
}

// Reset removes the overlay row, so the cell falls back to the file value.
func (s *Service) Reset(ctx context.Context, moderatorID int64, file, family, name, reason string) (Result, string, error) {
	if res, err := s.authorize(ctx, moderatorID); res != OK || err != nil {
		return res, "", err
	}
	if strings.TrimSpace(reason) == "" {
		return Invalid, "informe o motivo: restauração de taxa fica na auditoria", nil
	}
	row, res, message, err := s.cell(file, family, name)
	if res != OK || err != nil {
		return res, message, err
	}
	applied, err := s.store.DeleteContentRateOverride(ctx, row.File, row.Family, row.Name, moderatorID, reason)
	if err != nil {
		return Invalid, "", fmt.Errorf("ratesadmin: reset %s/%s/%s: %w", file, family, name, err)
	}
	if !applied {
		return NotFound, "esta célula não tem edição: já vale o valor do arquivo", nil
	}
	return OK, "", nil
}

// cell resolves (file, family, name) against the parsed tables. The overlay can
// never invent a cell: only what the file declares is editable.
func (s *Service) cell(file, family, name string) (ratetables.Row, Result, string, error) {
	if s.contentDir == "" {
		return ratetables.Row{}, Invalid, "web-api sem -content/W2PP_CONTENT: sem arquivos, não há célula para editar", nil
	}
	rows, err := ratetables.Rows(s.contentDir)
	if err != nil {
		return ratetables.Row{}, Invalid, "", fmt.Errorf("ratesadmin: read rate files: %w", err)
	}
	for _, row := range rows {
		if row.File == file && row.Family == family && row.Name == name {
			return row, OK, "", nil
		}
	}
	return ratetables.Row{}, NotFound, "célula inexistente: o arquivo não declara esse (arquivo, família, nome)", nil
}

func cellKey(file, family, name string) string {
	return file + "/" + family + "/" + name
}

// Flags returns the flags snapshot the given service recorded on its last boot.
// Empty available=false means the service never recorded one (or runs without
// -dbserver), which the UI shows as "sem snapshot" instead of an empty table.
func (s *Service) Flags(ctx context.Context, moderatorID int64, server string) (Result, []FlagView, time.Time, bool, error) {
	if res, err := s.authorize(ctx, moderatorID); res != OK || err != nil {
		return res, nil, time.Time{}, false, err
	}
	if server == "" {
		server = "tmserver"
	}
	raw, captured, ok, err := s.store.LatestInstanceFlags(ctx, server)
	if err != nil {
		return Invalid, nil, time.Time{}, false, fmt.Errorf("ratesadmin: latest instance flags: %w", err)
	}
	if !ok {
		return OK, nil, time.Time{}, false, nil
	}
	var flags []FlagView
	if err := json.Unmarshal(raw, &flags); err != nil {
		return Invalid, nil, time.Time{}, false, fmt.Errorf("ratesadmin: decode instance flags: %w", err)
	}
	return OK, flags, captured, true, nil
}

func (s *Service) authorize(ctx context.Context, moderatorID int64) (Result, error) {
	if moderatorID <= 0 {
		return Forbidden, nil
	}
	role, err := s.store.AccountRole(ctx, moderatorID)
	if errors.Is(err, store.ErrNotFound) {
		return Forbidden, nil
	}
	if err != nil {
		return Invalid, fmt.Errorf("ratesadmin: role lookup %d: %w", moderatorID, err)
	}
	if role != "moderator" && role != "admin" {
		return Forbidden, nil
	}
	return OK, nil
}
