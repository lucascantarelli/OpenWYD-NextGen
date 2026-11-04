package domain

import "time"

// TeleportRequirement values accepted by teleport_route.requirement.
const (
	TeleportRequirementNone  = "none"
	TeleportRequirementFairy = "fairy"
	TeleportRequirementKefra = "kefra"
)

// TeleportRoute is one moderator-editable teleport route (issue 42).
//
// It is an OVERLAY, not a copy of the world table: a row replaces the compiled
// teleportTable entry for the same origin tile, `Enabled=false` removes it, and
// an origin with no row keeps the compiled route. SrcX/SrcY are the rounded tile
// (multiple of 4) — the exact key teleportDest looks up.
type TeleportRoute struct {
	SrcX int `json:"src_x"`
	SrcY int `json:"src_y"`
	DstX int `json:"dst_x"`
	DstY int `json:"dst_y"`
	// Cost is gold charged on arrival (>= 0).
	Cost int `json:"cost"`
	// Requirement is one of TeleportRequirement*.
	Requirement string `json:"requirement"`
	Enabled     bool   `json:"enabled"`
	Note        string `json:"note"`
	UpdatedBy   int64  `json:"updated_by"`
	UpdatedAt   time.Time `json:"updated_at"`
}
