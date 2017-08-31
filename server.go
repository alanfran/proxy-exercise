package proxy

import (
	"bytes"
	"io/ioutil"
	"log"
	"net"
	"sync"

	"github.com/johnmcconnell/nop"
	"github.com/pkg/errors"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/packet"
)

// Proxy implements a proxy server that uses asymmetric PGP encryption.
type Proxy struct {
	localAddress  string
	remoteAddress string

	listener net.Listener
	client   net.Conn
	remote   net.Conn

	closed     bool
	closeMutex sync.Mutex
}

// NewProxy constructs a new Proxy given an address on which it will listen,
// and a remote address to/from which it will proxy traffic.
func NewProxy(localAddress, remoteAddress string) *Proxy {
	proxy := &Proxy{
		localAddress:  localAddress,
		remoteAddress: remoteAddress,
	}

	return proxy
}

// Run makes the proxy process input and send output.
// It:
//   * listens for clients dialing to localAddress,
//   * expects the client to send their public key in the first message,
//   * generates a keypair and sends the public key to the client,
//   * and begins proxying requests between the client and remoteAddress
func (p *Proxy) Run() (err error) {
	log.Println("[Server] Running proxy on ", p.localAddress)

	// Create a TCP listener on localAddress
	p.listener, err = net.Listen("tcp", p.localAddress)
	if err != nil {
		return errors.Wrap(err, "[Server] Error listening for clients.")
	}
	defer p.listener.Close()

	// Wait for a client to connect
	log.Println("[Server] Waiting for client to connect.")
	p.client, err = p.listener.Accept()
	if err != nil {
		return errors.Wrap(err, "[Server] Error accepting client connection.")
	}
	defer p.client.Close()
	log.Println("[Server] Accepted client. Awaiting public key.")

	// get client public key
	buf, err := MessageReadWriter{p.client}.ReadMessage()
	buffer := bytes.NewBuffer(buf)
	if err != nil {
		return errors.Wrap(err, "[Server] Error reading message from client.")
	}

	clientEntity, err := openpgp.ReadEntity(packet.NewReader(buffer))
	if err != nil {
		return errors.Wrap(err, "Error getting client's entity data.")
	}

	log.Println("[Server] Got client public key.")

	// generate and send server public key
	serverEntity, err := openpgp.NewEntity("proxy-server", "temporary", "", nil)
	if err != nil {
		return errors.Wrap(err, "Error generating new PGP entity.")
	}

	// Necessary to make the entity Serializable. See: https://github.com/golang/go/issues/6483
	err = serverEntity.SerializePrivate(nop.NewWriter(), nil)
	if err != nil {
		return errors.Wrap(err, "Error signing serverEntity.")
	}

	log.Println("[Server] Sending public key...")
	buffer.Reset()
	err = serverEntity.Serialize(buffer)
	if err != nil {
		return errors.Wrap(err, "Error serializing serverEntity.")
	}

	n, err := MessageReadWriter{p.client}.Write(buffer.Bytes())
	if err != nil {
		return errors.Wrap(err, "[Server] Error sending key to client.")
	}
	if n != buffer.Len() {
		return errors.New("[Server] Did not send entire key buffer to client")
	}
	log.Println("[Server] Sent public key.")

	// Connect to remoteAddress
	log.Println("[Server] Connecting to remote", p.remoteAddress)
	p.remote, err = net.Dial("tcp", p.remoteAddress)
	if err != nil {
		return errors.Wrap(err, "[Server] Error connecting to remote server.")
	}
	defer p.remote.Close()
	log.Println("[Server] Connected to remote", p.remoteAddress)

	log.Println("[Server] Proxy engaged.")

	errChan := make(chan error, 1)

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
				errChan <- errors.Wrap(err, "[Server] Error decrypting message from client.")
				return
			}

			msg, err := ioutil.ReadAll(message.UnverifiedBody)
			if err != nil {
				errChan <- errors.Wrap(err, "[Server] Error reading message from client.")
				return
			}

			_, err = p.remote.Write(msg)
			if err != nil {
				errChan <- errors.Wrap(err, "[Server] Error writing to remote.")
				return
			}
		}
	}()

	// remote -> client
	go func() {
		for {
			packet := make([]byte, 65535)
			n, err := p.remote.Read(packet)
			if err != nil {
				errChan <- errors.Wrap(err, "[Server] Error reading from remote.")
				return
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
				errChan <- errors.Wrap(err, "[Server] Error encrypting message to client.")
				return
			}

			_, err = encrypter.Write(packet[:n])
			if err != nil {
				errChan <- errors.Wrap(err, "[Server] Error writing encrypted message.")
				return
			}
			err = encrypter.Close()
			if err != nil {
				errChan <- errors.Wrap(err, "[Server] Error closing encrypter.")
				return
			}

			message, err := ioutil.ReadAll(buffer)
			if err != nil {
				errChan <- errors.Wrap(err, "[Server] Error reading encrypted message into buffer.")
				return
			}

			_, err = p.client.Write(message)
			if err != nil {
				errChan <- errors.Wrap(err, "[Server] Error writing message to client.")
				return
			}
		}
	}()

	// Catch errors and shut down the proxy.
	err = <-errChan

	// If the proxy has been closed, ignore the resulting connection errors.
	p.closeMutex.Lock()
	if p.closed {
		p.closeMutex.Unlock()
		return nil
	}
	p.closeMutex.Unlock()

	// Otherwise, close the connections and return an error.
	p.Close()
	return err
}

// Close stops the proxy from trying to process additional messages, and
// closes the underlying tcp connections and listeners.
func (p *Proxy) Close() {
	log.Println("[Server] Closing connections...")

	p.closeMutex.Lock()
	p.closed = true
	p.closeMutex.Unlock()

	p.remote.Close()
	p.client.Close()
	p.listener.Close()
}
