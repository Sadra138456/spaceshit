package main

import (
	"crypto/rand"
	"crypto/tls"
	"log"

	utls "github.com/refraction-networking/utls"
)

func NewClientTLSConfig(sni string) *utls.Config {
	return &utls.Config{
		ServerName:         sni,
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS12,
	}
}

func NewServerTLSConfig(certFile, keyFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}, nil
}

func GeneratePadding(maxSize int) []byte {
	if maxSize <= 0 {
		return nil
	}
	size := 1 + (int(randomByte()) % maxSize)
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		log.Printf("⚠️  Padding generation failed: %v", err)
	}
	return buf
}

func randomByte() byte {
	b := make([]byte, 1)
	rand.Read(b)
	return b[0]
}
