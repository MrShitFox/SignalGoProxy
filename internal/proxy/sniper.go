// Package proxy contains the core proxying logic.
package proxy

import (
	"bufio"
	"errors"
	"io"
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
		// If the error is EOF or UnexpectedEOF, it means the client sent
		// less data than we wanted to peek. This is not a fatal error;
		// we just might not be able to determine the protocol.
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			// Try to identify based on what we did get.
			if len(peekedBytes) > 0 {
				if peekedBytes[0] == 0x16 {
					return ProtoSignalTLS, nil, nil
				}
				// Not enough data for a reliable HTTP check, so we'll fall through.
			}
			return ProtoUnknown, nil, nil // Not enough data to determine.
		}
		// Any other error is a real problem.
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