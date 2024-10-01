package p2p

import (
	"net"
	"sync"
	"errors"
	"fmt"
	"log"

)

// tcpNode represents a peer node in the network

type TCPNode struct {

	net.Conn
	// if we dial a connection - outbound is true
	// if we accept a connection - outbound is false
	outbound bool 
	wg *sync.WaitGroup

}

func NEWTCPNode(conn net.Conn, outbound bool) *TCPNode {
	return &TCPNode{
		Conn: conn,
		outbound: outbound,
		wg: &sync.WaitGroup{},
	}
}

func (p *TCPNode) CloseStream(){
	p.wg.Done()
}

// Send sends bytes to the peer node
func (p *TCPNode) Send(data []byte) error {
	_, err := p.Write(data)
	return err
}

// tcp transport opts
type TCPTransportOpts struct {
	ListenerAddr string
	HandshakeFunc HandshakeFunc
	Decoder Decoder
	OnPeer func(Peer) error
}

// tcp transport
type TCPTransport struct {
	opts TCPTransportOpts
	listener net.Listener
	rpcch chan RPC
}


// new connection
func NEWTCPTransport(opts TCPTransportOpts) *TCPTransport {
	return &TCPTransport{
		opts: opts,
		rpcch: make(chan RPC, 1024), // buffer size of 1024 messages
	}
}

func (t *TCPTransport) Addr() string {
	return t.opts.ListenerAddr
}

func (t *TCPTransport) Consume() <- chan RPC {
	return t.listener.Close()
}

// Close implements the Transport interface
func (t *TCPTransport) Close() error {
	return t.listener.Close()
}


// new tcp transport