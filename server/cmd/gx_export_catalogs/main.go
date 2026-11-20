// gx_export_catalogs exports the retired SQLite definition tables without
// modifying the source database. GX World reads the resulting files at boot.
package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"maps"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"unicode"

	"github.com/Spelunkern/Shaiya-GX/internal/blessingrules"
	"github.com/Spelunkern/Shaiya-GX/internal/catalogfiles"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/enchantrules"
	"github.com/Spelunkern/Shaiya-GX/internal/itemboxes"
	"github.com/Spelunkern/Shaiya-GX/internal/maprules"
	"github.com/Spelunkern/Shaiya-GX/internal/mobaimessages"
	"github.com/Spelunkern/Shaiya-GX/internal/npcquest"
	"github.com/Spelunkern/Shaiya-GX/internal/teleportrunes"
	"github.com/Spelunkern/Shaiya-GX/internal/worlddrops"
	_ "modernc.org/sqlite"
)

func main() {
	db := flag.String("db", "", "Legacy SQLite database or verified backup from before migration 26")
	out := flag.String("out", "", "New output directory")
	npc := flag.String("npc-template", "", "Matching NpcQuest.SData for original texts and encoding")
	mapFile := flag.String("map-template", "", "Optional Map.ini whose additional keys should be retained")
	mob := flag.String("mob-translations", "", "UTF-8 MobAIMessages.ini used to translate Korean rows by index")
	flag.Parse()
	if *db == "" || *out == "" || *npc == "" || *mob == "" {
		flag.Usage()
		os.Exit(2)
	}
	files, err := export(context.Background(), *db, *npc, *mapFile, *mob)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.Mkdir(*out, 0755); err != nil {
		log.Fatal(err)
	}
	for _, name := range slices.Sorted(maps.Keys(files)) {
		if err := os.WriteFile(filepath.Join(*out, name), files[name], 0644); err != nil {
			log.Fatal(err)
		}
		fmt.Println(name)
	}
}

func encode[T any](value T, readErr error, format func(T) []byte) ([]byte, error) {
	if readErr != nil {
		return nil, readErr
	}
	return format(value), nil
}

func export(ctx context.Context, dbPath, npcFile, mapFile, mobFile string) (map[string][]byte, error) {
	abs, err := filepath.Abs(dbPath)
	if err != nil {
		return nil, err
	}
	u := url.URL{Scheme: "file", Path: filepath.ToSlash(abs), RawQuery: "mode=ro"}
	db, err := sql.Open("sqlite", u.String())
	if err != nil {
		return nil, err
	}
	defer db.Close()
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var legacy bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM sqlite_schema WHERE name='PS_GameDefs_EnchantAttack' AND type='table')`).Scan(&legacy); err != nil {
		return nil, err
	}
	if !legacy {
		return nil, errors.New("legacy catalog tables are absent; use a verified SQLite backup from before migration 26")
	}
	files := map[string][]byte{}
	ench, e := enchantrules.Read(ctx, tx)
	files[catalogfiles.Enchant], err = encode(ench, e, catalogfiles.EncodeEnchant)
	if err != nil {
		return nil, err
	}
	tiers, e := blessingrules.Read(ctx, tx)
	files[catalogfiles.KillStatus], err = encode(tiers, e, catalogfiles.EncodeKillStatus)
	if err != nil {
		return nil, err
	}
	drops, e := worlddrops.Read(ctx, tx)
	files[catalogfiles.WorldDrops], err = encode(drops, e, catalogfiles.EncodeWorldDrops)
	if err != nil {
		return nil, err
	}
	boxes, e := itemboxes.Read(ctx, tx)
	files[catalogfiles.ItemBoxes], err = encode(boxes, e, catalogfiles.EncodeItemBoxes)
	if err != nil {
		return nil, err
	}
	runes, e := teleportrunes.Read(ctx, tx)
	files[catalogfiles.TeleportRunes], err = encode(runes, e, catalogfiles.EncodeTeleportRunes)
	if err != nil {
		return nil, err
	}
	c, err := maprules.Read(ctx, tx)
	if err != nil {
		return nil, err
	}
	var mapOriginal []byte
	if mapFile != "" {
		mapOriginal, err = os.ReadFile(mapFile)
		if err != nil {
			return nil, err
		}
	}
	files[catalogfiles.Map], err = catalogfiles.EncodeMap(c, mapOriginal)
	if err != nil {
		return nil, err
	}
	messages, err := mobaimessages.Read(ctx, tx)
	if err != nil {
		return nil, err
	}
	translations, err := catalogfiles.Read(mobFile, catalogfiles.ParseMobMessages)
	if err != nil {
		return nil, err
	}
	english := map[uint16]string{}
	for _, m := range translations {
		english[m.Index] = m.Text
	}
	for i, m := range messages {
		if slices.ContainsFunc([]rune(m.Text), func(r rune) bool { return unicode.Is(unicode.Hangul, r) }) {
			text, ok := english[m.Index]
			if !ok {
				return nil, fmt.Errorf("Korean message %d has no translation", m.Index)
			}
			messages[i].Text = text
		}
	}
	files[catalogfiles.MobMessages] = catalogfiles.EncodeMobMessages(messages)
	if _, err := catalogfiles.ParseMobMessages(files[catalogfiles.MobMessages]); err != nil {
		return nil, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT "Index",Message,Frequency,COALESCE(FixedHour1,''),COALESCE(FixedHour2,''),COALESCE(FixedHour3,''),COALESCE(FixedHour4,''),COALESCE(FixedHour5,'') FROM PS_GameDefs_ServerNotices ORDER BY "Index"`)
	if err != nil {
		return nil, err
	}
	var notices []catalogfiles.Notice
	for rows.Next() {
		var n catalogfiles.Notice
		if err = rows.Scan(&n.Index, &n.Message, &n.Frequency, &n.Fixed[0], &n.Fixed[1], &n.Fixed[2], &n.Fixed[3], &n.Fixed[4]); err != nil {
			rows.Close()
			return nil, err
		}
		notices = append(notices, n)
	}
	if err = errors.Join(rows.Err(), rows.Close()); err != nil {
		return nil, err
	}
	files[catalogfiles.Notices] = catalogfiles.EncodeNotices(notices)
	original, err := os.ReadFile(npcFile)
	if err != nil {
		return nil, err
	}
	doc, err := catalogfiles.ParseNPCQuest(original)
	if err != nil {
		return nil, err
	}
	npcs, err := npcquest.Read(ctx, tx)
	if err != nil {
		return nil, err
	}
	if npcquest.Equal(doc.Catalog, npcs) {
		files[catalogfiles.NPCQuest] = original
	} else {
		raw, err := npcquest.Encode(npcs, doc, nil)
		if err != nil {
			return nil, err
		}
		files[catalogfiles.NPCQuest], err = sdata.EncodeLike(original, raw)
		if err != nil {
			return nil, err
		}
	}
	return files, nil
}
