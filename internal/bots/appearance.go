package bots

import (
	"cmp"
	"math/rand/v2"
	"slices"

	"github.com/Spelunkern/Shaiya-GX/internal/data/effect"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/itemtype"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

type ability struct {
	def                  sdata.Skill
	id                   uint16
	cast, flight, impact string
}

type loadout struct {
	look          scene.Appearance
	ranged, magic bool
	skills        []ability
}

type catalog struct {
	items  [25][]*sdata.Item
	skills []ability
	models map[scene.Appearance]characterModels
}

type characterModels struct {
	face, hair, helmet, upper, lower, hand, foot [256]bool
}

func loadCatalog(assets *scene.Assets) catalog {
	out := catalog{models: make(map[scene.Appearance]characterModels)}
	if data, err := assets.Arc.ReadFile("Item/Item.SData"); err == nil {
		if items, err := sdata.ParseItems(data); err == nil {
			for _, item := range items {
				kind := itemtype.Real(item.Type)
				if kind > 0 && int(kind) < len(out.items) {
					out.items[kind] = append(out.items[kind], item)
				}
			}
			for _, list := range out.items {
				slices.SortFunc(list, func(a, b *sdata.Item) int {
					return cmp.Or(cmp.Compare(a.Type, b.Type), cmp.Compare(a.TypeID, b.TypeID))
				})
			}
		}
	}
	if data, err := assets.Arc.ReadFile("Character/Skill.SData"); err == nil {
		if skills, err := sdata.ParseSkills(data); err == nil {
			for key, def := range skills {
				if key.Level == 1 && def.Level == 1 && def.Show != 0 && def.Target >= 1 && def.Target <= 5 &&
					def.Attack >= 1 && def.Attack <= 3 && (def.Effect == 1 || def.Effect == 3 || def.Effect == 5) {
					if impact := effect.ImpactLibrary(key.ID); impact != "" {
						out.skills = append(out.skills, ability{def, key.ID, effect.CastLibrary(key.ID), effect.ProjectileLibrary(key.ID), impact})
					}
				}
			}
			slices.SortFunc(out.skills, func(a, b ability) int { return cmp.Compare(a.id, b.id) })
		}
	}
	return out
}

func raceAllows(country, race uint8) bool {
	switch country {
	case 0, 1, 3:
		return country == race
	case 2:
		return race < 2
	case 4:
		return race == 2
	case 5:
		return race >= 2
	}
	return true
}

func itemAllows(item *sdata.Item, look scene.Appearance) bool {
	return item.Classes[look.Job] != 0 && raceAllows(item.Country, look.Race) &&
		(item.Special != 64 || look.Gender == 0) && (item.Special != 65 || look.Gender == 1)
}

func (c *catalog) pick(rng *rand.Rand, kind uint8, look scene.Appearance, available *[256]bool) *sdata.Item {
	var chosen *sdata.Item
	count := 0
	for _, item := range c.items[kind] {
		if itemAllows(item, look) && (available == nil || available[item.Model]) {
			count++
			if rng.IntN(count) == 0 {
				chosen = item
			}
		}
	}
	return chosen
}

func chooseModel(rng *rand.Rand, available [256]bool) uint8 {
	var selected uint8
	count := 0
	for i, ok := range available {
		if ok {
			count++
			if rng.IntN(count) == 0 {
				selected = uint8(i)
			}
		}
	}
	return selected
}

func (c *catalog) outfit(assets *scene.Assets, rng *rand.Rand, faction int) loadout {
	jobs := [4][4]uint8{{0, 1, 4, 5}, {2, 3, 4, 5}, {2, 3, 4, 5}, {0, 1, 2, 3}}
	race := uint8(2*faction + rng.IntN(2))
	key := scene.Appearance{Race: race, Gender: uint8(rng.IntN(2)), Job: jobs[race][rng.IntN(4)]}
	models, ok := c.models[key]
	if !ok {
		models = characterModels{
			face: assets.CharacterModels(key, "face"), hair: assets.CharacterModels(key, "hair"), helmet: assets.CharacterModels(key, "helmet"),
			upper: assets.CharacterModels(key, "upper"), lower: assets.CharacterModels(key, "lower"),
			hand: assets.CharacterModels(key, "hand"), foot: assets.CharacterModels(key, "foot"),
		}
		c.models[key] = models
	}
	look := key
	look.Height, look.Helmet = uint8(rng.IntN(5)), -1
	look.Face, look.Hair = chooseModel(rng, models.face), chooseModel(rng, models.hair)
	armor := func(kind uint8, models *[256]bool, dst *uint8) *sdata.Item {
		item := c.pick(rng, kind, look, models)
		if item != nil {
			*dst = item.Model
		}
		return item
	}
	chest := armor(17, &models.upper, &look.Upper)
	armor(18, &models.lower, &look.Lower)
	armor(20, &models.hand, &look.Hand)
	armor(21, &models.foot, &look.Foot)
	if helmet := c.pick(rng, 16, look, &models.helmet); helmet != nil && rng.IntN(4) != 0 {
		look.Helmet, look.ShowHelmet = int(helmet.Model), true
	}
	families := [6][]uint8{{1, 2, 3, 4, 15}, {1, 3, 7, 8}, {5, 9, 10, 13}, {11, 12, 14}, {6, 13}, {6, 7, 8}}
	group := families[look.Job]
	for range 16 {
		if item := c.pick(rng, group[rng.IntN(len(group))], look, nil); item != nil && assets.PrepareCharacterItem(item.Type, item.Model) {
			look.WeaponType, look.WeaponModel, look.WeaponEnchant = item.Type, item.Model, uint8(5+rng.IntN(11))
			break
		}
	}
	if look.WeaponType == 0 {
		for _, kind := range group {
			if assets.PrepareCharacterItem(kind, 0) {
				look.WeaponType, look.WeaponEnchant = kind, 5
				break
			}
		}
	}
	switch itemtype.Real(look.WeaponType) {
	case 1, 3, 7, 9:
		if item := c.pick(rng, 19, look, nil); item != nil && assets.PrepareCharacterItem(item.Type, item.Model) {
			look.ShieldType, look.ShieldModel = item.Type, item.Model
		}
	}
	if chest != nil && chest.ReqVG >= 1 && chest.ReqVG <= 8 {
		look.CloakFit = uint8(chest.ReqVG - 1)
	}
	for range 8 {
		if cloak := c.pick(rng, 24, look, nil); cloak != nil {
			look.CloakType, look.CloakModel = cloak.Type, cloak.Model
			if assets.PrepareCharacterCloak(look) {
				break
			}
			look.CloakType = 0
		}
	}
	if look.CloakType == 0 {
		look.CloakType, look.CloakModel = 24, 0
		if !assets.PrepareCharacterCloak(look) {
			look.CloakFit = 0
		}
	}

	family := itemtype.Real(look.WeaponType)
	magic := look.Job >= 4
	ranged := magic || family == 11 || family == 12 || family == 14
	result := loadout{look: look, ranged: ranged, magic: magic}
	for _, skill := range c.skills {
		if skill.def.Classes[look.Job] != 0 && raceAllows(skill.def.Country, look.Race) &&
			(magic == (skill.def.Attack == 3)) {
			result.skills = append(result.skills, skill)
		}
	}
	return result
}
