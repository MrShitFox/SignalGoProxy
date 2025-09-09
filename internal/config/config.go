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
	StealthNone  StealthMode = "none"
	StealthNginx StealthMode = "nginx"
)

// Config хранит все конфигурационные параметры.
type Config struct {
	Domain      string
	StealthMode StealthMode
	// В будущем можно добавить Email для Let's Encrypt, порты и т.д.
}

// New создает новую конфигурацию, считывая флаги и переменные окружения.
// Флаги имеют приоритет над переменными окружения.
func New() *Config {
	cfg := &Config{}

	var domain, stealthMode string

	// 1. Определяем флаги
	flag.StringVar(&domain, "domain", "", "Domain for the TLS certificate (required).")
	flag.StringVar(&stealthMode, "stealth-mode", "nginx", "Stealth mode: 'none' or 'nginx'.")
	flag.Parse()

	// 2. Если флаги не установлены, читаем переменные окружения
	if domain == "" {
		domain = os.Getenv("DOMAIN")
	}
	if stealthMode == "" || stealthMode == "nginx" && os.Getenv("STEALTH_MODE") != "" {
		stealthMode = os.Getenv("STEALTH_MODE")
	}

	// 3. Валидация
	if domain == "" {
		log.Fatal("Domain is required. Set it with -domain flag or DOMAIN environment variable.")
	}
	cfg.Domain = domain

	switch strings.ToLower(stealthMode) {
	case "nginx":
		cfg.StealthMode = StealthNginx
	case "none":
		cfg.StealthMode = StealthNone
	default:
		log.Fatalf("Invalid stealth mode: %s. Use 'none' or 'nginx'.", stealthMode)
	}

	return cfg
}