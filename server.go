package main

import (
	"crypto/tls"
	"io"
	"log"
	"net"

	"github.com/hashicorp/yamux"
)

func RunServer(cfg *Config) {
	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		log.Fatalf("Failed to load certificate: %v", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	listener, err := tls.Listen("tcp", cfg.ServerAddr, tlsConfig)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	defer listener.Close()

	log.Printf("[Ferrari] 🏎️  Server running on %s", cfg.ServerAddr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}
		go handleServerConnection(conn, cfg)
	}
}

func handleServerConnection(conn net.Conn, cfg *Config) {
	defer conn.Close()

	// XOR obfuscation
	obfsConn := NewObfuscatedConn(conn, cfg.PSK)

	// Yamux session
	session, err := yamux.Server(obfsConn, nil)
	if err != nil {
		log.Printf("Yamux server error: %v", err)
		return
	}
	defer session.Close()

	for {
		stream, err := session.Accept()
		if err != nil {
			return
		}
		go handleStream(stream)
	}
}

func handleStream(stream *yamux.Stream) {
	defer stream.Close()

	// ✅ دریافت target address از client
	buf := make([]byte, 512)
	n, err := stream.Read(buf)
	if err != nil {
		log.Printf("Failed to read target: %v", err)
		return
	}

	targetAddr := string(buf[:n])
	log.Printf("[Ferrari] 🎯 Connecting to: %s", targetAddr)

	// ✅ اتصال به target واقعی (نه 127.0.0.1:8080)
	target, err := net.Dial("tcp", targetAddr)
	if err != nil {
		log.Printf("Failed to connect to %s: %v", targetAddr, err)
		return
	}
	defer target.Close()

	// Forward traffic
	go io.Copy(target, stream)
	io.Copy(stream, target)
}
