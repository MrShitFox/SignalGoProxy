// Package config отвечает за конфигурацию приложения.
package config

import (
	"flag"
	"log"
	"os"
	"strings"
)

// StealthMode определяет режим маскировки.
type StealthMode string

const (
	StealthNone   StealthMode = "none"
	StealthNginx  StealthMode = "nginx"
	StealthApache StealthMode = "apache" // <-- ДОБАВЛЯЕМ НОВЫЙ РЕЖИМ
)

// Config хранит все конфигурационные параметры.
type Config struct {
	Domain      string
	StealthMode StealthMode
}

// New создает новую конфигурацию, считывая флаги и переменные окружения.
func New() *Config {
	cfg := &Config{}

	var domain, stealthMode string

	flag.StringVar(&domain, "domain", "", "Domain for the TLS certificate (required).")
	// Изменяем значение по умолчанию, чтобы показать все опции в --help
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

	// --- ОБНОВЛЯЕМ ЛОГИКУ ВЫБОРА РЕЖИМА ---
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