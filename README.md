# p2p-tooth (WIP)
A peer-to-peer content addressable distributed file system 


## Project Structure
```
crypto
│   └─── crypto.go
p2p
│   └─── transport.go
│   └─── tcp_transport.go
│   └─── message.go
│   └─── encoding.go
│   └─── handshake.go
store 
│   └─── store.go
main.go
|
server.go
```

## How to run 
- `go mod tidy` to install dependencies
- `go run .` to start the server

This will spin up 3 nodes, node on port 4000 will create 3 files and broadcast them to other nodes in the network. If ran successfully, you should see 3 folders `:3000_network`, `4000_network`, `5000_network` in the root directory.