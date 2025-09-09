package stealth

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestProxyRequest(t *testing.T) {
	// 1. Create a mock destination server
	destListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create destination listener: %v", err)
	}
	defer destListener.Close()
	destAddr := destListener.Addr().String()

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		destConn, err := destListener.Accept()
		if err != nil {
			t.Errorf("Destination failed to accept connection: %v", err)
			return
		}
		defer destConn.Close()

		// Read the request from the proxy
		_, err = http.ReadRequest(bufio.NewReader(destConn))
		if err != nil {
			t.Errorf("Destination failed to read request: %v", err)
			return
		}

		// Write a response
		response := "HTTP/1.1 200 OK\r\nContent-Length: 12\r\n\r\nHello, World"
		if _, err := destConn.Write([]byte(response)); err != nil {
			t.Errorf("Destination failed to write response: %v", err)
		}
	}()

	// 2. Create a mock client listener
	clientListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create client listener: %v", err)
	}
	defer clientListener.Close()
	clientAddr := clientListener.Addr().String()

	// 3. Run ProxyRequest in a goroutine
	go func() {
		clientConn, err := clientListener.Accept()
		if err != nil {
			t.Errorf("Proxy failed to accept client connection: %v", err)
			return
		}
		ProxyRequest(clientConn, fmt.Sprintf("http://%s", destAddr))
	}()

	// 4. Mock client connects to the proxy
	proxyConn, err := net.Dial("tcp", clientAddr)
	if err != nil {
		t.Fatalf("Client failed to connect to proxy: %v", err)
	}
	defer proxyConn.Close()

	// 5. Send a request from the client to the proxy
	request := "GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"
	if _, err := proxyConn.Write([]byte(request)); err != nil {
		t.Fatalf("Client failed to write request: %v", err)
	}

	// 6. Read the response from the proxy
	proxyConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	response, err := ioutil.ReadAll(proxyConn)
	if err != nil {
		t.Fatalf("Client failed to read response: %v", err)
	}

	// 7. Verify the response
	expectedResponse := "Hello, World"
	if !strings.Contains(string(response), expectedResponse) {
		t.Errorf("Expected response to contain '%s', but got '%s'", expectedResponse, string(response))
	}

	wg.Wait()
}
