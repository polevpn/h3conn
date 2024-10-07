package h3conn

import (
	"fmt"
	"net/http"

	"github.com/quic-go/quic-go/http3"
)

var ErrHTTP3NotSupported = fmt.Errorf("HTTP3 not supported")
var ErrHTTP3GetAddr = fmt.Errorf("HTTP3 get addr fail")
var ErrHTTP3Create = fmt.Errorf("HTTP3 create stream fail")

type Upgrader struct {
	StatusCode int
}

func (u *Upgrader) Accept(w http.ResponseWriter, r *http.Request) (*Conn, error) {

	if !r.ProtoAtLeast(3, 0) {
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

	stream, err := hijack.Connection().OpenStream()

	if err != nil {
		return nil, ErrHTTP3Create
	}

	_, err = stream.Write([]byte("h3"))

	if err != nil {
		return nil, ErrHTTP3Create
	}

	c := newConn(raddr, laddr, stream)

	w.WriteHeader(u.StatusCode)

	return c, nil
}

var defaultUpgrader = Upgrader{
	StatusCode: http.StatusOK,
}

func Accept(w http.ResponseWriter, r *http.Request) (*Conn, error) {
	return defaultUpgrader.Accept(w, r)
}
