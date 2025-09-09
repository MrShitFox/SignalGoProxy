// Package server управляет жизненным циклом TCP и HTTP серверов.
package server

import (
	"context"
	"crypto/tls"
	"errors" // <-- ДОБАВЛЕН ЭТОТ ИМПОРТ
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"golang.org/x/crypto/acme/autocert"
	"signalgoproxy/internal/config"
	"signalgoproxy/internal/proxy"
)

// Server - это наш главный серверный объект.
type Server struct {
	cfg         *config.Config
	httpServer  *http.Server
	tlsListener net.Listener
}

// New создает новый экземпляр сервера.
func New(cfg *config.Config) *Server {
	return &Server{
		cfg: cfg,
	}
}

// Start запускает все необходимые слушатели и ожидает сигнала о завершении.
func (s *Server) Start() {
	log.Println("Stage 1: Initializing...")

	certManager := &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(s.cfg.Domain),
		Cache:      autocert.DirCache("certs"),
	}

	tlsConfig := &tls.Config{
		GetCertificate: certManager.GetCertificate,
		NextProtos:     []string{"http/1.1", "acme-tls/1"},
	}

	// Создаем HTTP сервер для ACME challenge
	s.httpServer = &http.Server{
		Addr:    ":80",
		Handler: certManager.HTTPHandler(nil),
	}

	// Создаем TLS слушатель
	listener, err := tls.Listen("tcp", ":443", tlsConfig)
	if err != nil {
		log.Fatalf("Failed to listen on :443: %v", err)
	}
	s.tlsListener = listener

	// --- Stage 2: Запуск ---
	log.Println("Stage 2: Starting services...")
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		log.Println("Starting HTTP server on :80 for ACME challenges.")
		if err := s.httpServer.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
		log.Println("HTTP server stopped.")
	}()

	go func() {
		defer wg.Done()
		log.Println("Starting Signal TLS Proxy on :443.")
		s.acceptLoop()
		log.Println("TLS proxy stopped.")
	}()

	// --- Stage 3: Ожидание завершения ---
	log.Println("Stage 3: Running. Waiting for shutdown signal...")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutdown signal received...")
	s.stop()

	// Ждем, пока все горутины завершатся
	wg.Wait()
	log.Println("Server shut down gracefully.")
}

// acceptLoop принимает новые соединения и передает их обработчику.
func (s *Server) acceptLoop() {
	for {
		conn, err := s.tlsListener.Accept()
		if err != nil {
			// Если ошибка - это результат закрытия слушателя, то это нормальный выход.
			if errors.Is(err, net.ErrClosed) {
				break
			}
			log.Printf("Failed to accept connection: %v", err)
			continue
		}
		go proxy.HandleConnection(conn, s.cfg)
	}
}

// stop выполняет graceful shutdown.
func (s *Server) stop() {
	log.Println("Initiating graceful shutdown...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Сначала закрываем слушатель, чтобы перестать принимать новые соединения
	if err := s.tlsListener.Close(); err != nil {
		log.Printf("Error closing TLS listener: %v", err)
	}

	// Затем останавливаем HTTP сервер
	if err := s.httpServer.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}
}