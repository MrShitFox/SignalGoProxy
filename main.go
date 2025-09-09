package main

import (
	"log"
	"signalgoproxy/internal/config"
	"signalgoproxy/internal/server"
)

func main() {
	// Устанавливаем префикс для логов
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// 1. Создаем конфигурацию
	cfg := config.New()
	log.Printf("Configuration loaded for domain '%s' with stealth mode '%s'", cfg.Domain, cfg.StealthMode)

	// 2. Создаем сервер
	srv := server.New(cfg)

	// 3. Запускаем сервер (это блокирующая операция)
	srv.Start()
}