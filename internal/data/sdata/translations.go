package sdata

import (
	"bytes"
	"encoding/csv"
	"io"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Translations overlays presentation fields without changing SData identities or rules.
type Translations struct {
	Cash      map[string]TranslatedText
	Items     map[ItemKey]TranslatedText
	Monsters  map[uint16]string
	NPCs      map[NPCKey]TranslatedText
	Quests    map[uint16][]string
	Skills    map[SkillKey]TranslatedText
	NPCSkills map[SkillKey]TranslatedText
}

type TranslatedText struct{ Name, Description string }

func (t TranslatedText) Apply(name, description *string) {
	if t.Name != "" {
		*name = t.Name
	}
	if t.Description != "" {
		*description = t.Description
	}
}

func validTranslation(translated string) string {
	if strings.TrimSpace(translated) != "" && utf8.ValidString(translated) && !strings.ContainsAny(translated, "\x00\ufffd") {
		return translated
	}
	return ""
}

func (t Translations) Quest(id uint16, quest Quest) Quest {
	if fields := t.Quests[id]; len(fields) == 9 {
		TranslatedText{fields[0], fields[1]}.Apply(&quest.Name, &quest.Summary)
		for i := range quest.Descriptions {
			if fields[2+i] != "" {
				quest.Descriptions[i] = fields[2+i]
			}
		}
		for i := range quest.Completion {
			if fields[6+i] != "" {
				quest.Completion[i] = fields[6+i]
			}
		}
	}
	return quest
}

func LoadTranslations(read func(string) ([]byte, error), code string) Translations {
	if code == "" {
		return Translations{}
	}
	t := Translations{
		Cash: map[string]TranslatedText{}, Items: map[ItemKey]TranslatedText{},
		Monsters: map[uint16]string{}, NPCs: map[NPCKey]TranslatedText{},
		Quests: map[uint16][]string{}, Skills: map[SkillKey]TranslatedText{}, NPCSkills: map[SkillKey]TranslatedText{},
	}
	localized := func(name string) string { return name + "_" + code + ".csv" }
	load := func(file, header string, apply func([]string)) {
		raw, err := read("Languages/" + file)
		if err != nil {
			return
		}
		r := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(raw, []byte("\ufeff"))))
		columns, err := r.Read()
		if err != nil || !slices.Equal(columns, strings.Split(header, ",")) {
			return
		}
		for {
			row, err := r.Read()
			if err == io.EOF {
				return
			}
			if err == nil {
				apply(row)
			}
		}
	}
	id := func(s string, bits int) (uint16, bool) {
		v, err := strconv.ParseUint(s, 10, bits)
		return uint16(v), err == nil
	}
	load(localized("cash/cash"), "code,name,description", func(r []string) {
		t.Cash[r[0]] = TranslatedText{validTranslation(r[1]), validTranslation(r[2])}
	})
	load(localized("item/item"), "type,id,name,description", func(r []string) {
		typ, valid := id(r[0], 8)
		key, ok := id(r[1], 8)
		if valid && ok {
			t.Items[ItemKey{uint8(typ), uint8(key)}] = TranslatedText{validTranslation(r[2]), validTranslation(r[3])}
		}
	})
	load(localized("monster/monster"), "id,name", func(r []string) {
		if key, ok := id(r[0], 16); ok {
			t.Monsters[key] = validTranslation(r[1])
		}
	})
	npc := func(file, field string, apply func(*TranslatedText, string)) {
		load(localized("npc/"+file), "type,id,"+field, func(r []string) {
			typ, valid := id(r[0], 8)
			key, ok := id(r[1], 16)
			if valid && ok {
				k := NPCKey{uint8(typ), key}
				row := t.NPCs[k]
				apply(&row, validTranslation(r[2]))
				t.NPCs[k] = row
			}
		})
	}
	npc("npcnames", "name", func(row *TranslatedText, s string) { row.Name = s })
	npc("npcdescription", "description", func(row *TranslatedText, s string) { row.Description = s })
	load("npc/npcquest.csv", "language,id,name,summary,offer,journal,reminder,alternate,completion1,completion2,completion3", func(r []string) {
		if key, ok := id(r[1], 16); ok && r[0] == code {
			fields := r[2:]
			for i := range fields {
				fields[i] = validTranslation(fields[i])
			}
			t.Quests[key] = fields
		}
	})
	loadSkills := func(path string, dst map[SkillKey]TranslatedText) {
		load(localized(path), "id,level,name,description", func(r []string) {
			key, valid := id(r[0], 16)
			level, ok := id(r[1], 8)
			if valid && ok {
				dst[SkillKey{key, uint8(level)}] = TranslatedText{validTranslation(r[2]), validTranslation(r[3])}
			}
		})
	}
	loadSkills("skills/skills", t.Skills)
	loadSkills("npcskills/npcskills", t.NPCSkills)
	return t
}
