package proto

import (
	"fmt"
	"strings"
)

// GXServerNotice carries a global announcement as native length-prefixed chat text.
const GXServerNotice Opcode = 0x7F1D

func DecodeServerNotice(payload []byte) (string, error) {
	if Op(payload) != GXServerNotice || len(payload) < 5 || len(payload) > 131 || payload[len(payload)-1] != 0 {
		return "", fmt.Errorf("invalid server notice")
	}
	r := NewReader(payload)
	text := r.Text()
	if err := r.Err(); err != nil {
		return "", err
	}
	if r.Remaining() != 0 || strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("invalid server notice text")
	}
	return text, nil
}
