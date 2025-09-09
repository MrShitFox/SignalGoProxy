// Package config handles application configuration.
package config

import (
	"flag"
	"log"
	"net/url"
	"os"
	"strings"
)

// StealthMode defines the stealth mode for camouflage.
type StealthMode string

const (
	StealthNone   StealthMode = "none"
	StealthNginx  StealthMode = "nginx"
	StealthApache StealthMode = "apache"
	StealthProxy  StealthMode = "proxy"
)

// Config stores all configuration parameters.
type Config struct {
	Domain      string
	StealthMode StealthMode
	ProxyURL    string
}

// New creates a new configuration by reading flags and environment variables.
func New() *Config {
	cfg := &Config{}

	var domain, stealthMode, proxyURL string

	flag.StringVar(&domain, "domain", "", "Domain for the TLS certificate (required).")
	flag.StringVar(&stealthMode, "stealth-mode", "nginx", "Stealth mode: 'none', 'nginx', 'apache', or 'proxy'.")
	flag.StringVar(&proxyURL, "proxy-url", "", "Proxy URL for 'proxy' stealth mode.")
	flag.Parse()

	if domain == "" {
		domain = os.Getenv("DOMAIN")
	}
	if stealthMode == "" || stealthMode == "nginx" && os.Getenv("STEALTH_MODE") != "" {
		stealthMode = os.Getenv("STEALTH_MODE")
	}
	if proxyURL == "" {
		proxyURL = os.Getenv("PROXY_URL")
	}

	if domain == "" {
		log.Fatal("Domain is required. Set it with -domain flag or DOMAIN environment variable.")
	}
	cfg.Domain = domain
	cfg.ProxyURL = proxyURL

	switch strings.ToLower(stealthMode) {
	case "nginx":
		cfg.StealthMode = StealthNginx
	case "apache":
		cfg.StealthMode = StealthApache
	case "proxy":
		cfg.StealthMode = StealthProxy
		if proxyURL == "" {
			log.Fatal("Proxy URL is required for 'proxy' stealth mode. Set it with -proxy-url or PROXY_URL.")
		}
		u, err := url.Parse(proxyURL)
		if err != nil {
			log.Fatalf("Invalid proxy URL: %v", err)
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			log.Fatal("Proxy URL must have a scheme of 'http' or 'https'.")
		}
	case "none":
		cfg.StealthMode = StealthNone
	default:
		log.Fatalf("Invalid stealth mode: %s. Use 'none', 'nginx', 'apache', or 'proxy'.", stealthMode)
	}

	return cfg
}