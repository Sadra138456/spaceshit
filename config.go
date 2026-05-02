package main

import "time"

type Config struct {
	Mode       string
	ServerAddr string
	LocalAddr  string
	PSK        string
	SNI        string
}

const (
	AuthTimeout  = 4 * time.Second
	MaxJunkSize  = 1024
	BufferSize   = 32 * 1024
)
