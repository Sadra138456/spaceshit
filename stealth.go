package main

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

func selectSNI(domains []string) string {
	if len(domains) == 0 {
		return "video.aparat.com" // Default fallback
	}
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(domains))))
	return domains[n.Int64()]
}

// ایجاد پکت‌های فیک برای گمراه کردن سیستم‌های بازرسی (DPI)
func getFakePadding() []byte {
	size, _ := rand.Int(rand.Reader, big.NewInt(384)) // 0 to 384
	padding := make([]byte, size.Int64()+128)        // Min 128 bytes
	rand.Read(padding)
	return padding
}

func generateFakeHTTP3Header(host string) []byte {
	return []byte(fmt.Sprintf("GET /video/chunk/shd/%d.m4s HTTP/3\r\nHost: %s\r\n\r\n", 1000+makeInt(), host))
}

func makeInt() int64 {
	n, _ := rand.Int(rand.Reader, big.NewInt(9000))
	return n.Int64()
}
