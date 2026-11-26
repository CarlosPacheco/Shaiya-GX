package world

import (
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

const weatherDuration = 10 * time.Minute

var weatherStarted time.Time

func weatherAt(zone ZoneConfig, start, now time.Time) uint8 {
	if zone.WeatherType == 0 || start.IsZero() {
		return proto.WeatherClear
	}
	frequency := zone.WeatherFrequency
	if frequency == 0 {
		frequency = time.Hour
	}
	since := now.Sub(start)
	if since < frequency || since%frequency >= weatherDuration {
		return proto.WeatherClear
	}
	return zone.WeatherType
}

func (s *Session) updateWeather(now time.Time, force bool) {
	if s.char == nil || s.awaitingMapLoad || (!force && now.Before(s.weatherCheck)) {
		return
	}
	s.weatherCheck = now.Add(time.Second)
	current := weatherAt(mapConfig.Zones[realMap(s.char.MapID)], weatherStarted, now)
	if now.Before(s.weatherOverrideUntil) {
		current = s.weatherOverride
	}
	if current == s.lastWeather && !force {
		return
	}
	s.lastWeather = current
	w := packet.New(packet.Opcode(proto.GXWeather))
	w.U16(s.char.MapID)
	w.U8(current)
	s.send(w.MustFinish(), "GX local weather")
}

func (s *Session) gmWeather(args []string) error {
	if len(args) != 1 || (args[0] != "0" && args[0] != "1" && args[0] != "2") {
		return newGMError(12455)
	}
	now := time.Now()
	s.weatherOverride = args[0][0] - '0'
	if s.weatherOverride == proto.WeatherClear {
		s.weatherOverrideUntil = time.Time{}
		s.updateWeather(now, true)
		s.gmMessage(12445)
		return nil
	}
	s.weatherOverrideUntil = now.Add(weatherDuration)
	s.updateWeather(now, true)
	s.gmMessage(12446)
	return nil
}
