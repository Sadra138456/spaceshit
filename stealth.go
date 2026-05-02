package main

import (
	"fmt"
	"math/rand"
	"time"
)

func selectSNI(domains []string) string {
	if len(domains) == 0 {
		return "www.aparat.com"
	}
	rand.Seed(time.Now().UnixNano())
	return domains[rand.Intn(len(domains))]
}

func applyPadding(data []byte) []byte {
	rand.Seed(time.Now().UnixNano())
	chunkID := rand.Intn(9999)
	fakeHeader := fmt.Sprintf("GET /api/v2/stream/chunk_%d.m4s HTTP/3\r\n\r\n", chunkID)
	
	paddingSize := 128 + rand.Intn(385)
	padding := make([]byte, paddingSize)
	rand.Read(padding)
	
	result := make([]byte, 0, len(fakeHeader)+len(data)+len(padding))
	result = append(result, []byte(fakeHeader)...)
	result = append(result, data...)
	result = append(result, padding...)
	
	return result
}
