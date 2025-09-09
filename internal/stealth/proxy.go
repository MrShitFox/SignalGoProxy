package stealth

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
)

// ProxyRequest forwards the client's request to a specified proxy URL and streams the response.
func ProxyRequest(clientReader *bufio.Reader, clientConn net.Conn, proxyURL string) {
	defer clientConn.Close()

	// Read the full initial request from the client.
	req, err := http.ReadRequest(clientReader)
	if err != nil {
		if err != io.EOF {
			log.Printf("Error reading request from client: %v", err)
		}
		return
	}

	// Parse the target proxy URL.
	targetURL, err := url.Parse(proxyURL)
	if err != nil {
		log.Printf("Error parsing proxy URL '%s': %v", proxyURL, err)
		// Manually write a simple 500 error response.
		errorResponse := "HTTP/1.0 500 Internal Server Error\r\nConnection: close\r\n\r\n"
		clientConn.Write([]byte(errorResponse))
		return
	}

	// Create a new request to the target.
	outReq := &http.Request{
		Method: req.Method,
		URL:    targetURL,
		Header: req.Header,
		Body:   req.Body,
	}

	// Execute the request using the default HTTP client.
	log.Printf("Proxying request for %s to %s", req.RemoteAddr, targetURL)
	resp, err := http.DefaultClient.Do(outReq)
	if err != nil {
		log.Printf("Error forwarding request to proxy target '%s': %v", targetURL, err)
		// Manually write a simple 502 error response. This is more reliable
		// than http.Response.Write for simple, bodiless error responses.
		errorResponse := fmt.Sprintf("HTTP/1.0 %d %s\r\nConnection: close\r\n\r\n", http.StatusBadGateway, http.StatusText(http.StatusBadGateway))
		clientConn.Write([]byte(errorResponse))
		return
	}
	defer resp.Body.Close()

	// Write the response from the target back to the client.
	if err := resp.Write(clientConn); err != nil {
		log.Printf("Error writing proxy response to client: %v", err)
	}
}
