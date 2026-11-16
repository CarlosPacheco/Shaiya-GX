package proto

import "fmt"

const (
	MobSpeechText  Opcode = 0x1109
	MobSpeechIndex Opcode = 0x110a
)

type MobSpeech struct {
	MobID     uint32
	MessageID uint16
	Text      string
}

func DecodeMobSpeech(p []byte) (MobSpeech, error) {
	r := NewReader(p)
	var speech MobSpeech
	switch Op(p) {
	case MobSpeechText:
		if r.U8() != 1 {
			return speech, fmt.Errorf("invalid mob speech type")
		}
		speech.MobID = r.U32()
		speech.Text = r.Text()
	case MobSpeechIndex:
		speech.MobID = r.U32()
		speech.MessageID = r.U16()
	default:
		return speech, fmt.Errorf("invalid mob speech opcode")
	}
	if e := r.Err(); e != nil {
		return speech, e
	}
	if r.Remaining() != 0 {
		return speech, fmt.Errorf("trailing mob speech bytes")
	}
	return speech, nil
}
