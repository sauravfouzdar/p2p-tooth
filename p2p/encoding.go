package p2p

import (
	"io"
	"encoding/gob"

)

type Decoder interface {
	Decode (io.Reader, *RPC) error
}


type GOBDecoder struct {}
type DefaultDecoder struct {}

func (dec GOBDecoder) Decode(r io.Reader, msg *RPC) error {
	return gob.NewDecoder(r).Decode(msg)
}


func (dec DefaultDecoder) Decode(r io.Reader, msg *RPC) error {
	peekBuf := make([]byte, 1)
	if _, err := r.Read(peekBuf); err != nil {
		return nil
	}

	// For stream, we are not decoding the payload over network
	// We are just setting the stream flag to true
	stream := peekBuf[0] == IncomingStream

	if stream{
		msg.Stream = true
		return nil
	}

	buf := make([]byte, 1024)
	n, err := r.Read(buf)
	if err != nil {
		return err
	}

	msg.Payload = buf[:n]
	return nil

}