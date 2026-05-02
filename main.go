package main

import (
	"flag"
	"os"
)

func main() {
	mode := flag.String("mode", "client", "Launch mode: server or client")
	flag.Parse()

	// Pre-defined high-performance configuration
	cfg := Config{
		ServerAddr: "YOUR_IP:443", // Stealth over HTTPS port
		LocalAddr:  "127.0.0.1:1080",
		PSK:        "SUPER_SECURE_SPACE_PASS",
		SNI:        "www.google.com", // Mimic legitimate traffic
	}

	if *mode == "server" {
		StartSpaceServer(cfg)
	} else {
		StartSpaceClient(cfg)
	}
}
