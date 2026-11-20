package catalogfiles

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/enchantrules"
	"github.com/Spelunkern/Shaiya-GX/internal/gameini"
	"strconv"
	"strings"
)

func ParseEnchant(raw []byte) (enchantrules.Catalog, error) {
	ini, err := gameini.Parse(raw)
	if err != nil {
		return enchantrules.Catalog{}, err
	}
	cap := 20
	var weaponRate, defenseRate [enchantrules.MaxLevel + 1]uint32
	if raw, ok := ini["ENCHANT"]["MAXLEVEL"]; ok {
		n, err := strconv.ParseUint(raw, 10, 8)
		if err != nil || n > enchantrules.MaxLevel {
			return enchantrules.Catalog{}, fmt.Errorf("MaxLevel must be 0..%d", enchantrules.MaxLevel)
		}
		cap = int(n)
	}
	weaponCap, defenseCap := cap, cap
	for key, dst := range map[string]*int{"WEAPONMAXLEVEL": &weaponCap, "DEFENSEMAXLEVEL": &defenseCap} {
		if raw, ok := ini["ENCHANT"][key]; ok {
			n, err := strconv.ParseUint(raw, 10, 8)
			if err != nil || n > enchantrules.MaxLevel {
				return enchantrules.Catalog{}, fmt.Errorf("%s must be 0..%d", key, enchantrules.MaxLevel)
			}
			*dst = int(n)
		}
	}
	var weapon, armor [enchantrules.MaxLevel + 1]uint32
	for _, table := range []struct {
		section, prefix string
		dst             *[enchantrules.MaxLevel + 1]uint32
		limit           uint64
		cap             int
	}{
		{"LAPISIANENCHANTPERCENTRATE", "WEAPON", &weaponRate, 1000000, weaponCap},
		{"LAPISIANENCHANTPERCENTRATE", "DEFENSE", &defenseRate, 1000000, defenseCap},
		{"LAPISIANENCHANTADDVALUE", "WEAPON", &weapon, 2147483647, weaponCap},
		{"LAPISIANENCHANTADDVALUE", "DEFENSE", &armor, 2147483647, defenseCap},
	} {
		for step := 0; step <= enchantrules.MaxLevel; step++ {
			key := fmt.Sprintf("%sSTEP%02d", table.prefix, step)
			raw, ok := ini[table.section][key]
			if !ok {
				if step <= table.cap {
					return enchantrules.Catalog{}, fmt.Errorf("[%s] missing %s", table.section, key)
				}
				continue
			}
			n, err := strconv.ParseUint(raw, 10, 32)
			if err != nil || n > table.limit {
				return enchantrules.Catalog{}, fmt.Errorf("[%s] %s must be 0..%d", table.section, key, table.limit)
			}
			table.dst[step] = uint32(n)
		}
	}
	var rows enchantrules.Catalog
	for level := 1; level <= weaponCap; level++ {
		rows.Attack = append(rows.Attack, enchantrules.Level{Number: uint8(level), SuccessChance: weaponRate[level-1], Bonus: weapon[level]})
	}
	for level := 1; level <= defenseCap; level++ {
		rows.Defense = append(rows.Defense, enchantrules.Level{Number: uint8(level), SuccessChance: defenseRate[level-1], Bonus: armor[level]})
	}
	return rows, nil
}

func EncodeEnchant(rows enchantrules.Catalog) []byte {
	var b strings.Builder
	b.WriteString("; Successes per million for N -> N+1, and total bonuses at level N.\n")
	fmt.Fprintf(&b, "[Enchant]\nMaxLevel=%d\n", max(len(rows.Attack), len(rows.Defense)))
	if len(rows.Attack) != len(rows.Defense) {
		fmt.Fprintf(&b, "WeaponMaxLevel=%d\nDefenseMaxLevel=%d\n", len(rows.Attack), len(rows.Defense))
	}
	for _, rate := range []bool{true, false} {
		if rate {
			b.WriteString("\n[LapisianEnchantPercentRate]\n")
		} else {
			b.WriteString("\n[LapisianEnchantAddValue]\n")
		}
		for _, side := range []struct {
			prefix string
			levels []enchantrules.Level
		}{
			{"Weapon", rows.Attack}, {"Defense", rows.Defense},
		} {
			if !rate {
				fmt.Fprintf(&b, "%sStep00=0\n", side.prefix)
			}
			for i, row := range side.levels {
				n, value := i+1, row.Bonus
				if rate {
					n, value = i, row.SuccessChance
				}
				fmt.Fprintf(&b, "%sStep%02d=%d\n", side.prefix, n, value)
			}
			if rate {
				fmt.Fprintf(&b, "%sStep%02d=0\n", side.prefix, len(side.levels))
			}
		}
	}
	return []byte(b.String())
}
