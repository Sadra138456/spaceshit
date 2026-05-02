package main

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"log"
	"os"
)

type Config struct {
	Mode          string   `json:"mode"`
	ListenAddr    string   `json:"listen_addr"`
	ServerAddr    string   `json:"server_addr,omitempty"`
	PSK           string   `json:"psk"`
	SNIDomains    []string `json:"sni_domains"`
	EnableFallback bool    `json:"enable_fallback"`
}

func main() {
	configPath := flag.String("config", "config.json", "Path to config file")
	flag.Parse()

	data, err := os.ReadFile(*configPath)
	if err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("Failed to parse config: %v", err)
	}

	// Validate PSK
	pskBytes, err := hex.DecodeString(cfg.PSK)
	if err != nil || len(pskBytes) != 32 {
		log.Fatalf("PSK must be 64-char hex (32 bytes)")
	}

	if cfg.Mode == "client" {
		runClient(&cfg)
	} else if cfg.Mode == "server" {
		runServer(&cfg)
	} else {
		log.Fatalf("Invalid mode: %s", cfg.Mode)
	}
}
