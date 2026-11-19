package studio

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/Spelunkern/Shaiya-GX/internal/data/cash"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/textwire"
)

type Kind uint8

const (
	Cash Kind = iota
	Items
	Skills
	NPCSkills
	Enchant
	MobAIMessages
	Blessing
	WorldDrops
)

func (k Kind) DatabaseOnly() bool {
	return k == Enchant || k == MobAIMessages || k == Blessing || k == WorldDrops
}

func (k Kind) valid() bool { return k <= NPCSkills }

func (k Kind) RelativePath() string {
	switch k {
	case Items:
		return "Item/Item.SData"
	case Skills:
		return "Character/Skill.SData"
	case NPCSkills:
		return "Monster/NpcSkill.SData"
	default:
		return cash.SDataPath
	}
}

func (k Kind) filename() string { return filepath.Base(k.RelativePath()) }

func (k Kind) Table() string {
	switch k {
	case Items:
		return "PS_GameDefs_Items"
	case Skills, NPCSkills:
		return "PS_GameDefs_Skills"
	default:
		return ProductTable
	}
}

func (k Kind) Fields() []Field {
	switch k {
	case Items:
		return slices.Clone(itemFields)
	case Skills, NPCSkills:
		return slices.Clone(skillFields)
	default:
		return nil
	}
}

func (f Field) IsText() bool { return f.size == 0 }

type Record struct {
	ID, Level, Icon               uint32
	Name                          string
	DatabaseRow                   bool
	start, body, end, description int
}

func (r Record) sqlKey(kind Kind) (string, []any, bool) {
	switch kind {
	case Items:
		return "ItemID = ? AND Type = ? AND TypeID = ?", []any{r.ID, r.ID / 1000, r.ID % 1000}, true
	case Skills:
		return "SkillID = ? AND SkillLevel = ?", []any{r.ID, r.Level}, r.Level < 100
	case NPCSkills:
		return "SkillID = ? AND SkillLevel = 100", []any{r.ID}, r.Level == 1
	default:
		return "", nil, false
	}
}

type Definitions struct {
	Kind            Kind
	Path, DBPath    string
	Records         []Record
	Encrypted       bool
	original, plain []byte
	dbHash          string
}

// DefinitionEdits is indexed by the displayed record index and SData field name.
// Missing entries leave the exact original bytes and SQLite values intact.
type DefinitionEdits map[int]map[string]string

func OpenDefinitions(ctx context.Context, dataDir, database string, kind Kind) (*Definitions, error) {
	if kind == Cash || !kind.valid() {
		return nil, errors.New("unknown definition format")
	}
	path, err := FindFile(dataDir, kind.RelativePath())
	if err != nil {
		return nil, err
	}
	database, err = canonicalFile(database)
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
	records, err := definitionRecords(kind, plain)
	if err != nil {
		return nil, err
	}
	db, err := openDB(ctx, database, "ro")
	if err != nil {
		return nil, err
	}
	defer db.Close()
	dbHash, err := stateHash(ctx, db, kind)
	if err != nil {
		return nil, err
	}
	keys, err := definitionDBKeys(ctx, db, kind)
	if err != nil {
		return nil, err
	}
	for i := range records {
		r := &records[i]
		var key [2]uint32
		switch kind {
		case Items:
			key = [2]uint32{r.ID, 0}
		case NPCSkills:
			if r.Level != 1 {
				continue
			}
			key = [2]uint32{r.ID, 100}
		case Skills:
			key = [2]uint32{r.ID, r.Level}
		}
		r.DatabaseRow = keys[key] == 1
	}
	return &Definitions{Kind: kind, Path: path, DBPath: database, Records: records,
		Encrypted: !bytes.Equal(raw, plain), original: raw, plain: plain, dbHash: dbHash}, nil
}

func definitionDBKeys(ctx context.Context, db queryer, kind Kind) (map[[2]uint32]int, error) {
	query := "SELECT SkillID, SkillLevel FROM " + kind.Table()
	if kind == Items {
		query = "SELECT ItemID, 0 FROM " + kind.Table() + " WHERE ItemID = Type * 1000 + TypeID"
	} else if kind == NPCSkills {
		query += " WHERE SkillLevel = 100"
	} else {
		query += " WHERE SkillLevel < 100"
	}
	// Validate the complete mapping before presenting an editable form.
	columnRows, err := db.QueryContext(ctx, "SELECT d."+strings.Join(mappedColumns(kind), ",d.")+" FROM "+kind.Table()+" AS d LIMIT 0")
	if err != nil {
		return nil, err
	}
	if err := columnRows.Close(); err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	keys := map[[2]uint32]int{}
	for rows.Next() {
		var id, level uint32
		if err := rows.Scan(&id, &level); err != nil {
			return nil, err
		}
		keys[[2]uint32{id, level}]++
	}
	return keys, rows.Err()
}

