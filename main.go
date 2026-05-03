package main

import (
	"log"
	"os"
)

func main() {

	if len(os.Args) < 2 {
		log.Println("Usage:")
		log.Println("./spaceshit server")
		log.Println("./spaceshit client")
		log.Println("./spaceshit both")
		return
	}

	cfg := DefaultConfig()

	switch os.Args[1] {

	case "server":
		log.Println("Starting server...")
		if err := RunServer(cfg); err != nil {
			log.Fatal(err)
		}

	case "client":
		log.Println("Starting client...")
		if err := RunClient(cfg); err != nil {
			log.Fatal(err)
		}

	case "both":
		log.Println("Starting server + client...")

		go func() {
			if err := RunServer(cfg); err != nil {
				log.Fatal(err)
			}
		}()

		if err := RunClient(cfg); err != nil {
			log.Fatal(err)
		}

	default:
		log.Println("Unknown command")
	}
}
