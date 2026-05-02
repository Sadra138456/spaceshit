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

// StartSpaceClient connects to the mothership with SOCKS5 support
func StartSpaceClient(cfg Config) {
	// 1. Establish persistent tunnel connection
	var session *yamux.Session
	var reconnectAttempts int

	connectToServer := func() error {
		// TCP connection
		tcpConn, err := net.DialTimeout("tcp", cfg.ServerAddr, 10*time.Second)
		if err != nil {
			return fmt.Errorf("TCP dial failed: %w", err)
		}

		// uTLS with random browser fingerprint
		uConn := utls.UClient(tcpConn, GetUTLSConfig(cfg.SNI), utls.HelloChrome_Auto)
		if err := uConn.Handshake(); err != nil {
			tcpConn.Close()
			return fmt.Errorf("TLS handshake failed: %w", err)
		}

		// Send PSK + padding for DPI bypass
		if _, err := uConn.Write([]byte(cfg.PSK)); err != nil {
			uConn.Close()
			return fmt.Errorf("PSK send failed: %w", err)
		}
		if _, err := uConn.Write(GeneratePadding()); err != nil {
			uConn.Close()
			return fmt.Errorf("padding send failed: %w", err)
		}

		// Start yamux session
		newSession, err := yamux.Client(uConn, nil)
		if err != nil {
			uConn.Close()
			return fmt.Errorf("yamux session failed: %w", err)
		}

		session = newSession
		reconnectAttempts = 0
		log.Printf("[Ferrari] 🏎️  Tunnel established to %s", cfg.ServerAddr)
		return nil
	}

	// Initial connection
	if err := connectToServer(); err != nil {
		log.Fatal("[Ferrari] ❌ Initial connection failed: ", err)
	}

	// Auto-reconnect goroutine
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if session.IsClosed() {
				log.Printf("[Ferrari] 🔄 Reconnecting... (attempt %d)", reconnectAttempts+1)
				backoff := time.Duration(1<<reconnectAttempts) * time.Second
				if backoff > 60*time.Second {
					backoff = 60 * time.Second
				}
				time.Sleep(backoff)
				if err := connectToServer(); err != nil {
					log.Printf("[Ferrari] ⚠️  Reconnect failed: %v", err)
					reconnectAttempts++
				}
			}
		}
	}()

	// 2. Start SOCKS5 proxy listener
	ln, err := net.Listen("tcp", cfg.LocalAddr)
	if err != nil {
		log.Fatal("[Ferrari] ❌ Failed to start SOCKS5 listener: ", err)
	}
	log.Printf("[Ferrari] 🚀 SOCKS5 proxy ready at %s", cfg.LocalAddr)

	for {
		localConn, err := ln.Accept()
		if err != nil {
			log.Printf("[Ferrari] ⚠️  Accept error: %v", err)
			continue
		}
		go handleSOCKS5(localConn, session)
	}
}

