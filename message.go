package proxy

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
)

type MessageReadWriter struct {
	rw io.ReadWriter
}

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

func (m MessageReadWriter) Read(b []byte) (int, error) {
	// var buffer bytes.Buffer
	// nRead, err := m.rw.Read(buffer.Bytes())
	// if nRead == 0 || err != nil {
	// 	log.Println("Read nothing? n:", nRead, " err:", err)
	// 	return nRead, err
	// }

	// log.Println("Read buffer initial size: ", buffer.Len())

	// nb := bytes.NewBuffer(buffer.Next(4))

	// // read n (int32)
	// var n int32
	// err = binary.Read(nb, binary.LittleEndian, &n)
	// if err != nil {
	// 	return 0, err
	// }

	// // read rest of message
	// for {
	// 	if buffer.Len() == int(n) {
	// 		break
	// 	}
	// 	var chunk []byte
	// 	_, err := m.rw.Read(chunk)
	// 	if err != nil {
	// 		break
	// 	}
	// 	buffer.Write(chunk)
	// }

	// nRead, err = buffer.Read(b)

	// return nRead, err

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
