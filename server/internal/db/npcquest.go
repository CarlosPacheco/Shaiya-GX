package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/npcquest"
	"github.com/Spelunkern/Shaiya-GX/server/internal/config"
)

func initializeNPCQuest(ctx context.Context, tx *sql.Tx) error {
	raw, err := os.ReadFile(filepath.Join(config.DataDir(), "game", "NpcQuest.SData"))
	if errors.Is(err, os.ErrNotExist) {
		_, err = tx.ExecContext(ctx, npcquest.SeedSQL)
		return err
	}
	if err != nil {
		return err
	}
	plain, err := sdata.Decrypt(raw)
	if err != nil {
		return err
	}
	doc, err := npcquest.Parse(plain)
	if err != nil {
		return fmt.Errorf("import legacy NpcQuest: %w", err)
	}
	if err = npcquest.Insert(ctx, tx, doc.Catalog); err != nil {
		return err
	}
	copied, err := npcquest.Read(ctx, tx)
	if err != nil {
		return err
	}
	if !npcquest.Equal(doc.Catalog, copied) {
		return errors.New("imported NpcQuest does not match the legacy catalog")
	}
	return nil
}
