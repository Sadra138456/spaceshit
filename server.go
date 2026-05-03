package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"github.com/hashicorp/yamux"
)

func RunServer(cfg *Config) error {
	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return fmt.Errorf("TLS config failed: %v", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
		MaxVersion:   tls.VersionTLS13,
		NextProtos:   cfg.ALPN,
	}

	listener, err := tls.Listen("tcp", cfg.ServerAddr, tlsConfig)
	if err != nil {
		return fmt.Errorf("server listen failed: %v", err)
	}
	defer listener.Close()

	log.Printf("🚀 Ferrari Tunnel Server running on %s", cfg.ServerAddr)

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

	// Read PSK
	pskBuf := make([]byte, 64)
	n, err := SecureRead(conn, pskBuf, AuthTimeout)
	if err != nil || string(pskBuf[:n]) != cfg.PSK {
		log.Printf("❌ Auth failed from %s", conn.RemoteAddr())
		return
	}

	log.Printf("✅ Client authenticated: %s", conn.RemoteAddr())

	// Read and discard padding
	paddingLenBuf := make([]byte, 2)
	if _, err := io.ReadFull(conn, paddingLenBuf); err != nil {
		log.Printf("Padding read error: %v", err)
		return
	}
	paddingLen := int(paddingLenBuf[0])<<8 | int(paddingLenBuf[1])
	if paddingLen > 0 && paddingLen < 4096 {
		io.CopyN(io.Discard, conn, int64(paddingLen))
	}

	// Setup yamux
	muxConfig := yamux.DefaultConfig()
	muxConfig.KeepAliveInterval = cfg.MuxKeepAlive
	session, err := yamux.Server(conn, muxConfig)
	if err != nil {
		log.Printf("Yamux server error: %v", err)
		return
	}
	defer session.Close()

	log.Printf("🔗 Yamux session established with %s", conn.RemoteAddr())

	for {
		stream, err := session.Accept()
		if err != nil {
			log.Printf("Stream accept error: %v", err)
			return
		}

		go handleStream(stream)
	}
}

func handleStream(stream net.Conn) {
	defer stream.Close()

	// Read target address (SOCKS5 format: [ATYP][ADDR][PORT])
	addrTypeBuf := make([]byte, 1)
	if _, err := io.ReadFull(stream, addrTypeBuf); err != nil {
		log.Printf("Read address type error: %v", err)
		return
	}

	var targetAddr string
	switch addrTypeBuf[0] {
	case 0x01: // IPv4
		addrBuf := make([]byte, 4)
		if _, err := io.ReadFull(stream, addrBuf); err != nil {
			return
		}
		portBuf := make([]byte, 2)
		if _, err := io.ReadFull(stream, portBuf); err != nil {
			return
		}
		port := int(portBuf[0])<<8 | int(portBuf[1])
		targetAddr = fmt.Sprintf("%d.%d.%d.%d:%d", addrBuf[0], addrBuf[1], addrBuf[2], addrBuf[3], port)

	case 0x03: // Domain
		lenBuf := make([]byte, 1)
		if _, err := io.ReadFull(stream, lenBuf); err != nil {
			return
		}
		domainBuf := make([]byte, lenBuf[0])
		if _, err := io.ReadFull(stream, domainBuf); err != nil {
			return
		}
		portBuf := make([]byte, 2)
		if _, err := io.ReadFull(stream, portBuf); err != nil {
			return
		}
		port := int(portBuf[0])<<8 | int(portBuf[1])
		targetAddr = fmt.Sprintf("%s:%d", string(domainBuf), port)

	default:
		log.Printf("Unsupported address type: 0x%02x", addrTypeBuf[0])
		return
	}

	// Connect to target
	target, err := net.DialTimeout("tcp", targetAddr, 10*time.Second)
	if err != nil {
		log.Printf("Dial to %s failed: %v", targetAddr, err)
		stream.Write([]byte{0x05, 0x01}) // Connection refused
		return
	}
	defer target.Close()

	// Send success response
	stream.Write([]byte{0x05, 0x00}) // Success

	log.Printf("🔀 Forwarding: %s → %s", stream.RemoteAddr(), targetAddr)

	BidirectionalCopy(stream, target)
}
