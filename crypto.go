package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
)

func buildAuthHeader(pskHex, target string) []byte {
	pskHash := hashPSK(pskHex, target)
	targetBytes := []byte(target)
	
	header := make([]byte, 34+len(targetBytes))
	copy(header[0:32], pskHash)
	binary.BigEndian.PutUint16(header[32:34], uint16(len(targetBytes)))
	copy(header[34:], targetBytes)
	
	return header
}

func hashPSK(pskHex, target string) []byte {
	pskBytes, _ := hex.DecodeString(pskHex)
	combined := append(pskBytes, []byte(target)...)
	hash := sha256.Sum256(combined)
	return hash[:]
}

func validateAuthHeader(receivedHash []byte, pskHex, target string) bool {
	expectedHash := hashPSK(pskHex, target)
	return bytesEqual(receivedHash, expectedHash)
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
