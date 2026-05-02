package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/yamux"
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, 32*1024) // 32KB buffers
		return &buf
	},
}

// StartSpaceServer launches the Ferrari tunnel server
func StartSpaceServer(cfg Config) {
	cert, err := tls.LoadX509KeyPair("server.crt", "server.key")
	if err != nil {
		log.Fatal("[Ferrari] ❌ Certificate error: ", err)
	}

	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
		CipherSuites: []uint16{
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
		},
	}

	ln, err := net.Listen("tcp", cfg.ServerAddr)
	if err != nil {
		log.Fatal("[Ferrari] ❌ Listen error: ", err)
	}

	log.Printf("[Ferrari] 🏎️  Server started on %s", cfg.ServerAddr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("[Ferrari] ⚠️  Accept error: %v", err)
			continue
		}
		go handleFerrariClient(conn, tlsCfg, cfg)
	}
}

func handleFerrariClient(conn net.Conn, tlsCfg *tls.Config, cfg Config) {
	defer conn.Close()

	// TLS handshake
	tlsConn := tls.Server(conn, tlsCfg)
	tlsConn.SetReadDeadline(time.Now().Add(10 * time.Second))

	if err := tlsConn.Handshake(); err != nil {
		log.Printf("[Ferrari] ⚠️  TLS handshake failed: %v", err)
		return
	}

	// Validate PSK (silent authentication)
	authBuf := make([]byte, len(cfg.PSK))
	if _, err := io.ReadFull(tlsConn, authBuf); err != nil {
		log.Printf("[Ferrari] ⚠️  PSK read error: %v", err)
		return
	}

	if string(authBuf) != cfg.PSK {
		log.Printf("[Ferrari] 🚫 Invalid PSK from %s", conn.RemoteAddr())
		return
	}

	// Read padding (DPI bypass)
	paddingBuf := make([]byte, 512)
	tlsConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	if _, err := io.ReadFull(tlsConn, paddingBuf); err != nil {
		log.Printf("[Ferrari] ⚠️  Padding read error: %v", err)
		return
	}

	// Remove auth timeout
	tlsConn.SetReadDeadline(time.Time{})

	log.Printf("[Ferrari] ✅ Client authenticated: %s", conn.RemoteAddr())

	// Start yamux session
	session, err := yamux.Server(tlsConn, nil)
	if err != nil {
		log.Printf("[Ferrari] ⚠️  Yamux error: %v", err)
		return
	}
	defer session.Close()

	// Handle streams
	for {
		stream, err := session.Accept()
		if err != nil {
			if err != io.EOF {
				log.Printf("[Ferrari] ⚠️  Stream accept error: %v", err)
			}
			return
		}
		go handleStream(stream)
	}
}

func handleStream(stream net.Conn) {
	defer stream.Close()

	// Read target address from client (format: "host:port\n")
	stream.SetReadDeadline(time.Now().Add(10 * time.Second))
	reader := bufio.NewReader(stream)
	targetAddr, err := reader.ReadString('\n')
	if err != nil {
		log.Printf("[Ferrari] ⚠️  Target read error: %v", err)
		return
	}

	targetAddr = strings.TrimSpace(targetAddr)
	stream.SetReadDeadline(time.Time{})

	log.Printf("[Ferrari] 🎯 Connecting to: %s", targetAddr)

	// Connect to target
	target, err := net.DialTimeout("tcp", targetAddr, 15*time.Second)
	if err != nil {
		log.Printf("[Ferrari] ❌ Target dial failed (%s): %v", targetAddr, err)
		return
	}
	defer target.Close()

	// Enable TCP optimizations
	if tcpConn, ok := target.(*net.TCPConn); ok {
		tcpConn.SetNoDelay(true)
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(30 * time.Second)
	}

	log.Printf("[Ferrari] ✅ Tunnel established: %s", targetAddr)

	// Bidirectional relay with zero-copy I/O
	errChan := make(chan error, 2)

	// Stream → Target
	go func() {
		buf := bufferPool.Get().(*[]byte)
		defer bufferPool.Put(buf)
		_, err := io.CopyBuffer(target, stream, *buf)
		errChan <- err
	}()

	// Target → Stream
	go func() {
		buf := bufferPool.Get().(*[]byte)
		defer bufferPool.Put(buf)
		_, err := io.CopyBuffer(stream, target, *buf)
		errChan <- err
	}()

	// Wait for either direction to close
	err = <-errChan
	if err != nil && err != io.EOF {
		log.Printf("[Ferrari] ⚠️  Relay error (%s): %v", targetAddr, err)
	} else {
		log.Printf("[Ferrari] ✅ Connection closed: %s", targetAddr)
	}
}
