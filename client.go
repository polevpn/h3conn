package h3conn

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"golang.org/x/net/idna"
)

type Client struct {
	tlsConfig *tls.Config
}

func NewClient(tlsConfig *tls.Config) *Client {

	client := &Client{
		tlsConfig: tlsConfig,
	}
	return client
}

func (c *Client) getHost(urlStr string) (string, error) {

	u, err := url.Parse(urlStr)

	if err != nil {
		return "", err
	}

	host, port, err := net.SplitHostPort(u.Host)
	if err != nil {
		port = "443"
		host = u.Host
	}
	if a, err := idna.ToASCII(host); err == nil {
		host = a
	}
	// IPv6 address literal, without a port:
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		return host + ":" + port, nil
	}
	return net.JoinHostPort(host, port), nil
}

func (c *Client) Connect(urlStr string, timeout time.Duration, header http.Header) (*Conn, *http.Response, error) {

	reader, writer := io.Pipe()
	// Create a request object to send to the server
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	//create quic connect
	udpConn, err := net.ListenUDP("udp", nil)
	if err != nil {
		cancel()
		return nil, nil, err
	}

	transport := &quic.Transport{Conn: udpConn}

	dial := func(ctx context.Context, addr string, tlsCfg *tls.Config, cfg *quic.Config) (quic.EarlyConnection, error) {
		udpAddr, err := net.ResolveUDPAddr("udp", addr)
		if err != nil {
			return nil, err
		}
		return transport.DialEarly(ctx, udpAddr, tlsCfg, cfg)
	}

	host, err := c.getHost(urlStr)

	if err != nil {
		cancel()
		return nil, nil, err
	}

	c.tlsConfig.NextProtos = []string{"h3"}

	qconn, err := dial(ctx, host, c.tlsConfig, nil)

	if err != nil {
		cancel()
		return nil, nil, err
	}

	rt := &http3.SingleDestinationRoundTripper{
		Connection: qconn,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlStr, reader)
	if err != nil {
		cancel()
		return nil, nil, err
	}

	req.Header = header

	timer := time.NewTimer(timeout)

	go func() {
		<-timer.C
		cancel()
	}()

	// Perform the request
	resp, err := rt.RoundTrip(req)
	if err != nil {
		return nil, nil, err
	}
	timer.Stop()

	if resp.StatusCode != http.StatusOK {
		return nil, resp, errors.New("h3 handshake fail,code=" + strconv.Itoa(resp.StatusCode))
	}
	// Create a connection
	conn := newConn(qconn.RemoteAddr(), qconn.LocalAddr(), resp.Body, writer)

	return conn, resp, nil
}

var defaultClient = Client{
	tlsConfig: &tls.Config{InsecureSkipVerify: true},
}

func Connect(urlStr string, timeout time.Duration) (*Conn, *http.Response, error) {
	return defaultClient.Connect(urlStr, timeout, nil)
}
