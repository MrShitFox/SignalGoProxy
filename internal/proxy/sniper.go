// Package proxy содержит основную логику проксирования.
package proxy

import (
	"bufio"
	"io"
	"net/http"
)

// Protocol определяет тип обнаруженного протокола.
type Protocol int

const (
	ProtoSignalTLS Protocol = iota // Внутренний TLS-хендшейк Signal
	ProtoHTTP                    // Обычный HTTP/HTTPS запрос (от браузера)
	ProtoUnknown
)

// sniffProtocol "заглядывает" в соединение и определяет, что за трафик к нам пришел.
func sniffProtocol(reader *bufio.Reader) (Protocol, []byte, error) {
	// Peek() позволяет посмотреть первые байты, не "потребляя" их из буфера.
	// Это важно, т.к. эти байты (ClientHello) нам еще нужно будет переслать дальше.
	peekedBytes, err := reader.Peek(5)
	if err != nil {
		if err == io.EOF {
			return ProtoUnknown, nil, nil // Соединение закрылось до отправки данных
		}
		return ProtoUnknown, nil, err
	}

	// 0x16 = TLS Handshake. Если первый байт такой, значит это Signal пытается
	// установить внутреннее TLS-соединение.
	if peekedBytes[0] == 0x16 {
		return ProtoSignalTLS, nil, nil
	}

	// Если это не TLS, предполагаем, что это HTTP.
	// Пробуем разобрать как HTTP-запрос для уверенности.
	_, err = http.ReadRequest(reader)
	if err == nil {
		// Мы не можем вернуть прочитанные байты, так как ReadRequest их потребил.
		// Но для HTTP-заглушки это и не нужно.
		return ProtoHTTP, nil, nil
	}

	return ProtoUnknown, nil, nil
}