package main

import (
	"log"
	"os"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: ./spaceshit [server|client|both]")
	}

	mode := os.Args[1]

	cfg := &Config{
		ServerAddr: "0.0.0.0:443",
		LocalAddr:  "0.0.0.0:1080",
		PSK:        "8vN2xK9mP4wQ7jL5tR3yH6nB1sF0dA8c",
		SNI:        "cloudflare.com",
		CertFile:   "server.crt",
		KeyFile:    "server.key",
	}

	switch mode {
	case "server":
		RunServer(cfg)
	case "client":
		RunClient(cfg)
	case "both":
		go RunServer(cfg)
		time.Sleep(2 * time.Second)
		clientCfg := *cfg
		clientCfg.ServerAddr = "127.0.0.1:443"
		RunClient(&clientCfg)
	default:
		log.Fatal("Invalid mode. Use: server, client, or both")
	}
}
