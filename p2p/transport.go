package p2p

import (
	"net"
)


// PNode represents a peer node in the network
type Node interface {

	net.Conn
	Send ([]byte) error
	CloseStream() error

}

// Transport handles the communication between nodes 
type Transport interface {

	Addr()	string
	Dial(string) error
	ListenAndAccept() error
	Consume() <- chan RPC
	Close() error

}
