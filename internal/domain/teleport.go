package domain

import "time"

// Valores de TeleportRequirement aceitos por teleport_route.requirement.
const (
	TeleportRequirementNone  = "none"
	TeleportRequirementFairy = "fairy"
	TeleportRequirementKefra = "kefra"
)

// TeleportRoute é uma rota de teleporte editável pelo moderador.
//
// É um OVERLAY, não uma cópia da tabela do mundo: a linha substitui a entrada
// compilada de teleportTable no mesmo tile de origem, `Enabled=false` a remove e
// uma origem sem linha mantém a rota compilada. SrcX/SrcY são o tile arredondado
// (múltiplo de 4) — a chave exata que teleportDest consulta.
type TeleportRoute struct {
	SrcX int `json:"src_x"`
	SrcY int `json:"src_y"`
	DstX int `json:"dst_x"`
	DstY int `json:"dst_y"`
	// Cost é o ouro cobrado na chegada (>= 0).
	Cost int `json:"cost"`
	// Requirement é um dos TeleportRequirement*.
	Requirement string    `json:"requirement"`
	Enabled     bool      `json:"enabled"`
	Note        string    `json:"note"`
	UpdatedBy   int64     `json:"updated_by"`
	UpdatedAt   time.Time `json:"updated_at"`
}
