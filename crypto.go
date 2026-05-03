package main

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"io"
	"math/big"
	"net"
	"time"

	utls "github.com/refraction-networking/utls"
)

// GetUTLSConfig creates a uTLS config mimicking Chrome
func GetUTLSConfig(sni string, alpn []string) *utls.Config {
	return &utls.Config{
		ServerName:         sni,
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS12,
		MaxVersion:         tls.VersionTLS13,
		NextProtos:         alpn,
	}
}

// GeneratePadding creates random padding
func GeneratePadding(minSize, maxSize int) []byte {
	size := minSize
	if maxSize > minSize {
		diff, _ := rand.Int(rand.Reader, big.NewInt(int64(maxSize-minSize)))
		size += int(diff.Int64())
	}
	padding := make([]byte, size)
	rand.Read(padding)
	return padding
}

// FragmentedWrite sends data in fragments with delays
func FragmentedWrite(conn net.Conn, data []byte, fragmentSize int, delay, jitter time.Duration) error {
	if fragmentSize <= 0 || fragmentSize >= len(data) {
		_, err := conn.Write(data)
		return err
	}

	for i := 0; i < len(data); i += fragmentSize {
		end := i + fragmentSize
		if end > len(data) {
			end = len(data)
		}

		if _, err := conn.Write(data[i:end]); err != nil {
			return err
		}

		if i+fragmentSize < len(data) {
			sleepTime := delay
			if jitter > 0 {
				j, _ := rand.Int(rand.Reader, big.NewInt(int64(jitter)))
				sleepTime += time.Duration(j.Int64())
			}
			time.Sleep(sleepTime)
		}
	}
	return nil
}

// DeriveKey derives a key from PSK
func DeriveKey(psk string) []byte {
	hash := sha256.Sum256([]byte(psk))
	return hash[:]
}

// XORCipher applies XOR encryption
func XORCipher(data, key []byte) {
	for i := range data {
		data[i] ^= key[i%len(key)]
	}
}

// SecureRead reads with timeout
func SecureRead(conn net.Conn, buf []byte, timeout time.Duration) (int, error) {
	conn.SetReadDeadline(time.Now().Add(timeout))
	defer conn.SetReadDeadline(time.Time{})
	return conn.Read(buf)
}

// SecureWrite writes with timeout
func SecureWrite(conn net.Conn, data []byte, timeout time.Duration) error {
	conn.SetWriteDeadline(time.Now().Add(timeout))
	defer conn.SetWriteDeadline(time.Time{})
	_, err := conn.Write(data)
	return err
}

// BidirectionalCopy copies data between two connections
func BidirectionalCopy(conn1, conn2 net.Conn) {
	done := make(chan struct{}, 2)

	copy := func(dst, src net.Conn) {
		defer func() { done <- struct{}{} }()
		io.Copy(dst, src)
		dst.Close()
		src.Close()
	}

	go copy(conn1, conn2)
	go copy(conn2, conn1)

	<-done
	<-done
}

// GenerateXORKey generates XOR key from PSK
func GenerateXORKey(psk string) string {
	hash := sha256.Sum256([]byte(psk))
	return hex.EncodeToString(hash[:])
}
