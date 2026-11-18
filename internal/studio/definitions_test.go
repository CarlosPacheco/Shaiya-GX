package studio

import (
	"bytes"
	"encoding/binary"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

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
	_, err = db.ExecContext(ctx, "CREATE TABLE "+NPCSkills.Table()+" (SkillID INTEGER, SkillLevel INTEGER, "+strings.Join(declarations, ",")+")")
	must(t, err)
	_, err = db.ExecContext(ctx, "INSERT INTO "+NPCSkills.Table()+" (SkillID,SkillLevel,SkillName,SuccessValue) VALUES (1,1,'Player',70),(1,100,'NPC',15)")
	must(t, err)

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
		t.Fatal("NPC rank 1 should link to SQL level 100")
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
	must(t, db.QueryRowContext(ctx, "SELECT SuccessValue FROM "+NPCSkills.Table()+" WHERE SkillLevel=1").Scan(&player))
	must(t, db.QueryRowContext(ctx, "SELECT SuccessValue FROM "+NPCSkills.Table()+" WHERE SkillLevel=100").Scan(&npc))
	if player != 70 || npc != 19 {
		t.Fatalf("wrong namespace edited: player=%d, NPC=%d", player, npc)
	}
	original, err := readSData(filepath.Join(backup, NPCSkills.filename()))
	must(t, err)
	if !bytes.Equal(plain, original) {
		t.Fatal("backup does not match the original")
	}
}
