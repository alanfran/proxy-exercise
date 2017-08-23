package proxy

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
)

// MessageReadWriter wraps an io.ReadWriter and reads/writes
// sized messages.
type MessageReadWriter struct {
	rw io.ReadWriter
}

// Write writes the length of a message, and then the message to the underlying io.ReadWriter.
func (m MessageReadWriter) Write(b []byte) (int, error) {
	// write n (int32)
	var n int32
	n = int32(len(b))

	err := binary.Write(m.rw, binary.LittleEndian, n)
	if err != nil {
		return 0, err
	}

	// write message
	nWritten, err := m.rw.Write(b)

	return nWritten, err
}

// Read reads a sized message into the given byte slice.
func (m MessageReadWriter) Read(b []byte) (int, error) {
	var n int32
	var nN int
	for n == 0 {
		nBuf := make([]byte, 4)
		nN, err := io.ReadFull(m.rw, nBuf)
		if (nN != 4 || err != nil) && err != io.EOF {
			return nN, err
		}

		nb := bytes.NewBuffer(nBuf)
		err = binary.Read(nb, binary.LittleEndian, &n)
		if err != nil && err != io.EOF {
			return nN, err
		}
	}

	mBuf := make([]byte, n)
	buffer := bytes.NewBuffer(mBuf)

	nRead, err := io.ReadFull(m.rw, mBuf)
	// fmt.Println("ReadFull read: ", nRead, "/", n, " bytes.")
	if err != nil && err != io.EOF {
		return nN + nRead, err
	}

	// buffer := bytes.NewBuffer(mBuf)
	// // log.Println("Initial b: ", b)
	// nBody, err := buffer.Read(b)
	nBody, err := io.ReadAtLeast(buffer, b, nRead)
	// fmt.Println("Read: ", nBody, "/", n, " bytes.")

	// log.Println("Length of b: ", len(b))
	// log.Println("Message length: ", nBody)
	// log.Println("Read ", nBody)
	// log.Println("Post b: ", b)
	// log.Println("Buffer bytes: ", buffer.Bytes())

	return nN + nBody, err
}

// ReadMessage reads a sized message from the underlying io.ReadWriter
// and returns it in a byte slice.
func (m MessageReadWriter) ReadMessage() ([]byte, error) {
	var n int32
	for n == 0 {
		nBuf := make([]byte, 4)
		nN, err := io.ReadFull(m.rw, nBuf)
		if (nN != 4 || err != nil) && err != io.EOF {
			return []byte{}, err
		}

		nb := bytes.NewBuffer(nBuf)
		err = binary.Read(nb, binary.LittleEndian, &n)
		if err != nil && err != io.EOF {
			return []byte{}, err
		}
	}

	mBuf := make([]byte, n)

	nRead, err := io.ReadFull(m.rw, mBuf)
	if err != nil {
		return []byte{}, err
	}
	if nRead < int(n) {
		return []byte{}, errors.New("Did not read entire message.")
	}
	return mBuf, err
}
