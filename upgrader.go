package h3conn

import (
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/lucas-clemente/quic-go"
	"github.com/lucas-clemente/quic-go/http3"
)

var ErrHTTP3NotSupported = fmt.Errorf("HTTP3 not supported")
var ErrHTTP3GetAddr = fmt.Errorf("HTTP3 get addr fail")

type Upgrader struct {
	StatusCode int
}

func (u *Upgrader) Accept(w http.ResponseWriter, r *http.Request) (*Conn, error) {

	if !r.ProtoAtLeast(3, 0) {
		return nil, ErrHTTP3NotSupported
	}
	flusher, ok := w.(http.Flusher)

	if !ok {
		return nil, ErrHTTP3NotSupported
	}

	stream, ok := w.(http3.DataStreamer)

	if !ok {
		return nil, ErrHTTP3NotSupported
	}

	laddr := r.Context().Value(http.LocalAddrContextKey)

	if laddr == nil {
		return nil, ErrHTTP3GetAddr
	}

	raddr, err := net.ResolveUDPAddr("udp", r.RemoteAddr)

	if err != nil {
		return nil, ErrHTTP3GetAddr
	}

	c := newConn(raddr, laddr.(net.Addr), r.Body, &flushWrite{w: w, f: flusher, s: stream.DataStream()})

	w.WriteHeader(u.StatusCode)

	flusher.Flush()

	return c, nil
}

var defaultUpgrader = Upgrader{
	StatusCode: http.StatusOK,
}

func Accept(w http.ResponseWriter, r *http.Request) (*Conn, error) {
	return defaultUpgrader.Accept(w, r)
}

type flushWrite struct {
	w io.Writer
	f http.Flusher
	s quic.Stream
}

func (fw *flushWrite) Write(data []byte) (int, error) {
	n, err := fw.w.Write(data)
	fw.f.Flush()
	return n, err
}

func (fw *flushWrite) Close() error {
	return fw.s.Close()
}
