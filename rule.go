package con

// Handler - incoming message handler function
type Handler func(msg Msg) (ans []byte)

// Rule provides matching pattern with handler function
type Rule struct {
	handler   Handler
	once      bool
	called    bool
	msgID     []byte
	msgName   string
	msgAuthor string
}
