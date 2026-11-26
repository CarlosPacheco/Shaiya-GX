package app

import (
	"maps"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/data/text"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func loadContentMessages(read func(string) ([]byte, error), languageID int) (english, messages text.SysMsg) {
	base, _ := read("sysmsg-uni.txt")
	english = text.ParseSysMsg(base)
	return english, translatedMessages(english, read, languageID)
}

func translatedMessages(english text.SysMsg, read func(string) ([]byte, error), languageID int) text.SysMsg {
	if code := ui.MessageLanguageCode(languageID); code != "" {
		if data, err := read("Languages/Messages/sysmgtxt-" + code + ".txt"); err == nil {
			return english.WithTranslation(data)
		}
	}
	return english
}

// interfaceText keeps native labels in English independently of content text.
func (a *App) interfaceText(id int) string {
	if a.englishMsg != nil {
		return a.englishMsg.Get(id)
	}
	return a.msg.Get(id)
}

func (a *App) reloadContentMessages(languageID int) {
	if a.arc == nil {
		return
	}
	a.loadSDataTranslations(languageID)
	a.refreshMallText()
	a.questTracker.cachedState = nil
	a.tooltipContent = tooltipContent{}
	messages := translatedMessages(a.englishMsg, a.arc.ReadLiveFile, languageID)
	if maps.Equal(a.msg, messages) {
		return
	}
	if a.state != nil {
		for i := range a.state.Chat {
			line := &a.state.Chat[i]
			if line.Op != 0 && line.Presentation == nil {
				_, id := chatStyle(*line)
				presentation := chatPresentation(a.msg, *line, id)
				line.Presentation = &presentation
			}
		}
	}
	a.msg = messages
	if a.create != nil {
		a.create.SetMessages(messages)
	}
	if a.factionUI != nil {
		a.factionUI.SetMessages(messages)
	}
}

func (a *App) loadSDataTranslations(languageID int) {
	a.localized = sdata.LoadTranslations(a.arc.ReadLiveFile, ui.MessageLanguageCode(languageID))
	a.localizedItems = nil
}

func (a *App) npcDefinition(key sdata.NPCKey) sdata.NPC {
	def, ok := a.npcDefs[key]
	if ok {
		a.localized.NPCs[key].Apply(&def.Name, &def.Welcome)
	}
	return def
}

func (a *App) monsterName(id uint16, fallback string) string {
	if name := a.localized.Monsters[id]; name != "" {
		return name
	}
	return fallback
}

func (a *App) monsterDefinition(id uint16) (sdata.Monster, error) {
	def, err := a.assets.MonsterDefinition(id)
	def.Name = a.monsterName(id, def.Name)
	return def, err
}

func (a *App) questDefinition(id uint16) sdata.Quest {
	return a.localized.Quest(id, a.questDefs[id-1])
}

func (a *App) mallProductText(p sdata.CashProduct) sdata.CashProduct {
	a.localized.Cash[p.Code].Apply(&p.Name, &p.Description)
	return p
}

func (a *App) refreshMallText() {
	var positions [4]int
	for _, base := range a.mallCatalog {
		if base.Bag == 0 {
			continue
		}
		category := min(int(base.Bag)-1, 3)
		i := positions[category]
		positions[category]++
		if products := a.mall.products[category]; i < len(products) && products[i].Code == base.Code {
			p := a.mallProductText(base)
			products[i].Name, products[i].Description = p.Name, p.Description
		}
	}
}
