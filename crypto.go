package main

import (
	"crypto/rand"
	"math/big"
	"sync"

	utls "github.com/refraction-networking/utls"
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, BufferSize)
		return &buf
	},
}

// GetBuffer از pool می‌گیره (zero-allocation)
func GetBuffer() *[]byte {
	return bufferPool.Get().(*[]byte)
}

// PutBuffer برمی‌گردونه به pool
func PutBuffer(buf *[]byte) {
	bufferPool.Put(buf)
}

// GetUTLSConfig با fingerprint تصادفی
func GetUTLSConfig(sni string) *utls.Config {
	return &utls.Config{
		ServerName:         sni,
		MinVersion:         utls.VersionTLS13,
		InsecureSkipVerify: false,
	}
}

// GetRandomFingerprint انتخاب تصادفی Chrome/Firefox/Safari
func GetRandomFingerprint() utls.ClientHelloID {
	fps := []utls.ClientHelloID{
		utls.HelloChrome_Auto,
		utls.HelloFirefox_Auto,
		utls.HelloSafari_Auto,
	}
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(fps))))
	return fps[n.Int64()]
}

// GeneratePadding با traffic shaping (اندازه‌های استاندارد HTTPS)
func GeneratePadding() []byte {
	// Standard HTTPS packet sizes: 64, 128, 256, 512, 1024
	sizes := []int{64, 128, 256, 512, 1024}
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(sizes))))
	size := sizes[n.Int64()]

	padding := make([]byte, size)
	rand.Read(padding)
	return padding
}

// XORObfuscate lightweight encryption
func XORObfuscate(data []byte, key []byte) {
	for i := range data {
		data[i] ^= key[i%len(key)]
	}
}

// GenerateXORKey تولید کلید تصادفی
func GenerateXORKey() []byte {
	key := make([]byte, 32)
	rand.Read(key)
	return key
}

// GetRandomSNI انتخاب تصادفی از pool
func GetRandomSNI(domains []string) string {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(domains))))
	return domains[n.Int64()]
}

// GetRandomUserAgent
func GetRandomUserAgent() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(UserAgents))))
	return UserAgents[n.Int64()]
}
