package studio

import "fmt"

type Field struct {
	Label, Source, Column, Section string
	offset, size                   int
	signed                         bool
	sqlMax                         int64
}

func textField(label, source, column string, textIndex int) Field {
	return Field{Label: label, Source: source, Column: column, Section: "Identity", offset: textIndex}
}

func numeric(label, source, column, section string, offset, size int) Field {
	return Field{Label: label, Source: source, Column: column, Section: section, offset: offset, size: size}
}

var classes = []struct{ label, field string }{
	{"Fighter", "Attackfighter"}, {"Defender", "Defensefighter"}, {"Rogue", "Patrolrogue"},
	{"Archer", "Shootrogue"}, {"Mage", "Attackmage"}, {"Priest", "Defensemage"},
}

var itemFields = func() []Field {
	fields := []Field{textField("Name", "Name", "ItemName", 0), textField("Description", "Description", "", 1)}
	add := func(label, source, column, section string, offset, size int) {
		fields = append(fields, numeric(label, source, column, section, offset, size))
	}
	const visual, req, base, stats, shop = "Appearance", "Requirements", "Combat", "Bonuses", "Other"
	add("Model ID", "Model", "", visual, 2, 1)
	add("Inventory icon", "Icon", "", visual, 3, 1)
	add("Client type flag 2", "Type2", "", shop, 14, 1)
	add("Client type flag 3", "Type3", "", shop, 15, 1)
	add("Required level", "MinLevel", "Reqlevel", req, 4, 2)
	add("Faction", "Country", "Country", req, 6, 1)
	add("Legacy item property", "Grow", "Grow", req, 13, 1)
	for i, class := range classes {
		add(class.label+" eligibility", class.field, class.field, req, 7+i, 1)
	}
	for i, stat := range []string{"Str", "Dex", "Rec", "Int", "Wis", "Luc"} {
		column := "Req" + stat
		if stat == "Luc" {
			column = "Reqluc"
		}
		add("Required "+stat, "Req"+stat, column, req, 16+2*i, 2)
	}
	add("Special parameter VG", "ReqVG", "ReqVg", req, 28, 2)
	add("Guild bank restriction", "ReqOG", "ReqOg", req, 30, 1)
	add("Parameter IG", "ReqIg", "ReqIg", req, 31, 1)
	add("Range / linked skill ID (low byte)", "Range", "Range", base, 32, 1)
	add("Attack time / linked skill level", "AttackTime", "AttackTime", base, 33, 1)
	add("Element", "Attribute", "Attrib", base, 34, 1)
	add("Special effect", "Special", "Special", base, 35, 1)
	add("Socket count", "Slot", "Slot", base, 36, 1)
	add("Durability", "Quality", "Quality", base, 37, 2)
	for i, v := range []struct{ label, field, column string }{
		{"Base attack", "Attack", "Effect1"}, {"Attack variance", "AttackAdd", "Effect2"},
		{"Defense", "Def", "Effect3"}, {"Magic resistance", "Resist", "Effect4"},
		{"HP", "RecoveryHP", "ConstHP"}, {"SP", "RecoverySP", "ConstSP"}, {"MP", "RecoveryMP", "ConstMP"},
		{"Strength", "Str", "ConstStr"}, {"Dexterity", "Dex", "ConstDex"}, {"Reaction", "Rec", "ConstRec"},
		{"Intelligence", "Int", "ConstInt"}, {"Wisdom", "Wis", "ConstWis"}, {"Luck", "Luc", "ConstLuc"},
	} {
		add(v.label, v.field, v.column, stats, 39+2*i, 2)
	}
	add("Movement speed bonus", "Speed", "Speed", stats, 65, 1)
	add("Flat damage absorption", "Exp", "Exp", stats, 66, 1)
	add("Purchase price", "BuyPrice", "Buy", shop, 67, 4)
	add("Sale price", "SellPrice", "Sell", shop, 71, 4)
	add("Loot grade", "Grade", "Grade", shop, 75, 2)
	add("Drop limit selector", "Drop", "Drop", shop, 77, 2)
	add("Server flag", "Server", "Server", shop, 79, 1)
	add("Native stack flag / count", "Count", "Count", shop, 80, 1)
	for i := range fields {
		f := &fields[i]
		switch {
		case f.offset >= 16 && f.offset <= 28 && f.Column != "Reqluc", f.offset >= 39 && f.offset <= 63:
			f.sqlMax = 32767
		case f.offset == 67 || f.offset == 71:
			f.sqlMax = 2147483647
		}
	}
	return fields
}()

