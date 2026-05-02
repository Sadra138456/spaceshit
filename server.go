package main

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"io"
	"log"
	"net"

	"github.com/quic-go/quic-go"
)

func runServer(cfg *Config) {
	cert, err := tls.LoadX509KeyPair("server.crt", "server.key")
	if err != nil {
		log.Fatalf("Failed to load TLS cert: %v", err)
	}

	tlsConf := &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"h3"},
	}

	quicConf := &quic.Config{
		Allow0RTT:       true,
		EnableDatagrams: false,
	}

	listener, err := quic.ListenAddr(cfg.ListenAddr, tlsConf, quicConf)
	if err != nil {
		log.Fatalf("QUIC listen failed: %v", err)
	}
	defer listener.Close()

	log.Printf("[SERVER] Listening on %s (Black-Hole mode)", cfg.ListenAddr)

	for {
		session, err := listener.Accept(context.Background())
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}
		go handleSession(session, cfg)
	}
}

func handleSession(session quic.Connection, cfg *Config) {
	defer session.CloseWithError(0, "")

	for {
		stream, err := session.AcceptStream(context.Background())
		if err != nil {
			return
		}
		go handleStream(stream, cfg)
	}
}

func handleStream(stream quic.Stream, cfg *Config) {
	defer stream.Close()

	// Read auth header
	header := make([]byte, 32+2)
	if _, err := io.ReadFull(stream, header); err != nil {
		return // Silent drop
	}

	targetLen := binary.BigEndian.Uint16(header[32:34])
	targetBuf := make([]byte, targetLen)
	if _, err := io.ReadFull(stream, targetBuf); err != nil {
		return
	}
	target := string(targetBuf)

	if !validateAuthHeader(header[:32], cfg.PSK, target) {
		return // Silent drop (Black-Hole)
	}

	// Connect to target
	remote, err := net.Dial("tcp", target)
	if err != nil {
		log.Printf("Target dial failed: %v", err)
		return
	}
	defer remote.Close()

	// Bidirectional relay
	go io.Copy(stream, remote)
	io.Copy(remote, stream)
}
