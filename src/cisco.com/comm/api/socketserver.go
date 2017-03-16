package api

import (
	"cisco.com/comm/common"
)

// A socket server receives data from the API, serializes it, and sends it over
// the TCP channel to the socket server on the recipient side for the reverse
// set of operations. This interface is implemented by the
// cisco.com/comm/socket package. There is only one such implementation. This
// interface is needed to (1) avoid direct coupling to that implementation and
// more importantly (2) because socket has dependencies on the API package and
// you cannot have circular dependencies in GO >_<
type SocketServer interface {
	GetConnection(int) common.Connection
	GetConnections() []common.Connection
}
