package socket

import (
	"cisco.com/comm/log"
	"net"
	"sync"
)

// A bi-directional connection between two endpoints. Implements the
// WriteCloser interface and --indirectly through PayloadReader --the io.Reader
// interface.
type Pipe interface {
	Write([]byte) (int, error)
	Close() error
	Read([]byte) (int, error)
	Done()
	NextMessage() (*payloadReader, error)
}

type pipe struct {
	net.Conn

	mbody sync.Mutex
	body  *payloadReader
}

// Create a new Pipe. Do NOT share the net.Conn with any other goroutines.
// Doing so could result in undefined behavior.
func NewPipe(c net.Conn) Pipe {
	return &pipe{Conn: c}
}

func (s *pipe) Done() {
	s.mbody.Unlock()
}

// Consume the next message from the Pipe. Since each PayloadReader shares the
// same net.Conn (since there's only one network connection), calling
// NextMessage from different goroutines could result in undefined behavior.
func (s *pipe) NextMessage() (*payloadReader, error) {
	s.mbody.Lock()

	header, err := NewHeader(s)

	if err != nil {
		log.W("Error parsing header %v", err)
		return nil, err
	}

	log.D("Constructed new message. Using header %v", header)
	pr := &payloadReader{header: *header, connection: s}
	return pr, nil
}
