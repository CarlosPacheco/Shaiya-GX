package npcquest

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/korean"
)

const Groups = 13
const NPCWelcome = "NpcWelcome"

var QuestTexts = [...]string{"MsgSummary", "MsgComplete", "MsgResponse1", "MsgResponse2", "MsgTake", "MsgWelcome", "MsgIncomplete", "MsgResponse4"}

type TextKey struct {
	Table, Row int
	Field      string
}

type nativeText struct {
	raw   []byte
	value string
}

type Document struct {
	Catalog Catalog
	texts   map[TextKey]nativeText
}

func (doc *Document) Text(key TextKey) string { return doc.texts[key].value }

type reader struct {
	data   []byte
	offset int
	err    error
}

func (r *reader) take(n int) []byte {
	if r.err != nil {
		return nil
	}
	if n < 0 || n > len(r.data)-r.offset {
		r.err = fmt.Errorf("truncated NpcQuest at byte %d", r.offset)
		return nil
	}
	start := r.offset
	r.offset += n
	return r.data[start:r.offset]
}
func (r *reader) number(t ValueType) any {
	switch t {
	case U8:
		if p := r.take(1); p != nil {
			return int64(p[0])
		}
	case U16, I16:
		if p := r.take(2); p != nil {
			n := binary.LittleEndian.Uint16(p)
			if t == I16 {
				return int64(int16(n))
			}
			return int64(n)
		}
	case U32, I32, Float:
		if p := r.take(4); p != nil {
			n := binary.LittleEndian.Uint32(p)
			switch t {
			case I32:
				return int64(int32(n))
			case Float:
				return float64(math.Float32frombits(n))
			default:
				return int64(n)
			}
		}
	}
	return int64(0)
}
func (r *reader) count(limit int) int {
	n := r.number(U32).(int64)
	if n > int64(limit) {
		r.err = fmt.Errorf("invalid NpcQuest count %d at byte %d", n, r.offset)
		return 0
	}
	return int(n)
}

func decodeText(raw []byte) string {
	if i := bytes.IndexByte(raw, 0); i >= 0 {
		raw = raw[:i]
	}
	if utf8.Valid(raw) {
		return string(raw)
	}
	for i := 0; i+1 < len(raw); i++ {
		if raw[i] >= 0x80 && raw[i+1] >= 0x80 {
			if b, err := korean.EUCKR.NewDecoder().Bytes(raw); err == nil && !strings.ContainsRune(string(b), '\uFFFD') {
				return string(b)
			}
			break
		}
	}
	b, _ := charmap.Windows1252.NewDecoder().Bytes(raw)
	return string(b)
}

func (r *reader) text(d *Document, key TextKey) string {
	n := r.count(len(r.data))
	raw := r.take(n)
	value := decodeText(raw)
	d.texts[key] = nativeText{raw, value}
	return value
}

func Parse(plain []byte) (*Document, error) {
	d := &Document{texts: map[TextKey]nativeText{}}
	r := reader{data: plain}
	for group := int64(1); group <= Groups && r.err == nil; group++ {
		for row, count := int64(1), int64(r.count(65535)); row <= count && r.err == nil; row++ {
			index := len(d.Catalog[NPCs])
			v := Row{group, row, r.number(U8), r.number(U16)}
			var merchant any = int64(0)
			if group == 1 {
				merchant = r.number(U8)
			}
			for range 4 {
				v = append(v, r.number(I32))
			}
			v = append(v, r.text(d, TextKey{NPCs, index, "NpcName"}), merchant)
			d.Catalog[NPCs] = append(d.Catalog[NPCs], v)
			r.text(d, TextKey{NPCs, index, NPCWelcome})
			if group == 1 {
				for slot, n := int64(1), int64(r.count(65536)); slot <= n && r.err == nil; slot++ {
					d.Catalog[Products] = append(d.Catalog[Products], Row{group, row, slot, r.number(U8), r.number(U8)})
				}
			}
			if group == 2 {
				for slot := int64(1); slot <= 3; slot++ {
					g := Row{group, row, slot, r.number(U16), r.number(Float), r.number(Float), r.number(Float)}
					r.text(d, TextKey{Gates, len(d.Catalog[Gates]), "MapName"})
					d.Catalog[Gates] = append(d.Catalog[Gates], append(g, r.number(U32)))
				}
			}
			readLinks(&r, &d.Catalog[NPCLinks], group, row)
		}
	}
	for typ := int64(0); typ < 256 && r.err == nil; typ++ {
		for id := int64(0); id < 256 && r.err == nil; id++ {
			readLinks(&r, &d.Catalog[ItemLinks], typ, id)
		}
	}
	for row, n := int64(1), int64(r.count(65535)); row <= n && r.err == nil; row++ {
		index := int(row - 1)
		q := Row{row, r.number(U16), r.text(d, TextKey{Quests, index, "QuestName"})}
		r.text(d, TextKey{Quests, index, QuestTexts[0]})
		for _, field := range Tables[Quests].Fields[3:] {
			q = append(q, r.number(field.Type))
		}
		for _, name := range QuestTexts[1:] {
			r.text(d, TextKey{Quests, index, name})
		}
		d.Catalog[Quests] = append(d.Catalog[Quests], q)
	}
	if r.err != nil {
		return nil, r.err
	}
	if r.offset != len(plain) {
		return nil, fmt.Errorf("NpcQuest has %d trailing bytes", len(plain)-r.offset)
	}
	return d, nil
}

func readLinks(r *reader, rows *[]Row, a, b int64) {
	for direction := int64(0); direction <= 1; direction++ {
		for slot, n := int64(1), int64(r.count(65536)); slot <= n && r.err == nil; slot++ {
			*rows = append(*rows, Row{a, b, direction, slot, r.number(U16)})
		}
	}
}

func ParseValue(t ValueType, value string) (any, error) {
	if t == Text {
		if !utf8.ValidString(value) || strings.ContainsRune(value, 0) || len(value) > 65535 {
			return nil, errors.New("invalid text")
		}
		return value, nil
	}
	value = strings.TrimSpace(value)
	switch t {
	case Float:
		n, err := strconv.ParseFloat(value, 32)
		if err == nil && (math.IsNaN(n) || math.IsInf(n, 0)) {
			err = errors.New("expected finite coordinate")
		}
		return n, err
	case I16:
		return strconv.ParseInt(value, 10, 16)
	case I32:
		return strconv.ParseInt(value, 10, 32)
	case U8, U16, U32:
		bits := 8 << t
		n, err := strconv.ParseUint(value, 10, int(bits))
		return int64(n), err
	}
	return nil, errors.New("unknown native field type")
}

func FormatValue(v any) string {
	if n, ok := v.(float64); ok {
		return strconv.FormatFloat(n, 'g', -1, 32)
	}
	return fmt.Sprint(v)
}

func Clone(cat Catalog) Catalog {
	for table, rows := range cat {
		cat[table] = slices.Clone(rows)
		for i, row := range rows {
			cat[table][i] = slices.Clone(row)
		}
	}
	return cat
}
