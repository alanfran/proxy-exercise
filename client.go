package proxy

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net"
	"sync"

	"github.com/johnmcconnell/nop"
	"github.com/pkg/errors"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/packet"

	// Needed to encrypt messages.
	_ "golang.org/x/crypto/ripemd160"
)

// Client listens on a local port and sends encrypted messages to a proxy server.
type Client struct {
	localAddress  string
	serverAddress string

	server net.Conn
	client net.Conn

	listener net.Listener

	entity       *openpgp.Entity
	serverEntity *openpgp.Entity

	closed     bool
	closeMutex sync.Mutex
}

// NewClient returns a client with resolved TCP addresses.
func NewClient(localAddress, proxyAddress string) *Client {
	return &Client{
		localAddress:  localAddress,
		serverAddress: proxyAddress,
	}
}

// Run is the client's main loop. It:
//   * connects to the proxy server,
//   * exchanges public keys,
//   * sends encrypted messages to the server,
//   * and decrypts messages from the server.
func (c *Client) Run() (err error) {
	// Connect to the proxy server.
	log.Println("[Client] Connecting to proxy server: ", c.serverAddress)
	c.server, err = net.Dial("tcp", c.serverAddress)
	if err != nil {
		return errors.Wrap(err, "[Client] Error dialing to proxy server.")
	}

	// Generate keypair and send public key to proxy.
	c.entity, err = openpgp.NewEntity("client", "", "", nil)
	if err != nil {
		return errors.Wrap(err, "[Client] Error generating PGP keypair.")
	}

	err = c.entity.SerializePrivate(nop.NewWriter(), nil)
	if err != nil {
		return errors.Wrap(err, "[Client] Error signing PGP keypair.")
	}

	log.Println("[Client] Sending public key...")
	var buffer bytes.Buffer
	err = c.entity.Serialize(&buffer)
	if err != nil {
		return errors.Wrap(err, "[Client] Error serializing public key.")
	}

	_, err = buffer.WriteTo(MessageReadWriter{c.server})
	if err != nil && err != io.EOF {
		return errors.Wrap(err, "[Client] Error sending key to the proxy server.")
	}

	// Await the server's public key.
	log.Println("[Client] Awaiting server's public key...")
	buf, err := MessageReadWriter{c.server}.ReadMessage()
	if err != nil {
		return errors.Wrap(err, "[Client] Error reading server's entity data.")
	}
	message := bytes.NewBuffer(buf)
	c.serverEntity, err = openpgp.ReadEntity(packet.NewReader(message))
	if err != nil {
		return errors.Wrap(err, "[Client] Error creating server entity.")
	}

	log.Println("[Client] Listening on ", c.localAddress, " for requests.")
	// Listen on localAddress for requests.
	c.listener, err = net.Listen("tcp", c.localAddress)
	if err != nil {
		return errors.Wrap(err, "[Client] Error listening for client requests.")
	}

	c.client, err = c.listener.Accept()
	if err != nil {
		return errors.Wrap(err, "[Client] Error accepting client connection.")
	}

	log.Println("[Client] Request received. Proxying data...")

	errChan := make(chan error, 1)

	// server -> client
	go func() {
		for {
			// decrypt
			message, err := openpgp.ReadMessage(
				c.server,
				openpgp.EntityList{c.entity, c.serverEntity},
				nil,
				nil,
			)
			if err != nil {
				errChan <- errors.Wrap(err, "[Client] Error receiving encrypted message.")
				return
			}

			msg, err := ioutil.ReadAll(message.UnverifiedBody)
			if err != nil {
				errChan <- errors.Wrap(err, "[Client] Error reading message.")
				return
			}

			_, err = c.client.Write(msg)
			if err != nil {
				errChan <- errors.Wrap(err, "[Client] Error writing to client.")
				return
			}
		}
	}()

	// client -> server
	go func() {
		for {
			packet := make([]byte, 65535)
			n, err := c.client.Read(packet)
			if err != nil {
				errChan <- errors.Wrap(err, "[Client] Error reading packet from client.")
				return
			}

			buffer := new(bytes.Buffer)
			encrypter, err := openpgp.Encrypt(
				buffer,
				openpgp.EntityList{c.serverEntity},
				nil,
				nil,
				nil,
			)
			if err != nil {
				errChan <- errors.Wrap(err, "[Client] Error creating encrypter.")
				return
			}

			_, err = encrypter.Write(packet[:n])
			if err != nil {
				errChan <- errors.Wrap(err, "[Client] Error writing to encrypter.")
				return
			}
			err = encrypter.Close()
			if err != nil {
				errChan <- errors.Wrap(err, "[Client] Error closing encrypter.")
				return
			}

			message, err := ioutil.ReadAll(buffer)
			if err != nil {
				errChan <- errors.Wrap(err, "[Client] Error reading encrypted message to buffer.")
				return
			}

			n, err = c.server.Write(message)
			if err != nil {
				errChan <- errors.Wrap(err, "[Client] Error writing message to server.")
				return
			}
		}
	}()

	// Catch errors and shut down the client.
	err = <-errChan

	// If the client has been closed, ignore the resulting connection errors.
	c.closeMutex.Lock()
	if c.closed {
		c.closeMutex.Unlock()
		return nil
	}
	c.closeMutex.Unlock()

	// Otherwise, close all connections and return the error.
	c.Close()
	return err
}

// Close stops the client from processing additional messages
// and closes the client's TCP connections and listeners.
func (c *Client) Close() {
	log.Println("[Client] Closing connections...")

	c.closeMutex.Lock()
	c.closed = true
	c.closeMutex.Unlock()

	c.server.Close()
	c.client.Close()
	c.listener.Close()
}
