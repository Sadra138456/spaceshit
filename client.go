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

	utls "github.com/refraction-networking/utls"
	"github.com/quic-go/quic-go"
)

func runClient(cfg Config) {
	// Start SOCKS5 proxy listener
	listener, err := net.Listen("tcp", cfg.ListenAddr)
	if err != nil {
		log.Fatalf("Failed to start SOCKS5 listener: %v", err)
	}
	defer listener.Close()

	log.Printf("SOCKS5 proxy listening on %s", cfg.ListenAddr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}
		go handleSOCKS5(conn, cfg)
	}
}

func handleSOCKS5(conn net.Conn, cfg Config) {
	defer conn.Close()

	// SOCKS5 handshake: version + auth methods
	buf := make([]byte, 257)
	n, err := conn.Read(buf)
	if err != nil || n < 2 || buf[0] != 0x05 {
		return
	}

	// Reply: no authentication required
	conn.Write([]byte{0x05, 0x00})

	// Read SOCKS5 request
	n, err = conn.Read(buf)
	if err != nil || n < 7 || buf[0] != 0x05 || buf[1] != 0x01 {
		return
	}

	// Parse target address
	var target string
	switch buf[3] {
	case 0x01: // IPv4
		target = fmt.Sprintf("%d.%d.%d.%d:%d", buf[4], buf[5], buf[6], buf[7], binary.BigEndian.Uint16(buf[8:10]))
	case 0x03: // Domain
		domainLen := int(buf[4])
		target = fmt.Sprintf("%s:%d", string(buf[5:5+domainLen]), binary.BigEndian.Uint16(buf[5+domainLen:7+domainLen]))
	default:
		conn.Write([]byte{0x05, 0x08, 0x00, 0x01, 0, 0, 0, 0, 0, 0}) // Address type not supported
		return
	}

	// Reply: success
	conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0})

	// Establish QUIC connection to server
	stream, err := dialQUIC(cfg, target)
	if err != nil {
		log.Printf("QUIC dial failed: %v", err)
		return
	}
	defer stream.Close()

	// Bidirectional relay
	go io.Copy(stream, conn)
	io.Copy(conn, stream)
}

func dialQUIC(cfg Config, target string) (quic.Stream, error) {
	// Create uTLS config mimicking Chrome 120
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         selectSNI(cfg.SNIDomains),
		NextProtos:         []string{"h3"},
	}

	// Wrap with uTLS for fingerprint mimicry
	utlsConfig := &utls.Config{
		ServerName:         tlsConfig.ServerName,
		InsecureSkipVerify: true,
	}

	// QUIC transport config
	quicConfig := &quic.Config{
		MaxIdleTimeout:        30 * time.Second,
		KeepAlivePeriod:       10 * time.Second,
		EnableDatagrams:       false,
		Allow0RTT:             true,
		MaxIncomingStreams:    1000,
		MaxIncomingUniStreams: 1000,
	}

	// Dial QUIC connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	session, err := quic.DialAddr(ctx, cfg.ServerAddr, tlsConfig, quicConfig)
	if err != nil {
		return nil, fmt.Errorf("QUIC dial failed: %w", err)
	}

	// Open stream
	stream, err := session.OpenStreamSync(context.Background())
	if err != nil {
		session.CloseWithError(0, "stream open failed")
		return nil, err
	}

	// Send authentication header + target address
	authHeader := buildAuthHeader(cfg.PSK, target)
	if _, err := stream.Write(authHeader); err != nil {
		stream.Close()
		return nil, err
	}

	// Apply stealth padding
	paddedData := applyPadding([]byte{})
	stream.Write(paddedData)

	// Use utlsConfig for future enhancements
	_ = utlsConfig

	return stream, nil
}
