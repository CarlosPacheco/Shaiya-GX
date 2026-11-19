package studio

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Spelunkern/Shaiya-GX/internal/data/cash"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/npcquest"
)

type NPCQuestCatalog struct {
	Path, DBPath string
	Doc          *npcquest.Document
	original     []byte
	dbHash       string
}

func OpenNPCQuest(ctx context.Context, dataDir, dbPath string) (*NPCQuestCatalog, error) {
	path, err := FindFile(dataDir, NPCQuest.RelativePath())
	if err != nil {
		return nil, err
	}
	dbPath, err = canonicalFile(dbPath)
	if err != nil {
		return nil, err
	}
	raw, err := readSData(path)
	if err != nil {
		return nil, err
	}
	plain, err := sdata.Decrypt(raw)
	if err != nil {
		return nil, err
	}
	doc, err := npcquest.Parse(plain)
	if err != nil {
		return nil, err
	}
	db, err := openDB(ctx, dbPath, "ro")
	if err != nil {
		return nil, err
	}
	defer db.Close()
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	server, err := npcquest.Read(ctx, tx)
	if err != nil {
		return nil, err
	}
	if !npcquest.Equal(server, doc.Catalog) {
		return nil, errors.New("client NpcQuest and the SQLite catalog differ; open a matching installation")
	}
	encoded, err := json.Marshal(server)
	if err != nil {
		return nil, err
	}
	return &NPCQuestCatalog{Path: path, DBPath: dbPath, Doc: doc, original: raw, dbHash: hash(encoded)}, nil
}

func SaveNPCQuest(ctx context.Context, root string, project *NPCQuestCatalog, catalog npcquest.Catalog, texts map[npcquest.TextKey]string) (string, error) {
	plain, err := npcquest.Encode(catalog, project.Doc, texts)
	if err != nil {
		return "", err
	}
	if len(plain) > cash.MaxBytes {
		return "", errors.New("NpcQuest exceeds the catalog limit")
	}
	next, err := sdata.EncodeLike(project.original, plain)
	if err != nil {
		return "", err
	}
	if bytes.Equal(next, project.original) {
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
	return saveLocked(ctx, root, changeSet{
		kind: NPCQuest, path: project.Path, db: project.DBPath, fileHash: hash(project.original), dbHash: project.dbHash, next: next,
		verify: func(staged []byte) error {
			data, err := sdata.Decrypt(staged)
			if err != nil {
				return err
			}
			if !bytes.Equal(data, plain) {
				return errors.New("NpcQuest round-trip verification failed")
			}
			doc, err := npcquest.Parse(data)
			if err != nil {
				return err
			}
			if !npcquest.Equal(doc.Catalog, catalog) {
				return errors.New("NpcQuest shared data changed during serialization")
			}
			for key, value := range texts {
				if doc.Text(key) != value {
					return fmt.Errorf("NpcQuest text %s changed during serialization", key.Field)
				}
			}
			return nil
		},
		apply: func(tx *sql.Tx) error {
			if !npcquest.Equal(project.Doc.Catalog, catalog) {
				for i, table := range npcquest.Tables {
					var allowed []string
					if i == npcquest.NPCs {
						allowed = []string{npcquest.Tables[npcquest.Products].Name, npcquest.Tables[npcquest.Gates].Name, npcquest.Tables[npcquest.NPCLinks].Name}
					}
					if err := safeTable(ctx, tx, table.Name, false, allowed...); err != nil {
						return err
					}
				}
				if err := npcquest.Update(ctx, tx, project.Doc.Catalog, catalog); err != nil {
					return err
				}
			}
			current, err := npcquest.Read(ctx, tx)
			if err != nil {
				return err
			}
			if !npcquest.Equal(current, catalog) {
				return errors.New("saved SQLite NPC and quest data differs from the client")
			}
			return nil
		},
	})
}
