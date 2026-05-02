package main

import (
	"io"
	"log"
	"net"
	"time"

	"github.com/hashicorp/yamux"
	utls "github.com/refraction-networking/utls"
)

// StartSpaceClient با Ferrari mode
func StartSpaceClient(cfg Config) {
	reconnectCount := 0

	for {
		err := runClient(cfg)
		if err != nil {
			reconnectCount++
			if reconnectCount > cfg.ReconnectMax {
				log.Fatal("[SpaceShit] Max reconnect reached. Abort.")
			}

			backoff := ReconnectBackoff * time.Duration(reconnectCount)
			log.Printf("[SpaceShit] Connection lost. Retry in %v... (%d/%d)",
				backoff, reconnectCount, cfg.ReconnectMax)
			time.Sleep(backoff)
			continue
		}
		reconnectCount = 0
	}
}

func runClient(cfg Config) error {
	// 1. TCP connection
	tcpConn, err := net.DialTimeout("tcp", cfg.ServerAddr, 10*time.Second)
	if err != nil {
		return err
	}
	defer tcpConn.Close()

	// Enable TCP tuning
	if tcpConn, ok := tcpConn.(*net.TCPConn); ok {
		tcpConn.SetNoDelay(true)
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(30 * time.Second)
	}

	// 2. Random SNI + uTLS fingerprint
	sni := GetRandomSNI(cfg.SNIDomains)
	fingerprint := GetRandomFingerprint()
	uConn := utls.UClient(tcpConn, GetUTLSConfig(sni), fingerprint)

	if err := uConn.Handshake(); err != nil {
		return err
	}

	// 3. Send PSK + XOR obfuscated
	pskData := []byte(cfg.PSK)
	XORObfuscate(pskData, cfg.XORKey)
	if _, err := uConn.Write(pskData); err != nil {
		return err
	}

	// 4. Send padding (traffic shaping)
	padding := GeneratePadding()
	uConn.Write(padding)

	// 5. Yamux session
	session, err := yamux.Client(uConn, nil)
	if err != nil {
		return err
	}
	defer session.Close()

	// 6. Local listener
	ln, err := net.Listen("tcp", cfg.LocalAddr)
	if err != nil {
		return err
	}
	defer ln.Close()

	log.Printf("[SpaceShit] 🏎️ Ferrari Portal open at %s (SNI: %s)", cfg.LocalAddr, sni)

	// Health check goroutine
	go healthCheck(session, cfg.HealthCheck)

	// Accept connections
	for {
		localConn, err := ln.Accept()
		if err != nil {
			return err
		}

		go handleLocalConn(localConn, session)
	}
}

func handleLocalConn(localConn net.Conn, session *yamux.Session) {
	defer localConn.Close()

	stream, err := session.Open()
	if err != nil {
		log.Printf("[SpaceShit] Stream open failed: %v", err)
		return
	}
	defer stream.Close()

	// Zero-copy bidirectional relay
	done := make(chan struct{}, 2)

	go func() {
		buf := GetBuffer()
		defer PutBuffer(buf)
		io.CopyBuffer(stream, localConn, *buf)
		done <- struct{}{}
	}()

	go func() {
		buf := GetBuffer()
		defer PutBuffer(buf)
		io.CopyBuffer(localConn, stream, *buf)
		done <- struct{}{}
	}()

	<-done
}

func healthCheck(session *yamux.Session, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		if session.IsClosed() {
			log.Println("[SpaceShit] Session dead. Reconnecting...")
			return
		}
	}
}
