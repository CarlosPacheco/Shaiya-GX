package ini

import (
	"reflect"
	"testing"
	"unicode/utf16"
)

func TestParseKeepsSectionOrderAndComments(t *testing.T) {
	got := Parse([]byte("\xef\xbb\xbf; c\nloose=1\n[A]\nx = 1 \n# c\nX=2\n[B]\ny=3\nnoequals\n"))
	if len(got) != 3 || got[0].Name != "" || got[1].Name != "A" || got[2].Name != "B" {
		t.Fatalf("sections: %+v", got)
	}
	if v, _ := got[1].Get("x"); v != "1" || !reflect.DeepEqual(got[1].All("x"), []string{"1", "2"}) {
		t.Fatalf("keys: %+v", got[1])
	}
}

func TestDecodeUTF16(t *testing.T) {
	units := utf16.Encode([]rune("[A]\nWord=ñandú\n"))
	raw := []byte{0xff, 0xfe}
	for _, u := range units {
		raw = append(raw, byte(u), byte(u>>8))
	}
	got := Parse(raw)
	if v, _ := got[0].Get("word"); len(got) != 1 || v != "ñandú" {
		t.Fatalf("%+v", got)
	}
}