func mappedColumns(kind Kind) []string {
	var columns []string
	for _, f := range kind.Fields() {
		if f.Column != "" {
			columns = append(columns, quote(f.Column))
		}
	}
	return columns
}

func quote(column string) string { return `"` + column + `"` }

type layoutReader struct {
	raw []byte
	pos int
	err error
}

func (r *layoutReader) take(size int) []byte {
	if r.err != nil {
		return nil
	}
	if size < 0 || size > len(r.raw)-r.pos {
		r.err = fmt.Errorf("truncated SData at byte %d", r.pos)
		return nil
	}
	p := r.raw[r.pos : r.pos+size]
	r.pos += size
	return p
}

func (r *layoutReader) word() uint32 {
	raw := r.take(4)
	if raw == nil {
		return 0
	}
	return binary.LittleEndian.Uint32(raw)
}

func (r *layoutReader) text() string { return textwire.Decode(r.take(int(r.word()))) }

func definitionRecords(kind Kind, plain []byte) ([]Record, error) {
	r := layoutReader{raw: plain}
	groups := r.word()
	if groups == 0 || groups > 65535 || (kind == Items && groups > 255) {
		return nil, errors.New("invalid EP5 definition group count")
	}
	var rows []Record
	seen := map[[2]uint32]bool{}
	for group := uint32(1); group <= groups && r.err == nil; group++ {
		count, size := uint32(9), 116
		if kind == Items {
			count, size = r.word(), 81
			if count > 255 {
				return nil, fmt.Errorf("too many Item.SData records in family %d", group)
			}
		}
		for i := uint32(0); i < count && r.err == nil; i++ {
			row := Record{start: r.pos}
			row.Name = r.text()
			row.description = r.pos
			r.text()
			row.body = r.pos
			body := r.take(size)
			if r.err != nil {
				break
			}
			row.end = r.pos
			if kind == Items {
				typ, id := uint32(body[0]), uint32(body[1])
				if typ != group || id == 0 {
					return nil, fmt.Errorf("invalid item identity %d/%d in family %d", typ, id, group)
				}
				row.ID, row.Icon = typ*1000+id, uint32(body[3])
			} else {
				row.ID, row.Level, row.Icon = group, uint32(body[0]), uint32(binary.LittleEndian.Uint16(body[1:3]))
				if row.Level == 0 {
					continue // Reserved slot. Preserve it byte-for-byte.
				}
			}
			key := [2]uint32{row.ID, row.Level}
			if seen[key] {
				return nil, fmt.Errorf("duplicate definition %d/%d", row.ID, row.Level)
			}
			seen[key] = true
			rows = append(rows, row)
		}
	}
	if r.err != nil {
		return nil, r.err
	}
	if r.pos != len(plain) {
		return nil, fmt.Errorf("unexpected %d trailing bytes in %s", len(plain)-r.pos, kind.RelativePath())
	}
	return rows, nil
}

func (d *Definitions) Value(record int, f Field) string {
	r := d.Records[record]
	if f.IsText() {
		if f.offset == 0 {
			return r.Name
		}
		return textwire.Decode(d.plain[r.description+4 : r.body])
	}
	raw := d.plain[r.body+f.offset:]
	switch f.size {
	case 1:
		return strconv.FormatUint(uint64(raw[0]), 10)
	case 2:
		word := binary.LittleEndian.Uint16(raw)
		if f.signed {
			return strconv.FormatInt(int64(int16(word)), 10)
		}
		return strconv.FormatUint(uint64(word), 10)
	default:
		return strconv.FormatUint(uint64(binary.LittleEndian.Uint32(raw)), 10)
	}
}

