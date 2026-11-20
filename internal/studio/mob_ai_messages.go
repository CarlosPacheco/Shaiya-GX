package studio

import (
	"context"
	"database/sql"
	"errors"
	"github.com/Spelunkern/Shaiya-GX/internal/catalogfiles"
	"path/filepath"
	"slices"
	"unicode/utf8"

	"github.com/Spelunkern/Shaiya-GX/internal/mobaimessages"
)

type MobAIMessageCatalog struct {
	DBPath   string
	Messages []mobaimessages.Message
	File     *ServerFile
}

func OpenMobAIMessages(ctx context.Context, path string) (*MobAIMessageCatalog, error) {
	if filepath.Base(path) == catalogfiles.MobMessages {
		file, value, err := openServerCatalog(path, catalogfiles.ParseMobMessages)
		if err != nil {
			return nil, err
		}
		return &MobAIMessageCatalog{File: file, Messages: value}, nil
	}
	path, err := canonicalFile(path)
	if err != nil {
		return nil, err
	}
	db, err := openDB(ctx, path, "ro")
	if err != nil {
		return nil, err
	}
	defer db.Close()
	messages, err := mobaimessages.Read(ctx, db)
	if err != nil {
		return nil, err
	}
	return &MobAIMessageCatalog{DBPath: path, Messages: messages}, nil
}

func SaveMobAIMessages(ctx context.Context, root string, catalog *MobAIMessageCatalog, messages []mobaimessages.Message) (string, error) {
	if catalog.File != nil {
		next := catalogfiles.EncodeMobMessages(messages)
		return saveServerCatalog(root, catalog.File, next, catalogfiles.ParseMobMessages, func(v []mobaimessages.Message) bool { return slices.Equal(v, messages) })
	}
	if len(messages) != len(catalog.Messages) {
		return "", errors.New("keep the message indexes used by the AI scripts")
	}
	for i, message := range messages {
		if message.Index != catalog.Messages[i].Index || !utf8.ValidString(message.Text) {
			return "", errors.New("invalid AI message index or UTF-8 text")
		}
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
	db, err := openDB(ctx, catalog.DBPath, "rw")
	if err != nil {
		return "", err
	}
	defer db.Close()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	current, err := mobaimessages.Read(ctx, tx)
	if err != nil {
		return "", err
	}
	if !slices.Equal(current, catalog.Messages) {
		return "", errors.New("AI messages changed on disk; reopen the catalog before saving")
	}
	if err := safeTable(ctx, tx, mobaimessages.Table, false); err != nil {
		return "", err
	}
	backup, err := backupDatabaseCatalog(ctx, root, catalog.DBPath, "mob-ai-messages", func(snapshot *sql.DB) error {
		rows, err := mobaimessages.Read(ctx, snapshot)
		if err != nil || !slices.Equal(rows, current) {
			return errors.Join(errors.New("AI message backup failed verification"), err)
		}
		return nil
	})
	if err != nil {
		return backup, err
	}
	for i, message := range messages {
		if message.Text == current[i].Text {
			continue
		}
		if _, err := tx.ExecContext(ctx, `UPDATE `+mobaimessages.Table+` SET "Text"=? WHERE "Index"=?`, message.Text, message.Index); err != nil {
			return backup, err
		}
	}
	written, err := mobaimessages.Read(ctx, tx)
	if err != nil || !slices.Equal(written, messages) {
		return backup, errors.Join(errors.New("saved AI messages failed verification"), err)
	}
	return backup, tx.Commit()
}
