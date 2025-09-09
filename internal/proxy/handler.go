package proxy

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/cryptobyte"
	"signalgoproxy/internal/config"
	"signalgoproxy/internal/stealth"
)

// Routing map: SNI -> Signal server address.
var signalUpstreams = map[string]string{
	"chat.signal.org":         "chat.signal.org:443",
	"ud-chat.signal.org":      "chat.signal.org:443",
	"storage.signal.org":      "storage.signal.org:443",
	"cdn.signal.org":          "cdn.signal.org:443",
	"cdn2.signal.org":         "cdn2.signal.org:443",
	"cdn3.signal.org":         "cdn3.signal.org:443",
	"cdsi.signal.org":         "cdsi.signal.org:443",
	"contentproxy.signal.org": "contentproxy.signal.org:443",
	"sfu.voip.signal.org":     "sfu.voip.signal.org:443",
	"svr2.signal.org":         "svr2.signal.org:443",
	"svrb.signal.org":         "svrb.signal.org:443",
	"updates.signal.org":      "updates.signal.org:443",
	"updates2.signal.org":     "updates2.signal.org:443",
}

// HandleConnection is the main handler for incoming TLS connections.
func HandleConnection(conn net.Conn, cfg *config.Config) {
	defer conn.Close()

	bufReader := bufio.NewReader(conn)

	protocol, _, err := sniffProtocol(bufReader)
	if err != nil {
		log.Printf("Protocol sniffing error: %v", err)
		return
	}

	switch protocol {
	case ProtoSignalTLS:
		handleSignalProxy(bufReader, conn)
	case ProtoHTTP:
		handleStealth(conn, cfg)
	default:
		log.Printf("Unknown protocol from %s, closing connection.", conn.RemoteAddr())
	}
}

// handleSignalProxy handles traffic destined for Signal.
func handleSignalProxy(reader io.Reader, clientConn net.Conn) {
	serverName, rawClientHello, err := getSNI(reader)
	if err != nil {
		log.Printf("Failed to get inner SNI from %s: %v", clientConn.RemoteAddr(), err)
		return
	}
	log.Printf("Inner SNI '%s' detected from %s", serverName, clientConn.RemoteAddr())

	upstreamAddr, ok := signalUpstreams[strings.ToLower(serverName)]
	if !ok {
		log.Printf("Denied connection for unknown inner SNI: %s", serverName)
		return
	}

	upstreamConn, err := net.DialTimeout("tcp", upstreamAddr, 10*time.Second)
	if err != nil {
		log.Printf("Failed to connect to upstream %s: %v", upstreamAddr, err)
		return
	}
	defer upstreamConn.Close()

	if _, err = upstreamConn.Write(rawClientHello); err != nil {
		log.Printf("Failed to write inner ClientHello to upstream: %v", err)
		return
	}

	log.Printf("Proxying traffic for %s to %s", serverName, upstreamAddr)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		io.Copy(upstreamConn, clientConn)
		if tcpConn, ok := upstreamConn.(*net.TCPConn); ok {
			tcpConn.CloseWrite()
		}
	}()
	go func() {
		defer wg.Done()
		io.Copy(clientConn, upstreamConn)
		if tlsConn, ok := clientConn.(*tls.Conn); ok {
			tlsConn.CloseWrite()
		}
	}()
	wg.Wait()
	log.Printf("Connection for %s closed", serverName)
}

// handleStealth responds to HTTP requests with a stealth page to provide camouflage.
func handleStealth(conn net.Conn, cfg *config.Config) {
	var response []byte

	switch cfg.StealthMode {
	case config.StealthNginx:
		log.Printf("Stealth mode: Serving full fake Nginx page to %s", conn.RemoteAddr())
		response = stealth.GetNginxResponse()
	case config.StealthApache:
		log.Printf("Stealth mode: Serving full fake Apache page to %s", conn.RemoteAddr())
		response = stealth.GetApacheResponse()
	case config.StealthNone:
		// In "none" mode, just close the connection.
		return
	default:
		// Fallback for an unknown stealth mode.
		log.Printf("Unknown stealth mode '%s', closing connection.", cfg.StealthMode)
		return
	}

	_, err := conn.Write(response)
	if err != nil {
		log.Printf("Error writing stealth response: %v", err)
	}
}

