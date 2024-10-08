package h3conn

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
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

	wg := &sync.WaitGroup{}
	wg.Add(1)

	timer := time.NewTimer(timeout)

	go func() {
		<-timer.C
		cancel()
		wg.Done()
	}()

	c.tlsConfig.NextProtos = []string{"h3"}

	qconn, err := dial(ctx, host, c.tlsConfig, &quic.Config{KeepAlivePeriod: time.Second * 10, MaxIdleTimeout: time.Hour * 24})

	if err != nil {
		cancel()
		return nil, nil, err
	}

	var stream quic.Stream = nil

	rt := &http3.SingleDestinationRoundTripper{
		Connection: qconn,
		StreamHijacker: func(ft http3.FrameType, cti quic.ConnectionTracingID, s quic.Stream, err error) (bool, error) {

			stream = s
			wg.Done()
			return true, nil
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlStr, strings.NewReader("hello"))

	if err != nil {
		cancel()
		return nil, nil, err
	}

	req.Header = header

	// Perform the request
	resp, err := rt.RoundTrip(req)
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, resp, errors.New("h3 handshake fail,code=" + strconv.Itoa(resp.StatusCode))
	}

	wg.Wait()

	timer.Stop()

	if stream == nil {
		return nil, resp, errors.New("h3 create stream fail,timeout")
	}
	// Create a connection
	conn := newConn(qconn.RemoteAddr(), qconn.LocalAddr(), stream)

	return conn, resp, nil
}

var defaultClient = Client{
	tlsConfig: &tls.Config{InsecureSkipVerify: true},
}

func Connect(urlStr string, timeout time.Duration) (*Conn, *http.Response, error) {
	return defaultClient.Connect(urlStr, timeout, nil)
}
