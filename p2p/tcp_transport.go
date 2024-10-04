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
	ListenAddr string
	HandshakeFunc HandshakeFunc
	Decoder Decoder
	OnPeer func(PNode) error
}

// tcp transport
type TCPTransport struct {
	TCPTransportOpts
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
	return t.ListenAddr
}

func (t *TCPTransport) Consume() <- chan RPC {
	return t.rpcch
}

// Close implements the Transport interface
func (t *TCPTransport) Close() error {
	return t.listener.Close()
}

// Dial implements the Transport interface
func (t *TCPTransport) Dial(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}

	go t.handleConn(conn, true)
	return nil
}

func (t *TCPTransport) ListenAndAccept() error {
	var err error 
	
	t.listener, err = net.Listen("tcp", t.ListenAddr)

	if err != nil {
		return err
	}

	go t.startAcceptLoop()
	log.Printf("Listening on port: %s", t.ListenAddr)
	return nil

}


func (t *TCPTransport) startAcceptLoop() {

		for {
			conn, err := t.listener.Accept()
			if errors.Is(err, net.ErrClosed) {
				return
			}

			if err != nil {
				log.Printf("Error accepting connection: %s", err)
				continue
			}
			go t.handleConn(conn, false)
		}
}

func (t *TCPTransport) handleConn(conn net.Conn, outbound bool) {
	
	var err error
	defer func(){
		fmt.Print("Closing connection")
		conn.Close()
	}()

	node := NEWTCPNode(conn, outbound)
	if err = t.HandshakeFunc(node); err != nil {
		conn.Close()
		log.Printf("Handshake failed: %s", err)
		return
	}
	if t.OnPeer != nil {
		if err = t.OnPeer(node); err != nil {
			conn.Close()
			log.Printf("OnPeer failed: %s", err)
			return
		}
	}

	// Read loop
	for {
		rpc := RPC{}
		err := t.Decoder.Decode(conn, &rpc)
		if err != nil {
			log.Printf("Error decoding message: %s", err)
			return
		}
		rpc.FROM = conn.RemoteAddr().String()
		if rpc.Stream {
			node.wg.Add(1)
			fmt.Println("[%s] incoming stream waiting....\n", rpc.FROM)
			node.wg.Wait()
			fmt.Println("[%s] incoming stream done, resuming read loop.. \n", rpc.FROM)
			continue
		}
		t.rpcch <- rpc

	}
}