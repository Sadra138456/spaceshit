package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	utls "github.com/refraction-networking/utls"
	"github.com/hashicorp/yamux"
)

func RunClient(cfg *Config) {
	for {
		if err := runClientOnce(cfg); err != nil {
			log.Printf("⚠️  Client error: %v, retrying in 5s...", err)
			time.Sleep(5 * time.Second)
		}
	}
}

func runClientOnce(cfg *Config) error {
	// ✅ Connect to server
	tcpConn, err := net.DialTimeout("tcp", cfg.ServerAddr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("dial failed: %w", err)
	}
	defer tcpConn.Close()

	// ✅ uTLS handshake
	tlsConfig := NewClientTLSConfig(cfg.SNI)
	uConn := utls.UClient(tcpConn, tlsConfig, utls.HelloChrome_Auto)
	if err := uConn.Handshake(); err != nil {
		return fmt.Errorf("TLS handshake failed: %w", err)
	}

	// ✅ Send PSK
	if _, err := uConn.Write([]byte(cfg.PSK)); err != nil {
		return fmt.Errorf("PSK write failed: %w", err)
	}

	// ✅ Send padding
	padding := GeneratePadding(MaxJunkSize)
	padLen := uint16(len(padding))
	if err := binary.Write(uConn, binary.BigEndian, padLen); err != nil {
		return fmt.Errorf("padding length write failed: %w", err)
	}
	if padLen > 0 {
		if _, err := uConn.Write(padding); err != nil {
			return fmt.Errorf("padding write failed: %w", err)
		}
	}

	// ✅ Create yamux session
	session, err := yamux.Client(uConn, nil)
	if err != nil {
		return fmt.Errorf("yamux client failed: %w", err)
	}
	defer session.Close()

	log.Println("✅ Connected to server")

	// ✅ Start SOCKS5 proxy
	ln, err := net.Listen("tcp", cfg.LocalAddr)
	if err != nil {
		return fmt.Errorf("local listen failed: %w", err)
	}
	defer ln.Close()

	log.Printf("🚀 SOCKS5 proxy ready at %s", cfg.LocalAddr)

	for {
		localConn, err := ln.Accept()
		if err != nil {
			return fmt.Errorf("accept failed: %w", err)
		}
		go handleSOCKS5(localConn, session)
	}
}

func handleSOCKS5(localConn net.Conn, session *yamux.Session) {
	defer localConn.Close()

	// ✅ SOCKS5 greeting
	buf := make([]byte, 256)
	n, err := localConn.Read(buf)
	if err != nil || n < 2 {
		return
	}

	// Send: no auth required
	localConn.Write([]byte{0x05, 0x00})

	// ✅ Read request
	n, err = localConn.Read(buf)
	if err != nil || n < 7 {
		return
	}

	if buf[1] != 0x01 { // Only CONNECT
		localConn.Write([]byte{0x05, 0x07, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}

	// ✅ Parse target address
	var targetAddr string
	atyp := buf[3]

	switch atyp {
	case 0x01: // IPv4
		targetAddr = fmt.Sprintf("%d.%d.%d.%d:%d",
			buf[4], buf[5], buf[6], buf[7],
			binary.BigEndian.Uint16(buf[8:10]))
	case 0x03: // Domain
		domainLen := int(buf[4])
		targetAddr = fmt.Sprintf("%s:%d",
			string(buf[5:5+domainLen]),
			binary.BigEndian.Uint16(buf[5+domainLen:7+domainLen]))
	case 0x04: // IPv6
		targetAddr = fmt.Sprintf("[%x:%x:%x:%x:%x:%x:%x:%x]:%d",
			binary.BigEndian.Uint16(buf[4:6]),
			binary.BigEndian.Uint16(buf[6:8]),
			binary.BigEndian.Uint16(buf[8:10]),
			binary.BigEndian.Uint16(buf[10:12]),
			binary.BigEndian.Uint16(buf[12:14]),
			binary.BigEndian.Uint16(buf[14:16]),
			binary.BigEndian.Uint16(buf[16:18]),
			binary.BigEndian.Uint16(buf[18:20]),
			binary.BigEndian.Uint16(buf[20:22]))
	default:
		localConn.Write([]byte{0x05, 0x08, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}

	// ✅ Open stream
	stream, err := session.OpenStream()
	if err != nil {
		localConn.Write([]byte{0x05, 0x01, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}
	defer stream.Close()

	// ✅ Send target address to server
	addrBytes := []byte(targetAddr)
	addrLen := uint16(len(addrBytes))
	binary.Write(stream, binary.BigEndian, addrLen)
	stream.Write(addrBytes)

	// ✅ SOCKS5 success response
	localConn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0})

	// ✅ Bidirectional copy
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		io.Copy(stream, localConn)
	}()

	go func() {
		defer wg.Done()
		io.Copy(localConn, stream)
	}()

	wg.Wait()
}
