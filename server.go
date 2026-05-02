package main

import (
	"crypto/tls"
	"io"
	"log"
	"net"
	"time"

	"github.com/hashicorp/yamux"
)

// StartSpaceServer Ferrari mode
func StartSpaceServer(cfg Config) {
	cert, err := tls.LoadX509KeyPair("server.crt", "server.key")
	if err != nil {
		log.Fatal("[SpaceShit] Certificate error: ", err)
	}

	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
	}

	ln, err := net.Listen("tcp", cfg.ServerAddr)
	if err != nil {
		log.Fatal("[SpaceShit] Listen failed: ", err)
	}
	defer ln.Close()

	log.Printf("[SpaceShit] 🏎️ Ferrari Mission Control on %s", cfg.ServerAddr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("[SpaceShit] Accept error: %v", err)
			continue
		}

		go handleWarpCore(conn, tlsCfg, cfg)
	}
}

func handleWarpCore(conn net.Conn, tlsCfg *tls.Config, cfg Config) {
	defer conn.Close()

	// TCP tuning
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetNoDelay(true)
		tcpConn.SetKeepAlive(true)
	}

	// TLS handshake
	tlsConn := tls.Server(conn, tlsCfg)
	tlsConn.SetReadDeadline(time.Now().Add(AuthTimeout))

	if err := tlsConn.Handshake(); err != nil {
		return
	}

	// PSK validation (XOR obfuscated)
	authBuf := make([]byte, len(cfg.PSK))
	if _, err := io.ReadFull(tlsConn, authBuf); err != nil {
		return
	}

	XORObfuscate(authBuf, cfg.XORKey)
	if string(authBuf) != cfg.PSK {
		return // Silent drop
	}

	// Clear deadline
	tlsConn.SetReadDeadline(time.Time{})

	// Yamux session
	session, err := yamux.Server(tlsConn, nil)
	if err != nil {
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

func handleStream(stream net.Conn) {
	defer stream.Close()

	// Forward to backend
	target, err := net.DialTimeout("tcp", "127.0.0.1:8080", 5*time.Second)
	if err != nil {
		return
	}
	defer target.Close()

	// Zero-copy relay
	done := make(chan struct{}, 2)

	go func() {
		buf := GetBuffer()
		defer PutBuffer(buf)
		io.CopyBuffer(target, stream, *buf)
		done <- struct{}{}
	}()

	go func() {
		buf := GetBuffer()
		defer PutBuffer(buf)
		io.CopyBuffer(stream, target, *buf)
		done <- struct{}{}
	}()

	<-done
}
