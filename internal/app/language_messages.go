package app

import (
	"maps"

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
	a.tooltipContent = tooltipContent{}
}
