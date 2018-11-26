package con

import (
	"io"
	"net"
	"os"
	"runtime"
	"strings"
	"sync"
	"syscall"
)

// ConnectedClient - id, name and stream of connected client
type ConnectedClient struct {
	ID     string
	Name   string
	Stream net.Conn
}

// Server - server struct
type Server struct {
	sync.Mutex
	Clients []ConnectedClient
	rules   []Rule
}

// Listen start listening for incomming clients.
// Will block current goroutine.
func (s *Server) Listen(address string) error {
	// Get connetion type
	var connType string
	if strings.HasPrefix(address, "/") && strings.HasSuffix(address, ".sock") {
		connType = "unix"
	} else {
		connType = "tcp"
	}

	// Setup listener
	listener, err := setupListener(connType, address)
	if err != nil {
		return err
	}

	// Add handler for handshake
	s.Lock()
	s.rules = append(s.rules, Rule{
		handler: s.handshakeHandler,
		msgName: "handshake",
	})
	s.Unlock()

	// Listen for -> clients -> messages
	err = s.handleClients(listener)
	if err != nil {
		return err
	}

	return nil
}

// ListenAll - listen all addresses in separated goroutines.
func (s *Server) ListenAll(addresses []string) {
	for i := range addresses {
		go s.Listen(addresses[i])
	}
}

// On subscribes for some messages.
func (s *Server) On(clientName string, msgName string, handler Handler) {
	s.Lock()
	s.rules = append(s.rules, Rule{
		handler:   handler,
		msgAuthor: clientName,
		msgName:   msgName,
	})
	s.Unlock()
}

// Once subscribes for the next message.
func (s *Server) Once(clientName string, msgName string, handler Handler) {
	s.Lock()
	s.rules = append(s.rules, Rule{
		handler:   handler,
		msgAuthor: clientName,
		msgName:   msgName,
		once:      true,
	})
	s.Unlock()
}

// Broadcast sends message to all connected clients.
func (s *Server) Broadcast(msgName string, body []byte) error {
	s.Lock()
	for i := range s.Clients {
		c := s.Clients[i]
		var meta byte
		if body != nil {
			meta |= MsgWithBody
		}
		err := writeMsg(c.Stream, BID12(), meta, msgName, body)
		if err != nil {
			return err
		}
	}
	s.Unlock()
	return nil
}

// Send sends message to client by its name.
func (s *Server) Send(clientName string, msgName string, body []byte) error {
	s.Lock()
	for i := range s.Clients {
		c := s.Clients[i]
		if c.Name == clientName {
			var meta byte
			if body != nil {
				meta |= MsgWithBody
			}
			err := writeMsg(c.Stream, BID12(), meta, msgName, body)
			if err != nil {
				return err
			}
		}
	}
	s.Unlock()
	return nil
}

// Disconnect disconnects client by its id or name
func (s *Server) Disconnect(client string) error {
	s.Lock()
	for i := range s.Clients {
		if s.Clients[i].ID == client || s.Clients[i].Name == client {
			err := s.Clients[i].Stream.Close()
			if err != nil {
				return err
			}
			break
		}
	}
	s.Unlock()
	return nil
}

// Handle handshake message.
func (s *Server) handshakeHandler(msg Msg) []byte {
	i := 0
	s.Lock()
	for i = range s.Clients {
		c := &s.Clients[i]
		if c.ID == msg.Author {
			// Update client name
			c.Name = string(msg.Body)

			// Answer to client
			var id [12]byte
			copy(id[:], msg.ID)
			break
		}
	}
	s.Unlock()
	return []byte(s.Clients[i].ID)
}

// Try to setup listener. For unix socket, if got error
// 'address already in use' - will try to recreate socket.
func setupListener(connType string, address string) (listener net.Listener, err error) {
	listener, err = net.Listen(connType, address)
	if err == nil {
		return listener, nil
	}
	if opErr, ok := err.(*net.OpError); ok {
		if sysErr, ok := opErr.Err.(*os.SyscallError); ok {
			if errno, ok := sysErr.Err.(syscall.Errno); ok {
				if errno == syscall.EADDRINUSE || runtime.GOOS == "windows" && errno == 10048 {
					err = os.Remove(address)
					listener, err = net.Listen(connType, address)
				}
			}
		}
	}
	return
}

// Handle clients in current goroutine. Create new goroutine for
// each client.
func (s *Server) handleClients(listener net.Listener) error {
	for {
		stream, err := listener.Accept()
		if err != nil {
			return err
		}

		// Create and store new client
		clientID := UID()
		s.Lock()
		client := ConnectedClient{
			ID:     clientID,
			Name:   "",
			Stream: stream,
		}
		s.Clients = append(s.Clients, client)
		s.Unlock()

		// -> handleMessages
		go s.handleMessages(clientID, stream)
	}
}

// Handle client messages in current goroutine.
func (s *Server) handleMessages(clientID string, stream io.ReadWriter) {
	readStream(clientID, stream, func(msg Msg) bool {
		s.Lock()
		// Find handler
		for i := range s.rules {
			r := &s.rules[i]
			matched := true
			if r.msgID != nil {
				matched = matched && BinEq(r.msgID, msg.ID)
			}
			if r.msgName != "" {
				matched = matched && r.msgName == msg.Name
			}
			if r.msgAuthor != "" {
				for i := range s.Clients {
					c := &s.Clients[i]
					if c.ID == msg.Author {
						matched = matched && r.msgAuthor == c.Name
						break
					}
				}
			}
			if matched {
				// Call it and remove if needed
				if r.once {
					if !r.called {
						go s.handleMessage(msg, r, stream)
						r.called = true
					}
				} else {
					go s.handleMessage(msg, r, stream)
				}
			}
		}
		s.Unlock()

		return true
	})

	// Client was disconnected, cleanup
	s.Lock()
	var i int
	for i = range s.Clients {
		if s.Clients[i].ID == clientID {
			// Disconnected
			break
		}
	}
	s.Clients = append(s.Clients[:i], s.Clients[i+1:]...)
	s.Unlock()
}

func (s *Server) handleMessage(msg Msg, rule *Rule, stream io.ReadWriter) {
	handler := rule.handler
	ans := handler(msg)

	// Write answer
	if msg.Meta&MsgReq == MsgReq {
		var meta byte
		if ans != nil {
			meta = MsgWithBody
		}
		var id [12]byte
		copy(id[:], msg.ID)
		writeMsg(stream, id, meta, msg.Name, ans)
	}
}
