package proxy

import (
	"net"

	"github.com/johnmcconnell/nop"
	"golang.org/x/crypto/openpgp"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	localAddress  = "localhost:9001"
	remoteAddress = "localhost:9002"
)

var _ = Describe("Proxy", func() {
	var p *Proxy
	var client *net.TCPConn
	var remoteListener *net.TCPListener
	var remote *net.TCPConn

	Context("When initialized with valid endpoint addresses", func() {
		BeforeEach(func() {
			var err error

			localTCPAddress, err := net.ResolveTCPAddr("tcp", localAddress)
			Expect(err).ToNot(HaveOccurred())

			remoteTCPAddress, err := net.ResolveTCPAddr("tcp", remoteAddress)
			Expect(err).ToNot(HaveOccurred())

			p, err = NewProxy(localAddress, remoteAddress)
			Expect(err).ToNot(HaveOccurred())

			// Create a remote listener.
			remoteListener, err = net.ListenTCP("tcp", remoteTCPAddress)
			Expect(err).ToNot(HaveOccurred())

			// Run the proxy
			go p.Run()

			// Connect a client
			client, err = net.DialTCP("tcp", nil, localTCPAddress)
			Expect(err).ToNot(HaveOccurred())

			// Accept remote listener.
			remote, err = remoteListener.AcceptTCP()
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			client.Close()
			p.Close()
		})

		It("should take client's public key and send server's public key", func() {
			clientEntity, err := openpgp.NewEntity("client", "", "", nil)
			Expect(err).ToNot(HaveOccurred())

			// Necessary. See: https://github.com/golang/go/issues/6483
			err = clientEntity.SerializePrivate(nop.NewWriter(), nil)
			Expect(err).ToNot(HaveOccurred())

			// Send client public key.
			err = clientEntity.Serialize(client)
			Expect(err).ToNot(HaveOccurred())

			// Wait for server public key
			// Deadlock?

			//buffer := make([]byte, 65535)
			//n, err := client.Read(buffer)
			//Expect(err).ToNot(HaveOccurred())
			//log.Println(buffer[:n])
		})
	})
})
