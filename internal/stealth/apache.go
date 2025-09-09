// Package stealth provides modules for camouflaging the proxy as a standard web server.
package stealth

import (
	"fmt"
	"time"
)

// A standard Apache2 default page for Ubuntu.
const apacheHTMLBody = `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
  <head>
    <meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
    <title>Apache2 Ubuntu Default Page: It works</title>
    <style type="text/css" media="screen">
      * {
        margin: 0px 0px 0px 0px;
        padding: 0px 0px 0px 0px;
      }
      body, html {
        padding: 3px 3px 3px 3px;
        background-color: #D8DBE2;
        font-family: Verdana, sans-serif;
        font-size: 11pt;
      }
    </style>
  </head>
  <body>
    <div style="margin-left: auto; margin-right: auto; width: 760px; text-align: left;">
      <p style="text-align: center;">
        <b><span style="font-size: 14pt;">Apache2 Ubuntu Default Page</span></b>
      </p>
	  <p>
	    This is the default welcome page used to test the correct 
	    operation of the Apache2 server after installation on Ubuntu systems.
	  </p>
    </div>
  </body>
</html>`

// GetApacheResponse generates a full HTTP response that mimics a standard Apache server.
func GetApacheResponse() []byte {
	date := time.Now().UTC().Format(time.RFC1123)

	lastModified := generatePastDate()

	headers := fmt.Sprintf(
		"HTTP/1.1 200 OK\r\n"+
			"Date: %s\r\n"+
			"Server: Apache/2.4.41 (Ubuntu)\r\n"+
			"Last-Modified: %s\r\n"+
			"ETag: \"2d-4e9a49938b880\"\r\n"+
			"Accept-Ranges: bytes\r\n"+
			"Content-Length: %d\r\n"+
			"Vary: Accept-Encoding\r\n"+
			"Content-Type: text/html\r\n"+
			"Connection: close\r\n"+
			"\r\n",
		date,
		lastModified,
		len(apacheHTMLBody),
	)

	fullResponse := headers + apacheHTMLBody

	return []byte(fullResponse)
}