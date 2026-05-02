package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
)

// buildAuthHeader constructs the authentication header:
// [32 bytes PSK hash][2 bytes target length][N bytes target address]
func buildAuthHeader(psk, target string) []byte {
	pskHash := hashPSK(psk)
	targetBytes := []byte(target)
	targetLen := uint16(len(targetBytes))

	header := make([]byte, 34+len(targetBytes))
	copy(header[:32], pskHash)
	binary.BigEndian.PutUint16(header[32:34], targetLen)
	copy(header[34:], targetBytes)

	return header
}

// hashPSK converts hex PSK string to SHA256 hash
func hashPSK(psk string) []byte {
	pskBytes, _ := hex.DecodeString(psk)
	hash := sha256.Sum256(pskBytes)
	return hash[:]
}
