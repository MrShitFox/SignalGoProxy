package stealth

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestProxyRequest(t *testing.T) {
	// 1. Create a mock destination server
	mockDestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello, World")
	}))
	defer mockDestServer.Close()

	// 2. Create a mock client connection using a pipe
	clientConn, serverConn := net.Pipe()

	// 3. Run ProxyRequest in a goroutine with the server side of the pipe
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		ProxyRequest(serverConn, mockDestServer.URL)
	}()

	// 4. Write a sample HTTP request to the client side of the pipe
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	if err := req.Write(clientConn); err != nil {
		t.Fatalf("Failed to write request to pipe: %v", err)
	}

	// 5. Read the response from the client side of the pipe
	resp, err := http.ReadResponse(bufio.NewReader(clientConn), req)
	if err != nil {
		t.Fatalf("Failed to read response from pipe: %v", err)
	}
	defer resp.Body.Close()

	// 6. Verify the response
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, but got %d", http.StatusOK, resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	expectedBody := "Hello, World"
	if !strings.Contains(string(body), expectedBody) {
		t.Errorf("Expected response body to contain '%s', but got '%s'", expectedBody, string(body))
	}

	// Clean up
	clientConn.Close()
	wg.Wait()
}
