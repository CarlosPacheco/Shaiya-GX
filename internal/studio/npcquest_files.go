package studio

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Spelunkern/Shaiya-GX/internal/catalogfiles"
	"github.com/Spelunkern/Shaiya-GX/internal/data/cash"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/npcquest"
)

func saveNPCQuestFiles(root string, project *NPCQuestCatalog, catalog npcquest.Catalog, texts map[npcquest.TextKey]string, client []byte) (string, error) {
	server := project.Server.original
	if !npcquest.Equal(project.serverDoc.Catalog, catalog) {
		plain, err := npcquest.Encode(catalog, project.serverDoc, nil)
		if err != nil {
			return "", err
		}
		server, err = sdata.EncodeLike(server, plain)
		if err != nil {
			return "", err
		}
	}
	for _, raw := range [][]byte{client, server} {
		if len(raw) > cash.MaxBytes {
			return "", errors.New("NpcQuest file exceeds the catalog limit")
		}
		doc, err := catalogfiles.ParseNPCQuest(raw)
		if err != nil || !npcquest.Equal(doc.Catalog, catalog) {
			return "", errors.Join(errors.New("NpcQuest round-trip verification failed"), err)
		}
	}
	doc, err := catalogfiles.ParseNPCQuest(client)
	if err != nil {
		return "", err
	}
	for key, value := range texts {
		if doc.Text(key) != value {
			return "", fmt.Errorf("NpcQuest text %s changed during serialization", key.Field)
		}
	}
	if bytes.Equal(client, project.original) && bytes.Equal(server, project.Server.original) {
		return "", errors.New("there are no NPC or quest changes to save")
	}

	lock, err := lockRoot(root)
	if err != nil {
		return "", err
	}
	defer lock.Close()
	if pending, err := Pending(root); err != nil {
		return "", err
	} else if pending {
		return "", errors.New("recover the previous save before editing")
	}
	clientFile := &ServerFile{Path: project.Path, original: project.original}
	if err := clientFile.checkCurrent(); err != nil {
		return "", err
	}
	if err := project.Server.checkCurrent(); err != nil {
		return "", err
	}
	if clientFile.Path == project.Server.Path {
		return "", errors.New("select separate client and server NpcQuest files")
	}
	backup, err := catalogBackup(root, "npc-quest")
	if err != nil {
		return "", err
	}
	for name, raw := range map[string][]byte{"client-before.SData": project.original, "server-before.SData": project.Server.original,
		"client-next.SData": client, "server-next.SData": server} {
		if err := backupCatalogFile(filepath.Join(backup, name), raw); err != nil {
			return backup, err
		}
	}
	record := journal{Kind: NPCQuest, Backup: filepath.Base(backup), CashPath: project.Path, CashBefore: hash(project.original), CashAfter: hash(client),
		ServerPath: project.Server.Path, ServerBefore: hash(project.Server.original), ServerAfter: hash(server)}
	payload, err := json.Marshal(record)
	if err != nil {
		return backup, err
	}
	if err = writeAtomic(filepath.Join(root, pendingName), payload, 0600); err != nil {
		return backup, err
	}
	return finishNPCQuestFiles(root, record)
}

// Once the two verified replacements have a durable journal, recovery
// completes both writes. It rejects files externally modified after staging.
func finishNPCQuestFiles(root string, record journal) (string, error) {
	if record.Kind != NPCQuest || !filepath.IsAbs(record.CashPath) || !filepath.IsAbs(record.ServerPath) ||
		record.CashPath == record.ServerPath || record.Backup == "" || record.Backup == "." || record.Backup == ".." || record.Backup != filepath.Base(record.Backup) {
		return "", errors.New("invalid NPC and quest recovery journal")
	}
	dir := filepath.Join(root, "backups", record.Backup)
	targets := []struct {
		path, before, after, candidate string
		data                           []byte
		mode                           os.FileMode
	}{
		{path: record.CashPath, before: record.CashBefore, after: record.CashAfter, candidate: "client-next.SData"},
		{path: record.ServerPath, before: record.ServerBefore, after: record.ServerAfter, candidate: "server-next.SData"},
	}
	for i := range targets {
		t := &targets[i]
		current, err := readSData(t.path)
		if err != nil {
			return dir, err
		}
		switch hash(current) {
		case t.after:
			continue
		case t.before:
		default:
			return dir, fmt.Errorf("%s has an unexpected state; review the backups", t.path)
		}
		t.data, err = readSData(filepath.Join(dir, t.candidate))
		if err != nil || hash(t.data) != t.after {
			return dir, errors.Join(errors.New("NPC and quest recovery copy failed verification"), err)
		}
		t.mode, err = fileMode(t.path)
		if err != nil {
			return dir, err
		}
	}
	for _, t := range targets {
		if t.data != nil {
			if err := writeAtomic(t.path, t.data, t.mode); err != nil {
				return dir, err
			}
		}
		raw, err := readSData(t.path)
		if err != nil || hash(raw) != t.after {
			return dir, errors.Join(errors.New("saved NPC and quest file failed verification"), err)
		}
	}
	return dir, finish(root, dir)
}
