package world

import (
	"database/sql"
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestItemEnchantIniRatesBonusesCapAndWire(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "enchant-legacy.ini"))
	if err != nil {
		t.Fatal(err)
	}
	old, weapon, armor := enchantConfig, weaponEnchantAttackBonus, armorEnchantAbsorption
	t.Cleanup(func() { enchantConfig, weaponEnchantAttackBonus, armorEnchantAbsorption = old, weapon, armor })
	dir := t.TempDir()
	t.Setenv("RAPTURE_DATA_DIR", dir)
	if err := os.Mkdir(filepath.Join(dir, "game"), 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "game", "ItemEnchant.ini")
	write := func(s string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(s), 0600); err != nil {
			t.Fatal(err)
		}
	}
	load := func() error {
		rows, err := db.ReadLegacyEnchant(path)
		if err != nil {
			return err
		}
		return applyEnchantRules(rows)
	}
	// Custom cap, chance and armor absorption must reach execution and client.
	text := strings.Replace(string(raw), "MaxLevel=20", "MaxLevel=21", 1)
	text = strings.Replace(text, "WeaponStep00\t=900000", "WeaponStep00=123456", 1)
	text = strings.Replace(text, "DefenseStep20\t=100", "DefenseStep20=777", 1)
	text = strings.Replace(text, "[LapisianEnchantAddValue]", "WeaponStep21=0\nDefenseStep21=0\n[LapisianEnchantAddValue]", 1)
	text += "\nWeaponStep21=400\nDefenseStep21=888\n"
	write(text)
	if err := load(); err != nil {
		t.Fatal(err)
	}
	if enchantConfig.WeaponCap != 21 || enchantConfig.WeaponRate[0] != 123456 || armorEnchantAbsorption[20] != 777 || weaponEnchantAttackBonus[21] != 400 {
		t.Fatal("INI not authoritative")
	}
	if rate, ok := enchantRate(db.Item{Type: 1}, 1); !ok || rate != 123456 {
		t.Fatal("quote/execution ignored INI", rate, ok)
	}
	if _, ok := enchantRate(db.Item{Type: 1, CraftName: "00000000000000000021"}, 1); ok {
		t.Fatal("cap ignored")
	}
	oldDefs := db.MustDefs()
	t.Cleanup(func() { db.SetDefsForTest(oldDefs) })
	db.SetDefsForTest(&db.Defs{Items: map[db.ItemKey]*db.ItemDef{
		{Type: 1, TypeID: 1}:  {Type: 1, TypeID: 1, Server: 1, ReqWis: 1},
		{Type: 17, TypeID: 1}: {Type: 17, TypeID: 1, Server: 1, ReqWis: 1},
	}})
	equip := GetEquipmentStats([]db.Item{
		{Bag: 0, Slot: 5, Type: 1, TypeID: 1, CraftName: "00000000000000000021"},
		{Bag: 0, Slot: 1, Type: 17, TypeID: 1, CraftName: "00000000000000000071"},
	})
	if equip.ItemAttack != 400 || equip.Absorption != 888 {
		t.Fatalf("combat ignored INI bonuses: %+v", equip)
	}
	// Transport round-trip through a real server frame and game state.
	s, packets := gmWireSession(t, db.Character{ID: 990047}, false)
	s.sendEnchantRules()
	select {
	case p := <-packets:
		st := game.New()
		if _, err := st.Apply(p); err != nil {
			t.Fatal(err)
		}
		if st.EnchantRules == nil || st.EnchantRules.ArmorBonus[20] != 777 || st.EnchantRules.MaxLevel != 21 {
			t.Fatal("client rules differ", st.EnchantRules)
		}
	case <-time.After(time.Second):
		t.Fatal("missing rules frame")
	}
	good, w, a := enchantConfig, weaponEnchantAttackBonus, armorEnchantAbsorption
	for _, bad := range []string{
		strings.Replace(text, "MaxLevel=21", "MaxLevel=50", 1),
		strings.Replace(text, "WeaponStep00=123456", "WeaponStep00=1000001", 1),
		strings.Replace(text, "DefenseStep20=777", "DefenseStep20=-1", 1),
		strings.Replace(text, "DefenseStep20=777", "DefenseStep20=2147483648", 1),
		strings.Replace(text, "WeaponStep21=400", "", 1),
	} {
		write(bad)
		if load() == nil {
			t.Fatal("invalid INI accepted")
		}
		if enchantConfig != good || weaponEnchantAttackBonus != w || armorEnchantAbsorption != a {
			t.Fatal("partial invalid rules published")
		}
	}
	var full strings.Builder
	full.WriteString("[Enchant]\nMaxLevel=49\n")
	for _, section := range []string{"LapisianEnchantPercentRate", "LapisianEnchantAddValue"} {
		fmt.Fprintf(&full, "[%s]\n", section)
		for i := 0; i <= 49; i++ {
			fmt.Fprintf(&full, "WeaponStep%02d=1\nDefenseStep%02d=2\n", i, i)
		}
	}
	write(full.String())
	if err := load(); err != nil || enchantConfig.WeaponCap != 49 {
		t.Fatal("cap49 rejected", err)
	}
	write(strings.Replace(string(raw), "MaxLevel=20", "MaxLevel=0", 1))
	if err := load(); err != nil {
		t.Fatal(err)
	}
	if _, ok := enchantRate(db.Item{Type: 1}, 1); ok {
		t.Fatal("cap0 still allows enhancement")
	}
	if armorEnchantAbsorption[20] != 0 {
		t.Fatal("removed levels must have no bonus")
	}
	// Original file without the new section remains compatible, cap20.
	write(strings.Replace(string(raw), "[Enchant]\nMaxLevel=20\n", "", 1))
	if err := load(); err != nil || enchantConfig.WeaponCap != 20 {
		t.Fatal("legacy INI rejected", err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if load() == nil {
		t.Fatal("missing required INI silently accepted")
	}
}

func TestEnchantCatalogImportsLegacyOnce(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("RAPTURE_DATA_DIR", dir)
	t.Setenv("RAPTURE_DB_PATH", filepath.Join(dir, "rapture.sqlite"))
	seed, err := filepath.Abs("../../data/rapture.sqlite")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("RAPTURE_DB_SEED", seed)
	old, weapon, armor := enchantConfig, weaponEnchantAttackBonus, armorEnchantAbsorption
	t.Cleanup(func() { enchantConfig, weaponEnchantAttackBonus, armorEnchantAbsorption = old, weapon, armor })
	// Remove the enchant tables to simulate a database from before their migration.
	seedBytes, err := os.ReadFile(seed)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "rapture.sqlite"), seedBytes, 0600); err != nil {
		t.Fatal(err)
	}
	legacyDB, err := sql.Open("sqlite", filepath.Join(dir, "rapture.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := legacyDB.Exec("DROP TABLE PS_GameDefs_EnchantAttack; DROP TABLE PS_GameDefs_EnchantDefense"); err != nil {
		t.Fatal(err)
	}
	if err := legacyDB.Close(); err != nil {
		t.Fatal(err)
	}
	db.Close()
	t.Cleanup(func() { _ = db.Close() })
	raw, err := os.ReadFile(filepath.Join("testdata", "enchant-legacy.ini"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(gameDataDir(), 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(gameDataDir(), "ItemEnchant.ini")
	raw = []byte(strings.ReplaceAll(string(raw), "MaxLevel=20", "MaxLevel=0"))
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if err := loadEnchantRules(nil); err != nil {
		t.Fatal(err)
	}
	if enchantConfig.WeaponCap != 0 || enchantConfig.ArmorCap != 0 {
		t.Fatal("legacy import changed cap")
	}
	if err := os.WriteFile(path, []byte(strings.ReplaceAll(string(raw), "MaxLevel=0", "MaxLevel=20")), 0600); err != nil {
		t.Fatal(err)
	}
	if err := loadEnchantRules(nil); err != nil || enchantConfig.WeaponCap != 0 || enchantConfig.ArmorCap != 0 {
		t.Fatalf("the legacy INI was imported more than once: %v", err)
	}
	pool, err := db.Pool(db.GameDefs)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec("INSERT INTO PS_GameDefs_EnchantAttack VALUES(1, 0, 7)"); err != nil {
		t.Fatal(err)
	}
	if err := loadEnchantRules(nil); err != nil || enchantConfig.WeaponCap != 1 || enchantConfig.ArmorCap != 0 {
		t.Fatalf("SQLite-only reload failed: %v", err)
	}
	if _, ok := enchantRate(db.Item{Type: 1}, 1); !ok {
		t.Fatal("attack cap did not allow weapon enchant")
	}
	if _, ok := enchantRate(db.Item{Type: 17}, 1); ok {
		t.Fatal("defense cap allowed armor enchant")
	}
}
