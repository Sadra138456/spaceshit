package main

import (
	"fmt"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: spaceshit [server|client]")
		return
	}

	mode := os.Args[1]

	if mode == "server" {
		cfg := &Config{
			ServerAddr: "0.0.0.0:443",
			PSK:        "

7f8c2e41b9a5d3f0e812c6a49d7b5f2a1c0e3d5b8a9f4c2e7d1b6a0f3e2d5c8b",
			CertFile:   "server.crt",
			KeyFile:    "server.key",
		}
		RunServer(cfg)
	} else if mode == "client" {
		cfg := &Config{
			ServerAddr: "185.208.172.162:443",
			LocalAddr:  "0.0.0.0:1080",  // ✅ تغییر از 127.0.0.1 به 0.0.0.0
			PSK:        "

7f8c2e41b9a5d3f0e812c6a49d7b5f2a1c0e3d5b8a9f4c2e7d1b6a0f3e2d5c8b",
		}
		RunClient(cfg)
	} else {
		fmt.Println("Invalid mode. Use 'server' or 'client'")
	}
}
