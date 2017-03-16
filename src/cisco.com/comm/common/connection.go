package common

import (
	"net"
)

type Connection struct {
	Id     int                   `json:"id"`
	Remote net.Addr              `json:"remote"`
	Out    chan (EgressMessage)  `json:"-"`
	In     chan (IngressMessage) `json:"-"`
}

func (c *Connection) Close() {
	close(c.Out)
	close(c.In)
}
