package world

import (
	"context"
	"fmt"

	"github.com/Spelunkern/Shaiya-GX/internal/enchantrules"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/logx"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
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

func applyEnchantRules(rows enchantrules.Catalog) error {
	weaponCap, armorCap, err := rows.Caps()
	if err != nil {
		return err
	}
	next := enchantRulesConfig{WeaponCap: weaponCap, ArmorCap: armorCap}
	var weapon, armor [enchantMaxLevel + 1]uint32
	for _, row := range rows.Attack {
		next.WeaponRate[row.Number-1], weapon[row.Number] = row.SuccessChance, row.Bonus
	}
	for _, row := range rows.Defense {
		next.DefenseRate[row.Number-1], armor[row.Number] = row.SuccessChance, row.Bonus
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
	if err != nil {
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
