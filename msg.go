package con

// Message meta
const (
	MsgWithBody = byte(1 << 7)
	MsgReq      = byte(1 << 6)
)

// Msg type
type Msg struct {
	ID     []byte
	Author string
	Meta   byte
	Name   string
	Body   []byte
}
