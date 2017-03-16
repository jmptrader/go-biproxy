package socket

import (
	"cisco.com/comm/log"
	"bytes"
	"encoding/binary"
	"errors"
	"io"
)

//////////////////////////////////////////////////////////////////////////////////
// Header constants. All lengths are in bytes
//////////////////////////////////////////////////////////////////////////////////

// 'cisco'
var PREAMBLE = []byte{0x63, 0x69, 0x73, 0x63, 0x6f}

// Length of the message type field
const TYPE_LEN = 1

// Length of the payload length field
const LEN_LEN = 8

// Length of sequence identifier field
const SEQ_LEN = 8

// Total length of the header
var HEADER_LEN = len(PREAMBLE) + TYPE_LEN + LEN_LEN + SEQ_LEN

// Offset into header before length field
var LEN_OFF = len(PREAMBLE) + TYPE_LEN

// Offset into header before sequence field
var SEQ_OFF = LEN_OFF + LEN_LEN

const (
	MSG_TYPE_CONTROL = iota
	MSG_TYPE_DATA
)

type Header struct {
	Vendor string
	Type   byte
	Length uint64
	Seq    uint64
}

func (h *Header) ToBytes() []byte {
	lenbytes := make([]byte, LEN_LEN)
	seqbytes := make([]byte, SEQ_LEN)
	binary.BigEndian.PutUint64(lenbytes, h.Length)
	binary.BigEndian.PutUint64(seqbytes, h.Seq)
	res := append(append(append([]byte(h.Vendor), h.Type), lenbytes...), seqbytes...)
	log.D("Writing header %v to bytes %v", h, res)
	return res
}

// Construct a new header from a connection by reading the first
// HEADER_LEN bytes from the connection stream.
func NewHeader(conn io.Reader) (*Header, error) {
	header := make([]byte, HEADER_LEN)
	read := 0

	for read < HEADER_LEN {
		hbuf := make([]byte, HEADER_LEN-read)
		n, err := conn.Read(hbuf)

		for i := 0; i < n; i++ {
			header[read+i] = hbuf[i]
		}

		read += n
		log.D("Read %d header bytes. Total Header %d", n, read)
		if err != nil {
			log.W("No (or incomplete header) received")
			return nil, err
		}
	}

	if !bytes.Equal(header[:len(PREAMBLE)], PREAMBLE) {
		log.W("ERROR: Malformed header. Wrong preamble %v", header[:HEADER_LEN])
		return nil, errors.New("ERR_EQUALITY")
	}

	h := &Header{
		Vendor: string(header[:len(PREAMBLE)]),
		Type:   header[len(PREAMBLE)],
		Length: binary.BigEndian.Uint64(header[LEN_OFF : LEN_OFF+LEN_LEN]),
		Seq:    binary.BigEndian.Uint64(header[SEQ_OFF : SEQ_OFF+SEQ_LEN]),
	}
	return h, nil
}
