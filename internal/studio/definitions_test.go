package studio

import (
	"bytes"
	"encoding/binary"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func TestRaidTargetSavesInBothPlayerSkillCatalogs(t *testing.T) {
	ctx, root := t.Context(), t.TempDir()
	path := filepath.Join(root, "installed.sqlite")
	db, err := openDB(ctx, path, "rwc")
	must(t, err)
	defer db.Close()
	columns := mappedColumns(Skills)
	for i := range columns {
		columns[i] += " INTEGER DEFAULT 0"
	}
	columns[0] = quote("SkillName") + " TEXT"
	_, err = db.ExecContext(ctx, "CREATE TABLE "+Skills.Table()+" (SkillID INTEGER, SkillLevel INTEGER, "+strings.Join(columns, ",")+")")
	must(t, err)
	_, err = db.ExecContext(ctx, "INSERT INTO "+Skills.Table()+" (SkillID,SkillLevel,SkillName,TypeEffect,TargetType) VALUES (1,1,'Raid',4,4)")
	must(t, err)
	body := make([]byte, 116)
	body[0], body[skillEffectOffset], body[skillTargetOffset] = 1, 4, 4
	plain := binary.LittleEndian.AppendUint32(nil, 1)
	plain = binary.LittleEndian.AppendUint32(plain, 4)
	plain = append(plain, "Raid"...)
	plain = binary.LittleEndian.AppendUint32(plain, 0)
	plain = append(plain, body...)
	plain = append(plain, make([]byte, 8*124)...)
	file := filepath.Join(root, Skills.RelativePath())
	writeFixture(t, file, plain)
	d, err := OpenDefinitions(ctx, root, path, Skills)
	must(t, err)
	if err := d.ValidateEdit(0, map[string]string{"TargetType": "8", "TypeEffect": "1"}); err == nil {
		t.Fatal("an attack cannot use the whole-raid target")
	}
	_, err = SaveDefinitions(ctx, root, d, DefinitionEdits{0: {"TargetType": "8"}})
	must(t, err)
	saved, err := readSData(file)
	must(t, err)
	defs, err := sdata.ParseSkills(saved)
	must(t, err)
	var target int
	must(t, db.QueryRowContext(ctx, "SELECT TargetType FROM "+Skills.Table()+" WHERE SkillID=1 AND SkillLevel=1").Scan(&target))
	if target != proto.SkillTargetRaid || defs[sdata.SkillKey{ID: 1, Level: 1}].Target != proto.SkillTargetRaid {
		t.Fatal("whole-raid selector did not round-trip in both catalogs")
	}
}

func TestNPCSkillUpdatesOnlyItsOwnNamespace(t *testing.T) {
	ctx := t.Context()
	root := t.TempDir()
	path := filepath.Join(root, "installed.sqlite")
	db, err := openDB(ctx, path, "rwc")
	must(t, err)
	defer db.Close()
	columns := mappedColumns(NPCSkills)
	declarations := slices.Clone(columns)
	for i := range declarations {
		declarations[i] += " INTEGER DEFAULT 0"
	}
	declarations[0] = quote("SkillName") + " TEXT"
	for _, table := range []struct {
		kind       Kind
		rank, name string
		chance     int
	}{{Skills, "SkillLevel", "Player", 70}, {NPCSkills, "Rank", "NPC", 15}} {
		_, err = db.ExecContext(ctx, "CREATE TABLE "+table.kind.Table()+" (SkillID INTEGER, "+table.rank+" INTEGER, "+strings.Join(declarations, ",")+")")
		must(t, err)
		_, err = db.ExecContext(ctx, "INSERT INTO "+table.kind.Table()+" (SkillID,"+table.rank+",SkillName,SuccessValue) VALUES (1,1,?,?)", table.name, table.chance)
		must(t, err)
	}

	body := make([]byte, 116)
	body[0], body[50], body[51], body[55] = 1, 0xf1, 0xff, 15
	plain := binary.LittleEndian.AppendUint32(nil, 1)
	plain = binary.LittleEndian.AppendUint32(plain, 4)
	plain = append(plain, "NPC\x00"...)
	plain = binary.LittleEndian.AppendUint32(plain, 0)
	plain = append(plain, body...)
	plain = append(plain, make([]byte, 8*124)...)
	target := filepath.Join(root, NPCSkills.RelativePath())
	writeFixture(t, target, plain)
	project, err := OpenDefinitions(ctx, root, path, NPCSkills)
	must(t, err)
	if len(project.Records) != 1 || !project.Records[0].DatabaseRow {
		t.Fatal("NPC rank 1 should link to the NPC catalog")
	}
	backup, err := SaveDefinitions(ctx, root, project, DefinitionEdits{0: {"SuccessValue": "19"}})
	must(t, err)
	saved, err := readSData(target)
	must(t, err)
	expected := bytes.Clone(plain)
	expected[project.Records[0].body+55] = 19
	if !bytes.Equal(expected, saved) {
		t.Fatal("editing one skill parameter changed untouched SData bytes")
	}

	var player, npc int
	must(t, db.QueryRowContext(ctx, "SELECT SuccessValue FROM "+Skills.Table()+" WHERE SkillLevel=1").Scan(&player))
	must(t, db.QueryRowContext(ctx, "SELECT SuccessValue FROM "+NPCSkills.Table()+" WHERE Rank=1").Scan(&npc))
	if player != 70 || npc != 19 {
		t.Fatalf("wrong namespace edited: player=%d, NPC=%d", player, npc)
	}
	original, err := readSData(filepath.Join(backup, NPCSkills.filename()))
	must(t, err)
	if !bytes.Equal(plain, original) {
		t.Fatal("backup does not match the original")
	}
}

func TestReservedItemRowsSurviveDefinitionEdits(t *testing.T) {
	plain := binary.LittleEndian.AppendUint32(nil, 1)
	plain = binary.LittleEndian.AppendUint32(plain, 2)
	// Native reserved item slots use NUL-terminated empty strings and a zero body.
	for range 2 {
		plain = binary.LittleEndian.AppendUint32(plain, 1)
		plain = append(plain, 0)
	}
	plain = append(plain, make([]byte, 81)...)
	plain = binary.LittleEndian.AppendUint32(plain, 0)
	plain = binary.LittleEndian.AppendUint32(plain, 0)
	body := make([]byte, 81)
	body[0], body[1] = 1, 2
	plain = append(plain, body...)
	records, err := definitionRecords(Items, plain)
	must(t, err)
	if len(records) != 1 || records[0].ID != 1002 {
		t.Fatalf("unexpected editable item records: %+v", records)
	}
	records[0].DatabaseRow = true
	definitions := &Definitions{Kind: Items, Records: records, plain: plain}
	patched, _, err := definitions.apply(DefinitionEdits{0: {"Name": "Edited"}})
	must(t, err)
	if !bytes.Equal(patched[:records[0].start], plain[:records[0].start]) {
		t.Fatal("editing an item changed the preceding reserved bytes")
	}
}
