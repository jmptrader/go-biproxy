package api

import (
	"cisco.com/comm/common"
	"cisco.com/comm/log"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
)

type ConnectionsIndexResponse struct {
	Connections []common.Connection `json:"connections"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

type Controller struct {
	Server SocketServer
}

func jsonResponse(w http.ResponseWriter, payload interface{}) error {
	w.Header().Set("content-type", "application/json")
	res, err := json.Marshal(payload)
	w.Write(res)
	return err
}

//
// GET	/connections		Return a list of clients connected to this server.
//
func (c *Controller) Connections(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		res := ErrorResponse{Message: fmt.Sprintf("%s not allowed here", r.Method)}
		jsonResponse(w, res)
		return
	}

	cons := ConnectionsIndexResponse{c.Server.GetConnections()}
	res, _ := json.Marshal(cons)
	w.Write(res)
}

//
// PUT	/{id}		Send arbitrary data to a connected client given by {id}.
// GET /{id}		Synchronously receives data from connected client given by {id}
//							NOTE if client is not connected, both of these will fail fast.
//
// Ex: File upload via cURL: curl -XPUT localhost:3500/1 -F file=@test.bin
// Ex: Shell binary: curl -XPUT localhost:3500/1 -d 'asdf'
//
func (c *Controller) Transceiver(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[13:]
	rx, _ := regexp.Compile("^([0-9]{1,20})$")
	id := rx.FindString(path)

	if id == "" {
		w.WriteHeader(http.StatusPreconditionFailed)
		jsonResponse(w, ErrorResponse{Error: "ERR_NO_CUSTOMER_ID"})
		return
	}

	connid, err := strconv.ParseInt(id, 10, 32)

	if err != nil {
		w.WriteHeader(http.StatusNotAcceptable)
		jsonResponse(w, ErrorResponse{
			Error:   err.Error(),
			Message: "Failed to parse connection ID. Double check it against GET /connections"})
		return
	}

	switch r.Method {
	case "GET":
		c.Receive(w, r, int(connid))
	case "PUT":
		c.Transmit(w, r, int(connid))
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		res := ErrorResponse{Message: fmt.Sprintf("%s not allowed", r.Method)}
		jsonResponse(w, res)
	}
}

func (c *Controller) Transmit(
	w http.ResponseWriter,
	r *http.Request,
	connid int) {

	sz, err := strconv.ParseInt(r.Header.Get("content-length"), 10, 64)

	if err != nil {
		w.WriteHeader(http.StatusPreconditionFailed)
		jsonResponse(w, ErrorResponse{
			Error:   err.Error(),
			Message: "Failed to parse content-length header. Please make sure it's set."})
		return
	}

	X := common.EgressMessage{N: sz, R: r.Body, Binary: true}
	conn := c.Server.GetConnection(int(connid)).Out

	select {
	case conn <- X:
		log.D(" Sent data to connection %d. Waiting for replay", connid)
		c.Receive(w, r, connid)
	default:
		log.D("ERROR: Tried to send data to socket server but server is not ready")
		jsonResponse(w, errors.New("ERR_SOCKET_NOT_READY"))
	}
}

func (c *Controller) Receive(
	w http.ResponseWriter,
	r *http.Request,
	connid int) {

	log.D("Beginning Receive()")

	cn, ok := w.(http.CloseNotifier)

	if !ok {
		log.W("WARNING: Failed to get CloseNotifier. Can't reliably receive if client disconnects")
	}

	con := c.Server.GetConnection(connid).In

	select {
	case msg := <-con:
		log.D("RECV ingress msg from socket server")
		c.HandleResponse(w, r, msg, 0)
	case <-cn.CloseNotify():
		log.D("CloseNotify API client disconnected")
	}

	log.D("Exiting Receive()")
}

func (c *Controller) HandleResponse(
	w http.ResponseWriter,
	r *http.Request,
	resr common.IngressMessage,
	sz int64) {

	if resr.Err != nil {
		w.WriteHeader(http.StatusNotAcceptable)
		jsonResponse(w, ErrorResponse{
			Error:   resr.Err.Error(),
			Message: "Transmit failed. Make sure the socket is connected (hint: GET /connections)",
		})
	} else if resr.R == nil {
		w.WriteHeader(http.StatusInternalServerError)
		jsonResponse(w, ErrorResponse{
			Error:   "ERR_REMOTE_NA",
			Message: "Remote client went away before response was received.",
		})
	} else {
		io.Copy(w, resr.R)
	}
}
