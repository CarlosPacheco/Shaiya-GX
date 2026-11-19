package cash

import "github.com/Spelunkern/Shaiya-GX/internal/data/archive"

const (
	LegacySDataPath = "Cash.SData"
	SDataPath       = "Item/" + LegacySDataPath
)

// ReadSData reads the current catalog, or the root catalog in older packs.
func ReadSData(a *archive.Archive) ([]byte, error) {
	if _, ok := a.Lookup(SDataPath); ok {
		return a.ReadFile(SDataPath)
	}
	return a.ReadFile(LegacySDataPath)
}
