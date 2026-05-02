package main

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"github.com/quic-go/quic-go"
)

func runServer(cfg Config) {
	// TLS certificate (self-signed for now)
	cert, err := tls.LoadX509KeyPair("server.crt", "server.key")
	if err != nil {
		log.Fatalf("Failed to load TLS cert: %v (generate with: openssl req -x509 -newkey rsa:2048 -keyout server.key -out server.crt -days 365 -nodes)", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"h3"},
	}

	quicConfig := &quic.Config{
		MaxIdleTimeout:  60 * time.Second,
		KeepAlivePeriod: 15 * time.Second,
		Allow0RTT:       true,
	}

	// Start QUIC listener
	listener, err := quic.ListenAddr(cfg.ListenAddr, tlsConfig, quicConfig)
	if err != nil {
		log.Fatalf("Failed to start QUIC listener: %v", err)
	}
	defer listener.Close()

	log.Printf("QUIC server listening on %s", cfg.ListenAddr)

	for {
		session, err := listener.Accept(context.Background())
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}
		go handleSession(session, cfg)
	}
}

func handleSession(session quic.Connection, cfg Config) {
	defer session.CloseWithError(0, "session closed")

	for {
		stream, err := session.AcceptStream(context.Background())
		if err != nil {
			return
		}
		go handleStream(stream, cfg)
	}
}

func handleStream(stream quic.Stream, cfg Config) {
	defer stream.Close()

	// Read authentication header (first 64 bytes)
	authBuf := make([]byte, 64)
	if _, err := io.ReadFull(stream, authBuf); err != nil {
		log.Printf("Auth read failed: %v", err)
		return
	}

	// Validate PSK
	target, valid := validateAuthHeader(authBuf, cfg.PSK)
	if !valid {
		log.Printf("Invalid PSK - silent drop")
		return // Black-hole: no response
	}

	// Connect to target
	targetConn, err := net.DialTimeout("tcp", target, 10*time.Second)
	if err != nil {
		log.Printf("Target dial failed (%s): %v", target, err)
		return
	}
	defer targetConn.Close()

	log.Printf("Forwarding: %s → %s", stream.Context().Value("remote"), target)

	// Bidirectional relay
	go io.Copy(targetConn, stream)
	io.Copy(stream, targetConn)
}

func validateAuthHeader(header []byte, psk string) (string, bool) {
	// Extract PSK hash (first 32 bytes)
	pskHash := header[:32]

	// Validate against expected PSK
	expectedHash := hashPSK(psk)
	if !bytesEqual(pskHash, expectedHash) {
		return "", false
	}

	// Extract target address length and address
	targetLen := binary.BigEndian.Uint16(header[32:34])
	if int(targetLen) > len(header)-34 {
		return "", false
	}

	target := string(header[34 : 34+targetLen])
	return target, true
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
