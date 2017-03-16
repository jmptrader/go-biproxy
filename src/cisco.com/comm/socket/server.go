package socket

import (
	"cisco.com/comm/common"
	"cisco.com/comm/log"
	"fmt"
	"net"
	"sync"
)

type Server interface {
	Listen() error
	GetConnection(int) common.Connection
	GetConnections() []common.Connection
}

// A Server represents a listen-able endpoint. This is the endpoint that
// a Client will connect to. The common.ConnectionHandler passed in is responsible
// for responding to Client connections.
type server struct {
	Port     int
	Handler  ConnectionHandler
	m        sync.Mutex
	channels map[int]common.Connection
	i        int
}

func NewServer(port int, handler ConnectionHandler) Server {
	return &server{Port: port, Handler: handler, channels: make(map[int]common.Connection)}
}

// Start the server. This call will block until the server shuts down.
func (s *server) Listen() error {

	// Called by the handler when connection is closed
	teardown := func(i int) {
		s.m.Lock()
		delete(s.channels, i)
		s.m.Unlock()
	}

	lst, err := net.Listen(proto, fmt.Sprintf(":%d", s.Port))
	log.I("WAN server listening on %s", lst.Addr())

	if err != nil {
		return err
	}

	for {
		wan, err := lst.Accept()

		if err != nil {
			return err
		}

		s.m.Lock()
		s.i = (s.i + 1) % 65536

		s.channels[s.i] = common.Connection{
			Id:     s.i,
			Remote: wan.RemoteAddr(),
			Out:    make(chan (common.EgressMessage)),
			In:     make(chan common.IngressMessage)}

		c := s.channels[s.i]
		s.m.Unlock()

		go s.Handler.OnConnect(NewPipe(wan), c, teardown)
	}
}

func (s *server) GetConnections() []common.Connection {
	s.m.Lock()
	res := make([]common.Connection, len(s.channels))

	i := 0
	for _, value := range s.channels {
		res[i] = value
		i++
	}

	s.m.Unlock()
	return res
}

func (s *server) GetConnection(idx int) common.Connection {
	defer s.m.Unlock()
	s.m.Lock()
	return s.channels[idx]
}
