package p2p

import (
	"net"
)


// PNode represents a peer node in the network
type PNode interface {

	net.Conn
	Send ([]byte) error
	CloseStream() error

}

// Transport handles the communication between nodes 
type Transport interface {

	Addr()	string
	Dial(string) error
	ListAndAccept() error
	Consume() <- chan RPC
	Close() error

}
