package text

import (
	"encoding/binary"
	"testing"
	"unicode/utf16"
)

func TestWordFilterINIAndLiteralMatching(t *testing.T) {
	if ParseWordFilter(nil) != nil || ParseWordFilter([]byte("; example\n[Filter]\nWord=\n")) != nil {
		t.Fatal("empty filter activated the matcher")
	}
	data := []byte{0xff, 0xfe}
	for _, code := range utf16.Encode([]rune("[Filter]\r\nWord=c++\r\nWord=c++ test\r\nword=炎\n")) {
		data = binary.LittleEndian.AppendUint16(data, code)
	}
	if got := ParseWordFilter(data).Censor("C++ TEST 炎 cpp"); got != "******** * cpp" {
		t.Fatal(got)
	}
}
