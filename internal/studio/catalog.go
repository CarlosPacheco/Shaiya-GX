// Package studio coordinates the two Item Mall catalogs without loading GX runtime.
package studio

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/data/cash"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	_ "modernc.org/sqlite"
)

const ProductTable = "PS_GameDefs_ProductList"

// SQLProduct holds the fields shared by the client and the server.
type SQLProduct struct {
	Code, Name string
	Cost       uint32
	Items      [cash.Slots]cash.Item
}

func Shared(p cash.Product) SQLProduct {
	return SQLProduct{p.Code, p.Name, p.Cost, p.Items}
}

func (s SQLProduct) Apply(p cash.Product) cash.Product {
	p.Code, p.Name, p.Cost, p.Items = s.Code, s.Name, s.Cost, s.Items
	return p
}

type Project struct {
	DataDir, CashPath, DBPath string
	Products                  []cash.Product
	Server                    map[string]SQLProduct
	Encrypted                 bool
	cashRaw                   []byte
	fileHash, dbHash          string
}

func FindFile(root, relative string) (string, error) {
	current := root
	for _, component := range strings.Split(filepath.ToSlash(relative), "/") {
		entries, err := os.ReadDir(current)
		if err != nil {
			return "", err
		}
		var match string
		for _, entry := range entries {
			if strings.EqualFold(entry.Name(), component) {
				if match != "" {
					return "", fmt.Errorf("ambiguous filename %q in %s", component, current)
				}
				match = entry.Name()
			}
		}
		if match == "" {
			return "", fmt.Errorf("%s: %w", filepath.Join(root, relative), os.ErrNotExist)
		}
		current = filepath.Join(current, match)
	}
	return canonicalFile(current)
}

func canonicalFile(path string) (string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%s is not a regular file", path)
	}
	return path, nil
}

func readSData(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, cash.MaxBytes+1))
	if len(raw) > cash.MaxBytes {
		return nil, fmt.Errorf("%s exceeds the %d-byte catalog limit", path, cash.MaxBytes)
	}
	return raw, err
}

func Open(ctx context.Context, dataDir, database string) (*Project, error) {
	cashPath, err := FindFile(dataDir, cash.SDataPath)
	if errors.Is(err, os.ErrNotExist) {
		cashPath, err = FindFile(dataDir, cash.LegacySDataPath)
	}
	if err != nil {
		return nil, err
	}
	database, err = canonicalFile(database)
	if err != nil {
		return nil, err
	}
	raw, err := readSData(cashPath)
	if err != nil {
		return nil, err
	}
	plain, err := sdata.Decrypt(raw)
	if err != nil {
		return nil, err
	}
	products, err := sdata.ParseCash(raw)
	if err != nil {
		return nil, err
	}
	db, err := openDB(ctx, database, "ro")
	if err != nil {
		return nil, err
	}
	defer db.Close()
	state, err := readRows(ctx, db)
	if err != nil {
		return nil, err
	}
	server := make(map[string]SQLProduct, len(state))
	for _, row := range state {
		p := row.product()
		if _, exists := server[p.Code]; exists {
			return nil, fmt.Errorf("ambiguous SQLite product code %q", p.Code)
		}
		server[p.Code] = p
	}
	return &Project{
		DataDir: dataDir, CashPath: cashPath, DBPath: database,
		Products: products, Server: server, Encrypted: !bytes.Equal(raw, plain),
		cashRaw: raw, fileHash: hash(raw), dbHash: rowsHash(state),
	}, nil
}

// Detect returns runtime candidates. The distributed seed is selected only
// through the explicit database picker.
func Detect(ctx context.Context, dataDir string) []string {
	client := filepath.Dir(filepath.Clean(dataDir))
	bundle := filepath.Dir(client)
	paths := []string{
		os.Getenv("RAPTURE_DB_PATH"),
		filepath.Join(bundle, "Server", "runtime", "rapture.sqlite"),
		filepath.Join(client, "Server", "runtime", "rapture.sqlite"),
		filepath.Join(bundle, "GX Server", "runtime", "rapture.sqlite"),
		filepath.Join(client, "runtime", "rapture.sqlite"),
	}
	var found []string
	for _, candidate := range paths {
		if candidate == "" {
			continue
		}
		path, err := canonicalFile(candidate)
		if err != nil || slices.Contains(found, path) {
			continue
		}
		db, err := openDB(ctx, path, "ro")
		if err == nil {
			_, err = readRows(ctx, db)
			db.Close()
			if err == nil {
				found = append(found, path)
			}
		}
	}
	return found
}

type Change struct {
	Code, Summary string
}

func Differences(products []cash.Product, server map[string]SQLProduct) []Change {
	changes := make([]Change, 0)
	seen := make(map[string]bool, len(products))
	for _, product := range products {
		seen[product.Code] = true
		current, exists := server[product.Code]
		if !exists {
			changes = append(changes, Change{product.Code, "Add SQLite product"})
			continue
		}
		var fields []string
		if product.Name != current.Name {
			fields = append(fields, fmt.Sprintf("Name %q → %q", current.Name, product.Name))
		}
		if product.Cost != current.Cost {
			fields = append(fields, fmt.Sprintf("Price %d → %d", current.Cost, product.Cost))
		}
		for i, item := range product.Items {
			before := current.Items[i]
			if item != before {
				fields = append(fields, fmt.Sprintf("Slot %d: %d × %d → %d × %d", i+1, before.ItemID, before.Count, item.ItemID, item.Count))
			}
		}
		if len(fields) > 0 {
			changes = append(changes, Change{product.Code, strings.Join(fields, "\n")})
		}
	}
	var removed []string
	for code := range server {
		if !seen[code] {
			removed = append(removed, code)
		}
	}
	slices.Sort(removed)
	for _, code := range removed {
		changes = append(changes, Change{code, "Remove SQLite product"})
	}
	return changes
}

func dbURI(path, mode string) string {
	path = filepath.ToSlash(path)
	u := url.URL{Scheme: "file", Path: path}
	// SQLite requires a slash in file:///C:/... URIs.
	if !strings.HasPrefix(path, "/") {
		u.Path = "/" + path
	}
	if strings.HasPrefix(path, "//") {
		// Windows UNC support depends on how SQLite was compiled. Keep the
		// server component in the path rather than using it as URI authority.
		u.Path = "//" + strings.TrimPrefix(path, "//")
	}
	query := url.Values{
		"mode":    {mode},
		"_pragma": {"busy_timeout(3000)", "foreign_keys(ON)", "trusted_schema(OFF)", "synchronous(FULL)"},
		"_txlock": {"immediate"},
	}
	u.RawQuery = query.Encode()
	return u.String()
}

func openDB(ctx context.Context, path, mode string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbURI(path, mode))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err := db.PingContext(ctx); err != nil {
		return nil, errors.Join(err, db.Close())
	}
	return db, nil
}
