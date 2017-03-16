package socket

import (
	"bufio"
	"cisco.com/comm/common"
	"cisco.com/comm/log"
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

// Respond to events emitted from the socket server
type ConnectionHandler interface {

	// Called by the client or server once a new connection has been established.
	OnConnect(Pipe, common.Connection, func(int))
}

// ConnectionHandler that echoes back what it receives to the remote endpoint.
// This is only really useful in development as it bypasses normal message
// processing that would otherwise happen in higher levels. Use --handler echo.
type EchoHandler struct {
}

func (e *EchoHandler) OnConnect(p Pipe, c common.Connection, OnTeardown func(int)) {
	log.I("Connected to server")

	reader := bufio.NewReader(os.Stdin)

	// REPL
	for {
		fmt.Print(" [x] Enter message to send (or <Enter> to receive a message or Ctrl-C to exit) => ")

		// Get next message to send from stdin (newline delim)
		msg, _ := reader.ReadString('\n')

		h := Header{Vendor: string(PREAMBLE), Type: MSG_TYPE_CONTROL, Length: uint64(len(msg))}
		msgbytes := append(h.ToBytes(), []byte(msg)...)
		p.Write(msgbytes)

		log.D("Message sent. Waiting for reply...")

		r, err := p.NextMessage()

		if err != nil {
			log.I("Client got error in reply %v", err)
			p.Close()
			OnTeardown(c.Id)
			return
		}

		message, err := ioutil.ReadAll(r)
		echoMessage(r, err, string(message))
	}

	log.I("common.Connection closed")
	OnTeardown(c.Id)
	p.Close()
}

func echoMessage(reader *payloadReader, err error, text string) {
	var t string

	if reader.header.Type == MSG_TYPE_DATA {
		t = "DATA Message"
	} else if reader.header.Type == MSG_TYPE_CONTROL {
		t = "CONTROL Message"
	}

	fmt.Println(
		"\n ######################################################################\n",
		"# Message RECV\n # Type:", t, ".",
		"Length:", reader.header.Length, "\n # Payload:", text, err,
		"\n ######################################################################\n")
}

type ChannelHandler interface {
	OnConnect(Pipe, common.Connection, func(int))
}

// Handler that will pass messages to and from the In and Out channels in
// the Connection object.
type channelHandler struct {
	inflight map[uint64]chan common.IngressMessage
}

func NewChannelHandler() *channelHandler {
	return &channelHandler{inflight: make(map[uint64]chan common.IngressMessage)}
}

func (e *channelHandler) OnConnect(wan Pipe, c common.Connection, OnTeardown func(int)) {
	log.I("connected. Got channel %v", wan)

	go e.readFromWAN(wan, c, OnTeardown)
	go e.writeToWAN(wan, c)
	go listenForLANData(c)

	// Run this synchronously until it dies (which means the WAN has disconnected).
	listenForWANData(c)
}

func (e *channelHandler) writeToWAN(p Pipe, c common.Connection) {
	for m := range c.Out {
		var t byte
		if m.Binary {
			t = MSG_TYPE_DATA
		} else {
			t = MSG_TYPE_CONTROL
		}

		log.I("Got message to write to WAN: %v", m)
		h := Header{Seq: m.Seq, Vendor: string(PREAMBLE), Type: t, Length: uint64(m.N)}
		mr := &PrependHeaderReader{Header: h, InnerReader: m.R}
		n, err := io.Copy(p, mr)

		if err != nil {
			log.E("ERROR copying %v", err)
			c.In <- common.IngressMessage{Err: err}
		}

		e.inflight[m.Seq] = m.ResponseChan
		log.D("Wrote %d bytes. Err: %v", n, err)
	}

	log.I("Shutting down. Channel closed. Channel %v", c)
}

func (e *channelHandler) readFromWAN(p Pipe, conn common.Connection, OnTeardown func(int)) {
	for {
		log.D("READing WAN for NextMessage()")
		r, err := p.NextMessage()
		if err != nil {
			log.I("Got err on p.NextMessage() so closing connection %v", err)
			conn.Close()
			p.Close()
			OnTeardown(conn.Id)
			return
		}

		log.I("RECV new ingress message from WAN. Header: %v", r.header)

		ing := common.IngressMessage{
			Seq:    r.header.Seq,
			N:      int64(r.header.Length),
			R:      r,
			Binary: r.header.Type == MSG_TYPE_DATA,
		}

		c, ok := e.inflight[ing.Seq]

		if !ok {
			// This is a new message
			log.I("RECV done. Got NEW message. Sending on IN channel. Header %v", r.header)
			conn.In <- ing
		} else {
			// This is a response to a previous outbound message
			log.I("RECV done. Got RESPONSE message. Seq %d. Sending on channel %v", ing.Seq, c)

			// Send the response to the channel waiting for it.
			c <- ing
		}
	}
}
