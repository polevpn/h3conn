package h3conn

import (
	"io"
	"net"
	"sync"
)

// Conn is client/server symmetric connection.
// It implements the io.Reader/io.Writer/io.Closer to read/write or close the connection to the other side.
// It also has a Send/Recv function to use channels to communicate with the other side.
type Conn struct {
	remoteAddr net.Addr
	localAddr  net.Addr
	r          io.ReadCloser
	w          io.WriteCloser
	wLock      sync.Mutex
	rLock      sync.Mutex
}

func newConn(remoteAddr net.Addr, localAddr net.Addr, r io.ReadCloser, w io.WriteCloser) *Conn {
	return &Conn{
		remoteAddr: remoteAddr,
		localAddr:  localAddr,
		r:          r,
		w:          w,
	}
}

// Write writes data to the connection
func (c *Conn) Write(data []byte) (int, error) {
	c.wLock.Lock()
	defer c.wLock.Unlock()
	return c.w.Write(data)
}

// Read reads data from the connection
func (c *Conn) Read(data []byte) (int, error) {
	c.rLock.Lock()
	defer c.rLock.Unlock()
	return c.r.Read(data)
}

// LocalAddr returns the local network address.
func (c *Conn) LocalAddr() net.Addr {
	if c.localAddr != nil {
		return c.localAddr
	} else {
		return &net.TCPAddr{IP: []byte{}, Port: 0, Zone: ""}
	}
}

// RemoteAddr returns the remote network address.
func (c *Conn) RemoteAddr() net.Addr {
	if c.remoteAddr != nil {
		return c.remoteAddr
	} else {
		return &net.TCPAddr{IP: []byte{}, Port: 0, Zone: ""}
	}
}

// Close closes the connection
func (c *Conn) Close() error {
	c.r.Close()
	return c.w.Close()
}
