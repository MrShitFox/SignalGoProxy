// Package proxy contains the core proxying logic.
package proxy

import (
	"bufio"
	"io"
	"net/http"
	"strings"
)

// Protocol defines the type of the detected protocol.
type Protocol int

const (
	ProtoSignalTLS Protocol = iota // Inner TLS handshake from Signal
	ProtoHTTP                      // Standard HTTP/HTTPS request (from a browser)
	ProtoUnknown
)

// sniffProtocol peeks into the connection to determine the protocol being used
// without consuming any bytes from the reader.
func sniffProtocol(reader *bufio.Reader) (Protocol, []byte, error) {
	// Peek at the first few bytes to identify the protocol.
	// We peek at 8 bytes, which is enough to identify common HTTP methods
	// and the TLS handshake byte.
	peekedBytes, err := reader.Peek(8)
	if err != nil {
		// If we can't even peek a few bytes, it might be an EOF or a real error.
		if err == io.EOF {
			return ProtoUnknown, nil, nil // Connection closed before sending data.
		}
		return ProtoUnknown, nil, err
	}

	// 0x16 is the byte for a TLS Handshake record. If the first byte is this,
	// we assume it's the inner TLS handshake from Signal.
	if peekedBytes[0] == 0x16 {
		return ProtoSignalTLS, nil, nil
	}

	// If it's not a TLS handshake, check if it looks like a standard HTTP request.
	// We check for common HTTP methods.
	s := string(peekedBytes)
	if strings.HasPrefix(s, "GET ") ||
		strings.HasPrefix(s, "POST ") ||
		strings.HasPrefix(s, "HEAD ") ||
		strings.HasPrefix(s, "PUT ") ||
		strings.HasPrefix(s, "DELETE ") ||
		strings.HasPrefix(s, "OPTIONS ") ||
		strings.HasPrefix(s, "PATCH ") ||
		strings.HasPrefix(s, "CONNECT ") {
		return ProtoHTTP, nil, nil
	}

	// If it's neither, we don't know what it is.
	return ProtoUnknown, nil, nil
}