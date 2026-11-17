package world

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/logx"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
	"path/filepath"
	"strconv"
)

const (
	defaultEnchantMaxLevel = 20
	enchantMaxLevel        = 49 // armor encodes 50+level in the two-digit Craftname pair
)

// Runtime enchant configuration is loaded atomically from data/game/ItemEnchant.ini.
// These initial values retain the historical defaults before startup loading.
// Step N is the probability of N -> N+1, in successes per million.
type enchantRulesConfig struct {
	MaxLevel                uint8
	WeaponRate, DefenseRate [enchantMaxLevel + 1]uint32
}

var enchantConfig = enchantRulesConfig{
	MaxLevel:    defaultEnchantMaxLevel,
	WeaponRate:  [enchantMaxLevel + 1]uint32{900000, 800000, 700000, 600000, 400000, 200000, 100000, 80000, 60000, 10000, 6000, 2000, 1000, 600, 200, 200, 200, 200, 200, 200, 0},
	DefenseRate: [enchantMaxLevel + 1]uint32{990000, 980000, 970000, 800000, 600000, 400000, 200000, 100000, 80000, 60000, 10000, 6000, 2000, 1000, 600, 200, 200, 200, 200, 200, 0},
}

// Bonuses are indexed by the resulting enchantment level, starting at zero.
var weaponEnchantAttackBonus = [enchantMaxLevel + 1]uint32{0, 7, 14, 21, 31, 41, 51, 64, 77, 90, 106, 122, 138, 157, 176, 195, 217, 239, 261, 286, 311}
var armorEnchantAbsorption = [enchantMaxLevel + 1]uint32{0, 5, 10, 15, 20, 25, 30, 35, 40, 45, 50, 55, 60, 65, 70, 75, 80, 85, 90, 95, 100}

func validateEnchantRules() error {
	if enchantConfig.MaxLevel > enchantMaxLevel {
		return fmt.Errorf("enchant cap exceeds two-digit Craftname capacity (%d)", enchantMaxLevel)
	}
	for i := 0; i <= enchantMaxLevel; i++ {
		if enchantConfig.WeaponRate[i] > 1000000 || enchantConfig.DefenseRate[i] > 1000000 {
			return fmt.Errorf("enchant step %d exceeds 100%%", i)
		}
	}
	return nil
}

// loadItemEnchantIni restores the original sections, adding [Enchant] MaxLevel.
// Missing MaxLevel supports original INIs at +20. Invalid input never replaces
// the last valid rules; production startup treats an absent file as an error.
func loadItemEnchantIni(log *logx.Logger) error {
	ini, err := readIni(filepath.Join(gameDataDir(), "ItemEnchant.ini"))
	if err != nil {
		return err
	}
	next := enchantRulesConfig{MaxLevel: defaultEnchantMaxLevel}
	if raw, ok := ini["ENCHANT"]["MAXLEVEL"]; ok {
		n, err := strconv.ParseUint(raw, 10, 8)
		if err != nil || n > enchantMaxLevel {
			return fmt.Errorf("MaxLevel must be 0..%d", enchantMaxLevel)
		}
		next.MaxLevel = uint8(n)
	}
	var weapon, armor [enchantMaxLevel + 1]uint32
	for _, table := range []struct {
		section, prefix string
		dst             *[enchantMaxLevel + 1]uint32
		limit           uint64
	}{
		{"LAPISIANENCHANTPERCENTRATE", "WEAPON", &next.WeaponRate, 1000000},
		{"LAPISIANENCHANTPERCENTRATE", "DEFENSE", &next.DefenseRate, 1000000},
		{"LAPISIANENCHANTADDVALUE", "WEAPON", &weapon, 2147483647},
		{"LAPISIANENCHANTADDVALUE", "DEFENSE", &armor, 2147483647},
	} {
		for step := 0; step <= enchantMaxLevel; step++ {
			key := fmt.Sprintf("%sSTEP%02d", table.prefix, step)
			raw, ok := ini[table.section][key]
			if !ok {
				if step <= int(next.MaxLevel) {
					return fmt.Errorf("[%s] missing %s", table.section, key)
				}
				continue
			}
			n, err := strconv.ParseUint(raw, 10, 32)
			if err != nil || n > table.limit {
				return fmt.Errorf("[%s] %s must be 0..%d", table.section, key, table.limit)
			}
			table.dst[step] = uint32(n)
		}
	}
	enchantConfig, weaponEnchantAttackBonus, armorEnchantAbsorption = next, weapon, armor
	if log != nil {
		log.Printf("GX Server World loaded ItemEnchant.ini max_level=%d", next.MaxLevel)
	}
	return nil
}

func (s *Session) sendEnchantRules() {
	payload := proto.EnchantRulesPacket(proto.EnchantRules{MaxLevel: enchantConfig.MaxLevel, WeaponBonus: weaponEnchantAttackBonus, ArmorBonus: armorEnchantAbsorption})
	w := packet.New(packet.Opcode(proto.GXEnchantRules))
	w.Bytes(payload[2:])
	s.send(w.MustFinish(), "GX enchant rules")
}
