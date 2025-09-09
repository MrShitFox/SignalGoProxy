// Package config handles application configuration.
package config

import (
	"flag"
	"log"
	"os"
	"strings"
)

// StealthMode defines the stealth mode for camouflage.
type StealthMode string

const (
	StealthNone   StealthMode = "none"
	StealthNginx  StealthMode = "nginx"
	StealthApache StealthMode = "apache"
)

// Config stores all configuration parameters.
type Config struct {
	Domain      string
	StealthMode StealthMode
}

// New creates a new configuration by reading flags and environment variables.
func New() *Config {
	cfg := &Config{}

	var domain, stealthMode string

	flag.StringVar(&domain, "domain", "", "Domain for the TLS certificate (required).")
	flag.StringVar(&stealthMode, "stealth-mode", "nginx", "Stealth mode: 'none', 'nginx', or 'apache'.")
	flag.Parse()

	if domain == "" {
		domain = os.Getenv("DOMAIN")
	}
	if stealthMode == "" || stealthMode == "nginx" && os.Getenv("STEALTH_MODE") != "" {
		stealthMode = os.Getenv("STEALTH_MODE")
	}

	if domain == "" {
		log.Fatal("Domain is required. Set it with -domain flag or DOMAIN environment variable.")
	}
	cfg.Domain = domain

	switch strings.ToLower(stealthMode) {
	case "nginx":
		cfg.StealthMode = StealthNginx
	case "apache":
		cfg.StealthMode = StealthApache
	case "none":
		cfg.StealthMode = StealthNone
	default:
		log.Fatalf("Invalid stealth mode: %s. Use 'none', 'nginx', or 'apache'.", stealthMode)
	}

	return cfg
}