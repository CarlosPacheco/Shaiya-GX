package app

import "strconv"

// Numeric emphasis and omitted HP damage labels are requested GX presentation
// changes. Wording, non-HP resource labels and message IDs still come from data.
func (a *App) combatMessage(style uint8, id int, values map[string]string) {
	if value := values["v"]; value != "" {
		values["v"] = "{c3}" + value + "{/c}"
	}
	a.systemMessage(style, id, values)
}

// Pools in SkillResult are enumerated HP, MP, SP here, not in wire order.
func (a *App) combatPoolLabel(pool int, damage bool) string {
	if pool == 0 && damage {
		return ""
	}
	return a.msg.Get([]int{416, 418, 417}[pool])
}

// 0x577b90: periodic effects name their skill but have no source identity.
// The wire orders non-HP healing/debuff values differently; decode by effect.
func (a *App) periodicSystemChat() {
	r := a.state.LastPeriodicEffect
	def, ok := a.skillDefinition(r.Skill, r.Level)
	if !ok {
		return
	}
	own := r.Target == a.state.SelfID
	if !own && a.state.Party.Member(r.Target) == nil {
		return
	}
	damage := def.Effect != 4 || def.Detail == 84 // Fire Thorn is a buff that hurts
	id, style := 1310, uint8(1)
	if !own {
		id, style = 1211, 0
	}
	amounts := [3]uint16{r.HP, r.Resource1, r.Resource2}
	if damage {
		amounts[1], amounts[2] = amounts[2], amounts[1]
	}
	if !damage {
		id, style = 1103, 4
		if !own {
			id = 1206
		}
	}
	for pool, amount := range amounts {
		if amount > 0 {
			a.combatMessage(style, id, map[string]string{"s": def.Name, "p": a.chatEntityName(r.Target), "v": strconv.Itoa(int(amount)), "hp": a.combatPoolLabel(pool, damage)})
		}
	}
}

// 0x578d70 / 0x5798a0: additional damage is separate from the initial hit.
func (a *App) extraDamageSystemChat() {
	r := a.state.LastSkillExtra
	id, style := 1105, uint8(0)
	if r.Source != a.state.SelfID {
		if r.Target != a.state.SelfID {
			return
		}
		id, style = 1304, 1
		if r.Source == r.Target {
			id = 1310
		}
	}
	def, ok := a.skillDefinition(r.ID, r.Level)
	if !ok {
		return
	}
	for pool, amount := range []uint16{r.HP, r.MP, r.SP} {
		if amount > 0 {
			a.combatMessage(style, id, map[string]string{"s": def.Name, "p": a.chatEntityName(r.Source), "t": a.chatEntityName(r.Target), "v": strconv.Itoa(int(amount)), "hp": a.combatPoolLabel(pool, true)})
		}
	}
}
