package bot

// Tabela de comportamento: o que cada perfil faz em cada estado.
//
// As ações de combate, coleta, cura e comércio mutam apenas o modelo local do
// bot (Context) e contam métricas `sim_*`. Elas NÃO enviam pacotes de ataque,
// movimento ou comércio: esses formatos ainda não estão verificados contra o
// cliente 7662 e inventá-los produziria teste que prova nada. O tráfego real do
// bot hoje é conexão, INITCODE, login, seleção de personagem e leitura de
// frames — exatamente o caminho que o cliente gráfico usa e que o e2e já cobre.
// A troca dos `sim_*` por pacotes reais é o próximo passo do laboratório, com o formato de cada pacote provado antes de entrar.

// DefaultTransitions devolve a tabela de transições usada pelo executor.
func DefaultTransitions(met *Metrics) []Transition {
	inc := func(name string) {
		if met != nil {
			met.Inc(name)
		}
	}
	return []Transition{
		// Ciclo de vida da conexão.
		{From: StateConnecting, Event: EventConnected, To: StateAuthenticating, Name: "connect"},
		{From: StateAuthenticating, Event: EventAuthenticated, To: StateSelectingChar, Name: "auth"},
		{From: StateSelectingChar, Event: EventCharacterReady, To: StateEnteringWorld, Name: "select"},
		{From: StateEnteringWorld, Event: EventWorldEntered, To: StateMovingToHunt, Name: "enter-world"},

		// Deslocamento até a área de caça.
		{From: StateMovingToHunt, Event: EventReachedArea, To: StateHunting, Name: "reached-area"},
		{From: StateMovingToHunt, Event: EventNoTarget, To: StateHunting, Name: "no-target-area"},

		// Caça.
		{From: StateHunting, Event: EventTargetFound, To: StateAttacking, Name: "target-found",
			Action: func(c *Context) error {
				c.Target = 1
				c.HP -= 2
				if c.HP < 0 {
					c.HP = 0
				}
				inc("sim_attack")
				return nil
			}},
		{From: StateAttacking, Event: EventTargetDown, To: StateLooting, Name: "target-down-loot",
			Guard: func(c *Context) bool { return c.Loot > 0 }},
		{From: StateAttacking, Event: EventTargetDown, To: StateHunting, Name: "target-down",
			Action: func(c *Context) error {
				c.Target = 0
				c.Loot++
				inc("sim_kill")
				return nil
			}},
		{From: StateLooting, Event: EventLootPending, To: StateHunting, Name: "loot",
			Action: func(c *Context) error {
				if c.Loot > 0 {
					c.Loot--
				}
				if c.InventoryFree > 0 {
					c.InventoryFree--
				}
				c.Gold += 100
				inc("sim_loot")
				return nil
			}},

		// Sobrevivência.
		{From: StateHunting, Event: EventHpLow, To: StateHealing, Name: "hp-low"},
		{From: StateHealing, Event: EventHealed, To: StateHunting, Name: "healed",
			Action: func(c *Context) error {
				if c.Potions > 0 {
					c.Potions--
					c.HP = c.HPmax
					inc("sim_potion")
				} else {
					c.HP = c.HPmax / 4
					inc("sim_rest_recover")
				}
				return nil
			}},

		// Economia.
		{From: StateHunting, Event: EventInventoryFull, To: StateReturningToTown, Name: "inventory-full"},
		{From: StateReturningToTown, Event: EventInTown, To: StateSellingItems, Name: "in-town"},
		{From: StateSellingItems, Event: EventSold, To: StateRestocking, Name: "sold",
			Action: func(c *Context) error {
				c.InventoryFree = 8
				c.Gold += int64(c.InventoryFree) * 1000
				inc("sim_sell")
				return nil
			}},
		{From: StateHunting, Event: EventPotionMissing, To: StateRestocking, Name: "potion-missing"},
		{From: StateRestocking, Event: EventRestocked, To: StateMovingToHunt, Name: "restocked",
			Action: func(c *Context) error {
				c.Potions = 20
				c.Gold -= 500
				inc("sim_restock")
				return nil
			}},

		// Morte e retorno.
		{From: StateHunting, Event: EventDied, To: StateDead, Name: "died",
			Action: func(c *Context) error {
				c.Target = 0
				inc("sim_death")
				return nil
			}},
		{From: StateDead, Event: EventRespawned, To: StateReturningToTown, Name: "respawn",
			Action: func(c *Context) error {
				c.HP = c.HPmax
				c.MP = c.MPmax
				inc("sim_respawn")
				return nil
			}},

		// Encerramento.
		{From: StateHunting, Event: EventStop, To: StateDisconnecting, Name: "stop-hunting"},
		{From: StateMovingToHunt, Event: EventStop, To: StateDisconnecting, Name: "stop-moving"},
		{From: StateAttacking, Event: EventStop, To: StateDisconnecting, Name: "stop-attacking"},
		{From: StateLooting, Event: EventStop, To: StateDisconnecting, Name: "stop-looting"},
		{From: StateHealing, Event: EventStop, To: StateDisconnecting, Name: "stop-healing"},
		{From: StateIdle, Event: EventStop, To: StateDisconnecting, Name: "stop-idle"},
	}
}

// nextEvent escolhe o próximo estímulo a partir do estado e do modelo local.
// Um estado sem estímulo aplicável devolve Event("") e o passo é contado como
// ocioso, sem erro: é assim que o executor detecta roteiro incompleto.
func nextEvent(st State, c *Context, p Profile, g *RNG) Event {
	switch st {
	case StateMovingToHunt:
		if g.Chance(p.Aggro) {
			return EventReachedArea
		}
		return EventNoTarget
	case StateHunting:
		switch {
		case c.HPFraction() <= p.FleeBelow:
			return EventHpLow
		case c.Potions <= p.RestockBelow:
			return EventPotionMissing
		case c.InventoryFree <= p.InventoryFullAt:
			return EventInventoryFull
		case c.Loot > 0 && g.Chance(p.LootChance):
			return EventLootPending
		case g.Chance(p.Aggro):
			if c.HP <= 0 {
				return EventDied
			}
			return EventTargetFound
		default:
			return EventNoTarget
		}
	case StateAttacking:
		return EventTargetDown
	case StateLooting:
		return EventLootPending
	case StateHealing:
		return EventHealed
	case StateReturningToTown:
		return EventInTown
	case StateSellingItems:
		return EventSold
	case StateRestocking:
		return EventRestocked
	case StateDead:
		return EventRespawned
	case StateIdle:
		if g.Chance(p.Aggro) {
			return EventTargetFound
		}
		return ""
	}
	return ""
}