func (d *Definitions) SQLValues(ctx context.Context, record int) (map[string]string, error) {
	r := d.Records[record]
	if !r.DatabaseRow {
		return nil, nil
	}
	db, err := openDB(ctx, d.DBPath, "ro")
	if err != nil {
		return nil, err
	}
	defer db.Close()
	columns, where := mappedColumns(d.Kind), ""
	var args []any
	where, args, _ = r.sqlKey(d.Kind)
	values, targets := make([]any, len(columns)), make([]any, len(columns))
	for i := range targets {
		targets[i] = &values[i]
	}
	if err := db.QueryRowContext(ctx, "SELECT "+strings.Join(columns, ",")+" FROM "+d.Kind.Table()+" WHERE "+where, args...).Scan(targets...); err != nil {
		return nil, err
	}
	result := make(map[string]string, len(values))
	for i, value := range values {
		result[strings.Trim(columns[i], `"`)] = fmt.Sprint(value)
	}
	return result, nil
}

func stateHash(ctx context.Context, db queryer, kind Kind) (string, error) {
	if kind == Cash {
		rows, err := readRows(ctx, db)
		return rowsHash(rows), err
	}
	rows, err := db.QueryContext(ctx, "SELECT rowid, * FROM "+kind.Table()+" ORDER BY rowid")
	if err != nil {
		return "", err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return "", err
	}
	h := sha256.New()
	encoder := json.NewEncoder(h)
	values, targets := make([]any, len(cols)), make([]any, len(cols))
	for i := range targets {
		targets[i] = &values[i]
	}
	for rows.Next() {
		if err := rows.Scan(targets...); err != nil {
			return "", err
		}
		if err := encoder.Encode(values); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(h.Sum(nil)), rows.Err()
}

type definitionUpdate struct {
	record  Record
	columns []string
	values  []any
}

func SaveDefinitions(ctx context.Context, root string, definitions *Definitions, edits DefinitionEdits) (string, error) {
	plain, updates, err := definitions.apply(edits)
	if err != nil {
		return "", err
	}
	if bytes.Equal(plain, definitions.plain) {
		return "", errors.New("there are no definition changes to save")
	}
	raw, err := sdata.EncodeLike(definitions.original, plain)
	if err != nil {
		return "", err
	}
	if len(raw) > cash.MaxBytes {
		return "", errors.New("edited SData exceeds the catalog limit")
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
		kind: definitions.Kind, path: definitions.Path, db: definitions.DBPath, fileHash: hash(definitions.original), dbHash: definitions.dbHash, next: raw,
		verify: func(staged []byte) error {
			reopened, err := sdata.Decrypt(staged)
			if err != nil || !bytes.Equal(reopened, plain) {
				return errors.Join(errors.New("definition round-trip verification failed"), err)
			}
			_, err = definitionRecords(definitions.Kind, reopened)
			if err != nil {
				return err
			}
			if definitions.Kind == Items {
				_, err = sdata.ParseItems(staged)
			} else {
				_, err = sdata.ParseSkills(staged)
			}
			return err
		},
		apply: func(tx *sql.Tx) error { return applyDefinitions(ctx, tx, definitions.Kind, updates) },
	})
}

func (d *Definitions) apply(edits DefinitionEdits) ([]byte, []definitionUpdate, error) {
	indexes := make([]int, 0, len(edits))
	for index := range edits {
		if index < 0 || index >= len(d.Records) {
			return nil, nil, fmt.Errorf("invalid definition index %d", index)
		}
		indexes = append(indexes, index)
	}
	slices.Sort(indexes)
	plain := make([]byte, 0, len(d.plain))
	var updates []definitionUpdate
	cursor := 0
	for _, index := range indexes {
		r := d.Records[index]
		patch := edits[index]
		body := bytes.Clone(d.plain[r.body:r.end])
		name := d.plain[r.start:r.description]
		desc := d.plain[r.description:r.body]
		update := definitionUpdate{record: r}
		matched := 0
		for _, field := range d.Kind.Fields() {
			value, changed := patch[field.Source]
			if !changed {
				continue
			}
			matched++
			if value == d.Value(index, field) {
				continue
			}
			if field.Column != "" && !r.DatabaseRow {
				return nil, nil, fmt.Errorf("definition %d/%d has no unique SQLite row for %s", r.ID, r.Level, field.Label)
			}
			sqlValue, err := encodeField(field, value, body, &name, &desc)
			if err != nil {
				return nil, nil, fmt.Errorf("definition %d/%d, %s: %w", r.ID, r.Level, field.Label, err)
			}
			if field.Column != "" {
				update.columns = append(update.columns, quote(field.Column))
				update.values = append(update.values, sqlValue)
			}
		}
		if matched != len(patch) {
			return nil, nil, fmt.Errorf("unsupported field in definition %d/%d", r.ID, r.Level)
		}
		if len(update.columns) != 0 {
			updates = append(updates, update)
		}
		plain = append(plain, d.plain[cursor:r.start]...)
		plain = append(plain, name...)
		plain = append(plain, desc...)
		plain = append(plain, body...)
		cursor = r.end
	}
	return append(plain, d.plain[cursor:]...), updates, nil
}

