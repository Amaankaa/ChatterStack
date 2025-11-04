package main

import (
	"flag"
	"log"
	"os"

	"chatterstack/internal/config"
)

func main() {
	mode := flag.String("mode", "api", "service mode: api or websocket")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config load error: %v", err)
	}

	log.Printf("starting ChatterStack in %s mode on %s", *mode, cfg.HTTP.Address())

	switch *mode {
	case "api":
		log.Println("API server bootstrap incomplete; wire handlers in internal/delivery/http")
	case "websocket":
		log.Println("WebSocket server bootstrap incomplete; implement hub startup in internal/delivery/websocket")
	default:
		log.Printf("unknown mode %q", *mode)
		os.Exit(1)
	}
}
