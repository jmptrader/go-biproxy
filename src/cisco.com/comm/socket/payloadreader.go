package socket

import (
	"errors"
	log "github.com/Sirupsen/logrus"
	"io"
	"sync"
)

// A PayloadReader is a wrapper around a standard io.Reader which is capable
// of understanding the message protocol such that it will only read a single
// message from the underlying io.Reader stream before returning EOF. The
// underlying io.Reader may or may not still have data in its buffer (i.e.,
// subsequent reads directly from the io.Reader do not necessarily return EOF
// just because PayloadReader did). If you use PayloadReader, do should NOT
// share the underlying io.Reader with anything else until PayloadReader reads
// it to its own completion.
type payloadReader struct {
	header     Header
	connection Pipe
	progress   uint64
	closed     bool
	mlock      sync.Mutex
}

// Close the PayloadReader. This will NOT close the underlying io.Reader. It
// only serves to close the current "Message" for this PayloadReader so that
// no subsequent Read()s from it are possible. Calling Read() after closing
// the PayloadReader will have no effect. A PayloadReader cannot be reopened
// once it is closed. Close() will be called automatically from Read() upon
// exhaustion of the reader so there's little reason to invoke it directly.
func (p *payloadReader) Close() {
	p.closed = true
	p.connection.Done()
}

// Read a message. This function implements the io.Reader interface for
// PayloadReader. It will return EOF once the message has been consumed. This
// does not necessarily mean the underlying io.Reader has been exhausted.
func (p *payloadReader) Read(output []byte) (int, error) {
	if p.closed {
		log.Println("[socket.payloadreader] ERR Reading from depleted socket")
		return 0, errors.New("ERR_SOCKET_RE_READ")
	}

	h := p.header
	conn := p.connection

	buf := make([]byte, Min(int64(len(output)), int64(h.Length-p.progress)))

	n, err := conn.Read(buf)
	log.Debug("[socket.payloadreader] Read ", n, " bytes")

	if n <= 0 {
		log.Info("[socket.payloadreader] PayloadReader depleted. Message has been consumed.", n)
		p.Close()
		return n, io.EOF
	}

	p.progress += uint64(n)
	copy(output, buf)

	if err != nil {
		log.Debug("Connection Closed")
		p.Close()
		return n, err
	}

	return n, nil
}
