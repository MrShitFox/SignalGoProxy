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

	"signalgoproxy/internal/config"
)

// ... (структуры clientHelloInfo и fakeConn и функция newFakeConn остаются такими же, как в предыдущем примере)
type clientHelloInfo struct {
	Raw        []byte
	ServerName string
}

type fakeConn struct {
	*bytes.Buffer
}
func (c *fakeConn) Close() error                     { return nil }
func (c *fakeConn) LocalAddr() net.Addr              { return nil }
func (c *fakeConn) RemoteAddr() net.Addr             { return nil }
func (c *fakeConn) SetDeadline(t time.Time) error    { return nil }
func (c *fakeConn) SetReadDeadline(t time.Time) error  { return nil }
func (c *fakeConn) SetWriteDeadline(t time.Time) error { return nil }
func newFakeConn(data []byte) *fakeConn {
	return &fakeConn{Buffer: bytes.NewBuffer(data)}
}

// Карта маршрутизации Signal.
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

// HandleConnection - главный обработчик входящих TLS-соединений.
func HandleConnection(conn net.Conn, cfg *config.Config) {
	defer conn.Close()

	// Оборачиваем соединение в bufio.Reader для использования Peek()
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

// handleSignalProxy обрабатывает трафик для Signal.
func handleSignalProxy(reader io.Reader, clientConn net.Conn) {
	innerHello, err := getInnerClientHelloInfo(reader)
	if err != nil {
		log.Printf("Failed to get inner SNI from %s: %v", clientConn.RemoteAddr(), err)
		return
	}

	log.Printf("Inner SNI '%s' detected from %s", innerHello.ServerName, clientConn.RemoteAddr())

	upstreamAddr, ok := signalUpstreams[strings.ToLower(innerHello.ServerName)]
	if !ok {
		log.Printf("Denied connection for unknown inner SNI: %s", innerHello.ServerName)
		return
	}

	upstreamConn, err := net.DialTimeout("tcp", upstreamAddr, 10*time.Second)
	if err != nil {
		log.Printf("Failed to connect to upstream %s: %v", upstreamAddr, err)
		return
	}
	defer upstreamConn.Close()

	// Сначала отправляем уже прочитанный ClientHello
	if _, err = upstreamConn.Write(innerHello.Raw); err != nil {
		log.Printf("Failed to write inner ClientHello to upstream: %v", err)
		return
	}

	log.Printf("Proxying traffic for %s to %s", innerHello.ServerName, upstreamAddr)

	// Копируем данные в обе стороны
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
	log.Printf("Connection for %s closed", innerHello.ServerName)
}

// handleStealth отвечает на HTTP-запросы заглушкой для маскировки.
func handleStealth(conn net.Conn, cfg *config.Config) {
	log.Printf("Stealth mode: Serving fake Nginx page to %s", conn.RemoteAddr())
	
	response := "HTTP/1.1 200 OK\r\n" +
		"Server: nginx/1.21.3\r\n" +
		"Content-Type: text/html\r\n" +
		"Content-Length: 151\r\n" +
		"Connection: close\r\n" +
		"\r\n" +
		"<!DOCTYPE html>\n<html>\n<head>\n<title>Welcome to nginx!</title>\n</head>\n<body>\n<p><em>Thank you for using nginx.</em></p>\n</body>\n</html>"

	_, err := conn.Write([]byte(response))
	if err != nil {
		log.Printf("Error writing stealth response: %v", err)
	}
}

// getInnerClientHelloInfo парсит внутренний ClientHello.
func getInnerClientHelloInfo(reader io.Reader) (*clientHelloInfo, error) {
	// Эта функция остается такой же, как в последнем рабочем варианте
	header := make([]byte, 5)
	if _, err := io.ReadFull(reader, header); err != nil {
		return nil, err
	}
	if header[0] != 0x16 {
		return nil, errors.New("not a TLS handshake record")
	}
	recordLen := int(binary.BigEndian.Uint16(header[3:]))
	recordBody := make([]byte, recordLen)
	if _, err := io.ReadFull(reader, recordBody); err != nil {
		return nil, err
	}
	if len(recordBody) < 1 || recordBody[0] != 0x01 {
		return nil, errors.New("not a ClientHello message")
	}
	fullRecord := append(header, recordBody...)
	var serverName string
	config := &tls.Config{
		GetConfigForClient: func(hello *tls.ClientHelloInfo) (*tls.Config, error) {
			serverName = hello.ServerName
			return nil, fmt.Errorf("extracting SNI")
		},
	}
	tlsConn := tls.Server(newFakeConn(fullRecord), config)
	err := tlsConn.Handshake()
	if err != nil && !strings.Contains(err.Error(), "extracting SNI") {
		return nil, err
	}
	if serverName == "" {
		return nil, errors.New("SNI not found in inner ClientHello")
	}
	return &clientHelloInfo{
		Raw:        fullRecord,
		ServerName: serverName,
	}, nil
}