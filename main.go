package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"p2p-tooth/p2p"
	"time"

	"crypto"
	"p2p"
	"store"
)


func createServer(listenerAddr string, nodes ...string) *FileServer {
		TCPTransportOpts := p2p.TCPTransportOpts {
			ListenAddr: listenerAddr,
			HandshakeFunc: p2p.NOPHandshake,
			Decoder: p2p.DefaultDecoder{},
		}
		TCPTransport := p2p.NEWTCPTransport(TCPTransportOpts)
		fileServerOpts := FileServerOpts {
				EncKey : crypto.NewEncryptionKey(),
				StorageRoot: listenerAddr + "_network",
				DefaulPathTransformFunc: store.CASPathTransformFunc,
				Transport: TCPTransport,
				BootstrapNodes: nodes,
		}
		s := NewFileServer(fileServerOpts)
		tcpTransport.OnPeer = s.onPeer 
		return s 
}

func main() {

	server_1 := createServer(":3000", "")
	server_2 := createServer(":5000", "3000")
	server_3 := createServer(":7000", ":5000", ":3000")

	go func () {log.Fatal(s1.Start())}()

	time.Sleep(time.Millisecond * 500)

	go server_3.Start()

	time.Sleep(time.Second * 1)

	for i :=0; i<5; i++ {
			key := fmt.Sprintf("picture_%d",i)
			data := bytes.NewReader([]byte("your file goes here"))
			server_2.Store(key, data)
			time.Sleep(time.Millisecond * 5)

			if err := server_2.store.Delete(server_2.ID, key); err != nil {
				log.Fatal(err)
			}
			b, err := io.ReadAll(r)
			if err != nil {
				log.Fatal(err)
			}
			log.Println("read bytes from the file system:", string (b))
	}
}