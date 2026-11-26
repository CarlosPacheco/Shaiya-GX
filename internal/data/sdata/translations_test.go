package sdata

import (
	"io/fs"
	"testing"
)

func TestPartialSDataTranslations(t *testing.T) {
	read := func(path string) ([]byte, error) {
		files := map[string]string{
			"Languages/monster/monster_es.csv":    "\"id,name\"\n1\n",
			"Languages/item/item_es.csv":          "type,id,name,description\n1,1,Espada,\n",
			"Languages/npc/npcnames_es.csv":       "type,id,name\n1,1,Erina\n",
			"Languages/npc/npcdescription_es.csv": "type,id,description\n1,1,\"Hola, viajero\"\n",
			"Languages/npc/npcquest.csv":          "language,id,name,summary,offer,journal,reminder,alternate,completion1,completion2,completion3\nfr,1,Français,,,,,,,,\nes,1,Primera misión,,\"Línea 1\nLínea 2\",,,,,,\n",
		}
		if file, ok := files[path]; ok {
			return []byte(file), nil
		}
		return nil, fs.ErrNotExist
	}
	tr := LoadTranslations(read, "es")
	item := Item{Name: "Sword", Description: "English description", Model: 10}
	tr.Items[ItemKey{1, 1}].Apply(&item.Name, &item.Description)
	if item.Name != "Espada" || item.Description != "English description" || item.Model != 10 {
		t.Fatalf("item did not preserve English description and model: %+v", item)
	}
	npc := tr.NPCs[NPCKey{1, 1}]
	if npc.Name != "Erina" || npc.Description != "Hola, viajero" || len(tr.Monsters) != 0 {
		t.Fatalf("NPC fields or missing monster file were not handled: %+v", npc)
	}
	q := tr.Quest(1, Quest{Summary: "English summary", PVPKills: 5, Completion: [3]string{"Done"}})
	if q.Name != "Primera misión" || q.Summary != "English summary" || q.Descriptions[0] != "Línea 1\nLínea 2" || q.Completion[0] != "Done" || q.PVPKills != 5 {
		t.Fatalf("quest overlay is incorrect: %+v", q)
	}
	LoadTranslations(func(string) ([]byte, error) {
		t.Fatal("English tried to read a translation")
		return nil, nil
	}, "")
}
