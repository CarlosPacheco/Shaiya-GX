package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/text"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func loadContentMessages(read func(string) ([]byte, error), languageID int) (english, messages text.SysMsg) {
	base, _ := read("sysmsg-uni.txt")
	english = text.ParseSysMsg(base)
	if code := ui.MessageLanguageCode(languageID); code != "" {
		if data, err := read("Languages/Messages/sysmgtxt-" + code + ".txt"); err == nil {
			return english, english.WithTranslation(data)
		}
	}
	return english, english
}

// interfaceText keeps native labels in English independently of content text.
func (a *App) interfaceText(id int) string {
	if a.englishMsg != nil {
		return a.englishMsg.Get(id)
	}
	return a.msg.Get(id)
}
