package db

import (
	"context"
	"testing"
	"time"
)

func TestWorldOwnershipAndAccountLeases(t *testing.T) {
	sqliteFixture(t)
	release, err := AcquireWorld(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	c, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if another, err := AcquireWorld(c); err == nil {
		another()
		t.Fatal("two simulations own one runtime")
	}
	uid, err := CreateAccount("lease_account", "development", false)
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := AcquireAccount(uint32(uid), 1); err != nil || !ok {
		t.Fatal(ok, err)
	}
	if ok, err := AcquireAccount(uint32(uid), 2); err != nil || ok {
		t.Fatal("duplicate active account", ok, err)
	}
	if err := ReleaseAccount(uint32(uid), 2); err != nil {
		t.Fatal(err)
	}
	if result, err := TryGameLogin("lease_account", "development", 3, "127.0.0.1"); err != nil || result.Status != 1 {
		t.Fatal("active login overwritten", result, err)
	}
	if err := ReleaseAccount(uint32(uid), 1); err != nil {
		t.Fatal(err)
	}
	if result, err := TryGameLogin("lease_account", "development", 3, "127.0.0.1"); err != nil || result.Status != 0 {
		t.Fatal(result, err)
	}
}
func TestFailedSnapshotPreservesQuickSlotsAndBuffs(t *testing.T) {
	p := sqliteFixture(t)
	ch, _ := economyCharacter(t)
	slots := []QuickSlot{{QuickBar: 0, QuickSlot: 0, Number: 1}, {QuickBar: 0, QuickSlot: 1, Number: 2}}
	if err := SaveQuickSlots(ch.ID, slots); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Exec(`CREATE TRIGGER GX_QuickFailure BEFORE INSERT ON PS_GameData_CharQuickSlots WHEN NEW.Number=3 BEGIN SELECT RAISE(ABORT,'injected');END`); err != nil {
		t.Fatal(err)
	}
	if err := SaveQuickSlots(ch.ID, []QuickSlot{{Number: 3}}); err == nil {
		t.Fatal("failed snapshot committed")
	}
	actual, err := QuickSlots(ch.ID)
	if err != nil || len(actual) != 2 {
		t.Fatal("old bar lost", actual, err)
	}
	original := []AppliedBuff{{SkillID: 1, Level: 1, LeftSeconds: 30}}
	if err := SaveAppliedBuffs(ch.ID, original); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Exec(`CREATE TRIGGER GX_BuffFailure BEFORE INSERT ON PS_GameData_CharApplySkills WHEN NEW.SkillID=2 BEGIN SELECT RAISE(ABORT,'injected');END`); err != nil {
		t.Fatal(err)
	}
	if err := SaveAppliedBuffs(ch.ID, []AppliedBuff{{SkillID: 2, Level: 1, LeftSeconds: 5}}); err == nil {
		t.Fatal("failed buffs committed")
	}
	buffs, err := AppliedBuffs(ch.ID)
	if err != nil || len(buffs) != 1 || buffs[0] != original[0] {
		t.Fatal("old buffs lost", buffs, err)
	}
}
func TestSkillCostFailureDoesNotGrantSkill(t *testing.T) {
	p := sqliteFixture(t)
	ch, _ := economyCharacter(t)
	after := ch
	after.SkillPoint--
	if _, err := p.Exec(`CREATE TRIGGER GX_SkillFailure BEFORE UPDATE OF SkillPoint ON PS_GameData_Chars BEGIN SELECT RAISE(ABORT,'injected');END`); err != nil {
		t.Fatal(err)
	}
	if err := CommitLearnSkill(ch, after, nil, CharacterSkill{SkillID: 1, Level: 1, Number: 1}); err == nil {
		t.Fatal("free skill granted")
	}
	skills, err := CharacterSkills(ch.ID)
	if err != nil || len(skills) != 0 {
		t.Fatal("partial skill survived", skills, err)
	}
}