const (
	skillEffectOffset = 21
	skillTargetOffset = 56
)

var skillFields = func() []Field {
	fields := []Field{textField("Name", "Name", "SkillName", 0), textField("Description", "Description", "", 1)}
	add := func(label, field, section string, offset, size int) {
		fields = append(fields, numeric(label, field, field, section, offset, size))
	}
	client := func(label, field string, offset, size int) {
		fields = append(fields, numeric(label, field, "", "Appearance", offset, size))
	}
	const req, cast, dmg, effects = "Requirements", "Cast and target", "Damage and healing", "Effects"
	client("Skill icon", "Icon", 1, 2)
	client("Animation ID", "Animation", 3, 1)
	client("Visual effect ID", "VisualCode low byte", 4, 1)
	client("Toggle flag / visual high byte", "Toggle / VisualCode high byte", 5, 1)
	client("Sound ID", "Sound", 6, 2)
	add("Required character level", "ReqLevel", req, 8, 2)
	add("Faction", "Country", req, 10, 1)
	for i, class := range classes {
		add(class.label+" eligibility", class.field, req, 11+i, 1)
	}
	add("Skill point cost", "SkillPoint", req, 18, 1)
	add("Visibility / learnability", "TypeShow", req, 19, 1)
	add("Attack kind", "TypeAttack", cast, 20, 1)
	add("Effect kind", "TypeEffect", effects, skillEffectOffset, 1)
	add("Effect detail", "TypeDetail", effects, 22, 2)
	for i := 0; i < 15; i++ {
		add(fmt.Sprintf("Weapon eligibility %d", i+1), fmt.Sprintf("NeedWeapon%d", i+1), req, 24+i, 1)
	}
	add("Shield required", "Shield", req, 39, 1)
	add("SP cost", "SP", cast, 40, 2)
	add("MP cost", "MP", cast, 42, 2)
	add("Cast time", "ReadyTime", cast, 44, 1)
	add("Cooldown in seconds", "ResetTime", cast, 45, 2)
	add("Attack range", "AttackRange", cast, 47, 1)
	add("Status type", "StateType", effects, 48, 1)
	add("Element override", "AttrType", dmg, 49, 1)
	add("Required skill on cooldown", "PrevSkillID", req, 52, 2)
	add("Success mode", "SuccessType", cast, 54, 1)
	add("Success value", "SuccessValue", cast, 55, 1)
	add("Target type", "TargetType", cast, skillTargetOffset, 1)
	add("Effect radius", "ApplyRange", cast, 57, 1)
	add("Multi-hit setting", "MultiAttack", cast, 58, 1)
	add("Duration (hours if FixRange = 1, seconds otherwise)", "KeepTime", effects, 59, 2)
	add("Passive weapon type 1", "Weapon1", effects, 61, 1)
	add("Passive weapon type 2", "Weapon2", effects, 62, 1)
	add("Passive weapon bonus", "Weaponvalue", effects, 63, 1)
	add("Bag parameter", "Bag", effects, 64, 1)
	add("Aggro value", "Arrow", effects, 65, 2)
	add("Damage mode", "DamageType", dmg, 67, 1)
	add("Damage over time mode", "TimeDamageType", dmg, 74, 1)
	for _, group := range []struct {
		label, column string
		offset        int
	}{
		{"Damage", "Damage", 68}, {"Damage over time", "TimeDamage", 75}, {"Extra damage", "AddDamage", 81},
		{"Healing", "Heal", 96}, {"Healing over time", "TimeHeal", 102},
	} {
		for i, resource := range []string{"HP", "SP", "MP"} {
			add(group.label+" · "+resource, group.column+resource, dmg, group.offset+2*i, 2)
		}
	}
	for i := 0; i < 3; i++ {
		add(fmt.Sprintf("Ability %d type", i+1), fmt.Sprintf("AbilityType%d", i+1), effects, 87+3*i, 1)
		add(fmt.Sprintf("Ability %d amount", i+1), fmt.Sprintf("AbilityValue%d", i+1), effects, 88+3*i, 2)
		fields[len(fields)-1].signed = true
	}
	add("Defense kind", "DefenceType", effects, 108, 1)
	add("Defense amount", "DefenceValue", effects, 109, 1)
	add("HP threshold", "LimitHP", effects, 110, 1)
	add("Duration / death behavior", "FixRange", effects, 111, 1)
	add("Transformation / change type", "ChangeType", effects, 112, 2)
	add("Transformation / change value", "ChangeLevel", effects, 114, 2)
	return fields
}()
