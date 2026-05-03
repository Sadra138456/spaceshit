package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"github.com/hashicorp/yamux"
	utls "github.com/refraction-networking/utls"
)

func RunClient(cfg *Config) error {
	for {
		if err := runClientSession(cfg); err != nil {
			log.Printf("❌ Client error: %v. Reconnecting in %v...", err, cfg.ReconnectDelay)
			time.Sleep(cfg.ReconnectDelay)
			continue
		}
	}
}

func runClientSession(cfg *Config) error {
	// Connect to server
	rawConn, err := net.DialTimeout("tcp", cfg.ServerAddr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("dial failed: %v", err)
	}

	// uTLS handshake
	utlsConfig := GetUTLSConfig(cfg.SNI, cfg.ALPN)
	utlsConn := utls.UClient(rawConn, utlsConfig, utls.HelloChrome_Auto)

	if err := utlsConn.Handshake(); err != nil {
		rawConn.Close()
		return fmt.Errorf("TLS handshake failed: %v", err)
	}

	log.Printf("🔐 TLS handshake successful with %s (SNI: %s)", cfg.ServerAddr, cfg.SNI)

	// Send PSK
	if err := SecureWrite(utlsConn, []byte(cfg.PSK), AuthTimeout); err != nil {
		utlsConn.Close()
		return fmt.Errorf("PSK send failed: %v", err)
	}

	// Send padding
	padding := GeneratePadding(cfg.MinPaddingSize, cfg.MaxPaddingSize)
	paddingLen := make([]byte, 2)
	binary.BigEndian.PutUint16(paddingLen, uint16(len(padding)))
	if err := SecureWrite(utlsConn, paddingLen, AuthTimeout); err != nil {
		utlsConn.Close()
		return fmt.Errorf("padding length send failed: %v", err)
	}
	if err := FragmentedWrite(utlsConn, padding, cfg.FragmentSize, cfg.FragmentDelay, cfg.TimingJitter); err != nil {
		utlsConn.Close()
		return fmt.Errorf("padding send failed: %v", err)
	}

	log.Printf("✅ Authentication successful")

	// Setup yamux
	muxConfig := yamux.DefaultConfig()
	muxConfig.KeepAliveInterval = cfg.MuxKeepAlive
	session, err := yamux.Client(utlsConn, muxConfig)
	if err != nil {
		utlsConn.Close()
		return fmt.Errorf("yamux client failed: %v", err)
	}
	defer session.Close()

	log.Printf("🔗 Yamux session established")

	// Start SOCKS5 proxy
	listener, err := net.Listen("tcp", cfg.LocalAddr)
	if err != nil {
		return fmt.Errorf("local listen failed: %v", err)
	}
	defer listener.Close()

	log.Printf("🚀 SOCKS5 Proxy ready at %s", cfg.LocalAddr)

	for {
		localConn, err := listener.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}

		go handleSOCKS5(localConn, session)
	}
}

func handleSOCKS5(local net.Conn, session *yamux.Session) {
	defer local.Close()

	buf := make([]byte, 262)

	// SOCKS5 greeting
	n, err := local.Read(buf)
	if err != nil || n < 2 || buf[0] != 0x05 {
		return
	}

	// No auth required
	local.Write([]byte{0x05, 0x00})

	// SOCKS5 request
	n, err = local.Read(buf)
	if err != nil || n < 7 || buf[0] != 0x05 || buf[1] != 0x01 {
		return
	}

	atyp := buf[3]

	// Open yamux stream
	stream, err := session.Open()
	if err != nil {
		local.Write([]byte{0x05, 0x01, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}
	defer stream.Close()

	// Send address to server
	switch atyp {
	case 0x01: // IPv4
		stream.Write([]byte{0x01})
		stream.Write(buf[4:8])  // IP
		stream.Write(buf[8:10]) // Port

	case 0x03: // Domain
		length := int(buf[4])
		stream.Write([]byte{0x03})
		stream.Write([]byte{byte(length)})
		stream.Write(buf[5 : 5+length])       // Domain
		stream.Write(buf[5+length : 7+length]) // Port

	case 0x04: // IPv6
		stream.Write([]byte{0x04})
		stream.Write(buf[4:20])  // IPv6
		stream.Write(buf[20:22]) // Port

	default:
		local.Write([]byte{0x05, 0x08, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}

	// Wait for server response
	resp := make([]byte, 2)
	if _, err := io.ReadFull(stream, resp); err != nil {
		local.Write([]byte{0x05, 0x01, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}

	if resp[1] != 0x00 {
		local.Write([]byte{0x05, 0x01, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}

	// Send success to client
	local.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0})

	// Bidirectional copy
	BidirectionalCopy(local, stream)
}
