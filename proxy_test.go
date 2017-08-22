package proxy

import (
	"log"
	"net"
	"time"

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
	var remote *net.TCPConn

	Context("When initialized with valid endpoint addresses", func() {
		BeforeEach(func() {
			var err error

			// proxyTCPAddress, err := net.ResolveTCPAddr("tcp", proxyAddress)
			// Expect(err).ToNot(HaveOccurred())

			// clientTCPAddress, err := net.ResolveTCPAddr("tcp", clientAddress)
			// Expect(err).ToNot(HaveOccurred())

			remoteTCPAddress, err := net.ResolveTCPAddr("tcp", remoteAddress)
			Expect(err).ToNot(HaveOccurred())

			// Create a remote listener.
			remoteListener, err = net.ListenTCP("tcp", remoteTCPAddress)
			Expect(err).ToNot(HaveOccurred())

			// Create and run the proxy
			proxy, err = NewProxy(proxyAddress, remoteAddress)
			Expect(err).ToNot(HaveOccurred())
			go proxy.Run()

			// Create and run a client
			client, err = NewClient(clientAddress, proxyAddress)
			Expect(err).ToNot(HaveOccurred())
			go client.Run()

			// Accept remote listener.
			log.Println("[Remote] Awaiting connection from server...")
			remote, err = remoteListener.AcceptTCP()
			Expect(err).ToNot(HaveOccurred())
			log.Println("[Remote] Server connected.")
		})

		AfterEach(func() {
			client.Close()
			proxy.Close()
			remote.Close()
		})

		It("should proxy a connection from client, through the proxy, to remote", func() {
			// send message through client
			time.Sleep(time.Second * 10)
			// expect remote to receive message
		})
	})
})
