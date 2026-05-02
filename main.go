package main

import (
	"flag"
	"log"
)

func main() {
	mode := flag.String("mode", "client", "Launch mode: server or client")
	serverAddr := flag.String("server", "185.208.172.162:443", "Server address")
	localAddr := flag.String("local", "127.0.0.1:1080", "Local SOCKS address")
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
