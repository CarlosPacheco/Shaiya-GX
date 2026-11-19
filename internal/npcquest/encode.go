package npcquest

import (
	"encoding/binary"
	"fmt"
	"math"
	"slices"
)

type writer struct {
	plain []byte
	doc   *Document
	edits map[TextKey]string
}

func (w *writer) number(t ValueType, value any) {
	if t == Float {
		w.plain = binary.LittleEndian.AppendUint32(w.plain, math.Float32bits(float32(value.(float64))))
		return
	}
	v := value.(int64)
	switch t {
	case U8:
		w.plain = append(w.plain, byte(v))
	case U16, I16:
		w.plain = binary.LittleEndian.AppendUint16(w.plain, uint16(v))
	default:
		w.plain = binary.LittleEndian.AppendUint32(w.plain, uint32(v))
	}
}
func (w *writer) count(n int) { w.number(U32, int64(n)) }
func (w *writer) text(key TextKey, value string) {
	var old nativeText
	if w.doc != nil {
		old = w.doc.texts[key]
	}
	if edit, ok := w.edits[key]; ok {
		value = edit
	}
	raw := old.raw
	if old.value != value {
		raw = []byte(value)
		if len(old.raw) > 0 && old.raw[len(old.raw)-1] == 0 {
			raw = append(raw, 0)
		}
	}
	w.count(len(raw))
	w.plain = append(w.plain, raw...)
}
func (w *writer) nativeFields(table, start int, row Row) {
	for j, f := range Tables[table].Fields[start:] {
		w.number(f.Type, row[start+j])
	}
}

type stream struct {
	rows []Row
	pos  int
}

func (s *stream) take(keys ...int64) []Row {
	start := s.pos
	for s.pos < len(s.rows) {
		row := s.rows[s.pos]
		match := true
		for i, key := range keys {
			if row[i] != key {
				match = false
				break
			}
		}
		if !match {
			break
		}
		s.pos++
	}
	return s.rows[start:s.pos]
}

func (w *writer) links(s *stream, a, b int64) {
	for direction := int64(0); direction <= 1; direction++ {
		rows := s.take(a, b, direction)
		w.count(len(rows))
		for _, row := range rows {
			w.number(U16, row[4])
		}
	}
}

// Encode reconstructs the native catalog. Document supplies the client-only
// strings and original string encodings. A nil Document omits client text.
func Encode(cat Catalog, doc *Document, edits map[TextKey]string) ([]byte, error) {
	for table, rows := range cat {
		for i, row := range rows {
			if len(row) != len(Tables[table].Fields) {
				return nil, fmt.Errorf("invalid %s row %d width", Tables[table].Name, i+1)
			}
			for j, field := range Tables[table].Fields {
				switch field.Type {
				case Text:
					if _, ok := row[j].(string); !ok {
						return nil, fmt.Errorf("invalid %s", field.Name)
					}
				case Float:
					n, ok := row[j].(float64)
					if !ok || math.IsNaN(n) || math.IsInf(n, 0) || float64(float32(n)) != n {
						return nil, fmt.Errorf("invalid %s coordinate", field.Name)
					}
				default:
					n, ok := row[j].(int64)
					if !ok {
						return nil, fmt.Errorf("invalid %s", field.Name)
					}
					switch field.Type {
					case U8:
						ok = n >= 0 && n <= math.MaxUint8
					case U16:
						ok = n >= 0 && n <= math.MaxUint16
					case U32:
						ok = n >= 0 && n <= math.MaxUint32
					case I16:
						ok = n >= math.MinInt16 && n <= math.MaxInt16
					case I32:
						ok = n >= math.MinInt32 && n <= math.MaxInt32
					}
					if !ok {
						return nil, fmt.Errorf("out-of-range %s", field.Name)
					}
				}
			}
		}
	}
	for key, value := range edits {
		if doc == nil {
			return nil, fmt.Errorf("missing source text for %s", key.Field)
		}
		if _, ok := doc.texts[key]; !ok {
			return nil, fmt.Errorf("unknown source text %s", key.Field)
		}
		if _, err := ParseValue(Text, value); err != nil {
			return nil, err
		}
	}
	w := writer{doc: doc, edits: edits}
	var streams [len(Tables)]stream
	for t, rows := range cat {
		streams[t].rows = rows
	}
	for g := int64(1); g <= Groups; g++ {
		npcs := streams[NPCs].take(g)
		w.count(len(npcs))
		for i, n := range npcs {
			row := int64(i + 1)
			index := streams[NPCs].pos - len(npcs) + i
			w.number(U8, n[2])
			w.number(U16, n[3])
			if g == 1 {
				w.number(U8, n[9])
			}
			for _, v := range n[4:8] {
				w.number(I32, v)
			}
			w.text(TextKey{NPCs, index, "NpcName"}, n[8].(string))
			key := TextKey{NPCs, index, NPCWelcome}
			w.text(key, w.oldText(key))
			if g == 1 {
				products := streams[Products].take(g, row)
				w.count(len(products))
				for _, product := range products {
					w.nativeFields(Products, 3, product)
				}
			}
			if g == 2 {
				gates := streams[Gates].take(g, row)
				if len(gates) != 3 {
					return nil, fmt.Errorf("gatekeeper row %d must have three native slots", row)
				}
				for j, gate := range gates {
					w.number(U16, gate[3])
					for _, v := range gate[4:7] {
						w.number(Float, v)
					}
					key := TextKey{Gates, streams[Gates].pos - len(gates) + j, "MapName"}
					w.text(key, w.oldText(key))
					w.number(U32, gate[7])
				}
			}
			w.links(&streams[NPCLinks], g, row)
		}
	}
	for typ := int64(0); typ < 256; typ++ {
		for id := int64(0); id < 256; id++ {
			w.links(&streams[ItemLinks], typ, id)
		}
	}
	w.count(len(cat[Quests]))
	for i, q := range cat[Quests] {
		w.number(U16, q[1])
		w.text(TextKey{Quests, i, "QuestName"}, q[2].(string))
		key := TextKey{Quests, i, QuestTexts[0]}
		w.text(key, w.oldText(key))
		w.nativeFields(Quests, 3, q)
		for _, field := range QuestTexts[1:] {
			key := TextKey{Quests, i, field}
			w.text(key, w.oldText(key))
		}
	}
	streams[Quests].pos = len(cat[Quests])
	for i, s := range streams {
		if s.pos != len(s.rows) {
			return nil, fmt.Errorf("%s has orphan or unordered rows", Tables[i].Name)
		}
	}
	roundtrip, err := Parse(w.plain)
	if err != nil {
		return nil, err
	}
	if !Equal(cat, roundtrip.Catalog) {
		return nil, fmt.Errorf("NpcQuest round trip changed a native field, row or slot")
	}
	return w.plain, nil
}

func (w *writer) oldText(k TextKey) string {
	if w.doc == nil {
		return ""
	}
	return w.doc.Text(k)
}

func Equal(a, b Catalog) bool {
	for i := range a {
		if !slices.EqualFunc(a[i], b[i], slices.Equal[Row, any]) {
			return false
		}
	}
	return true
}
