package p2p

const (
	IncomingStream = 1
	IncomingMessage = 2
)

type RPC struct {
	FROM string
	Payload []byte
	Stream bool
}

