package main

import (
	"fmt"
	"log"
	"time"
)

func main() {
	cfg := &Config{
		ServerAddr: "0.0.0.0:443",
		LocalAddr:  "0.0.0.0:1080",
		PSK:        "7f8c2e41b9a5d3f0e812c6a49d7b5f2a1c0e3d5b8a9f4c2e7d1b6a0f3e2d5c8b",
		CertFile:   "server.crt",
		KeyFile:    "server.key",
	}

	// ✅ Server در goroutine جداگانه
	go func() {
		log.Println("[Ferrari] 🏎️  Starting server...")
		RunServer(cfg)
	}()

	// ✅ کمی صبر می‌کنیم تا server بالا بیاد
	time.Sleep(2 * time.Second)

	// ✅ Client در همین thread اصلی
	log.Println("[Ferrari] 🚀 Starting client...")
	clientCfg := &Config{
		ServerAddr: "127.0.0.1:443",  // به خودش وصل می‌شه
		LocalAddr:  "0.0.0.0:1080",
		PSK:        "7f8c2e41b9a5d3f0e812c6a49d7b5f2a1c0e3d5b8a9f4c2e7d1b6a0f3e2d5c8b",
	}
	RunClient(clientCfg)
}
