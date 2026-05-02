package main

import (
	"crypto/tls"
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

func StartSpaceClient(cfg Config) {
	var session *yamux.Session
	var sessionMu sync.RWMutex

	// Auto-reconnect loop
	go func() {
		backoff := time.Second
		for {
			log.Printf("[Ferrari] 🔌 Connecting to %s...", cfg.ServerAddr)
			
			s, err := connectToServer(cfg)
			if err != nil {
				log.Printf("[Ferrari] ❌ Connection failed: %v (retry in %v)", err, backoff)
				time.Sleep(backoff)
				backoff = min(backoff*2, 30*time.Second)
				continue
			}

			sessionMu.Lock()
			session = s
			sessionMu.Unlock()

			backoff = time.Second
			log.Printf("[Ferrari] ✅ Tunnel established!")

			<-session.CloseChan()
			log.Printf("[Ferrari] ⚠️  Tunnel disconnected, reconnecting...")
		}
	}()

	time.Sleep(2 * time.Second)

	ln, err := net.Listen("tcp", cfg.LocalAddr)
	if err != nil {
		log.Fatal("[Ferrari] ❌ Listen error: ", err)
	}

	log.Printf("[Ferrari] 🏎️  SOCKS5 Proxy ready at %s", cfg.LocalAddr)
	log.Printf("[Ferrari] 📱 Connect from any device: socks5://%s", cfg.LocalAddr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("[Ferrari] ⚠️  Accept error: %v", err)
			continue
		}

		sessionMu.RLock()
		currentSession := session
		sessionMu.RUnlock()

		if currentSession == nil || currentSession.IsClosed() {
			conn.Close()
			continue
		}

		go handleSOCKS5(conn, currentSession)
	}
}

func connectToServer(cfg Config) (*yamux.Session, error) {
	conn, err := net.DialTimeout("tcp", cfg.ServerAddr, 10*time.Second)
	if err != nil {
		return nil, err
	}

	uConn := utls.UClient(conn, &utls.Config{
		ServerName:         cfg.SNI,
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS13,
	}, utls.HelloChrome_Auto)

	if err := uConn.Handshake(); err != nil {
		conn.Close()
		return nil, err
	}

	if _, err := uConn.Write([]byte(cfg.PSK)); err != nil {
		uConn.Close()
		return nil, err
	}

	padding := GeneratePadding()
	if _, err := uConn.Write(padding); err != nil {
		uConn.Close()
		return nil, err
	}

	session, err := yamux.Client(uConn, nil)
	if err != nil {
		uConn.Close()
		return nil, err
	}

	return session, nil
}

func handleSOCKS5(clientConn net.Conn, session *yamux.Session) {
	defer clientConn.Close()

	buf := make([]byte, 257)
	n, err := clientConn.Read(buf)
	if err != nil || n < 2 {
		return
	}

	ver, nmethods := buf[0], buf[1]
	if ver != 0x05 || n < 2+int(nmethods) {
		return
	}

	clientConn.Write([]byte{0x05, 0x00})

	n, err = clientConn.Read(buf)
	if err != nil || n < 7 {
		return
	}

	ver, cmd, atyp := buf[0], buf[1], buf[3]
	if ver != 0x05 || cmd != 0x01 {
		clientConn.Write([]byte{0x05, 0x07, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}

	var host string
	var port uint16

	switch atyp {
	case 0x01: // IPv4
		if n < 10 {
			return
		}
		host = fmt.Sprintf("%d.%d.%d.%d", buf[4], buf[5], buf[6], buf[7])
		port = binary.BigEndian.Uint16(buf[8:10])

	case 0x03: // Domain
		addrLen := int(buf[4])
		if n < 5+addrLen+2 {
			return
		}
		host = string(buf[5 : 5+addrLen])
		port = binary.BigEndian.Uint16(buf[5+addrLen : 7+addrLen])

	case 0x04: // IPv6
		if n < 22 {
			return
		}
		host = net.IP(buf[4:20]).String()
		port = binary.BigEndian.Uint16(buf[20:22])

	default:
		clientConn.Write([]byte{0x05, 0x08, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}

	targetAddr := fmt.Sprintf("%s:%d", host, port)

	stream, err := session.Open()
	if err != nil {
		clientConn.Write([]byte{0x05, 0x01, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}
	defer stream.Close()

	stream.Write([]byte(targetAddr + "\n"))

	clientConn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0})

	log.Printf("[Ferrari] 🎯 %s → %s", clientConn.RemoteAddr(), targetAddr)

	errChan := make(chan error, 2)

	go func() {
		buf := bufferPool.Get().(*[]byte)
		defer bufferPool.Put(buf)
		_, err := io.CopyBuffer(stream, clientConn, *buf)
		errChan <- err
	}()

	go func() {
		buf := bufferPool.Get().(*[]byte)
		defer bufferPool.Put(buf)
		_, err := io.CopyBuffer(clientConn, stream, *buf)
		errChan <- err
	}()

	<-errChan
}

func min(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
