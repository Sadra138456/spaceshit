package main

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/quic-go/quic-go"
	utls "github.com/refraction-networking/utls"
)

func runClient(cfg *Config) {
	listener, err := net.Listen("tcp", cfg.ListenAddr)
	if err != nil {
		log.Fatalf("SOCKS5 listen failed: %v", err)
	}
	defer listener.Close()
	log.Printf("[CLIENT] SOCKS5 listening on %s", cfg.ListenAddr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}
		go handleSOCKS5(conn, cfg)
	}
}

func handleSOCKS5(conn net.Conn, cfg *Config) {
	defer conn.Close()

	// SOCKS5 handshake
	buf := make([]byte, 256)
	n, err := conn.Read(buf)
	if err != nil || n < 2 || buf[0] != 0x05 {
		return
	}
	conn.Write([]byte{0x05, 0x00})

	// Read request
	n, err = conn.Read(buf)
	if err != nil || n < 7 || buf[0] != 0x05 || buf[1] != 0x01 {
		return
	}

	var target string
	switch buf[3] {
	case 0x01: // IPv4
		target = fmt.Sprintf("%d.%d.%d.%d:%d", buf[4], buf[5], buf[6], buf[7], binary.BigEndian.Uint16(buf[8:10]))
	case 0x03: // Domain
		domainLen := int(buf[4])
		target = fmt.Sprintf("%s:%d", buf[5:5+domainLen], binary.BigEndian.Uint16(buf[5+domainLen:7+domainLen]))
	default:
		conn.Write([]byte{0x05, 0x08, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}

	// QUIC connection
	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"h3"},
		ServerName:         selectSNI(cfg.SNIDomains),
	}

	// Apply uTLS fingerprinting
	utlsConf := &utls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"h3"},
		ServerName:         tlsConf.ServerName,
	}
	_ = utlsConf // Placeholder for future uTLS integration

	quicConf := &quic.Config{
		Allow0RTT:          true,
		EnableDatagrams:    false,
		MaxIdleTimeout:     0,
		KeepAlivePeriod:    0,
	}

	session, err := quic.DialAddr(context.Background(), cfg.ServerAddr, tlsConf, quicConf)
	if err != nil {
		log.Printf("QUIC dial failed: %v", err)
		conn.Write([]byte{0x05, 0x05, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}
	defer session.CloseWithError(0, "")

	stream, err := session.OpenStreamSync(context.Background())
	if err != nil {
		log.Printf("Stream open failed: %v", err)
		conn.Write([]byte{0x05, 0x05, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}
	defer stream.Close()

	// Send auth header
	authHeader := buildAuthHeader(cfg.PSK, target)
	if _, err := stream.Write(authHeader); err != nil {
		log.Printf("Auth header write failed: %v", err)
		return
	}

	// Success response
	conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0})

	// Bidirectional relay
	go func() {
		buf := make([]byte, 32*1024)
		for {
			n, err := conn.Read(buf)
			if err != nil {
				return
			}
			padded := applyPadding(buf[:n])
			if _, err := stream.Write(padded); err != nil {
				return
			}
		}
	}()

	io.Copy(conn, stream)
}
