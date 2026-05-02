package main

import (
	"context"
	"crypto/tls"
	"io"
	"log"
	"net"
	"encoding/binary"

	"github.com/quic-go/quic-go"
)

func runServer(cfg *Config) {
	cert, err := tls.LoadX509KeyPair("server.crt", "server.key")
	if err != nil {
		log.Fatalf("Failed to load TLS certs: %v", err)
	}

	tlsConf := &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"h3"},
	}

	quicConf := &quic.Config{
		Allow0RTT: true,
	}

	listener, err := quic.ListenAddr(cfg.ListenAddr, tlsConf, quicConf)
	if err != nil {
		log.Fatal(err)
	}

	for {
		conn, err := listener.Accept(context.Background())
		if err != nil {
			continue
		}
		go handleConnection(conn, cfg)
	}
}

func handleConnection(conn quic.Connection, cfg *Config) {
	for {
		stream, err := conn.AcceptStream(context.Background())
		if err != nil {
			return
		}
		go handleStream(stream, cfg)
	}
}

func handleStream(stream quic.Stream, cfg *Config) {
	defer stream.Close()

	// 1. خواندن هش PSK
	authHash := make([]byte, 32)
	if _, err := io.ReadFull(stream, authHash); err != nil {
		return
	}

	// 2. تایید هویت (Black-Hole)
	if !validateAuthHeader(cfg.PSK, authHash) {
		return // سایلنت دراپ
	}

	// 3. خواندن آدرس مقصد
	lenBuf := make([]byte, 2)
	if _, err := io.ReadFull(stream, lenBuf); err != nil {
		return
	}
	targetLen := binary.BigEndian.Uint16(lenBuf)
	targetBuf := make([]byte, targetLen)
	if _, err := io.ReadFull(stream, targetBuf); err != nil {
		return
	}
	targetAddr := string(targetBuf)

	// 4. اتصال به مقصد نهایی
	targetConn, err := net.Dial("tcp", targetAddr)
	if err != nil {
		return
	}
	defer targetConn.Close()

	// 5. جابجایی ترافیک (Relay)
	done := make(chan struct{})
	go func() {
		io.Copy(targetConn, stream)
		close(done)
	}()
	go func() {
		io.Copy(stream, targetConn)
		close(done)
	}()
	<-done
}
