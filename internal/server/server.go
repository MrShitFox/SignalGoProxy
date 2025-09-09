// Package server manages the lifecycle of TCP and HTTP servers.
package server

import (
	"context"
	"crypto/tls"
	"errors"
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

// Server is the main server object.
type Server struct {
	cfg         *config.Config
	httpServer  *http.Server
	tlsListener net.Listener
}

// New creates a new server instance.
func New(cfg *config.Config) *Server {
	return &Server{
		cfg: cfg,
	}
}

// Start launches all necessary listeners and waits for a shutdown signal.
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

	// Create an HTTP server for the ACME challenge
	s.httpServer = &http.Server{
		Addr:    ":80",
		Handler: certManager.HTTPHandler(nil),
	}

	// Create a TLS listener
	listener, err := tls.Listen("tcp", ":443", tlsConfig)
	if err != nil {
		log.Fatalf("Failed to listen on :443: %v", err)
	}
	s.tlsListener = listener

	// --- Stage 2: Startup ---
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

	// --- Stage 3: Running ---
	log.Println("Stage 3: Running. Waiting for shutdown signal...")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutdown signal received...")
	s.stop()

	// Wait for all goroutines to finish
	wg.Wait()
	log.Println("Server shut down gracefully.")
}

// acceptLoop accepts new connections and passes them to the handler.
func (s *Server) acceptLoop() {
	for {
		conn, err := s.tlsListener.Accept()
		if err != nil {
			// If the error is due to the listener being closed, it's a clean exit.
			if errors.Is(err, net.ErrClosed) {
				break
			}
			log.Printf("Failed to accept connection: %v", err)
			continue
		}
		go proxy.HandleConnection(conn, s.cfg)
	}
}

// stop performs a graceful shutdown.
func (s *Server) stop() {
	log.Println("Initiating graceful shutdown...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// First, close the listener to stop accepting new connections
	if err := s.tlsListener.Close(); err != nil {
		log.Printf("Error closing TLS listener: %v", err)
	}

	// Then, shut down the HTTP server
	if err := s.httpServer.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}
}