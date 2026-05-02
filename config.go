package main

import (
	"time"
)

// Config defines Ferrari-grade tunnel parameters
type Config struct {
	ServerAddr   string        `json:"server_addr"`
	LocalAddr    string        `json:"local_addr"`
	PSK          string        `json:"psk"`
	SNIDomains   []string      `json:"sni_domains"`
	HealthCheck  time.Duration `json:"health_check"`
	ReconnectMax int           `json:"reconnect_max"`
	XORKey       []byte        `json:"-"` // Dynamic XOR key
}

const (
	MaxJunkSize      = 256
	AuthTimeout      = 4 * time.Second
	BufferSize       = 32 * 1024 // 32KB buffers
	HealthCheckInt   = 30 * time.Second
	ReconnectBackoff = 2 * time.Second
	MaxReconnect     = 10
)

// Fake domain fronting pool
var DefaultSNIDomains = []string{
	"www.google.com",
	"www.cloudflare.com",
	"www.microsoft.com",
	"www.apple.com",
}

// User-Agent pool for HTTP mimicry
var UserAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) Safari/605.1.15",
	"Mozilla/5.0 (X11; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/121.0",
}
