package proxy

import (
	"bytes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Message", func() {
	It("works", func() {
		var buffer bytes.Buffer
		messager := MessageReadWriter{rw: &buffer}

		msg := []byte("test")
		result := make([]byte, len(msg))

		// log.Println("Writing: ", msg)
		n, err := messager.Write(msg)
		Expect(err).ToNot(HaveOccurred())
		Expect(n).To(Equal(len(msg)))

		n, err = messager.Read(result)
		// log.Println("Read: ", result)

		Expect(err).ToNot(HaveOccurred())
		Expect(n).ToNot(BeZero())

		Expect(result).To(Equal(msg))

	})
})
