package proxy

import (
	"bytes"
	"io/ioutil"
	"log"
	"net"

	"github.com/johnmcconnell/nop"
	"github.com/pkg/errors"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/packet"
)

// Proxy implements a proxy server that uses asymmetric PGP encryption.
type Proxy struct {
	localAddress  *net.TCPAddr
	remoteAddress *net.TCPAddr

	listener *net.TCPListener
	client   *net.TCPConn
	remote   *net.TCPConn
}

// NewProxy constructs a new Proxy given the following parameters, and will return an error
// if it cannot resolve the given addresses.
//
// Parameters:
//   * a local address
//   * a remote address
func NewProxy(localAddress, remoteAddress string) (*Proxy, error) {
	local, err := net.ResolveTCPAddr("tcp", localAddress)
	if err != nil {
		return nil, err
	}

	remote, err := net.ResolveTCPAddr("tcp", remoteAddress)
	if err != nil {
		return nil, err
	}

	proxy := &Proxy{
		localAddress:  local,
		remoteAddress: remote,
	}

	return proxy, err
}

// Run makes the proxy process input, and send output.
// It:
//   * listens for clients dialing to localAddress,
//   * expects the client to send their public key in the first message,
//   * generates a keypair and sends the public key to the client,
//   * and begins proxying requests between the client and remoteAddress
func (p *Proxy) Run() {
	var err error

	log.Println("[Server] Running proxy on ", p.localAddress.String())

	// Create a TCP listener on localAddress
	p.listener, err = net.ListenTCP("tcp", p.localAddress)
	if err != nil {
		log.Fatal(err)
	}
	defer p.listener.Close()

	// Wait for a client to connect
	log.Println("[Server] Waiting for client to connect.")
	p.client, err = p.listener.AcceptTCP()
	if err != nil {
		log.Fatal(err)
	}
	defer p.client.Close()
	log.Println("[Server] Accepted client. Awaiting public key.")

	// get client public key
	clientEntity, err := openpgp.ReadEntity(packet.NewReader(p.client))
	if err != nil {
		log.Fatal(errors.Wrap(err, "Error getting client's entity data."))
	}

	log.Println("[Server] Got client public key.")

	// generate and send server public key
	serverEntity, err := openpgp.NewEntity("proxy-server", "temporary", "", nil)
	if err != nil {
		log.Fatal(errors.Wrap(err, "Error generating new PGP entity."))
	}

	// Necessary to make the entity Serializable. See: https://github.com/golang/go/issues/6483
	err = serverEntity.SerializePrivate(nop.NewWriter(), nil)
	if err != nil {
		log.Fatal(errors.Wrap(err, "Error signing serverEntity."))
	}

	log.Println("[Server] Sending public key...")
	err = serverEntity.Serialize(p.client)
	if err != nil {
		log.Fatal(errors.Wrap(err, "Error serializing serverEntity and sending to client."))
	}

	log.Println("[Server] Sent public key.")

	// Connect to remoteAddress
	log.Println("Connecting to remote", p.remoteAddress.String())
	p.remote, err = net.DialTCP("tcp", nil, p.remoteAddress)
	if err != nil {
		log.Fatal(err)
	}
	defer p.remote.Close()
	log.Println("[Server] Connected to remote", p.remoteAddress.String())

	// begin proxying
	log.Println("[Server] Proxy engaged.")
	// client -> remote
	go func() {
		for {
			// decrypt
			message, err := openpgp.ReadMessage(
				p.client,
				openpgp.EntityList{clientEntity, serverEntity},
				nil,
				nil,
			)
			if err != nil {
				log.Fatal(err)
			}

			msg, err := ioutil.ReadAll(message.UnverifiedBody)
			if err != nil {
				log.Fatal(err)
			}

			_, err = p.remote.Write(msg)
			if err != nil {
				log.Fatal(err)
			}

		}
	}()

	// remote -> client
	for {
		packet := make([]byte, 65535)
		_, err := p.remote.Read(packet)
		if err != nil {
			log.Fatal(err)
		}

		buffer := new(bytes.Buffer)
		encrypter, err := openpgp.Encrypt(
			buffer,
			openpgp.EntityList{clientEntity},
			nil,
			nil,
			nil,
		)
		if err != nil {
			log.Fatal(err)
		}

		_, err = encrypter.Write(packet)
		if err != nil {
			log.Fatal(err)
		}
		err = encrypter.Close()
		if err != nil {
			log.Fatal(err)
		}

		message, err := ioutil.ReadAll(buffer)
		if err != nil {
			log.Fatal(err)
		}

		_, err = p.client.Write(message)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func (p *Proxy) Close() {
	p.remote.Close()
	p.client.Close()
	p.listener.Close()
}
