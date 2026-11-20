package studio

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/catalogfiles"
)

type ServerFile struct {
	Path     string
	original []byte
}

func OpenServerFile(path string) (*ServerFile, error) {
	path, err := canonicalFile(path)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return &ServerFile{Path: path, original: data}, nil
}

func (f *ServerFile) Text() string { return string(f.original) }

func openServerCatalog[T any](path string, parse func([]byte) (T, error)) (*ServerFile, T, error) {
	var zero T
	f, err := OpenServerFile(path)
	if err != nil {
		return nil, zero, err
	}
	value, err := parse(f.original)
	return f, value, err
}

func (f *ServerFile) checkCurrent() error {
	raw, err := os.ReadFile(f.Path)
	if err != nil {
		return err
	}
	if !bytes.Equal(raw, f.original) {
		return fmt.Errorf("%s changed on disk; reopen it before saving", filepath.Base(f.Path))
	}
	return nil
}

func catalogBackup(root, label string) (string, error) {
	parent := filepath.Join(root, "backups")
	if err := os.MkdirAll(parent, 0700); err != nil {
		return "", err
	}
	return os.MkdirTemp(parent, time.Now().UTC().Format("20060102T150405Z")+"-"+label+"-")
}

func backupCatalogFile(path string, data []byte) error {
	if err := writeAtomic(path, data, 0600); err != nil {
		return err
	}
	copied, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(copied, data) {
		return errors.Join(errors.New("catalog backup failed verification"), err)
	}
	return nil
}

func SaveServerFile(root string, f *ServerFile, next []byte) (string, error) {
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
	if err := f.checkCurrent(); err != nil {
		return "", err
	}
	mode, err := fileMode(f.Path)
	if err != nil {
		return "", err
	}
	backup, err := catalogBackup(root, "server-catalog")
	if err != nil {
		return "", err
	}
	if err := backupCatalogFile(filepath.Join(backup, filepath.Base(f.Path)), f.original); err != nil {
		return backup, err
	}
	if err := writeAtomic(f.Path, next, mode); err != nil {
		return backup, err
	}
	raw, err := os.ReadFile(f.Path)
	if err != nil || !bytes.Equal(raw, next) {
		return backup, errors.Join(errors.New("saved server file failed verification"), err, writeAtomic(f.Path, f.original, mode))
	}
	f.original = bytes.Clone(next)
	return backup, nil
}

func saveServerCatalog[T any](root string, file *ServerFile, next []byte, parse func([]byte) (T, error), matches func(T) bool) (string, error) {
	reloaded, err := parse(next)
	if err != nil {
		return "", err
	}
	if !matches(reloaded) {
		return "", errors.New("server catalog failed round-trip verification")
	}
	return SaveServerFile(root, file, next)
}

func ServerCatalogName(kind Kind) string {
	switch kind {
	case Enchant:
		return catalogfiles.Enchant
	case MobAIMessages:
		return catalogfiles.MobMessages
	case Blessing:
		return catalogfiles.KillStatus
	case WorldDrops:
		return catalogfiles.WorldDrops
	case ItemBoxes:
		return catalogfiles.ItemBoxes
	case NPCQuest:
		return catalogfiles.NPCQuest
	case MapRules:
		return catalogfiles.Map
	case TeleportRunes:
		return catalogfiles.TeleportRunes
	case Notices:
		return catalogfiles.Notices
	}
	return ""
}

func ResolveServerGameDir(explicit, dbPath, clientData string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		return filepath.Abs(explicit)
	}
	var dirs []string
	if dbPath != "" {
		base := filepath.Dir(dbPath)
		dirs = append(dirs, filepath.Join(base, "game"), filepath.Join(base, "..", "data", "game"))
	}
	if clientData != "" {
		dirs = append(dirs, filepath.Join(clientData, "..", "..", "GX Server", "data", "game"), filepath.Join(clientData, "..", "..", "Server", "data", "game"))
	}
	for _, dir := range dirs {
		if info, err := os.Stat(filepath.Join(dir, catalogfiles.Map)); err == nil && !info.IsDir() {
			return filepath.Abs(dir)
		}
	}
	return "", errors.New("choose the server data/game folder")
}
