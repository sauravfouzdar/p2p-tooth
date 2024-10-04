package p2p
 
type HandshakeFunc func (Node) error


func NOPHandshakeFunc(Node) error { return nil }

