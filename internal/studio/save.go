package studio

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/cash"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
)

const pendingName = "pending-save.json"

type journal struct {
	Backup, CashPath, DBPath string
	Kind                     Kind
	CashBefore, CashAfter    string
	SQLBefore, SQLAfter      string
}

func hash(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func rowsHash(rows []sqlRow) string {
	raw, _ := json.Marshal(rows) // sqlRow contains only primitive values.
	return hash(raw)
}

func Pending(root string) (bool, error) {
	_, err := os.Stat(filepath.Join(root, pendingName))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

// Save requires the client and GX Server to be offline. A published journal
// remains in place until both targets have been verified.
func Save(ctx context.Context, root string, project *Project, products []cash.Product) (string, error) {
	if project.Encrypted {
		return "", errors.New("encrypted Cash.SData is read-only")
	}
	next, err := encodeCash(project.cashRaw, products)
	if err != nil {
		return "", err
	}
	if err := validateClientItems(project, products); err != nil {
		return "", err
	}
	if len(next) > cash.MaxBytes {
		return "", errors.New("generated Cash.SData exceeds the catalog limit")
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
		path: project.CashPath, db: project.DBPath, fileHash: project.fileHash, dbHash: project.dbHash, next: next,
		verify: func(staged []byte) error {
			parsed, err := sdata.ParseCash(staged)
			if err != nil || !slices.Equal(products, parsed) {
				return errors.Join(errors.New("staged Cash.SData failed round-trip verification"), err)
			}
			return nil
		},
		apply: func(tx *sql.Tx) error {
			before, err := readRows(ctx, tx)
			if err != nil {
				return err
			}
			if err := validateItems(ctx, tx, products); err != nil {
				return err
			}
			if err := writeRows(ctx, tx, before, products); err != nil {
				return err
			}
			after, err := readRows(ctx, tx)
			if err != nil {
				return err
			}
			return verifyShared(after, products)
		},
	})
}

type changeSet struct {
	kind                       Kind
	path, db, fileHash, dbHash string
	next                       []byte
	verify                     func([]byte) error
	apply                      func(*sql.Tx) error
}

func saveLocked(ctx context.Context, root string, change changeSet) (string, error) {
	db, err := openDB(ctx, change.db, "rw")
	if err != nil {
		return "", err
	}
	defer db.Close()
	tx, err := db.BeginTx(ctx, nil) // BEGIN IMMEDIATE reserves the SQLite writer.
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	before, err := stateHash(ctx, tx, change.kind)
	if err != nil {
		return "", err
	}
	if before != change.dbHash {
		return "", errors.New("the SQLite catalog changed on disk. Reopen the installation before saving")
	}
	old, err := readSData(change.path)
	if err != nil {
		return "", err
	}
	if hash(old) != change.fileHash {
		return "", errors.New("the SData catalog changed on disk. Reopen the installation before saving")
	}

	backup, err := makeBackup(ctx, root, change, old)
	if err != nil {
		return backup, err
	}
	if err := change.apply(tx); err != nil {
		return backup, err
	}
	after, err := stateHash(ctx, tx, change.kind)
	if err != nil {
		return backup, err
	}
	mode, err := fileMode(change.path)
	if err != nil {
		return backup, err
	}
	record := journal{
		Backup: filepath.Base(backup), CashPath: change.path, DBPath: change.db, Kind: change.kind,
		CashBefore: hash(old), CashAfter: hash(change.next), SQLBefore: before, SQLAfter: after,
	}
	current, err := readSData(change.path)
	if err != nil {
		return backup, err
	}
	if hash(current) != record.CashBefore {
		return backup, errors.New("Cash.SData changed while creating the backup")
	}
	payload, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return backup, err
	}
	if err := writeAtomic(filepath.Join(root, pendingName), payload, 0600); err != nil {
		return backup, err
	}
	if record.CashAfter != record.CashBefore {
		if err := writeAtomic(change.path, change.next, mode); err != nil {
			return backup, err
		}
	}
	if err := tx.Commit(); err != nil {
		return backup, err
	}
	if err := verifyDisk(ctx, db, record, record.CashAfter, record.SQLAfter); err != nil {
		return backup, err
	}
	if err := finish(root, backup); err != nil {
		return backup, err
	}
	return backup, nil
}

func verifyShared(rows []sqlRow, products []cash.Product) error {
	state := make(map[string]SQLProduct, len(rows))
	for _, row := range rows {
		p := row.product()
		state[p.Code] = p
	}
	if len(rows) != len(products) || len(Differences(products, state)) != 0 {
		return errors.New("SQLite contents differ from the proposed Cash catalog")
	}
	return nil
}

