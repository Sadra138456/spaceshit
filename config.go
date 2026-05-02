package main

// Config defines the space-grade parameters for our tunnel
type Config struct {
	ServerAddr string `json:"server_addr"`
	LocalAddr  string `json:"local_addr"`
	PSK        string `json:"psk"` // Pre-Shared Key for stealth authentication
	SNI        string `json:"sni"` // Server Name Indication (e.g., speedtest.net)
}

const (
	MaxJunkSize = 256              // Maximum bytes for anti-DPI padding
	AuthTimeout = 4 * 1000000000   // 4 seconds timeout for handshake
)
