package main

import (
	"flag"
	"log"
)

func main() {
	mode := flag.String("mode", "client", "Launch mode: server or client")
	serverAddr := flag.String("server", "185.208.172.162:443", "Server address")
	clientLocalAddr := flag.String("local", "0.0.0.0:1080", "Client local SOCKS5 address")
	psk := flag.String("psk", "SUPER_SECURE_SPACE_PASS", "Pre-shared key")
	flag.Parse()

	// Generate dynamic XOR key
	xorKey := GenerateXORKey()

	cfg := Config{
		ServerAddr:   *serverAddr,
		LocalAddr:    *localAddr,
		PSK:          *psk,
		SNIDomains:   DefaultSNIDomains,
		HealthCheck:  HealthCheckInt,
		ReconnectMax: MaxReconnect,
		XORKey:       xorKey,
	}

	log.Printf("[SpaceShit] 🚀 Ferrari Tunnel v2.0")
	log.Printf("[SpaceShit] Mode: %s", *mode)

	if *mode == "server" {
		StartSpaceServer(cfg)
	} else {
		StartSpaceClient(cfg)
	}
}
