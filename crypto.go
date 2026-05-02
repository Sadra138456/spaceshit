package main

import (
	"crypto/rand"
	utls "github.com/refraction-networking/utls"
)

// GetUTLSConfig generates a Chrome-mimicking TLS configuration
func GetUTLSConfig(sni string) *utls.Config {
	return &utls.Config{
		ServerName: sni,
		MinVersion: utls.VersionTLS13, // Only TLS 1.3 for maximum security
	}
}

// GeneratePadding creates random noise to defeat statistical traffic analysis
func GeneratePadding() []byte {
	sizeBuf := make([]byte, 1)
	rand.Read(sizeBuf)
	size := int(sizeBuf[0]) % MaxJunkSize
	padding := make([]byte, size)
	rand.Read(padding)
	return padding
}
