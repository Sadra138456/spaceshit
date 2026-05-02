package main

import (
	"flag"
	"log"
)

func main() {
	mode := flag.String("mode", "client", "Mode: client or server")
	serverAddr := flag.String("server", "127.0.0.1:443", "Server address")
	localAddr := flag.String("local", "0.0.0.0:1080", "Client local SOCKS5 address")
	psk := flag.String("psk", "MySecretKey123", "Pre-shared key")
	sni := flag.String("sni", "www.google.com", "SNI domain for TLS")
	flag.Parse()

	cfg := Config{
		Mode:       *mode,
		ServerAddr: *serverAddr,
		LocalAddr:  *localAddr,
		PSK:        *psk,
		SNI:        *sni,
	}

	if cfg.Mode == "server" {
		StartSpaceServer(cfg)
	} else {
		StartSpaceClient(cfg)
	}
}
