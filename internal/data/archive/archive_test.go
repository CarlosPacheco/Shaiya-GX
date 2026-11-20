package archive

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// 0x40af40: DJB2 over case-folded signed bytes.
func TestHashFoldsCase(t *testing.T) {
	if Hash("Login.TGA") != Hash("login.tga") {
		t.Fatal("hash must fold A-Z")
	}
	h := uint32(0x1505)
	for _, c := range "ab" {
		h = h*33 + uint32(c)
	}
	if Hash("AB") != h {
		t.Fatalf("hash %#x, want %#x", Hash("AB"), h)
	}
	if Hash("\xe9") == Hash("\xc9") {
		t.Fatal("bytes above 0x7f are not folded")
	}
}

func TestReadPrefixKeepsArchiveEntryIntact(t *testing.T) {
	file := filepath.Join(t.TempDir(), "data.saf")
	if err := os.WriteFile(file, []byte("abcdef"), 0600); err != nil {
		t.Fatal(err)
	}
	handle, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer handle.Close()
	entry := &File{Name: "model.smod", Length: 6}
	a := &Archive{saf: handle, safSize: 6, Root: &Dir{fileByHash: map[uint32]*File{Hash(entry.Name): entry}}}
	prefix, err := a.ReadPrefix("MODEL.SMOD", 2)
	if err != nil || string(prefix) != "ab" || entry.Length != 6 {
		t.Fatal("prefix altered archive entry", string(prefix), err)
	}
	all, err := a.ReadPrefix("model.smod", 100)
	if err != nil || string(all) != "abcdef" {
		t.Fatal("prefix beyond file", string(all), err)
	}
	if _, err := a.ReadPrefix("model.smod", -1); err == nil {
		t.Fatal("negative prefix")
	}
}

func TestLegacyArchiveRejectsInvalidEntryRanges(t *testing.T) {
	dir := t.TempDir()
	sah, saf := filepath.Join(dir, "data.sah"), filepath.Join(dir, "data.saf")
	if err := os.WriteFile(saf, []byte("abcdef"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name           string
		offset, length uint64
		valid          bool
	}{
		{"within archive", 2, 4, true},
		{"zero at EOF", 6, 0, true},
		{"negative length", 0, 0xffffffff, false},
		{"beyond EOF", 2, 5, false},
		{"negative offset", ^uint64(0), 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			index := append([]byte("SAH"), make([]byte, 48)...)
			index = binary.LittleEndian.AppendUint32(index, 0)
			index = binary.LittleEndian.AppendUint32(index, 1)
			index = binary.LittleEndian.AppendUint32(index, 2)
			index = append(index, 'a', 0)
			index = binary.LittleEndian.AppendUint64(index, tc.offset)
			index = binary.LittleEndian.AppendUint32(index, uint32(tc.length))
			index = binary.LittleEndian.AppendUint32(index, 0)
			index = binary.LittleEndian.AppendUint32(index, 0)
			if err := os.WriteFile(sah, index, 0600); err != nil {
				t.Fatal(err)
			}
			a, err := OpenFiles(sah, saf)
			if (err == nil) != tc.valid {
				t.Fatalf("OpenFiles error=%v, expected valid=%v", err, tc.valid)
			}
			if a != nil {
				a.Close()
			}
		})
	}
}
