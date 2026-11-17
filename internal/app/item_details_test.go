package app

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/data/text"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func TestCraftValuesDecodeAndRejectInvalid(t *testing.T) {
	v, ok := parseCraftValues("01020304050607080953  ")
	if !ok || v.stats != [9]uint16{1, 2, 3, 4, 5, 6, 700, 800, 900} || v.enchant != 3 {
		t.Fatalf("unexpected values: %+v %t", v, ok)
	}
	for _, bad := range []string{"", "0102030405060708095", "010203040506070809x3"} {
		if _, ok := parseCraftValues(bad); ok {
			t.Fatalf("invalid craft accepted: %q", bad)
		}
	}
	if got := craftSummary("01000000000001000000"); got != "STR +1, HP +100" {
		t.Fatal(got)
	}
}

func TestItemDetailUsesInstanceAndGemDefinitions(t *testing.T) {
	def := &sdata.Item{Name: "Sword", Quality: 1200, Attack: 12}
	item := &proto.InventoryItem{Count: 1, Quality: 17, Craft: "01000000000000000002", Gems: [6]uint8{5, 9}}
	rows := itemDetailLines(def, item, 1, "Sell: 10 gold", func(id uint8) string {
		if id == 5 {
			return "Power Lapis"
		}
		return ""
	})
	var texts []string
	for _, row := range rows {
		texts = append(texts, row.text)
	}
	joined := strings.Join(texts, "|")
	if strings.Contains(joined, "endurance") || strings.Contains(joined, "Durability") {
		t.Fatal("legacy durability remains visible", joined)
	}
	for _, want := range []string{"Sword [2]", "STR +1", "Lapis: Power Lapis", "Lapis: Lapis 9"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in %q", want, joined)
		}
	}
	shop := itemDetailLines(def, nil, 1, "Buy: 100 gold", nil)
	for _, row := range shop {
		if strings.Contains(row.text, "endurance") || strings.Contains(row.text, "Lapis:") {
			t.Fatal("shop definition invented mutable instance state")
		}
	}
}

func TestItemDescriptionRemovesColorControls(t *testing.T) {
	got := stripItemColors("{c5}Enhancement{/c}: {custom} +2")
	if got != "Enhancement: {custom} +2" {
		t.Fatal(got)
	}
}

func TestTooltipPreservesInlineColorsAndCompleteDescription(t *testing.T) {
	glyphs := tooltipGlyphs("Plain {c3}gold{/c} tail", white)
	var plain strings.Builder
	yellow := 0
	for _, g := range glyphs {
		plain.WriteRune(g.char)
		if g.color == ([4]float32{1, 1, 0, 1}) {
			yellow++
		}
	}
	if plain.String() != "Plain gold tail" || yellow != 4 {
		t.Fatal("inline description colours lost", plain.String(), yellow)
	}
	rows := itemDetailLines(&sdata.Item{Name: "Item", Description: "first\\n{c3}second{/c}\nthird"}, nil, 1, "", nil)
	if len(rows) != 4 || rows[2].text != "{c3}second{/c}" || rows[3].text != "third" {
		t.Fatal("item description was truncated", rows)
	}
}

func TestEquipmentTooltipHasNoModeRequirement(t *testing.T) {
	rows := itemDetailLines(&sdata.Item{Name: "Goddess Anor Helmet", Grow: 3}, nil, 1, "", nil)
	for _, row := range rows {
		if strings.Contains(row.text, "Required mode") {
			t.Fatal("obsolete mode requirement visible")
		}
	}
}

