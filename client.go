package h3conn

import (
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/lucas-clemente/quic-go/http3"
)

// Client provides HTTP2 client side connection with special arguments
type Client struct {
	// Header enables sending custom headers to the server
	Header http.Header
	// Client is a custom HTTP client to be used for the connection.
	// The client must have an http2.Transport as it's transport.
	Client *http.Client
}

// Connect establishes a full duplex communication with an HTTP2 server with custom client.
// See h2conn.Connect documentation for more info.
func (c *Client) Connect(urlStr string) (*Conn, *http.Response, error) {
	reader, writer := io.Pipe()

	// Create a request object to send to the server
	req, err := http.NewRequest(http.MethodPost, urlStr, reader)
	if err != nil {
		return nil, nil, err
	}

	// Apply custom headers
	if c.Header != nil {
		req.Header = c.Header
	}

	// If an http client was not defined, use the default http client
	httpClient := c.Client
	if httpClient == nil {
		httpClient = defaultClient.Client
	}

	// Perform the request
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, resp, errors.New("h3 handshake fail,code=" + strconv.Itoa(resp.StatusCode))
	}
	// Create a connection
	conn := newConn(nil, nil, resp.Body, writer)

	return conn, resp, nil
}

var defaultClient = Client{
	Client: &http.Client{
		Transport: &http3.RoundTripper{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
		Timeout:   time.Second * 5,
	},
}

func Connect(urlStr string) (*Conn, *http.Response, error) {
	return defaultClient.Connect(urlStr)
}
