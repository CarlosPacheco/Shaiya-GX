package world

import (
	"context"
	"errors"
	"fmt"

	"github.com/Spelunkern/Shaiya-GX/internal/enchantrules"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/logx"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
	"path/filepath"
	"strconv"
)

const (
	defaultEnchantMaxLevel = 20
	enchantMaxLevel        = enchantrules.MaxLevel // armor encodes 50+level in the two-digit Craftname pair
)

// Runtime enchant configuration is loaded atomically from SQLite.
// These initial values retain the historical defaults before startup loading.
// Step N is the probability of N -> N+1, in successes per million.
type enchantRulesConfig struct {
	WeaponCap, ArmorCap     uint8
	WeaponRate, DefenseRate [enchantMaxLevel + 1]uint32
}

var enchantConfig = enchantRulesConfig{
	WeaponCap: defaultEnchantMaxLevel, ArmorCap: defaultEnchantMaxLevel,
	WeaponRate:  [enchantMaxLevel + 1]uint32{900000, 800000, 700000, 600000, 400000, 200000, 100000, 80000, 60000, 10000, 6000, 2000, 1000, 600, 200, 200, 200, 200, 200, 200, 0},
	DefenseRate: [enchantMaxLevel + 1]uint32{990000, 980000, 970000, 800000, 600000, 400000, 200000, 100000, 80000, 60000, 10000, 6000, 2000, 1000, 600, 200, 200, 200, 200, 200, 0},
}

// Bonuses are indexed by the resulting enchantment level, starting at zero.
var weaponEnchantAttackBonus = [enchantMaxLevel + 1]uint32{0, 7, 14, 21, 31, 41, 51, 64, 77, 90, 106, 122, 138, 157, 176, 195, 217, 239, 261, 286, 311}
var armorEnchantAbsorption = [enchantMaxLevel + 1]uint32{0, 5, 10, 15, 20, 25, 30, 35, 40, 45, 50, 55, 60, 65, 70, 75, 80, 85, 90, 95, 100}

func validateEnchantRules() error {
	if max(enchantConfig.WeaponCap, enchantConfig.ArmorCap) > enchantMaxLevel {
		return fmt.Errorf("enchant cap exceeds two-digit Craftname capacity (%d)", enchantMaxLevel)
	}
	for i := 0; i <= enchantMaxLevel; i++ {
		if enchantConfig.WeaponRate[i] > 1000000 || enchantConfig.DefenseRate[i] > 1000000 {
			return fmt.Errorf("enchant step %d exceeds 100%%", i)
		}
	}
	return nil
}

// readLegacyEnchant is used only to initialize a pre-migration installation.
func readLegacyEnchant(path string) (enchantrules.Catalog, error) {
	ini, err := readIni(path)
	if err != nil {
		return enchantrules.Catalog{}, err
	}
	cap := defaultEnchantMaxLevel
	var next enchantRulesConfig
	if raw, ok := ini["ENCHANT"]["MAXLEVEL"]; ok {
		n, err := strconv.ParseUint(raw, 10, 8)
		if err != nil || n > enchantMaxLevel {
			return enchantrules.Catalog{}, fmt.Errorf("MaxLevel must be 0..%d", enchantMaxLevel)
		}
		cap = int(n)
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
				if step <= cap {
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
	for level := 0; level <= cap; level++ {
		rows.Attack = append(rows.Attack, enchantrules.Level{Number: uint8(level), SuccessChance: next.WeaponRate[level], Bonus: weapon[level]})
		rows.Defense = append(rows.Defense, enchantrules.Level{Number: uint8(level), SuccessChance: next.DefenseRate[level], Bonus: armor[level]})
	}
	return rows, nil
}

func applyEnchantRules(rows enchantrules.Catalog) error {
	weaponCap, armorCap, err := rows.Caps()
	if err != nil {
		return err
	}
	next := enchantRulesConfig{WeaponCap: weaponCap, ArmorCap: armorCap}
	var weapon, armor [enchantMaxLevel + 1]uint32
	for _, row := range rows.Attack {
		next.WeaponRate[row.Number], weapon[row.Number] = row.SuccessChance, row.Bonus
	}
	for _, row := range rows.Defense {
		next.DefenseRate[row.Number], armor[row.Number] = row.SuccessChance, row.Bonus
	}
	enchantConfig, weaponEnchantAttackBonus, armorEnchantAbsorption = next, weapon, armor
	return nil
}

func loadEnchantRules(log *logx.Logger) error {
	ctx, cancel := context.WithTimeout(context.Background(), db.Timeout)
	defer cancel()
	pool, err := db.Pool(db.GameDefs)
	if err != nil {
		return err
	}
	rows, err := enchantrules.Read(ctx, pool)
	if errors.Is(err, enchantrules.ErrEmpty) {
		tx, err := pool.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		rows, err = enchantrules.Read(ctx, tx)
		if errors.Is(err, enchantrules.ErrEmpty) {
			rows, err = readLegacyEnchant(filepath.Join(gameDataDir(), "ItemEnchant.ini"))
			if err != nil {
				return fmt.Errorf("initialize empty enchant tables from legacy ItemEnchant.ini: %w", err)
			}
			if err := enchantrules.Replace(ctx, tx, rows); err != nil {
				return err
			}
			stored, err := enchantrules.Read(ctx, tx)
			if err != nil || !stored.Equal(rows) {
				return errors.Join(errors.New("legacy enchant import verification failed"), err)
			}
		} else if err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	if err := applyEnchantRules(rows); err != nil {
		return err
	}
	if log != nil {
		log.Printf("GX Server World loaded enchant rules weapon_cap=%d armor_cap=%d", enchantConfig.WeaponCap, enchantConfig.ArmorCap)
	}
	return nil
}

func (s *Session) sendEnchantRules() {
	// The legacy packet has one display bound. Quotes and attempts use
	// their equipment-specific cap on the server.
	payload := proto.EnchantRulesPacket(proto.EnchantRules{MaxLevel: max(enchantConfig.WeaponCap, enchantConfig.ArmorCap), WeaponBonus: weaponEnchantAttackBonus, ArmorBonus: armorEnchantAbsorption})
	w := packet.New(packet.Opcode(proto.GXEnchantRules))
	w.Bytes(payload[2:])
	s.send(w.MustFinish(), "GX enchant rules")
}
