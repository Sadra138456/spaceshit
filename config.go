package main

import "time"

type Config struct {
	ServerAddr string
	LocalAddr  string
	PSK        string
	SNI        string
	
	// Anti-DPI features
	FragmentSize    int
	FragmentDelay   time.Duration
	TimingJitter    time.Duration
	MinPaddingSize  int
	MaxPaddingSize  int
	EnableMux       bool
	MuxKeepAlive    time.Duration
	ReconnectDelay  time.Duration
	
	// TLS
	CertFile string
	KeyFile  string
	ALPN     []string
}

func DefaultConfig() *Config {
	return &Config{
		ServerAddr:      "185.208.172.162:443",
		LocalAddr:       "0.0.0.0:1080",
		PSK:             "SUPER_SECURE_SPACE_KEY_2025",
		SNI:             "www.speedtest.net",
		FragmentSize:    100,
		FragmentDelay:   5 * time.Millisecond,
		TimingJitter:    10 * time.Millisecond,
		MinPaddingSize:  64,
		MaxPaddingSize:  256,
		EnableMux:       true,
		MuxKeepAlive:    30 * time.Second,
		ReconnectDelay:  5 * time.Second,
		CertFile:        "server.crt",
		KeyFile:         "server.key",
		ALPN:            []string{"h2", "http/1.1"},
	}
}

const (
	AuthTimeout = 4 * time.Second
)
