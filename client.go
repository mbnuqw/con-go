package con

import (
	"sync"
)

// Client struct.
type Client struct {
	sync.RWMutex
	ID     string
	stream Conn
	rules  []Rule
}

// Connect - try to connect to provided address.
func (c *Client) Connect(address string, name string) (err error) {
	c.stream, err = Dial(address)
	if err != nil {
		return err
	}

	// Handshake (sync)
	err = c.handshake(name)
	if err != nil {
		return err
	}

	// Start msg listener
	go c.handleResponses()
	return nil
}

// Send - without waiting of answer.
func (c *Client) Send(name string, body []byte) error {
	if c.ID == "" {
		return ErrDisconnected
	}

	id := BID12()
	meta := byte(0)
	if body != nil || len(body) > 0 {
		meta = MsgWithBody
	}

	c.Lock()
	err := writeMsg(c.stream, id, meta, name, body)
	c.Unlock()
	if err != nil {
		return err
	}

	return nil
}

// Req - with answer
func (c *Client) Req(name string, body []byte, ch chan Msg) error {
	if c.ID == "" {
		return ErrDisconnected
	}

	id := BID12()
	meta := MsgReq
	if body != nil {
		meta |= MsgWithBody
	}

	// Add handler for answer
	c.Lock()
	c.rules = append(c.rules, Rule{
		handler: func(msg Msg) (ans []byte) {
			ch <- msg
			return
		},
		once:    true,
		msgID:   id[:],
		msgName: name,
	})

	err := writeMsg(c.stream, id, meta, name, body)
	c.Unlock()
	if err != nil {
		return err
	}

	return nil
}

// On subscribes on message by its name.
func (c *Client) On(msgName string, h Handler) {
	c.Lock()
	c.rules = append(c.rules, Rule{
		handler: h,
		msgName: msgName,
	})
	c.Unlock()
}

// Disconnect - close connection.
func (c *Client) Disconnect() error {
	c.Lock()
	c.ID = ""
	err := c.stream.Close()
	c.Unlock()
	return err
}

func (c *Client) handleResponses() {
	readStream("server", c.stream, func(msg Msg) bool {
		c.Lock()
		// Find handler
		for i := range c.rules {
			r := &c.rules[i]
			matched := true
			if r.msgID != nil {
				matched = matched && BinEq(r.msgID, msg.ID)
			}
			if r.msgName != "" {
				matched = matched && r.msgName == msg.Name
			}
			if matched {
				// Call it and remove if needed
				handler := r.handler
				if r.once {
					if !r.called {
						go handler(msg)
						r.called = true
					}
				} else {
					go handler(msg)
				}
			}
		}
		c.Unlock()

		return true
	})
}

// Handshake with server
func (c *Client) handshake(name string) error {
	id := BID12()
	err := writeMsg(c.stream, id, MsgWithBody|MsgReq, "handshake", []byte(name))
	if err != nil {
		return err
	}

	// Listen for answer
	readStream("server", c.stream, func(msg Msg) bool {
		// Skip non-handshake responses
		if msg.Name != "handshake" {
			return true
		}

		c.Lock()
		c.ID = string(msg.Body)
		c.Unlock()
		return false
	})

	return nil
}
