package socket

import (
	"cisco.com/comm/common"
	"fmt"
	"cisco.com/comm/log"
	"net"
	"sync"
)

// The Client half of a client-server connection model. Note that since there's
// only one network connection, the Client must necessarily share this
// connection across all methods. Therefore, calling methods on the client from
// multiple goroutines concurrently might result in undefined behavior.
type Client interface {
	Connect() error
	GetConnections() []common.Connection
	GetConnection(int) common.Connection
}

type client struct {
	Handler     ConnectionHandler
	Addr        *net.TCPAddr
	conn        *net.TCPConn
	mconnection sync.Mutex
	connection  *common.Connection
}

func NewClient(addr string, port int, h ConnectionHandler) (Client, error) {
	n, err := net.ResolveTCPAddr(proto, fmt.Sprintf("%s:%d", addr, port))

	if err != nil {
		return nil, err
	}

	c := &client{Handler: h, Addr: n}
	return c, nil
}

// Connect to the server and process data. This function blocks until it is
// shutdown either by the server closing the connection or program terminate.
func (c *client) Connect() error {
	conn, err := c.connect()

	if err != nil {
		return err
	}

	OnTeardown := func(int) {
		c.mconnection.Lock()
		c.connection = nil
		c.mconnection.Unlock()
	}

	c.connection = &common.Connection{
		Remote: conn.RemoteAddr(),
		Out:    make(chan common.EgressMessage),
		In:     make(chan common.IngressMessage)}

	c.Handler.OnConnect(NewPipe(conn), *c.connection, OnTeardown)

	log.I("client.Connect() shutting down")
	return nil
}

// Return a list of connections. Since the client can only be connected to one
// server (for now), this is a degenerate list of <= 1 connections.
func (c *client) GetConnections() []common.Connection {
	defer c.mconnection.Unlock()
	c.mconnection.Lock()
	return []common.Connection{*c.connection}
}

// Get channel by its ID. Since the client can only be connected to one endpoint
// currently, the id field is not used here.
func (c *client) GetConnection(id int) common.Connection {
	defer c.mconnection.Unlock()
	c.mconnection.Lock()
	return *c.connection
}

// Establish a connection to the server and cache the connection in
// receiver.conn
func (c *client) connect() (*net.TCPConn, error) {
	if c.conn != nil {
		return c.conn, nil
	}

	conn, err := net.DialTCP(proto, nil, c.Addr)

	if err != nil {
		log.E("ServerConnection failed to connect to server %v", err)
		return nil, err
	}

	c.conn = conn
	return c.conn, nil
}
