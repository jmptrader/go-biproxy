package common

import "io"

// An outbound message over the TCP channel
type EgressMessage struct {

	// Sequence identifier for the message
	Seq uint64

	// Length of this message payload
	N int64

	// Underlying reader
	R io.Reader

	// Whether or not the message payload should be interpreted as binary
	Binary bool

	// Channel to receive the corresponding response message
	ResponseChan chan IngressMessage
}

// An inbound message over the TCP channel
type IngressMessage struct {

	// Sequence identifier for the message
	Seq uint64

	// Length of this message
	N int64

	// Underlying reader
	R io.Reader

	// If the message couldn't be read, report that here instead of a reader
	Err error

	// Whether or not the message payload should be interpreted as binary
	Binary bool
}