func TestLapisTooltipTargetsFollowPlacementFlags(t *testing.T) {
	for _, typ := range []uint8{30, 95} {
		def := &sdata.Item{Type: typ, Name: "Gem", MinLevel: 1, Country: 1,
			Classes: [6]uint8{1, 0, 1, 0, 1, 0}, Grow: 1,
			RequiredStats: [6]uint16{1}, Description: "original description"}
		rows := itemDetailLines(def, nil, 1, "", nil)
		var joined string
		for _, row := range rows {
			joined += "|" + row.text
		}
		for _, want := range []string{"Item possible to equip", "Weapons", "Helmets", "Upper armor", "Gauntlet", "Shields", "Necklaces", "Coat", "Bracelets"} {
			if !strings.Contains(joined, "|"+want) {
				t.Fatalf("type %d missing %s: %s", typ, want, joined)
			}
		}
		for _, absent := range []string{"Lower armor", "Shoes", "Rings", "level 1"} {
			if strings.Contains(joined, "|"+absent) {
				t.Fatalf("type %d advertises unsupported %s", typ, absent)
			}
		}
		if rows[len(rows)-1].text != def.Description {
			t.Fatal("description must remain last for visual tooltip splitting")
		}
	}
}

func TestSpeedTooltipUsesNativeFieldsActiveLapisAndSysMsg(t *testing.T) {
	a := inputApp()
	a.msg = map[int]string{5197: "ATTACK NORM", 5276: "NORM", 5198: "ATTACK QUICK", 5209: "MOVE UP"}
	weapon := &sdata.Item{Type: 1, TypeID: 1, Name: "Test weapon", AttackTime: 5, Speed: 1, Attribute: 1}
	a.items = map[sdata.ItemKey]*sdata.Item{
		{Type: 1, TypeID: 1}:  weapon,
		{Type: 30, TypeID: 1}: {Type: 30, TypeID: 1, Name: "Active speed", AttackTime: 1, Speed: 2},
		{Type: 30, TypeID: 2}: {Type: 30, TypeID: 2, Name: "Inactive element", Attribute: 2, AttackTime: 3, Speed: 3},
	}
	it := proto.InventoryItem{Type: 1, TypeID: 1, Gems: [6]uint8{1, 2}}
	a.drawItemDetails(weapon, &it, 1, 0, 0, 1024, 768, "")
	has := func(want string) bool {
		for _, r := range a.tooltip {
			if r.text == want {
				return true
			}
		}
		return false
	}
	if !has("ATTACK QUICK") || !has("MOVE UP +3") {
		t.Fatal("native/socket speed missing or inactive lapis applied", a.tooltip)
	}
	if weapon.AttackTime != 5 || weapon.Speed != 1 {
		t.Fatal("tooltip mutated cached definition")
	}
	for _, d := range []*sdata.Item{{Type: 22, AttackTime: 1}, {Type: 30, AttackTime: 2}} {
		rows := itemDetailLines(d, nil, 1, "", nil, a.msg)
		found := false
		for _, r := range rows {
			found = found || r.text == "ATTACK +"+fmt.Sprint(d.AttackTime)
		}
		if !found {
			t.Fatal("native attack bonus missing", rows)
		}
	}
	rows := itemDetailLines(&sdata.Item{Type: 25, AttackTime: 1, Speed: 9}, nil, 1, "", nil, a.msg)
	for _, r := range rows {
		if strings.HasPrefix(r.text, "ATTACK") || strings.HasPrefix(r.text, "MOVE UP") {
			t.Fatal("consumable skill level displayed as equipment speed")
		}
	}
}

func TestEquipmentAbsorptionTooltip(t *testing.T) {
	for _, tt := range []struct {
		typ, absorb uint8
		want        bool
	}{
		{17, 30, true}, {1, 255, true}, {40, 10, true}, {17, 0, false},
		{25, 30, false}, {30, 30, false}, {100, 30, false},
	} {
		def := &sdata.Item{Type: tt.typ, ExpAbsorb: tt.absorb}
		rows := itemDetailLines(def, nil, 1, "", nil, text.SysMsg{485: "ABSORB"})
		found := false
		for _, row := range rows {
			found = found || row.text == fmt.Sprintf("ABSORB +%d", tt.absorb)
		}
		if found != tt.want {
			t.Fatalf("type=%d Exp=%d: absorption shown=%v, want %v", tt.typ, tt.absorb, found, tt.want)
		}
	}
}
