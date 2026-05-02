package main

import (
	"crypto/tls"
	"io"
	"log"
	"net"
	"time"
	"github.com/hashicorp/yamux"
)

// StartSpaceServer launches the stealth listener
func StartSpaceServer(cfg Config) {
	cert, err := tls.LoadX509KeyPair("server.crt", "server.key")
	if err != nil {
		log.Fatal("Certificate error: ", err)
	}

	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
	}

	ln, _ := net.Listen("tcp", cfg.ServerAddr)
	log.Printf("[SpaceShit] Mission Control started on %s", cfg.ServerAddr)

	for {
		conn, _ := ln.Accept()
		go handleWarpCore(conn, tlsCfg, cfg.PSK)
	}
}

func handleWarpCore(conn net.Conn, tlsCfg *tls.Config, psk string) {
	tlsConn := tls.Server(conn, tlsCfg)
	tlsConn.SetReadDeadline(time.Now().Add(time.Duration(AuthTimeout)))

	// Validate PSK (Silent Authentication)
	authBuf := make([]byte, len(psk))
	if _, err := io.ReadFull(tlsConn, authBuf); err != nil || string(authBuf) != psk {
		tlsConn.Close() // Drop connection if PSK is invalid
		return
	}

	// Initialize Yamux Multiplexing for high-speed streams
	session, _ := yamux.Server(tlsConn, nil)
	for {
		stream, err := session.Accept()
		if err != nil {
			return
		}
		go func(st net.Conn) {
			defer st.Close()
			// Forwarding to local backend or proxy (e.g. SOCKS5 server)
			target, _ := net.Dial("tcp", "127.0.0.1:8080") 
			go io.Copy(target, st)
			io.Copy(st, target)
		}(stream)
	}
}
