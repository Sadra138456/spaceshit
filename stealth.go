package main

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// selectSNI picks a random SNI domain from the list
func selectSNI(domains []string) string {
	if len(domains) == 0 {
		return "https://www.aparat.com" // Default fallback
	}
	idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(domains))))
	return domains[idx.Int64()]
}

// applyPadding adds random padding (128-512 bytes) to mimic video streaming
func applyPadding(data []byte) []byte {
	paddingSize, _ := rand.Int(rand.Reader, big.NewInt(385)) // 128 to 512
	padding := make([]byte, 128+paddingSize.Int64())
	rand.Read(padding)

	// Prepend fake HTTP/3 header
	fakeHeader := generateFakeHTTP3Header()
	return append(append(fakeHeader, data...), padding...)
}

// generateFakeHTTP3Header creates a fake HTTP/3 GET request header
func generateFakeHTTP3Header() []byte {
	chunkID, _ := rand.Int(rand.Reader, big.NewInt(999999))
	header := fmt.Sprintf("GET /api/v2/stream/chunk_%d.m4s HTTP/3\r\nHost: video.aparat.com\r\n\r\n", chunkID.Int64())
	return []byte(header)
}
