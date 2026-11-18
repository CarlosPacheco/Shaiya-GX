package sdata

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
)

// EncodeLike preserves the input's plaintext or regular EP5 SEED envelope.
// The checksum and round order follow Parsec (credited in NOTICE).
func EncodeLike(original, plain []byte) ([]byte, error) {
	if !bytes.HasPrefix(original, []byte(seedSignature)) {
		return plain, nil
	}
	old, err := Decrypt(original)
	if err != nil {
		return nil, err
	}
	checksum := binary.LittleEndian.Uint32(original[40:44])
	nextChecksum := crc32.ChecksumIEEE(plain)
	if checksum == 0 || checksum != crc32.ChecksumIEEE(old) || nextChecksum == 0 {
		return nil, errors.New("sdata: unsupported SEED header or checksum")
	}
	out := make([]byte, 64+(len(plain)+15)/16*16)
	copy(out[:64], original[:64])
	binary.LittleEndian.PutUint32(out[40:44], nextChecksum)
	binary.LittleEndian.PutUint32(out[44:48], uint32(len(plain)))
	copy(out[64:], plain)
	for offset := 64; offset < len(out); offset += 16 {
		encryptChunk(out[offset : offset+16])
	}
	decoded, err := Decrypt(out)
	if err != nil || !bytes.Equal(decoded, plain) {
		return nil, errors.New("sdata: encrypted round-trip verification failed")
	}
	return out, nil
}

func encryptChunk(block []byte) {
	l0, l1, r0, r1 := be32(block), be32(block[4:]), be32(block[8:]), be32(block[12:])
	for k := 0; k < 32; k += 4 {
		round(&l0, &l1, r0, r1, k)
		round(&r0, &r1, l0, l1, k+2)
	}
	binary.BigEndian.PutUint32(block, r0)
	binary.BigEndian.PutUint32(block[4:], r1)
	binary.BigEndian.PutUint32(block[8:], l0)
	binary.BigEndian.PutUint32(block[12:], l1)
}
