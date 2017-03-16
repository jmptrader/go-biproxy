package socket

import (
	"bufio"
	"cisco.com/comm/common"
	"cisco.com/comm/log"
	"io"
	"net"
	"net/textproto"
	"regexp"
	"strconv"
	"strings"
)

type RespondableMessage struct {
	Seq             int
	R               io.Reader
	ResponseChannel chan io.Reader
}

type routeKey int

type proxyRequest struct {
	len  int
	dest routeKey
}

func parseRequestLine(line string) (method, requestURI, proto string, ok bool) {
	s1 := strings.Index(line, " ")
	s2 := strings.Index(line[s1+1:], " ")
	if s1 < 0 || s2 < 0 {
		return
	}
	s2 += s1 + 1
	return line[:s1], line[s1+1 : s2], line[s2+1:], true
}

func getRequestLength(conn io.Reader) (int64, io.Reader) {
	rd := bufio.NewReader(conn)
	line, hdr := parseHeader(rd)
	headerstring := writeHeaderToString(line, hdr)

	blen, _ := strconv.ParseInt(hdr.Get("content-length"), 10, 64)
	totallen := int64(len(headerstring)) + blen

	r1 := strings.NewReader(headerstring)
	r2 := &io.LimitedReader{N: blen, R: rd}
	r3 := io.MultiReader(r1, r2)

	return totallen, r3
}

// Encode a MIMEHeader to its HTTP 1.1 text form
func writeHeaderToString(line string, hdr *textproto.MIMEHeader) string {
	headerstring := line + "\r\n"
	for key, value := range *hdr {
		headerstring += key + ": "

		for _, item := range value {
			headerstring += item
		}

		headerstring += "\r\n"
	}

	headerstring += "\r\n"
	return headerstring
}

// Read HTTP 1.1 header from an io.Reader.
// Returns the MIMEHeader as well as the first line of preceeding text.
func parseHeader(rd *bufio.Reader) (string, *textproto.MIMEHeader) {
	tp := textproto.NewReader(rd)
	line, _ := tp.ReadLine()
	hdr, err := tp.ReadMIMEHeader()

	if err != nil {
		log.E("ERR_HEADER_PARSE %v", err)
		return "", nil
	}

	return line, &hdr
}

func onNewWANRequest(conn common.IngressMessage) (*common.EgressMessage, error) {
	egress, err := net.Dial("tcp", "localhost:8080")

	if err != nil {
		log.E("ERR_CON_OPEN %v", err)
		return nil, err
	}

	// Write to the LAN connection
	io.Copy(egress, conn.R)
	log.D("Wrote message to LAN client")

	// Read the response
	tlen, r := getRequestLength(egress)
	log.D("Request body total length %d", tlen)

	res := &common.EgressMessage{
		Seq: conn.Seq,
		N:   tlen,
		R:   r,
	}

	// Send response back to caller
	return res, nil
}

func listenForWANData(conn common.Connection) {
	for {
		log.I("Client waiting for new data from WAN")
		in := <-conn.In

		if in.R == nil {
			log.E("WARNING: Got nil in reader. Stopping.")
			return
		}

		log.D("Got new data from WAN. Opening channel to LAN client. In message:", in)
		res, err := onNewWANRequest(in)

		if err != nil {
			log.E("ERROR handling new WAN request. Aborting %v", err)
			return
		}

		log.D("Sending response back to WAN")
		conn.Out <- *res
	}
}

// Return the sequence number from the net.Conn
func seqFromConn(c net.Conn) uint64 {
	r := regexp.MustCompile("([0-9])+$")
	str := r.FindString(c.RemoteAddr().String())
	id, _ := strconv.ParseInt(str, 10, 32)
	log.D("Generated sequence %v from connection %s", id, c.RemoteAddr().String())
	return uint64(id)
}

// Called when a LAN client connects to this server to send a new message.
func onLANRead(lan net.Conn, wan common.Connection) {
	tlen, r := getRequestLength(lan)
	log.D("Sending request with total length %d", tlen)
	c := make(chan common.IngressMessage)

	// Send the message
	wan.Out <- common.EgressMessage{
		Seq:          seqFromConn(lan),
		N:            tlen,
		R:            r,
		ResponseChan: c}

	// Wait for the response
	res := <-c

	log.D("Got response message %v", res)
	log.I("Closing LAN connection")

	// Send the response back to the caller
	io.Copy(lan, res.R)

	lan.Close()
}

// Respond to TCP connections from the LAN side (sending new messages out).
func listenForLANData(wan common.Connection) {
	lst, err := net.Listen("tcp", ":0")
	if err != nil {
		log.E("ERR_LISTEN %v", err)
	}

	log.I("Listening for LAN connections on TCP %v", lst.Addr())

	for {
		lan, _ := lst.Accept()
		log.I("Got new LAN connection %v", lan.RemoteAddr())
		go onLANRead(lan, wan)
	}
}
