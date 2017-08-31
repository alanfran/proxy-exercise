package proxy

import (
	"bytes"
	"log"
	"net"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	clientAddress = "localhost:9001"
	proxyAddress  = "localhost:9002"
	remoteAddress = "localhost:9003"
)

var _ = Describe("Proxy", func() {
	var proxy *Proxy
	var client *Client
	var remoteListener *net.TCPListener
	var remoteConn *net.TCPConn

	var clientConn *net.TCPConn

	Context("When initialized with valid endpoint addresses", func() {
		BeforeEach(func() {
			var err error

			// proxyTCPAddress, err := net.ResolveTCPAddr("tcp", proxyAddress)
			// Expect(err).ToNot(HaveOccurred())

			clientTCPAddress, err := net.ResolveTCPAddr("tcp", clientAddress)
			Expect(err).ToNot(HaveOccurred())

			remoteTCPAddress, err := net.ResolveTCPAddr("tcp", remoteAddress)
			Expect(err).ToNot(HaveOccurred())

			// Create a remote listener.
			remoteListener, err = net.ListenTCP("tcp", remoteTCPAddress)
			Expect(err).ToNot(HaveOccurred())

			// Create and run the proxy
			proxy = NewProxy(proxyAddress, remoteAddress)
			go proxy.Run()

			// Create and run a client
			client = NewClient(clientAddress, proxyAddress)
			go client.Run()

			// Accept remote listener.
			log.Println("[Remote] Awaiting connection from server...")
			remoteConn, err = remoteListener.AcceptTCP()
			Expect(err).ToNot(HaveOccurred())
			log.Println("[Remote] Server connected.")

			// Connect to client
			clientConn, err = net.DialTCP("tcp", nil, clientTCPAddress)
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			client.Close()
			proxy.Close()
			remoteConn.Close()
			clientConn.Close()
		})

		It("should proxy a connection between client and remote", func() {
			msg := []byte("this is a test message")

			By("sending a message from client to remote")

			nWritten, err := clientConn.Write(msg)
			Expect(err).ToNot(HaveOccurred())

			var buffer bytes.Buffer
			for buffer.Len() != nWritten {
				b := make([]byte, 65535)
				n, err := remoteConn.Read(b)
				Expect(err).ToNot(HaveOccurred())

				_, err = buffer.Write(b[:n])
				Expect(err).ToNot(HaveOccurred())
			}

			Expect(buffer.Bytes()).To(Equal(msg))

			By("sending a message from remote to client")

			nWritten, err = remoteConn.Write(msg)
			Expect(err).ToNot(HaveOccurred())

			buffer.Reset()
			for buffer.Len() != nWritten {
				b := make([]byte, 65535)
				n, err := clientConn.Read(b)
				Expect(err).ToNot(HaveOccurred())

				n, err = buffer.Write(b[:n])
				Expect(err).ToNot(HaveOccurred())
				if n != 0 {
					// log.Println("[Test] Remote received ", n)
				}
			}

			Expect(buffer.Bytes()).To(Equal(msg))
		})
	})
})
