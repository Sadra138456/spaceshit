package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
)

type Config struct {
	Mode           string   `json:"mode"`            // "client" or "server"
	ListenAddr     string   `json:"listen_addr"`     // Local address to listen
	ServerAddr     string   `json:"server_addr"`     // Remote server address (for client)
	PSK            string   `json:"psk"`             // 64 hex characters
	SNIDomains     []string `json:"sni_domains"`     // List of domains for camouflage
	EnableFallback bool     `json:"enable_fallback"` // Hide behind a real site
}

func main() {
	configPath := flag.String("config", "config.json", "path to config file")
	flag.Parse()

	file, err := os.Open(*configPath)
	if err != nil {
		log.Fatalf("Failed to open config: %v", err)
	}
	defer file.Close()

	var cfg Config
	if err := json.NewDecoder(file).Decode(&cfg); err != nil {
		log.Fatalf("Failed to decode config: %v", err)
	}

	if len(cfg.PSK) != 64 {
		log.Fatal("PSK must be 64 hex characters (32 bytes)")
	}

	switch cfg.Mode {
	case "client":
		log.Printf("Starting Client mode on %s -> Tunneling to %s", cfg.ListenAddr, cfg.ServerAddr)
		runClient(&cfg) // اضافه کردن & برای رفع ارور Pointer
	case "server":
		log.Printf("Starting Server mode on %s (Black-Hole active)", cfg.ListenAddr)
		runServer(&cfg)
	default:
		log.Fatal("Invalid mode! Use 'client' or 'server'")
	}
}
