package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
)

// Config represents the unified configuration for both client and server
type Config struct {
	Mode         string   `json:"mode"`          // "client" or "server"
	ListenAddr   string   `json:"listen_addr"`   // Client: SOCKS5 addr, Server: QUIC listen addr
	ServerAddr   string   `json:"server_addr"`   // Client only: remote server address
	PSK          string   `json:"psk"`           // Pre-Shared Key (32-byte hex string)
	SNIDomains   []string `json:"sni_domains"`   // List of SNI domains for spoofing
	EnableFallback bool   `json:"enable_fallback"` // Enable WebSocket/TLS fallback
}

func main() {
	configPath := flag.String("config", "config.json", "Path to configuration file")
	flag.Parse()

	// Load configuration
	data, err := os.ReadFile(*configPath)
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("Failed to parse config: %v", err)
	}

	// Validate PSK length (must be 64 hex chars = 32 bytes)
	if len(cfg.PSK) != 64 {
		log.Fatalf("PSK must be 64 hex characters (32 bytes)")
	}

	// Route to client or server
	switch cfg.Mode {
	case "client":
		if cfg.ServerAddr == "" {
			log.Fatal("server_addr is required in client mode")
		}
		fmt.Printf("🚀 SpaceShit Client starting on %s → %s\n", cfg.ListenAddr, cfg.ServerAddr)
		runClient(cfg)
	case "server":
		fmt.Printf("🛸 SpaceShit Server listening on %s\n", cfg.ListenAddr)
		runServer(cfg)
	default:
		log.Fatalf("Invalid mode: %s (must be 'client' or 'server')", cfg.Mode)
	}
}
