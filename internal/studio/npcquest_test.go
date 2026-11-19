package studio

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/npcquest"
)

func TestNPCQuestSaveKeepsClientTextOutOfSQLite(t *testing.T) {
	ctx := t.Context()
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	path := filepath.Join(dataDir, NPCQuest.RelativePath())
	must(t, os.MkdirAll(filepath.Dir(path), 0700))
	dbPath := filepath.Join(root, "rapture.sqlite")
	db, err := openDB(ctx, dbPath, "rwc")
	must(t, err)
	_, err = db.ExecContext(ctx, npcquest.MigrationSQL+npcquest.SeedSQL)
	must(t, err)
	base, err := npcquest.Read(ctx, db)
	must(t, err)
	must(t, db.Close())
	plain, err := npcquest.Encode(base, nil, nil)
	must(t, err)
	must(t, os.WriteFile(path, plain, 0600))
	catalog, err := OpenNPCQuest(ctx, dataDir, dbPath)
	must(t, err)
	edited := npcquest.Clone(base)
	edited[npcquest.NPCs][0][8] = "Nombre de prueba"
	edited[npcquest.Quests][0][2] = "Misión de prueba"
	key := npcquest.TextKey{Table: npcquest.Quests, Row: 0, Field: "MsgSummary"}
	backup, err := SaveNPCQuest(ctx, root, catalog, edited, map[npcquest.TextKey]string{key: "Texto de la misión\nDetalle."})
	must(t, err)
	next, err := OpenNPCQuest(ctx, dataDir, dbPath)
	must(t, err)
	if !npcquest.Equal(next.Doc.Catalog, edited) || next.Doc.Text(key) != "Texto de la misión\nDetalle." {
		t.Fatal("client and server catalog differ after saving")
	}
	oldDB, err := openDB(ctx, filepath.Join(backup, "rapture.sqlite"), "ro")
	must(t, err)
	defer oldDB.Close()
	before, err := npcquest.Read(ctx, oldDB)
	must(t, err)
	if !npcquest.Equal(before, base) {
		t.Fatal("backup changed the original rows")
	}
	oldRaw, err := os.ReadFile(filepath.Join(backup, NPCQuest.filename()))
	must(t, err)
	oldDoc, err := npcquest.Parse(oldRaw)
	must(t, err)
	if oldDoc.Text(key) != "" || oldDoc.Catalog[npcquest.Quests][0][2] == next.Doc.Catalog[npcquest.Quests][0][2] {
		t.Fatal("backup changed original client dialog or quest name")
	}
}
