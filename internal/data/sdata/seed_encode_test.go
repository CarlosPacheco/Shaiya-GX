package sdata

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"testing"
)

func TestEncodeLikeKeepsEP5SeedEnvelope(t *testing.T) {
	original := []byte("test studio seed")
	header := make([]byte, 80)
	copy(header, seedSignature)
	binary.LittleEndian.PutUint32(header[40:], crc32.ChecksumIEEE(original))
	binary.LittleEndian.PutUint32(header[44:], uint32(len(original)))
	copy(header[64:], original)
	encryptChunk(header[64:])
	next := []byte("new payload with a different block count")
	encoded, err := EncodeLike(header, next)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(encoded, []byte(seedSignature)) || binary.LittleEndian.Uint32(encoded[40:]) != crc32.ChecksumIEEE(next) {
		t.Fatal("invalid encrypted envelope")
	}
	decoded, err := Decrypt(encoded)
	if err != nil || !bytes.Equal(decoded, next) {
		t.Fatalf("decoded %q: %v", decoded, err)
	}
	header[40] ^= 0x80
	if _, err := EncodeLike(header, next); err == nil {
		t.Fatal("accepted a mismatched original checksum")
	}
}
