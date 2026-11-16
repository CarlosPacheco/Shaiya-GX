// Package authwire defines GX's bounded UTF-8 credential extension. Original
// 0xA102 layouts remain unchanged; long passwords use a distinct opcode.
package authwire

import (
	"encoding/binary"
	"errors"
	"strings"
	"unicode/utf8"
)

const Opcode uint16 = 0x7F12
const MaxUser = 18
const MaxPassword = 128

func Validate(user, password string) error {
	if len(user) < 1 || len(user) > MaxUser || len(password) < 1 || len(password) > MaxPassword || !utf8.ValidString(user) || !utf8.ValidString(password) || strings.ContainsRune(user, 0) || strings.ContainsRune(password, 0) {
		return errors.New("credentials exceed UTF-8 byte limits (18/128) or contain invalid text")
	}
	return nil
}
func Encode(user, password string) ([]byte, error) {
	if err := Validate(user, password); err != nil {
		return nil, err
	}
	out := binary.LittleEndian.AppendUint16(nil, Opcode)
	out = append(out, byte(len(user)), byte(len(password)))
	out = append(out, user...)
	out = append(out, password...)
	return out, nil
}
func Decode(payload []byte) (string, string, error) {
	if len(payload) < 4 || binary.LittleEndian.Uint16(payload) != Opcode {
		return "", "", errors.New("invalid credential extension")
	}
	u, p := int(payload[2]), int(payload[3])
	if len(payload) != 4+u+p {
		return "", "", errors.New("invalid credential lengths")
	}
	user, password := string(payload[4:4+u]), string(payload[4+u:])
	return user, password, Validate(user, password)
}
