package proto

import "fmt"

const GXWeather Opcode = 0x7F1C

const (
	WeatherClear = iota
	WeatherRain
	WeatherSnow
)

type Weather struct {
	MapID uint16
	Type  uint8
}

func DecodeWeather(p []byte) (Weather, error) {
	r := NewReader(p)
	v := Weather{r.U16(), r.U8()}
	if err := r.Err(); err != nil {
		return Weather{}, err
	}
	if v.Type > WeatherSnow || r.Remaining() != 0 {
		return Weather{}, fmt.Errorf("invalid weather snapshot")
	}
	return v, nil
}
