package stealth

import (
	"bufio"
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
		// This can happen if the client disconnects, it's not always a server error.
		if err != io.EOF {
			log.Printf("Error reading request from client: %v", err)
		}
		return
	}

	// Parse the target proxy URL.
	targetURL, err := url.Parse(proxyURL)
	if err != nil {
		log.Printf("Error parsing proxy URL '%s': %v", proxyURL, err)
		// Inform the client of the error.
		resp := &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       http.NoBody,
		}
		resp.Write(clientConn)
		return
	}

	// Create a new request to the target.
	// Copy over the essential parts of the original request.
	outReq := &http.Request{
		Method: req.Method,
		URL:    targetURL,
		Header: req.Header,
		Body:   req.Body,
	}
	// The Host header is implicitly set by the http.Client when it makes the request.
	// We can also set it explicitly if needed: outReq.Host = targetURL.Host

	// Execute the request using the default HTTP client.
	// DefaultClient handles HTTPS and certificate validation.
	log.Printf("Proxying request for %s to %s", req.RemoteAddr, targetURL)
	resp, err := http.DefaultClient.Do(outReq)
	if err != nil {
		log.Printf("Error forwarding request to proxy target '%s': %v", targetURL, err)
		resp := &http.Response{
			StatusCode: http.StatusBadGateway,
			Body:       http.NoBody,
		}
		resp.Write(clientConn)
		return
	}
	defer resp.Body.Close()

	// Write the response from the target back to the client.
	if err := resp.Write(clientConn); err != nil {
		log.Printf("Error writing proxy response to client: %v", err)
	}
}