// handleSOCKS5 implements SOCKS5 protocol (RFC 1928)
func handleSOCKS5(client net.Conn, session *yamux.Session) {
	defer client.Close()

	// Set timeout for handshake
	client.SetDeadline(time.Now().Add(10 * time.Second))

	// Step 1: Version identification/method selection
	// Client sends: [VER(1) | NMETHODS(1) | METHODS(1-255)]
	buf := make([]byte, 257)
	n, err := client.Read(buf)
	if err != nil || n < 2 {
		log.Printf("[SOCKS5] ❌ Handshake read error: %v", err)
		return
	}

	version := buf[0]
	if version != 0x05 {
		log.Printf("[SOCKS5] ❌ Unsupported version: %d", version)
		return
	}

	// Step 2: Server chooses method (0x00 = NO AUTH)
	// Server sends: [VER(1) | METHOD(1)]
	if _, err := client.Write([]byte{0x05, 0x00}); err != nil {
		log.Printf("[SOCKS5] ❌ Method response error: %v", err)
		return
	}

	// Step 3: Request
	// Client sends: [VER(1) | CMD(1) | RSV(1) | ATYP(1) | DST.ADDR(var) | DST.PORT(2)]
	n, err = client.Read(buf)
	if err != nil || n < 7 {
		log.Printf("[SOCKS5] ❌ Request read error: %v", err)
		return
	}

	if buf[0] != 0x05 {
		log.Printf("[SOCKS5] ❌ Invalid request version: %d", buf[0])
		return
	}

	cmd := buf[1]
	if cmd != 0x01 { // Only support CONNECT
		client.Write([]byte{0x05, 0x07, 0x00, 0x01, 0, 0, 0, 0, 0, 0}) // Command not supported
		log.Printf("[SOCKS5] ❌ Unsupported command: %d", cmd)
		return
	}

	atyp := buf[3]
	var host string
	var port uint16
	var addrStart int

	switch atyp {
	case 0x01: // IPv4
		if n < 10 {
			client.Write([]byte{0x05, 0x01, 0x00, 0x01, 0, 0, 0, 0, 0, 0}) // General failure
			return
		}
		host = fmt.Sprintf("%d.%d.%d.%d", buf[4], buf[5], buf[6], buf[7])
		port = binary.BigEndian.Uint16(buf[8:10])
		addrStart = 4

	case 0x03: // Domain name
		if n < 5 {
			client.Write([]byte{0x05, 0x01, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
			return
		}
		domainLen := int(buf[4])
		if n < 5+domainLen+2 {
			client.Write([]byte{0x05, 0x01, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
			return
		}
		host = string(buf[5 : 5+domainLen])
		port = binary.BigEndian.Uint16(buf[5+domainLen : 5+domainLen+2])
		addrStart = 4

	case 0x04: // IPv6
		if n < 22 {
			client.Write([]byte{0x05, 0x01, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
			return
		}
		host = net.IP(buf[4:20]).String()
		port = binary.BigEndian.Uint16(buf[20:22])
		addrStart = 4

	default:
		client.Write([]byte{0x05, 0x08, 0x00, 0x01, 0, 0, 0, 0, 0, 0}) // Address type not supported
		log.Printf("[SOCKS5] ❌ Unsupported address type: %d", atyp)
		return
	}

	target := fmt.Sprintf("%s:%d", host, port)
	log.Printf("[SOCKS5] 🎯 CONNECT %s", target)

	// Remove handshake timeout
	client.SetDeadline(time.Time{})

	// Step 4: Open stream through Ferrari tunnel
	stream, err := session.Open()
	if err != nil {
		client.Write([]byte{0x05, 0x01, 0x00, 0x01, 0, 0, 0, 0, 0, 0}) // General failure
		log.Printf("[SOCKS5] ❌ Tunnel stream error: %v", err)
		return
	}
	defer stream.Close()

	// Send target address to server through tunnel
	if _, err := stream.Write([]byte(target + "\n")); err != nil {
		client.Write([]byte{0x05, 0x01, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		log.Printf("[SOCKS5] ❌ Target send error: %v", err)
		return
	}

	// Step 5: Send success response
	// [VER(1) | REP(1) | RSV(1) | ATYP(1) | BND.ADDR(var) | BND.PORT(2)]
	response := []byte{0x05, 0x00, 0x00}
	response = append(response, buf[3:addrStart+1]...) // Copy ATYP + address
	if atyp == 0x03 {
		domainLen := int(buf[4])
		response = append(response, buf[4:5+domainLen+2]...)
	} else if atyp == 0x01 {
		response = append(response, buf[4:10]...)
	} else if atyp == 0x04 {
		response = append(response, buf[4:22]...)
	}

	if _, err := client.Write(response); err != nil {
		log.Printf("[SOCKS5] ❌ Response write error: %v", err)
		return
	}

	// Step 6: Relay data (zero-copy I/O)
	errChan := make(chan error, 2)

	// Client → Tunnel
	go func() {
		_, err := io.Copy(stream, client)
		errChan <- err
	}()

	// Tunnel → Client
	go func() {
		_, err := io.Copy(client, stream)
		errChan <- err
	}()

	// Wait for either direction to close
	<-errChan
	log.Printf("[SOCKS5] ✅ Connection closed: %s", target)
}
