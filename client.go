package h3conn

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/lucas-clemente/quic-go/http3"
)

type Client struct {
	RoundTripper *http3.RoundTripper
}

func NewClient(tlsConfig *tls.Config) *Client {
	client := &Client{
		RoundTripper: &http3.RoundTripper{TLSClientConfig: tlsConfig},
	}
	return client
}

func (c *Client) Connect(urlStr string, timeout time.Duration, header http.Header) (*Conn, *http.Response, error) {
	reader, writer := io.Pipe()
	// Create a request object to send to the server
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlStr, reader)
	if err != nil {
		return nil, nil, err
	}

	req.Header = header

	timer := time.NewTimer(timeout)

	go func() {
		<-timer.C
		cancel()
	}()

	// Perform the request
	resp, err := c.RoundTripper.RoundTrip(req)
	if err != nil {
		return nil, nil, err
	}
	timer.Stop()

	if resp.StatusCode != http.StatusOK {
		return nil, resp, errors.New("h3 handshake fail,code=" + strconv.Itoa(resp.StatusCode))
	}
	// Create a connection
	conn := newConn(nil, nil, resp.Body, writer)

	return conn, resp, nil
}

var defaultClient = Client{
	RoundTripper: &http3.RoundTripper{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
}

func Connect(urlStr string, timeout time.Duration) (*Conn, *http.Response, error) {
	return defaultClient.Connect(urlStr, timeout, nil)
}