// getSNI reads from the connection, parses the TLS ClientHello message,
// and extracts the Server Name Indication (SNI) extension.
// It returns the found server name, the raw ClientHello bytes, and any error.
// This implementation uses cryptobyte for robust and efficient parsing.
func getSNI(reader io.Reader) (string, []byte, error) {
	// Read the TLS record header.
	header := make([]byte, 5)
	if _, err := io.ReadFull(reader, header); err != nil {
		return "", nil, fmt.Errorf("failed to read TLS record header: %w", err)
	}

	// Check if it's a TLS handshake record.
	if header[0] != 0x16 { // 0x16 = Handshake
		return "", nil, errors.New("not a TLS handshake record")
	}

	// Read the rest of the record.
	recordLen := int(binary.BigEndian.Uint16(header[3:]))
	recordBody := make([]byte, recordLen)
	if _, err := io.ReadFull(reader, recordBody); err != nil {
		return "", nil, fmt.Errorf("failed to read TLS record body: %w", err)
	}

	fullRecord := append(header, recordBody...)

	// Wrap the record body in a cryptobyte.String for parsing.
	s := cryptobyte.String(recordBody)

	// Parse the ClientHello message.
	// See RFC 8446, Section 4.1.2.
	var msgType uint8
	var clientHello cryptobyte.String
	if !s.ReadUint8(&msgType) || msgType != 1 || !s.ReadUint24LengthPrefixed(&clientHello) { // 1 = ClientHello
		return "", nil, errors.New("not a ClientHello message")
	}

	// Skip legacy version and random.
	if !clientHello.Skip(2) || !clientHello.Skip(32) {
		return "", nil, errors.New("error parsing ClientHello header")
	}

	// Skip legacy session id.
	if !clientHello.Skip(1) {
		return "", nil, errors.New("error parsing session id")
	}

	// Skip cipher suites.
	var cipherSuites cryptobyte.String
	if !clientHello.ReadUint16LengthPrefixed(&cipherSuites) {
		return "", nil, errors.New("error parsing cipher suites")
	}

	// Skip compression methods.
	var compressionMethods cryptobyte.String
	if !clientHello.ReadUint8LengthPrefixed(&compressionMethods) {
		return "", nil, errors.New("error parsing compression methods")
	}

	// Check for extensions.
	if clientHello.Empty() {
		return "", nil, errors.New("no extensions found")
	}

	// Parse extensions.
	var extensions cryptobyte.String
	if !clientHello.ReadUint16LengthPrefixed(&extensions) {
		return "", nil, errors.New("error parsing extensions")
	}

	var serverName string
	for !extensions.Empty() {
		var extType uint16
		var extData cryptobyte.String
		if !extensions.ReadUint16(&extType) || !extensions.ReadUint16LengthPrefixed(&extData) {
			return "", nil, errors.New("error parsing extension")
		}

		if extType == 0 { // 0 = server_name
			var serverNameList cryptobyte.String
			if !extData.ReadUint16LengthPrefixed(&serverNameList) || serverNameList.Empty() {
				return "", nil, errors.New("error parsing server_name extension")
			}

			var nameType uint8
			var hostName cryptobyte.String
			if !serverNameList.ReadUint8(&nameType) || nameType != 0 || !serverNameList.ReadUint16LengthPrefixed(&hostName) || hostName.Empty() { // 0 = host_name
				return "", nil, errors.New("error parsing host_name")
			}
			serverName = string(hostName)
			break // Found it
		}
	}

	if serverName == "" {
		return "", nil, errors.New("SNI not found in ClientHello")
	}

	return serverName, fullRecord, nil
}