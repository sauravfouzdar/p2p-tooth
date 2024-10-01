package main

import (
	"fmt"
)


func createServer(listenerAddr string, nodes ...string) *FileServer {
	
	// Create a new server instance
	server := &FileServer{
		listenerAddr: listenerAddr,
		nodes:        nodes,



}

func main() {

	server_1 := createServer(":5000")
	server_2 := createServer(":3000")
	server_3 := createServer(":7000", ":5000", ":3000")

	
}