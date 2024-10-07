package main

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/sauravfouzdar/p2p-tooth/crypto"
	"github.com/sauravfouzdar/p2p-tooth/p2p"
	"github.com/sauravfouzdar/p2p-tooth/store"
)


type FileServerOpts struct {
		ID string
		EncKey []byte
		StorageRoot string 
		PathTransformFunc store.PathTransformFunc
		Transport p2p.Transport
		BootstrapNodes []string 
}

type FileServer struct {
		FileServerOpts

		peerLock sync.Mutex 
		peers map[string]p2p.Node 
		store *store.Store 
		quitch chan struct {}
}

func NewFileServer(opts FileServerOpts) *FileServer {
		storeOpts := store.StoreOpts {
				Root: opts.StorageRoot,
				PathTransformFunc: opts.PathTransformFunc,
		}

		if len(opts.ID) == 0 { opts.ID = crypto.GenerateID() }

		return &FileServer {
				FileServerOpts: opts,
				store: store.NewStore(storeOpts),
				quitch: make(chan struct{}),
				peers: make(map[string]p2p.Node),
		}
}

func (s *FileServer) broadcast (msg *Message) error {
		buf := new(bytes.Buffer)
		if err := gob.NewEncoder(buf).Encode(msg); err != nil {
				return fmt.Errorf("error while encoding broadcast %v", err)
		}
		for _, peer := range s.peers {
				peer.Send([]byte{p2p.IncomingMessage})
				if err := peer.Send(buf.Bytes()); err != nil {
						return err 
				}
		}
		return nil 
}

type Message struct {
	Payload any
}

type MessageStoreFile struct {
	ID string 
	Key string 
	Size int64
}

type MessageGetFile struct {
	ID string 
	Key string 
}

type MessageDeleteFile struct {
	ID string 
	Key string 
}

func (s *FileServer) Get (key string) (io.Reader, error) {
		if s.store.Has(s.ID, key) {
				log.Printf("[%s] serving file (%s) from local disk\n", s.Transport.Addr(), key)
				_, r, err := s.store.Read(s.ID, key)
				return r, err 
		}

		fmt.Printf("[%s] doesn't have file (%s) locally, fetching from network....\n", s.Transport.Addr(), key)

		msg := Message {
				Payload: MessageGetFile{
						ID: s.ID,
						Key: crypto.HashKey(key),
				},
		}

		//Broadcast the key to other nodes to check
		// if they have the file 
		if err := s.broadcast(&msg); err != nil {
				return nil, err 
		}

		time.Sleep(time.Millisecond * 500)

		for _, peer := range s.peers {
			// check file size
			var fileSize int64 
			binary.Read(peer, binary.LittleEndian, &fileSize)
			n, err := s.store.WriteDecrypt(s.EncKey, s.ID, key, io.LimitReader(peer, fileSize))
			if err != nil {
				return nil, err 
			}
			fmt.Printf("[%s] received (%d) bytes over the network from [%s]\n", s.Transport.Addr(), n, peer.RemoteAddr().String())

			peer.CloseStream()
		}

		_, r, err := s.store.Read(s.ID, key)
		return r, err
}

func (s *FileServer) Store (key string, r io.Reader) error {
		var (
				fileBuffer = new(bytes.Buffer)
				tee = io.TeeReader(r, fileBuffer)
		)

		size, err := s.store.Write(s.ID, key, tee);
		if err != nil {
				return err 
		}
		msg := Message {
				Payload: MessageStoreFile {
					ID: s.ID,
					Key: crypto.HashKey(key),
					Size: size + 16,
				},
		}

		// Broadcast key,size of msg to all nodes
		if err := s.broadcast(&msg); err != nil {
				return err 
		}

		time.Sleep(time.Millisecond * 5)

		peers := []io.Writer{}
		for _, peer := range s.peers {
				peers = append(peers, peer)
		}
		mw := io.MultiWriter(peers...)
		mw.Write([]byte{p2p.IncomingStream})
		n, err := crypto.CopyEncrypt(s.EncKey, fileBuffer, mw)
		if err != nil {
				return err 
		}

		fmt.Printf("[%s] Received and written (%d) bytes to disk\n", s.Transport.Addr(), n)

		return nil 
}

func (s *FileServer) Delete (key string) error {
		err := s.store.Delete(s.ID, key);
		if err != nil {
				return err
		}
			
		msg := Message {
				Payload: MessageDeleteFile {
						ID: s.ID,
						Key: crypto.HashKey(key),
				},
		}
		

		//sending key,size of msg to all nodes
		fmt.Printf("[%s] sending delete command to all nodes in the network\n", s.Transport.Addr())
		if err := s.broadcast(&msg); err != nil {
				return err 
		}

		time.Sleep(time.Millisecond * 5)

		return nil 
}

