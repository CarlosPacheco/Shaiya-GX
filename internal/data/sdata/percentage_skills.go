package sdata

import "fmt"

// NormalizePercentageSkills updates GX's native catalog presentation. The
// server catalog is migrated separately and owns the actual combat effects.
func NormalizePercentageSkills(skills map[SkillKey]Skill, npc bool) {
	for key, skill := range skills {
		if npc {
			if key == (SkillKey{190, 1}) && skill.Name == "Ill Fate" && skill.AbilityType == [3]uint8{20, 21, 22} && skill.AbilityValue == [3]int16{25, 25, 25} {
				skill.AbilityValue = [3]int16{20, 20, 20}
				skills[key] = skill
			}
			continue
		}
		switch {
		case key.Level == 1 && skill.AbilityType[0] == 35 &&
			(key.ID == 222 || key.ID == 223 || key.ID == 250 || key.ID == 251 || key.ID == 272):
			if skill.AbilityValue[0] == 1 {
				skill.AbilityValue[0] = 50
			}
			skill.Description = experienceDescription(int(skill.AbilityValue[0]), skill.KeepSeconds())
		case key.Level == 1 && key.ID >= 231 && key.ID <= 233 && skill.Effect == 4 &&
			(skill.AbilityType[0] == 35 || skill.AbilityType[0] == uint8(key.ID-191)):
			if skill.AbilityType[0] != 35 {
				skill.AbilityType[0], skill.AbilityValue[0] = 35, 20
			}
			skill.Description = experienceDescription(int(skill.AbilityValue[0]), skill.KeepSeconds())
		case key.ID == 136 && skill.AbilityType == [3]uint8{20, 21} && skill.AbilityValue == [3]int16{500, 500}:
			skill.AbilityValue[0], skill.AbilityValue[1] = 100, 100
		case key.ID == 189 && skill.AbilityType == [3]uint8{29, 30} &&
			(key.Level == 2 && skill.AbilityValue == [3]int16{180, 180} || key.Level == 3 && skill.AbilityValue == [3]int16{240, 240}):
			skill.AbilityValue[0], skill.AbilityValue[1] = 100, 100
		case key.Level == 1 && key.ID >= 212 && key.ID <= 214 && skill.AbilityType[0] == uint8(key.ID-192):
			category := []string{"melee", "ranged", "magic"}[key.ID-212]
			if skill.AbilityValue[0] == 300 {
				skill.AbilityValue[0] = 100
			}
			skill.Description = fmt.Sprintf("Adds %d percentage points of %s accuracy for 4 hours. Expires on death.", skill.AbilityValue[0], category)
		case key.ID == 25 && key.Level >= 1 && key.Level <= 3 && skill.AbilityType == [3]uint8{29, 30}:
			skill.Description = fmt.Sprintf("Below %d%% HP, adds %d percentage points of melee and ranged evasion for 30 seconds. Can trigger once every 3 minutes.",
				skill.LimitHP, skill.AbilityValue[0])
		case key.ID == 31 && key.Level >= 1 && key.Level <= 3 && skill.AbilityType[0] == 20:
			skill.Description = fmt.Sprintf("Adds %d percentage points of melee accuracy for 5 minutes.", skill.AbilityValue[0])
		default:
			continue
		}
		skills[key] = skill
	}
}

func experienceDescription(bonus int, seconds uint32) string {
	duration := fmt.Sprintf("%d hours", seconds/3600)
	switch {
	case seconds == 3600:
		duration = "1 hour"
	case seconds == 86400:
		duration = "1 day"
	case seconds >= 86400 && seconds%86400 == 0:
		duration = fmt.Sprintf("%d days", seconds/86400)
	}
	return fmt.Sprintf("Gain %d%% more EXP from defeating monsters for %s. Persists after death.", bonus, duration)
}

// NormalizePercentageItems replaces the legacy text on the affected consumables.
func NormalizePercentageItems(items map[ItemKey]*Item) {
	for id, hours := range map[uint8]uint32{42: 168, 43: 720, 72: 240, 73: 720, 74: 720, 108: 1, 109: 12, 135: 24} {
		if item := items[ItemKey{100, id}]; item != nil {
			bonus := 50
			if id >= 72 && id <= 74 {
				bonus = 20
			}
			item.Description = experienceDescription(bonus, hours*3600)
			if id == 135 && item.Range == 16 && item.AttackTime == 1 {
				item.Range = 272
			}
			if id == 135 {
				item.Description += " <Cannot be traded>"
			}
		}
	}
	for id, category := range map[uint8]string{11: "ranged", 12: "magic", 13: "melee"} {
		if item := items[ItemKey{100, id}]; item != nil {
			item.Description = fmt.Sprintf("Adds 100 percentage points of %s accuracy for 4 hours. Expires on death.", category)
		}
	}
}
