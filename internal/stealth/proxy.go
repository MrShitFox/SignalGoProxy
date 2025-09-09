package stealth

import (
	"bufio"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"sync"
)

// ProxyRequest forwards the client's request to a specified proxy URL and streams the response.
func ProxyRequest(clientConn net.Conn, proxyURL string) {
	// Ensure the client connection is closed when the function exits.
	defer clientConn.Close()

	// Parse the provided proxy URL to extract the host.
	parsedURL, err := url.Parse(proxyURL)
	if err != nil {
		log.Printf("Error parsing proxy URL '%s': %v", proxyURL, err)
		return
	}
	host := parsedURL.Host
	if parsedURL.Port() == "" {
		host = net.JoinHostPort(host, "443")
	}

	// Establish a connection to the proxy destination.
	destConn, err := net.Dial("tcp", host)
	if err != nil {
		log.Printf("Error connecting to proxy host '%s': %v", host, err)
		return
	}
	defer destConn.Close()

	// Read the full initial request from the client.
	clientReader := bufio.NewReader(clientConn)
	req, err := http.ReadRequest(clientReader)
	if err != nil {
		log.Printf("Error reading request from client: %v", err)
		return
	}

	// Forward the initial request to the destination server.
	if err := req.Write(destConn); err != nil {
		log.Printf("Error writing request to destination: %v", err)
		return
	}

	log.Printf("Proxying request to %s", host)

	// Use a WaitGroup to wait for both directions of the proxy to complete.
	var wg sync.WaitGroup
	wg.Add(2)

	// Goroutine to copy data from the client to the destination.
	go func() {
		defer wg.Done()
		io.Copy(destConn, clientConn)
		if tcpConn, ok := destConn.(*net.TCPConn); ok {
			tcpConn.CloseWrite()
		}
	}()

	// Goroutine to copy data from the destination back to the client.
	go func() {
		defer wg.Done()
		io.Copy(clientConn, destConn)
		if tcpConn, ok := clientConn.(*net.TCPConn); ok {
			tcpConn.CloseWrite()
		}
	}()

	// Wait for both copy operations to finish.
	wg.Wait()
	log.Printf("Proxy connection to %s closed", host)
}
