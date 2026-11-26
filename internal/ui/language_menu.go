package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

const EnglishLanguageID = 1

// IDs are persisted. Keep existing IDs stable when adding languages.
var gameLanguages = [...]struct {
	id   int
	name string
	flag string
}{
	{EnglishLanguageID, "English", "🇺🇸"},
	{2, "Español", "🇪🇸"},
	{3, "Português (Brasil)", "🇧🇷"},
	{4, "Français", "🇫🇷"},
	{5, "Deutsch", "🇩🇪"},
	{6, "Italiano", "🇮🇹"},
	{7, "Polski", "🇵🇱"},
	{8, "Türkçe", "🇹🇷"},
	{9, "Русский", "🇷🇺"},
	{10, "Tiếng Việt", "🇻🇳"},
}

const (
	languageX = 177
	languageY = 42
	languageW = 175
	languageH = 27
)

func languageIndex(id int) int {
	for i, language := range gameLanguages {
		if id == language.id {
			return i
		}
	}
	return 0
}

func NormalizeLanguageID(id int) int { return gameLanguages[languageIndex(id)].id }

func (m *GameMenu) PopupOpen() bool {
	return m.DisplayPopupOpen() || m != nil && m.Page == MenuUser && m.languageOpen
}

func (m *GameMenu) updateLanguage(in *platform.Input, x, y int) bool {
	m.languageButton.OriginX, m.languageButton.OriginY = x, y
	if m.languageButton.Update(in) {
		m.languageOpen = !m.languageOpen
		m.languageCursor = languageIndex(m.Settings.LanguageID)
		m.languageHeld = -1
		return true
	}
	if !m.languageOpen {
		return false
	}
	if in.HasKey(platform.KeyEscape) {
		m.languageOpen = false
		return true
	}
	if in.HasKey(platform.KeyUp) {
		m.languageCursor = max(0, m.languageCursor-1)
	}
	if in.HasKey(platform.KeyDown) {
		m.languageCursor = min(len(gameLanguages)-1, m.languageCursor+1)
	}

	top := y + languageY + languageH
	row := -1
	if in.MouseX >= x+languageX && in.MouseX < x+languageX+languageW && in.MouseY >= top && in.MouseY < top+len(gameLanguages)*languageH {
		row = (in.MouseY - top) / languageH
	}
	m.languageHover = row
	if in.Pressed {
		m.languageHeld = row
		if row < 0 && !m.languageButton.hit(in) {
			m.languageOpen = false
		}
	}
	if in.Clicked && row >= 0 && m.languageHeld == row {
		m.languageCursor = row
		m.Settings.LanguageID = gameLanguages[row].id
		m.languageOpen = false
	}
	if in.HasKey(platform.KeyEnter) {
		m.Settings.LanguageID = gameLanguages[m.languageCursor].id
		m.languageOpen = false
	}
	if in.Clicked {
		m.languageHeld = -1
	}
	return true
}

func (m *GameMenu) drawLanguage(s *render.Sprites, f *Font, solid *render.Texture, x, y int) {
	if m.Page != MenuUser || solid == nil {
		return
	}
	px, py := x+languageX, y+languageY
	fill := TipFill
	fill[3] = 1
	drawRow := func(index, rowY int) {
		language := gameLanguages[index]
		f.Draw(s, language.flag, px+9, rowY+6, white, 27)
		f.Draw(s, language.name, px+39, rowY+6, white, languageW-59)
	}
	DrawTipFrame(s, solid, float32(px), float32(py), languageW, languageH, 1, fill, TipBorder)
	drawRow(languageIndex(m.Settings.LanguageID), py)
	f.Draw(s, "▾", px+languageW-15, py+6, white, 12)
	if !m.languageOpen {
		return
	}

	top := py + languageH
	DrawTipFrame(s, solid, float32(px), float32(top), languageW, float32(len(gameLanguages)*languageH), 1, fill, TipBorder)
	highlight := m.languageCursor
	if m.languageHover >= 0 {
		highlight = m.languageHover
	}
	s.Draw(solid, float32(px+2), float32(top+highlight*languageH+1), languageW-4, languageH-2, 0, 0, 1, 1, RGBColor(0x534a33))
	for i := range gameLanguages {
		drawRow(i, top+i*languageH)
	}
}
