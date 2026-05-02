package main

import (
	"bufio"
	"crypto/tls"
	"io"
	"log"
	"net"
	"strings"
	"time"

	"github.com/hashicorp/yamux"
)

func StartSpaceServer(cfg Config) {
	cert, err := tls.LoadX509KeyPair("server.crt", "server.key")
	if err != nil {
		log.Fatal("[Ferrari] ❌ TLS cert error: ", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
	}

	ln, err := tls.Listen("tcp", cfg.ServerAddr, tlsConfig)
	if err != nil {
		log.Fatal("[Ferrari] ❌ Listen error: ", err)
	}

	log.Printf("[Ferrari] 🏎️  Server listening on %s", cfg.ServerAddr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("[Ferrari] ⚠️  Accept error: %v", err)
			continue
		}

		go handleClient(conn, cfg)
	}
}

func handleClient(conn net.Conn, cfg Config) {
	defer conn.Close()

	tlsConn := conn.(*tls.Conn)
	tlsConn.SetReadDeadline(time.Now().Add(AuthTimeout))

	pskBuf := make([]byte, len(cfg.PSK))
	if _, err := io.ReadFull(tlsConn, pskBuf); err != nil {
		return
	}

	if string(pskBuf) != cfg.PSK {
		return
	}

	junkBuf := make([]byte, MaxJunkSize)
	tlsConn.Read(junkBuf)

	tlsConn.SetReadDeadline(time.Time{})

	session, err := yamux.Server(tlsConn, nil)
	if err != nil {
		return
	}
	defer session.Close()

	log.Printf("[Ferrari] ✅ Client authenticated: %s", conn.RemoteAddr())

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

	reader := bufio.NewReader(stream)
	targetAddr, err := reader.ReadString('\n')
	if err != nil {
		return
	}

	targetAddr = strings.TrimSpace(targetAddr)

	backend, err := net.DialTimeout("tcp", targetAddr, 10*time.Second)
	if err != nil {
		log.Printf("[Ferrari] ❌ Backend dial failed: %s → %v", targetAddr, err)
		return
	}
	defer backend.Close()

	log.Printf("[Ferrari] 🔗 Forwarding to %s", targetAddr)

	errChan := make(chan error, 2)

	go func() {
		buf := bufferPool.Get().(*[]byte)
		defer bufferPool.Put(buf)
		_, err := io.CopyBuffer(backend, stream, *buf)
		errChan <- err
	}()

	go func() {
		buf := bufferPool.Get().(*[]byte)
		defer bufferPool.Put(buf)
		_, err := io.CopyBuffer(stream, backend, *buf)
		errChan <- err
	}()

	<-errChan
}