func (s *FileServer) Stop() {
	close (s.quitch)
}

func (s *FileServer) onPeer (p p2p.Node) error {
	s.peerLock.Lock()
	defer s.peerLock.Unlock()
	s.peers[p.RemoteAddr().String()] = p 

	log.Printf("connected with remote: %s", p.RemoteAddr())

	return nil
}

func (s *FileServer) loop() {
		defer func () {
				fmt.Println("file server stopped due to error or user quit action")
				s.Transport.Close()
		}()
		for {
				select {
				case rpc := <- s.Transport.Consume():
						var msg Message
						if err := gob.NewDecoder(bytes.NewReader(rpc.Payload)).Decode(&msg); err != nil {
								log.Println("error while decoding received message:", err)
						}
						if err := s.handleMessage(rpc.FROM, &msg); err != nil {
								log.Println("handle message error:", err)
						}
				case <- s.quitch:
						return
				}
		}
}

func (s *FileServer) handleMessage (from string, msg* Message) error {
		switch v := msg.Payload.(type) {
		case MessageStoreFile:
				return s.handleMessageStoreFile(from, v)
		case MessageGetFile:
				return s.handleMessageGetFile(from, v)
		case MessageDeleteFile:
				return s.handleMessageDeleteFile(from, v)
		}
		return nil
}

func (s *FileServer) handleMessageGetFile (from string, msg MessageGetFile) error {
		if !s.store.Has(msg.ID, msg.Key) {
				return fmt.Errorf("[%s] need to serve file (%s) but it does not exist on disk", s.Transport.Addr(), msg.Key)
		}

		log.Printf("[%s] serving file (%s) over the network\n", s.Transport.Addr(), msg.Key)

		fileSize, r, err := s.store.Read(msg.ID, msg.Key);
		if err != nil {
				return err 
		}

		if rc, ok := r.(io.ReadCloser); ok {
				log.Println("closing ReadCloser")
				defer rc.Close()
		}

		peer, ok := s.peers[from]
		if !ok {
				return fmt.Errorf("peer %s not in map", peer)
		}

		// Send IncomingStream byte, then send fileSize
		peer.Send([]byte{p2p.IncomingMessage})
		binary.Write(peer, binary.LittleEndian, fileSize)
		n, err := io.Copy(peer, r)
		if err != nil {
				return err 
		}
		fmt.Printf("[%s] written (%d) bytes over the network to %s\n", s.Transport.Addr(), n, from)

		return nil
}

func (s *FileServer) handleMessageStoreFile (from string, msg MessageStoreFile) error {
		peer, ok := s.peers[from]
		if !ok {
				return fmt.Errorf("peer (%s) could not be found in the peer map", from)
		}
		n, err := s.store.Write(msg.ID, msg.Key, io.LimitReader(peer, msg.Size));
		if err != nil {
				return err 
		}
		log.Printf("[%s] written (%d) bytes to disk\n", s.Transport.Addr(), n)

		peer.CloseStream()

		return nil
}

func (s *FileServer) handleMessageDeleteFile (from string, msg MessageDeleteFile) error {
		if !s.store.Has(msg.ID, msg.Key) {
				return fmt.Errorf("[%s] need to delete file (%s) but it does not exist on disk", s.Transport.Addr(), msg.Key)
		}

		peer, ok := s.peers[from]
		if !ok {
				return fmt.Errorf("peer %s not in map", peer)
		}

		log.Printf("[%s] successfully deleted file (%s)", s.Transport.Addr(), msg.Key)

		if err := s.store.Delete(msg.ID, msg.Key); err != nil {
			return fmt.Errorf("[%s] error while deleting file (%s): %v", s.Transport.Addr(), msg.Key, err)
		}

		log.Printf("[%s] successfully deleted file (%s)", s.Transport.Addr(), msg.Key)

		return nil
}

func (s *FileServer) BootstrapNetwork() error {
		for _, addr := range s.BootstrapNodes {
				if len(addr) == 0 {
					continue
				}
				go func (addr string) {
					fmt.Printf("[%s] attemting to connect with remote:%s\n", s.Transport.Addr(), addr)
					if err := s.Transport.Dial(addr); err != nil {
							fmt.Println("dial error", err)
					}
				}(addr)
		}
		return nil 
}

func (s *FileServer) Start () error {
		if err := s.Transport.ListenAndAccept(); err != nil {
				return err 
		}
		s.BootstrapNetwork()
		s.loop()
		return nil 
}

func init () {
		gob.Register(MessageStoreFile{})
		gob.Register(MessageDeleteFile{})
		gob.Register(MessageGetFile{})
}