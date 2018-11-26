package con

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"
	"math/rand"
	"time"
)

var alph = [64]rune{
	'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l',
	'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z',
	'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L',
	'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z',
	'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '-', '_',
}

// UID generates simple uid string
func UID() string {
	// Get time part
	ns := uint64(time.Now().UnixNano())

	// Get rand part
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	rd := rnd.Uint64()

	// Result string
	var output bytes.Buffer

	// Rand part
	for i := 0; i < 7; i++ {
		output.WriteRune(alph[rd&63])
		rd >>= 6
	}

	// Time part
	for i := 0; i < 5; i++ {
		output.WriteRune(alph[ns&63])
		ns >>= 6
	}

	return output.String()
}

// BID12 generates 12-bytes id
func BID12() [12]byte {
	var out [12]byte

	// Get time part
	ns := uint64(time.Now().UnixNano())
	out[0] = byte((ns & 0xff000000) >> 24)
	out[1] = byte((ns & 0x00ff0000) >> 16)
	out[2] = byte((ns & 0x0000ff00) >> 8)
	out[3] = byte(ns & 0x000000ff)

	// Get rand part
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	rd := rnd.Uint64()
	out[4] = byte((rd & 0xff00000000000000) >> 56)
	out[5] = byte((rd & 0x00ff000000000000) >> 48)
	out[6] = byte((rd & 0x0000ff0000000000) >> 40)
	out[7] = byte((rd & 0x000000ff00000000) >> 32)
	out[8] = byte((rd & 0x00000000ff000000) >> 24)
	out[9] = byte((rd & 0x0000000000ff0000) >> 16)
	out[10] = byte((rd & 0x000000000000ff00) >> 8)
	out[11] = byte(rd & 0x00000000000000ff)

	return out
}

// BinEq check equality of bin slices
func BinEq(a, b []byte) bool {
	if (a == nil) != (b == nil) {
		return false
	}

	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

// Write message to stream
func writeMsg(stream io.Writer, id [12]byte, meta byte, name string, body []byte) error {
	metaArr := [1]byte{meta}
	nameBytes := []byte(name)
	nameLen := len(nameBytes)
	bodyLen := uint64(len(body))
	buf := make([]byte, 0, 22+nameLen+len(body))
	msgBuff := bytes.NewBuffer(buf)

	if nameLen > 255 {
		log.Fatal("Cannot use message name longer then 255 bytes")
	}

	_, err := msgBuff.Write(id[0:12])
	if err != nil {
		return err
	}
	_, err = msgBuff.Write(metaArr[0:1])
	if err != nil {
		return err
	}
	err = binary.Write(msgBuff, binary.BigEndian, uint8(nameLen))
	if err != nil {
		return err
	}
	_, err = msgBuff.Write(nameBytes[:])
	if err != nil {
		return err
	}
	if bodyLen > 0 {
		err = binary.Write(msgBuff, binary.BigEndian, bodyLen)
		if err != nil {
			return err
		}
		_, err = msgBuff.Write(body[:bodyLen])
		if err != nil {
			return err
		}
	}

	_, err = stream.Write(msgBuff.Bytes())
	return err
}

// Continuously read stream and parse
// incomming data to messages. Will Stop if
// msgHandler return false.
func readStream(clientID string, stream io.Reader, msgHandler func(msg Msg) bool) {
	msgBuff := make([]byte, 0, 1024)
	chunkBuff := make([]byte, 256)
	var readLen uint64
	var id []byte
	var meta byte
	var withBody bool
	var nameLen uint64
	var name string
	var nameEnd uint64
	var bodyStart uint64
	var bodyEnd uint64
	var bodyLen uint64
	var body []byte
	var msgEnd uint64

	for {
		n, err := stream.Read(chunkBuff[:])
		if err != nil {
			break
		}

		msgBuff = append(msgBuff, chunkBuff[0:n]...)
		readLen += uint64(n)

		for len(msgBuff) > 0 {
			// Id
			if id == nil && readLen >= 12 {
				id = make([]byte, 12)
				copy(id, msgBuff[0:12])
			}

			// Meta
			if meta == 0 && readLen >= 13 {
				meta = msgBuff[12]
				withBody = (meta & MsgWithBody) == MsgWithBody
			}

			// Name len
			if nameLen == 0 && readLen >= 14 {
				nameLen = uint64(msgBuff[13])
				nameEnd = 14 + nameLen
				if !withBody {
					msgEnd = nameEnd
				}
			}

			// Name
			if nameLen > 0 && readLen >= nameEnd {
				name = string(msgBuff[14:nameEnd])
			}

			// Body len
			if withBody && nameLen > 0 && bodyLen == 0 && readLen >= nameEnd+8 {
				bodyLen = binary.BigEndian.Uint64(msgBuff[nameEnd : nameEnd+8])
				bodyStart = nameEnd + 8
				bodyEnd = bodyStart + bodyLen
				msgEnd = bodyEnd
			}

			// Body
			if withBody && bodyLen > 0 && readLen >= bodyEnd {
				body = make([]byte, bodyLen)
				copy(body, msgBuff[bodyStart:bodyEnd])
			}

			// Full message
			if msgEnd > 0 && readLen >= msgEnd {
				if !msgHandler(Msg{
					ID:     id,
					Author: clientID,
					Meta:   meta,
					Name:   name,
					Body:   body,
				}) {
					return
				}

				// Cleanup
				id = nil
				meta = 0
				nameLen = 0
				bodyLen = 0
				name = ""
				body = nil
				if readLen == msgEnd {
					msgBuff = msgBuff[:0]
					readLen = 0
					msgEnd = 0
					break
				} else {
					msgBuff = msgBuff[msgEnd:]
					readLen -= msgEnd
					msgEnd = 0
					continue
				}
			} else {
				break
			}
		}
	}

	msgHandler(Msg{
		Author: clientID,
		Name:   "disconnect",
	})
}
