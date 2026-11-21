package text

import (
	"encoding/binary"
	"testing"
	"unicode/utf16"
)

func TestWordFilterNativeFormatAndLiteralMatching(t *testing.T) {
	if ParseWordFilter(nil) != nil || ParseWordFilter([]byte("// example\n3\t\t\"\"")) != nil {
		t.Fatal("empty table activated the matcher")
	}
	data := []byte{0xff, 0xfe}
	for _, code := range utf16.Encode([]rune("1\t\t\"c++\"\r\n2\t\t\"c++ test\"\r\n3\t\t\"炎\"\n")) {
		data = binary.LittleEndian.AppendUint16(data, code)
	}
	if got := ParseWordFilter(data).Censor("C++ TEST 炎 cpp"); got != "******** * cpp" {
		t.Fatal(got)
	}
}
