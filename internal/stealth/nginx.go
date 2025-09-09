package stealth

import (
	"fmt"
	"time"
)

const nginxHTMLBody = `<!DOCTYPE html>
<html>
<head>
<title>Welcome to nginx!</title>
<style>
    body {
        width: 35em;
        margin: 0 auto;
        font-family: Tahoma, Verdana, Arial, sans-serif;
    }
</style>
</head>
<body>
<h1>Welcome to nginx!</h1>
<p>If you see this page, the nginx web server is successfully installed and
working. Further configuration is required.</p>

<p>For online documentation and support please refer to
<a href="http://nginx.org/">nginx.org</a>.<br/>
Commercial support is available at
<a href="http://nginx.com/">nginx.com</a>.</p>

<p><em>Thank you for using nginx.</em></p>
</body>
</html>`

// GetNginxResponse generates a full HTTP response that mimics a standard Nginx server.
func GetNginxResponse() []byte {
	// Format the current date into the standard GMT format for HTTP headers.
	date := time.Now().UTC().Format(time.RFC1123)

	// Assemble headers that closely resemble a real Nginx response.
	headers := fmt.Sprintf(
		"HTTP/1.1 200 OK\r\n"+
			"Server: nginx/1.18.0 (Ubuntu)\r\n"+
			"Date: %s\r\n"+
			"Content-Type: text/html\r\n"+
			"Content-Length: %d\r\n"+
			"Last-Modified: Tue, 01 Sep 2025 12:00:00 GMT\r\n"+
			"Connection: close\r\n"+
			"ETag: \"5f4e3a9c-265\"\r\n"+
			"Accept-Ranges: bytes\r\n"+
			"\r\n",
		date,
		len(nginxHTMLBody),
	)

	// Combine headers and body to form the full response.
	fullResponse := headers + nginxHTMLBody

	return []byte(fullResponse)
}