package main

import (
	"crypto/tls"
	"encoding/binary"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/hashicorp/yamux"
)

func RunServer(cfg *Config) {
	tlsConfig, err := NewServerTLSConfig(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		log.Fatalf("❌ TLS config failed: %v", err)
	}

	ln, err := tls.Listen("tcp", cfg.ServerAddr, tlsConfig)
	if err != nil {
		log.Fatalf("❌ Server listen failed: %v", err)
	}
	defer ln.Close()

	log.Printf("🚀 Server listening on %s", cfg.ServerAddr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("⚠️  Accept error: %v", err)
			continue
		}
		go handleServerConn(conn, cfg)
	}
}

func handleServerConn(conn net.Conn, cfg *Config) {
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(AuthTimeout))

	// ✅ Read PSK
	pskBuf := make([]byte, len(cfg.PSK))
	if _, err := io.ReadFull(conn, pskBuf); err != nil {
		log.Printf("⚠️  PSK read failed: %v", err)
		return
	}
	if string(pskBuf) != cfg.PSK {
		log.Println("⚠️  Invalid PSK")
		return
	}

	// ✅ Read padding length
	var padLen uint16
	if err := binary.Read(conn, binary.BigEndian, &padLen); err != nil {
		log.Printf("⚠️  Padding length read failed: %v", err)
		return
	}

	// ✅ Discard padding
	if padLen > 0 {
		if _, err := io.CopyN(io.Discard, conn, int64(padLen)); err != nil {
			log.Printf("⚠️  Padding discard failed: %v", err)
			return
		}
	}

	conn.SetDeadline(time.Time{})

	// ✅ Create yamux session
	session, err := yamux.Server(conn, nil)
	if err != nil {
		log.Printf("⚠️  Yamux server failed: %v", err)
		return
	}
	defer session.Close()

	log.Println("✅ Client authenticated")

	for {
		stream, err := session.AcceptStream()
		if err != nil {
			return
		}
		go handleStream(stream)
	}
}

func handleStream(stream net.Conn) {
	defer stream.Close()

	// ✅ Read target address from client
	var addrLen uint16
	if err := binary.Read(stream, binary.BigEndian, &addrLen); err != nil {
		log.Printf("⚠️  Address length read failed: %v", err)
		return
	}

	addrBuf := make([]byte, addrLen)
	if _, err := io.ReadFull(stream, addrBuf); err != nil {
		log.Printf("⚠️  Address read failed: %v", err)
		return
	}

	targetAddr := string(addrBuf)
	log.Printf("🔗 Connecting to %s", targetAddr)

	// ✅ Connect to real target
	backend, err := net.DialTimeout("tcp", targetAddr, 10*time.Second)
	if err != nil {
		log.Printf("⚠️  Backend dial failed: %v", err)
		return
	}
	defer backend.Close()

	// ✅ Bidirectional copy
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		io.Copy(backend, stream)
	}()

	go func() {
		defer wg.Done()
		io.Copy(stream, backend)
	}()

	wg.Wait()
}
