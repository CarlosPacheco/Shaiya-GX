package studio

import (
	"bytes"
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

func TestNPCQuestFileSavePreservesSeparateTexts(t *testing.T) {
	ctx := t.Context()
	root := t.TempDir()
	clientDir := filepath.Join(root, "client")
	client := filepath.Join(clientDir, NPCQuest.RelativePath())
	server := filepath.Join(root, "game", "NpcQuest.SData")
	seed, err := os.ReadFile(filepath.Join("..", "..", "server", "data", "game", "NpcQuest.SData"))
	must(t, err)
	for _, path := range []string{client, server} {
		must(t, os.MkdirAll(filepath.Dir(path), 0700))
		must(t, os.WriteFile(path, seed, 0600))
	}
	project, err := OpenNPCQuest(ctx, clientDir, server)
	must(t, err)
	edited := npcquest.Clone(project.Doc.Catalog)
	edited[npcquest.NPCs][0][8] = "NPC File Test"
	key := npcquest.TextKey{Table: npcquest.Quests, Row: 0, Field: "MsgSummary"}
	serverText := project.serverDoc.Text(key)
	backup, err := SaveNPCQuest(ctx, root, project, edited, map[npcquest.TextKey]string{key: "Client-only dialog."})
	must(t, err)
	next, err := OpenNPCQuest(ctx, clientDir, server)
	must(t, err)
	if !npcquest.Equal(next.Doc.Catalog, edited) || !npcquest.Equal(next.serverDoc.Catalog, edited) ||
		next.Doc.Text(key) != "Client-only dialog." || next.serverDoc.Text(key) != serverText {
		t.Fatal("the shared rules or client/server text changed")
	}
	for _, name := range []string{"client-before.SData", "server-before.SData"} {
		raw, err := os.ReadFile(filepath.Join(backup, name))
		must(t, err)
		if !bytes.Equal(raw, seed) {
			t.Fatal("backup does not match the original")
		}
	}
	// Simulate an interruption after the client replacement.
	must(t, os.WriteFile(server, seed, 0600))
	must(t, os.Rename(filepath.Join(backup, "save.json"), filepath.Join(root, pendingName)))
	_, err = Recover(ctx, root)
	must(t, err)
	restored, err := OpenNPCQuest(ctx, clientDir, server)
	must(t, err)
	if !npcquest.Equal(restored.serverDoc.Catalog, edited) || restored.Doc.Text(key) != "Client-only dialog." {
		t.Fatal("recovery did not finish the two-file save")
	}

}
