package main

import (
	"io"
	"log"
	"net"
	"github.com/hashicorp/yamux"
	utls "github.com/refraction-networking/utls"
)

// StartSpaceClient connects to the mothership
func StartSpaceClient(cfg Config) {
	// 1. Establish TCP connection
	tcpConn, err := net.Dial("tcp", cfg.ServerAddr)
	if err != nil {
		log.Fatal("Launch failed: ", err)
	}

	// 2. Wrap with uTLS (Chrome Fingerprint)
	uConn := utls.UClient(tcpConn, GetUTLSConfig(cfg.SNI), utls.HelloChrome_Auto)
	uConn.Handshake()

	// 3. Send PSK + Junk Data to bypass DPI
	uConn.Write([]byte(cfg.PSK))
	uConn.Write(GeneratePadding())

	// 4. Start Multiplexing Session
	session, _ := yamux.Client(uConn, nil)

	// 5. Local Listener (Entry point for your apps)
	ln, _ := net.Listen("tcp", cfg.LocalAddr)
	log.Printf("[SpaceShit] Warp Portal open at %s", cfg.LocalAddr)

	for {
		localConn, _ := ln.Accept()
		go func(lc net.Conn) {
			stream, _ := session.Open() // Multiplexing: no new handshake needed
			go io.Copy(stream, lc)
			io.Copy(lc, stream)
		}(localConn)
	}
}
