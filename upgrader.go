package h3conn

import (
	"fmt"
	"io"
	"net/http"

	"github.com/quic-go/quic-go/http3"
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

	hijack, ok := w.(http3.Hijacker)

	if !ok {
		return nil, ErrHTTP3NotSupported
	}

	laddr := hijack.Connection().LocalAddr()

	if laddr == nil {
		return nil, ErrHTTP3GetAddr
	}

	raddr := hijack.Connection().RemoteAddr()

	c := newConn(raddr, laddr, r.Body, &flushWrite{w: w, f: flusher, c: hijack.Connection()})

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
	c http3.Connection
}

func (fw *flushWrite) Write(data []byte) (int, error) {
	n, err := fw.w.Write(data)
	fw.f.Flush()
	return n, err
}

func (fw *flushWrite) Close() error {
	return fw.c.CloseWithError(0, "closed")
}
