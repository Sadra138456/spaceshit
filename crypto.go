package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"bytes"
)

// تولید هدر امن برای تانل
func buildAuthHeader(psk string, target string) []byte {
	hash := hashPSK(psk)
	targetBytes := []byte(target)
	targetLen := uint16(len(targetBytes))

	header := make([]byte, 32+2+len(targetBytes))
	copy(header[0:32], hash)
	binary.BigEndian.PutUint16(header[32:34], targetLen)
	copy(header[34:], targetBytes)

	return header
}

func hashPSK(psk string) []byte {
	key, _ := hex.DecodeString(psk)
	hash := sha256.Sum256(key)
	return hash[:]
}

// حل مشکل ارور undefined در سرور
func validateAuthHeader(psk string, receivedHash []byte) bool {
	expectedHash := hashPSK(psk)
	return bytes.Equal(expectedHash, receivedHash)
}
