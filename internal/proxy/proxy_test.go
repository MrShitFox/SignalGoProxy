// Package proxy contains the core proxying logic.
package proxy

import (
	"bufio"
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/cryptobyte"
)

// TestSniffProtocol tests the protocol sniffing logic.
func TestSniffProtocol(t *testing.T) {
	testCases := []struct {
		name             string
		input            []byte
		expectedProtocol Protocol
		expectError      bool
	}{
		{
			name:             "Signal TLS Handshake",
			input:            []byte{0x16, 0x03, 0x01, 0x02, 0x00, 0x01, 0x00, 0x01},
			expectedProtocol: ProtoSignalTLS,
			expectError:      false,
		},
		{
			name:             "HTTP GET Request",
			input:            []byte("GET / HTTP/1.1\r\n"),
			expectedProtocol: ProtoHTTP,
			expectError:      false,
		},
		{
			name:             "HTTP POST Request",
			input:            []byte("POST /submit HTTP/1.1\r\n"),
			expectedProtocol: ProtoHTTP,
			expectError:      false,
		},
		{
			name:             "Unknown Protocol",
			input:            []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07},
			expectedProtocol: ProtoUnknown,
			expectError:      false,
		},
		{
			name:             "Empty Input",
			input:            []byte{},
			expectedProtocol: ProtoUnknown,
			expectError:      false,
		},
		{
			name:             "Short Input TLS",
			input:            []byte{0x16, 0x03, 0x01},
			expectedProtocol: ProtoSignalTLS, // Should still be detected
			expectError:      false,
		},
		{
			name:             "Short Input Other",
			input:            []byte{0x01, 0x02, 0x03},
			expectedProtocol: ProtoUnknown,
			expectError:      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reader := bufio.NewReader(bytes.NewReader(tc.input))
			protocol, _, err := sniffProtocol(reader)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedProtocol, protocol)
			}
		})
	}
}

// buildTestClientHello creates a syntactically correct ClientHello record
// using cryptobyte, which helps avoid manual length calculation errors.
func buildTestClientHello(t *testing.T, serverName string) []byte {
	var body, extensions, serverNameExt cryptobyte.Builder

	// --- Build Extensions ---
	if serverName != "" {
		// Server Name extension (the one we care about)
		serverNameExt.AddUint16(0) // server_name extension type
		serverNameExt.AddUint16LengthPrefixed(func(b *cryptobyte.Builder) {
			b.AddUint16LengthPrefixed(func(b *cryptobyte.Builder) {
				b.AddUint8(0) // name_type = host_name
				b.AddUint16LengthPrefixed(func(b *cryptobyte.Builder) {
					b.AddBytes([]byte(serverName))
				})
			})
		})
		extensions.AddBytes(serverNameExt.BytesOrPanic())
	}

	// A dummy extension to ensure the list is not empty if SNI is not present
	// This makes the parsing logic slightly different between the two cases.
	if serverName == "" {
		extensions.AddUint16(0x0017) // padding extension
		extensions.AddUint16(0)      // length 0
	}

	// --- Build ClientHello Body ---
	body.AddUint16(0x0303) // legacy_version (TLS 1.2)
	body.AddBytes(make([]byte, 32)) // random
	body.AddUint8LengthPrefixed(func(b *cryptobyte.Builder) { // session_id
		// empty
	})
	body.AddUint16LengthPrefixed(func(b *cryptobyte.Builder) { // cipher_suites
		b.AddUint16(0xc02b) // some cipher
	})
	body.AddUint8LengthPrefixed(func(b *cryptobyte.Builder) { // compression_methods
		b.AddUint8(0)
	})
	body.AddUint16LengthPrefixed(func(b *cryptobyte.Builder) { // extensions
		b.AddBytes(extensions.BytesOrPanic())
	})

	// --- Build Handshake Message ---
	var handshakeMsg cryptobyte.Builder
	handshakeMsg.AddUint8(1) // ClientHello message type
	handshakeMsg.AddUint24LengthPrefixed(func(b *cryptobyte.Builder) {
		b.AddBytes(body.BytesOrPanic())
	})

	// --- Build TLS Record ---
	var record cryptobyte.Builder
	record.AddUint8(0x16) // Handshake record type
	record.AddUint16(0x0301) // legacy_record_version
	record.AddUint16LengthPrefixed(func(b *cryptobyte.Builder) {
		b.AddBytes(handshakeMsg.BytesOrPanic())
	})

	return record.BytesOrPanic()
}

// TestGetSNI tests the SNI parsing from a ClientHello message.
func TestGetSNI(t *testing.T) {
	validCH := buildTestClientHello(t, "test.example.com")
	noSniCH := buildTestClientHello(t, "")

	testCases := []struct {
		name           string
		input          io.Reader
		fullRecord     []byte
		expectedSNI    string
		expectError    bool
		expectedErrMsg string
	}{
		{
			name:        "Valid ClientHello with SNI",
			input:       bytes.NewReader(validCH),
			fullRecord:  validCH,
			expectedSNI: "test.example.com",
			expectError: false,
		},
		{
			name:           "ClientHello without SNI",
			input:          bytes.NewReader(noSniCH),
			fullRecord:     noSniCH,
			expectError:    true,
			expectedErrMsg: "SNI not found",
		},
		{
			name:           "Malformed - Not a handshake record",
			input:          bytes.NewReader([]byte{0x17, 0x03, 0x01, 0x00, 0x01}),
			expectError:    true,
			expectedErrMsg: "not a TLS handshake record",
		},
		{
			name:           "Empty Input",
			input:          bytes.NewReader([]byte{}),
			expectError:    true,
			expectedErrMsg: "failed to read TLS record header: EOF",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sni, raw, err := getSNI(tc.input)

			if tc.expectError {
				require.Error(t, err)
				if tc.expectedErrMsg != "" {
					assert.Contains(t, err.Error(), tc.expectedErrMsg)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedSNI, sni)
				assert.Equal(t, tc.fullRecord, raw, "The full raw ClientHello should be returned")
			}
		})
	}
}
