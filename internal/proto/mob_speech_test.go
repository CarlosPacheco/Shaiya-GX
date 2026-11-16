package proto

import "testing"

func TestMobSpeechRejectsIncompleteOrTrailingData(t *testing.T) {
	for _, op := range []Opcode{MobSpeechText, MobSpeechIndex} {
		w := NewWriter(op)
		if op == MobSpeechText {
			w.U8(1)
		}
		w.U32(123)
		if op == MobSpeechText {
			w.Text("Boss speech")
		} else {
			w.U16(6401)
		}
		p := w.Bytes()
		if _, err := DecodeMobSpeech(p); err != nil {
			t.Fatal(err)
		}
		for i := range len(p) {
			if _, err := DecodeMobSpeech(p[:i]); err == nil {
				t.Fatalf("accepted truncated %04x at %d", op, i)
			}
		}
		if _, err := DecodeMobSpeech(append(p, 0)); err == nil {
			t.Fatal("accepted trailing data")
		}
	}
}
