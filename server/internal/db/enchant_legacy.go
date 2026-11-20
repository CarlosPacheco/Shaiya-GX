package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Spelunkern/Shaiya-GX/internal/catalogfiles"
	"github.com/Spelunkern/Shaiya-GX/internal/enchantrules"
	"github.com/Spelunkern/Shaiya-GX/server/internal/config"
)

// ReadLegacyEnchant is used only to initialize a pre-migration installation.
func ReadLegacyEnchant(path string) (enchantrules.Catalog, error) {
	return catalogfiles.Read(path, catalogfiles.ParseEnchant)
}

// initializeEnchant11 runs inside migration 11 and never imports an INI again
// after that migration is recorded, including when both caps are later zero.
func initializeEnchant11(ctx context.Context, tx *sql.Tx) error {
	rules, err := enchantrules.Read(ctx, tx)
	if err != nil || len(rules.Attack) != 0 || len(rules.Defense) != 0 {
		return err
	}
	rules, err = ReadLegacyEnchant(filepath.Join(config.DataDir(), "game", "ItemEnchant.ini"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("import legacy enchant rules: %w", err)
	}
	return enchantrules.Replace(ctx, tx, rules)
}
