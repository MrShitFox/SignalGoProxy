package main

import (
	"log"
	"signalgoproxy/internal/config"
	"signalgoproxy/internal/server"
)

func main() {
	// Set a prefix for logs to include file and line number.
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// 1. Create the configuration
	cfg := config.New()
	log.Printf("Configuration loaded for domain '%s' with stealth mode '%s'", cfg.Domain, cfg.StealthMode)

	// 2. Create the server
	srv := server.New(cfg)

	// 3. Start the server (this is a blocking operation)
	srv.Start()
}