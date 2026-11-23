package studio

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/catalogfiles"
	"github.com/Spelunkern/Shaiya-GX/internal/enchantrules"
	"github.com/Spelunkern/Shaiya-GX/internal/gameini"
)

func TestSaveEnchantINIUpdatesOnlyIntendedSettings(t *testing.T) {
	root := t.TempDir()
	ctx := t.Context()
	path := filepath.Join(root, catalogfiles.Enchant)
	original := []byte("\ufeff" + strings.ReplaceAll(`; Customized installation
[Enchant]
MaxLevel=1
DefenseMaxLevel=0
LocalSwitch=enabled

[LapisianEnchantPercentRate]
WeaponStep00=900000
WeaponStep01=42
WeaponStep08=418
DefenseStep00=33

[LapisianEnchantAddValue]
WeaponStep00=19
WeaponStep01=02147483647
WeaponStep08=27
DefenseStep00=11

[Other]
Note="  // keep /* this */; \"quoted\"\nline  "
`, "\n", "\r\n"))
	must(t, os.WriteFile(path, original, 0600))
	catalog, err := OpenEnchant(ctx, path)
	must(t, err)
	stale, err := OpenEnchant(ctx, path)
	must(t, err)
	unchanged, err := catalogfiles.UpdateEnchant(catalog.Rules, original)
	must(t, err)
	if !bytes.Equal(unchanged, original) {
		t.Fatal("unmodified encoding changed the original bytes")
	}

	settings, err := gameini.Parse(original)
	must(t, err)
	assertSave := func(next enchantrules.Catalog) {
		t.Helper()
		before, err := os.ReadFile(path)
		must(t, err)
		backup, err := SaveEnchant(ctx, root, catalog, next)
		must(t, err)
		backupBytes, err := os.ReadFile(filepath.Join(backup, catalogfiles.Enchant))
		must(t, err)
		if !bytes.Equal(backupBytes, before) {
			t.Fatal("backup differs from the pre-save file")
		}
		reopened, err := OpenEnchant(ctx, path)
		must(t, err)
		if !reopened.Rules.Equal(next) {
			t.Fatalf("reopened rules = %+v, want %+v", reopened.Rules, next)
		}
		raw := []byte(reopened.File.Text())
		actual, err := gameini.Parse(raw)
		must(t, err)
		if !reflect.DeepEqual(actual, settings) {
			t.Fatalf("reopened settings = %v, want %v", actual, settings)
		}
		catalog.Rules = next
	}

	attack := []enchantrules.Level{{Number: 1, SuccessChance: 125000, Bonus: 2147483647}}
	settings["LAPISIANENCHANTPERCENTRATE"]["WEAPONSTEP00"] = "125000"
	assertSave(enchantrules.Catalog{Attack: attack})
	if _, err := SaveEnchant(ctx, root, stale, enchantrules.Catalog{Attack: attack}); err == nil {
		t.Fatal("stale editor was allowed to save")
	}

	defense := []enchantrules.Level{{Number: 1, SuccessChance: 1, Bonus: 8}}
	settings["ENCHANT"]["DEFENSEMAXLEVEL"] = "1"
	settings["LAPISIANENCHANTPERCENTRATE"]["DEFENSESTEP00"] = "1"
	settings["LAPISIANENCHANTPERCENTRATE"]["DEFENSESTEP01"] = "0"
	settings["LAPISIANENCHANTADDVALUE"]["DEFENSESTEP01"] = "8"
	assertSave(enchantrules.Catalog{Attack: attack, Defense: defense})

	defense = append(defense, enchantrules.Level{Number: 2, SuccessChance: 1000000, Bonus: 10})
	settings["ENCHANT"]["MAXLEVEL"] = "2"
	settings["ENCHANT"]["WEAPONMAXLEVEL"] = "1"
	settings["ENCHANT"]["DEFENSEMAXLEVEL"] = "2"
	settings["LAPISIANENCHANTPERCENTRATE"]["DEFENSESTEP01"] = "1000000"
	settings["LAPISIANENCHANTPERCENTRATE"]["DEFENSESTEP02"] = "0"
	settings["LAPISIANENCHANTADDVALUE"]["DEFENSESTEP02"] = "10"
	assertSave(enchantrules.Catalog{Attack: attack, Defense: defense})

	settings["ENCHANT"]["MAXLEVEL"] = "1"
	settings["ENCHANT"]["DEFENSEMAXLEVEL"] = "0"
	assertSave(enchantrules.Catalog{Attack: attack})
}

func TestSaveEnchantBacksUpAndRejectsGaps(t *testing.T) {
	ctx := t.Context()
	root := t.TempDir()
	path := filepath.Join(root, "local.sqlite")
	db, err := openDB(ctx, path, "rwc")
	must(t, err)
	defer db.Close()
	_, err = db.Exec(enchantrules.CreateSQL)
	must(t, err)
	tx, err := db.BeginTx(ctx, nil)
	must(t, err)
	initial := enchantrules.Catalog{Defense: []enchantrules.Level{{Number: 1, Bonus: 5}}}
	must(t, enchantrules.Replace(ctx, tx, initial))
	must(t, tx.Commit())
	catalog, err := OpenEnchant(ctx, path)
	must(t, err)
	bad := enchantrules.Catalog{Attack: []enchantrules.Level{{Number: 2}}, Defense: initial.Defense}
	if _, err := SaveEnchant(ctx, root, catalog, bad); err == nil {
		t.Fatal("gap in attack levels was accepted")
	}
	next := enchantrules.Catalog{Attack: []enchantrules.Level{{Number: 1, SuccessChance: 900000, Bonus: 7}}}
	backup, err := SaveEnchant(ctx, root, catalog, next)
	must(t, err)
	old, err := OpenEnchant(ctx, filepath.Join(backup, "rapture.sqlite"))
	must(t, err)
	saved, err := OpenEnchant(ctx, path)
	must(t, err)
	attack, defense, err := saved.Rules.Caps()
	must(t, err)
	if attack != 1 || defense != 0 || !old.Rules.Equal(initial) || !saved.Rules.Equal(next) {
		t.Fatal("backup or independent caps differ from the requested catalogs")
	}
	if _, err := SaveEnchant(ctx, root, catalog, next); err == nil {
		t.Fatal("stale draft was accepted")
	}
}
