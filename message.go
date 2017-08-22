package proxy

import (
	"encoding/binary"
	"net"
)

type MessageReadWriter struct {
	conn *net.TCPConn
}

func (m MessageReadWriter) Write(b []byte) (int, error) {
	// write n (int32)
	var n int32
	n = int32(len(b))

	err := binary.Write(m.conn, binary.LittleEndian, n)
	if err != nil {
		return 0, err
	}

	// write message
	nWritten, err := m.conn.Write(b)

	return nWritten, err
}

func (m MessageReadWriter) Read(b []byte) (int, error) {
	// read n (int32)
	var n int32
	err := binary.Read(m.conn, binary.LittleEndian, &n)
	if err != nil {
		return 0, err
	}

	// read message
	var buffer []byte

	for {
		if len(buffer) == int(n) {
			break
		}
		var chunk []byte
		n, err := m.conn.Read(chunk)
		if err != nil {
			return len(buffer) + n, err
		}
		buffer = append(buffer, chunk...)
	}

	return len(buffer), err
}
