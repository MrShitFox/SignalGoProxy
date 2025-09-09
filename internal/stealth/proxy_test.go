package stealth

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGeneratePastDate checks that the generated date is in the correct format.
func TestGeneratePastDate(t *testing.T) {
	dateStr := generatePastDate()
	_, err := time.Parse(time.RFC1123, dateStr)
	assert.NoError(t, err, "The generated date should be in RFC1123 format")
}

// TestGetNginxResponse checks the fake Nginx response.
func TestGetNginxResponse(t *testing.T) {
	responseBytes := GetNginxResponse()
	response, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(responseBytes)), nil)
	require.NoError(t, err)

	assert.Equal(t, "200 OK", response.Status)
	assert.Equal(t, "nginx/1.18.0 (Ubuntu)", response.Header.Get("Server"))
	assert.Contains(t, response.Header.Get("Content-Type"), "text/html")

	body, err := ioutil.ReadAll(response.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "Welcome to nginx!")
}

// TestGetApacheResponse checks the fake Apache response.
func TestGetApacheResponse(t *testing.T) {
	responseBytes := GetApacheResponse()
	response, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(responseBytes)), nil)
	require.NoError(t, err)

	assert.Equal(t, "200 OK", response.Status)
	assert.Equal(t, "Apache/2.4.41 (Ubuntu)", response.Header.Get("Server"))
	assert.Contains(t, response.Header.Get("Content-Type"), "text/html")

	body, err := ioutil.ReadAll(response.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "Apache2 Ubuntu Default Page")
}

// TestProxyRequest from original file
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
		ProxyRequest(bufio.NewReader(serverConn), serverConn, mockDestServer.URL)
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

// TestProxyRequest_BadGateway tests how ProxyRequest handles a failing target.
func TestProxyRequest_BadGateway(t *testing.T) {
	// Create a mock server and immediately close it to simulate a connection error.
	mockTargetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	mockTargetServer.Close()

	clientConn, proxyConn := net.Pipe()

	var wg sync.WaitGroup
	wg.Add(1)

	var respBytes []byte
	var readErr error

	// This goroutine will be the "client"
	go func() {
		defer wg.Done()
		defer clientConn.Close()

		// The client writes a request...
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)
		err = req.Write(clientConn)
		require.NoError(t, err)

		// ...and then reads the response from the server
		respBytes, readErr = ioutil.ReadAll(clientConn)
	}()

	// The "server" side runs the function under test
	ProxyRequest(bufio.NewReader(proxyConn), proxyConn, mockTargetServer.URL)

	wg.Wait()

	// Now, check the results after the goroutine has finished.
	require.NoError(t, readErr)
	require.True(t, len(respBytes) > 0, "Should have read some bytes")
	assert.True(t, strings.HasPrefix(string(respBytes), "HTTP/1.0 502 Bad Gateway"), "Response should be 502")
}