func encodeField(field Field, value string, body []byte, name, desc *[]byte) (any, error) {
	if field.IsText() {
		if !utf8.ValidString(value) || strings.ContainsRune(value, 0) || len(value) > 65535 {
			return nil, errors.New("expected UTF-8 text without NUL (up to 65,535 bytes)")
		}
		original := name
		if field.offset == 1 {
			original = desc
		}
		// Only touched strings are encoded. Reuse the original convention
		// for NUL termination and leave every untouched string as raw bytes.
		data := []byte(value)
		if bytes.IndexByte((*original)[4:], 0) >= 0 {
			data = append(data, 0)
		}
		*original = append(binary.LittleEndian.AppendUint32(nil, uint32(len(data))), data...)
		return value, nil
	}
	value = strings.TrimSpace(value)
	var n int64
	var err error
	if field.signed {
		n, err = strconv.ParseInt(value, 10, field.size*8)
	} else {
		var unsigned uint64
		unsigned, err = strconv.ParseUint(value, 10, field.size*8)
		n = int64(unsigned)
	}
	if err != nil {
		return nil, fmt.Errorf("invalid %d-bit integer", field.size*8)
	}
	if field.sqlMax > 0 && n > field.sqlMax {
		return nil, fmt.Errorf("maximum value supported by GX Server is %d", field.sqlMax)
	}
	switch field.size {
	case 1:
		body[field.offset] = byte(n)
	case 2:
		binary.LittleEndian.PutUint16(body[field.offset:], uint16(n))
	case 4:
		binary.LittleEndian.PutUint32(body[field.offset:], uint32(n))
	}
	if field.Source == "Count" && n > 1 {
		return int64(255), nil // GX treats any stackable native count as a 255 stack.
	}
	return n, nil
}

func applyDefinitions(ctx context.Context, tx *sql.Tx, kind Kind, updates []definitionUpdate) error {
	if len(updates) == 0 {
		return nil
	}
	if err := safeTable(ctx, tx, kind.Table(), true); err != nil {
		return err
	}
	for _, update := range updates {
		where, key, valid := update.record.sqlKey(kind)
		if !valid {
			return errors.New("definition is outside the selected SQLite namespace")
		}
		var set, verify []string
		for _, column := range update.columns {
			set = append(set, column+" = ?")
			verify = append(verify, column+" COLLATE BINARY IS ?")
		}
		values := append(slices.Clone(update.values), key...)
		result, err := tx.ExecContext(ctx, "UPDATE "+kind.Table()+" SET "+strings.Join(set, ",")+" WHERE "+where, values...)
		if err != nil {
			return err
		}
		affected, err := result.RowsAffected()
		if err != nil || affected != 1 {
			return errors.Join(fmt.Errorf("expected one updated SQLite row for definition %d/%d", update.record.ID, update.record.Level), err)
		}
		var count int
		err = tx.QueryRowContext(ctx, "SELECT count(*) FROM "+kind.Table()+" WHERE "+strings.Join(verify, " AND ")+" AND "+where, values...).Scan(&count)
		if err != nil || count != 1 {
			return errors.Join(fmt.Errorf("could not verify SQLite values for definition %d/%d", update.record.ID, update.record.Level), err)
		}
	}
	return nil
}

func (d *Definitions) ValidateEdit(index int, edit map[string]string) error {
	_, _, err := d.apply(DefinitionEdits{index: edit})
	return err
}

// SQLValue reports the stored representation shown in the save review.
func (f Field) SQLValue(value string) (string, error) {
	body := make([]byte, 116)
	name, description := make([]byte, 4), make([]byte, 4)
	stored, err := encodeField(f, value, body, &name, &description)
	if err != nil {
		return "", err
	}
	return fmt.Sprint(stored), nil
}
