package app

import (
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

// loadLanguage sets the catalog for the lifetime of this client.
func (a *App) loadLanguage(id int) {
	a.activeLanguageID = ui.NormalizeLanguageID(id)
	code := ui.MessageLanguageCode(a.activeLanguageID)
	a.assets.UseInterfaceLanguage(code)
	a.englishMsg, a.msg = loadContentMessages(a.arc.ReadFile, a.activeLanguageID)
	a.localized = sdata.LoadTranslations(a.arc.ReadFile, code)
	a.loadLocalizedLoadingTips(a.activeLanguageID)
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
