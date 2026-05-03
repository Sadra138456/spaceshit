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

		go handle