func makeBackup(ctx context.Context, root string, change changeSet, old []byte) (string, error) {
	parent := filepath.Join(root, "backups")
	if err := os.MkdirAll(parent, 0700); err != nil {
		return "", err
	}
	if err := syncDirectory(root); err != nil {
		return "", err
	}
	dir, err := os.MkdirTemp(parent, time.Now().UTC().Format("20060102T150405Z")+"-")
	if err != nil {
		return "", err
	}
	if err := syncDirectory(parent); err != nil {
		return dir, err
	}
	oldFile := filepath.Join(dir, change.kind.filename())
	if err := writeAtomic(oldFile, old, 0600); err != nil {
		return dir, err
	}
	copied, err := readSData(oldFile)
	if err != nil || hash(copied) != hash(old) {
		return dir, errors.Join(errors.New("Cash.SData backup verification failed"), err)
	}

	nextFile := filepath.Join(dir, "next-"+change.kind.filename())
	if err := writeAtomic(nextFile, change.next, 0600); err != nil {
		return dir, err
	}
	staged, err := readSData(nextFile)
	if err != nil {
		return dir, err
	}
	if hash(staged) != hash(change.next) {
		return dir, errors.New("staged SData bytes failed verification")
	}
	if err := change.verify(staged); err != nil {
		return dir, err
	}

	source, err := openDB(ctx, change.db, "ro")
	if err != nil {
		return dir, err
	}
	defer source.Close()
	path := filepath.Join(dir, "rapture.sqlite")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err != nil {
		return dir, err
	}
	if err := file.Close(); err != nil {
		return dir, err
	}
	if err := backupDB(ctx, source, path); err != nil {
		return dir, err
	}
	snapshot, err := openDB(ctx, path, "ro")
	if err != nil {
		return dir, err
	}
	defer snapshot.Close()
	if err := integrity(ctx, snapshot); err != nil {
		return dir, err
	}
	state, err := stateHash(ctx, snapshot, change.kind)
	if err != nil || state != change.dbHash {
		return dir, errors.Join(errors.New("SQLite backup verification failed"), err)
	}
	if err := syncFile(path); err != nil {
		return dir, err
	}
	return dir, syncDirectory(dir)
}

func verifyDisk(ctx context.Context, db queryer, record journal, cashHash, sqlHash string) error {
	raw, err := readSData(record.CashPath)
	if err != nil {
		return err
	}
	state, err := stateHash(ctx, db, record.Kind)
	if err != nil {
		return err
	}
	if hash(raw) != cashHash || state != sqlHash {
		return errors.New("saved catalogs failed verification; recovery is pending")
	}
	return nil
}

// Recover uses SQLite's committed state to decide whether to finish the save
// or restore the previous client file. Unexpected states require manual review.
func Recover(ctx context.Context, root string) (string, error) {
	lock, err := lockRoot(root)
	if err != nil {
		return "", err
	}
	defer lock.Close()
	payload, err := os.ReadFile(filepath.Join(root, pendingName))
	if err != nil {
		return "", err
	}
	var record journal
	if err := json.Unmarshal(payload, &record); err != nil {
		return "", err
	}
	if (!record.Kind.valid() && record.Kind != NPCQuest) || !filepath.IsAbs(record.CashPath) || !filepath.IsAbs(record.DBPath) ||
		record.Backup == "" || record.Backup == "." || record.Backup == ".." || record.Backup != filepath.Base(record.Backup) ||
		len(record.CashBefore) != 64 || len(record.CashAfter) != 64 || len(record.SQLBefore) != 64 || len(record.SQLAfter) != 64 {
		return "", errors.New("invalid recovery journal")
	}
	backup := filepath.Join(root, "backups", record.Backup)
	db, err := openDB(ctx, record.DBPath, "rw")
	if err != nil {
		return backup, err
	}
	defer db.Close()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return backup, err
	}
	defer tx.Rollback()
	sqlState, err := stateHash(ctx, tx, record.Kind)
	if err != nil {
		return backup, err
	}
	raw, err := readSData(record.CashPath)
	if err != nil {
		return backup, err
	}
	cashState := hash(raw)
	if cashState != record.CashBefore && cashState != record.CashAfter {
		return backup, errors.New("The SData file has an unexpected state. Review the backups before changing it")
	}

	target, candidate := record.CashBefore, record.Kind.filename()
	switch {
	case sqlState == record.SQLAfter && (record.SQLBefore != record.SQLAfter || cashState == record.CashAfter):
		target, candidate = record.CashAfter, "next-"+record.Kind.filename()
	case sqlState == record.SQLBefore:
	default:
		return backup, errors.New("The SQLite catalog has an unexpected state. Review the backups before changing it")
	}
	if target != cashState {
		data, err := readSData(filepath.Join(backup, candidate))
		if err != nil || hash(data) != target {
			return backup, errors.Join(errors.New("recovery copy failed verification"), err)
		}
		mode, err := fileMode(record.CashPath)
		if err != nil {
			return backup, err
		}
		if err := writeAtomic(record.CashPath, data, mode); err != nil {
			return backup, err
		}
	}
	if err := verifyDisk(ctx, tx, record, target, sqlState); err != nil {
		return backup, err
	}
	if err := tx.Commit(); err != nil {
		return backup, err
	}
	return backup, finish(root, backup)
}

func finish(root, backup string) error {
	if err := replace(filepath.Join(root, pendingName), filepath.Join(backup, "save.json")); err != nil {
		return err
	}
	return errors.Join(syncDirectory(backup), syncDirectory(root))
}

func lockRoot(root string) (*os.File, error) {
	lock, err := os.OpenFile(filepath.Join(root, ".gxstudio.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := lockFile(lock); err != nil {
		lock.Close()
		return nil, fmt.Errorf("GX Studio save lock: %w", err)
	}
	return lock, nil
}

func fileMode(path string) (os.FileMode, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Mode().Perm(), nil
}

func syncFile(path string) error {
	file, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	return errors.Join(file.Sync(), file.Close())
}

func writeAtomic(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	file, err := os.CreateTemp(dir, ".gxstudio-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	err = file.Chmod(mode)
	if err == nil {
		_, err = file.Write(data)
	}
	if err == nil {
		err = file.Sync()
	}
	if err := errors.Join(err, file.Close()); err != nil {
		return err
	}
	if err := replace(file.Name(), path); err != nil {
		return err
	}
	return syncDirectory(dir)
}
