package main

import "time"

type Config struct {
	ServerAddr string
	LocalAddr  string
	PSK        string
	SNI        string
	CertFile   string
	KeyFile    string
}

const (
	MaxJunkSize = 4096
	AuthTimeout = 5 * time.Second
)
