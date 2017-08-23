package proxy

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net"

	"github.com/johnmcconnell/nop"
	"github.com/pkg/errors"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/packet"

	// Needed to encrypt messages.
	_ "golang.org/x/crypto/ripemd160"
)

type Client struct {
	localAddress  *net.TCPAddr
	serverAddress *net.TCPAddr

	server *net.TCPConn
	client *net.TCPConn

	listener *net.TCPListener

	entity       *openpgp.Entity
	serverEntity *openpgp.Entity

	closed bool
}

func NewClient(localAddress, proxyAddress string) (*Client, error) {
	localTCPAddress, err := net.ResolveTCPAddr("tcp", localAddress)
	if err != nil {
		return nil, err
	}

	serverTCPAddress, err := net.ResolveTCPAddr("tcp", proxyAddress)
	if err != nil {
		return nil, err
	}

	return &Client{
		localAddress:  localTCPAddress,
		serverAddress: serverTCPAddress,
	}, err
}

func (c *Client) Run() {
	// Connect to the proxy server.
	log.Println("[Client] Connecting to proxy server: ", c.serverAddress.String())
	server, err := net.DialTCP("tcp", nil, c.serverAddress)
	if err != nil {
		log.Fatal(errors.Wrap(err, "[Client] "))
	}
	c.server = server

	// Generate keypair and send public key to proxy.
	clientEntity, err := openpgp.NewEntity("client", "", "", nil)
	if err != nil {
		log.Fatal(errors.Wrap(err, "[Client] "))
	}

	err = clientEntity.SerializePrivate(nop.NewWriter(), nil)
	if err != nil {
		log.Fatal(errors.Wrap(err, "[Client] "))
	}

	log.Println("[Client] Sending public key...")
	var buffer bytes.Buffer
	err = clientEntity.Serialize(&buffer)
	if err != nil {
		log.Fatal(errors.Wrap(err, "[Client] Serializing public key."))
	}

	_, err = buffer.WriteTo(MessageReadWriter{server})
	// _, err = MessageReadWriter{server}.Write(buffer.Bytes())
	if err != nil && err != io.EOF {
		log.Fatal(errors.Wrap(err, "[Client] Sending serialized key."))
	}

	// Await the server's public key.
	log.Println("[Client] Awaiting server's public key...")
	buf, err := MessageReadWriter{server}.ReadMessage()
	if err != nil {
		log.Fatal(errors.Wrap(err, "[Client] Reading server's entity data."))
	}
	message := bytes.NewBuffer(buf)
	serverEntity, err := openpgp.ReadEntity(packet.NewReader(message))
	if err != nil {
		log.Fatal(errors.Wrap(err, "[Client] Creating server entity."))
	}

	log.Println("[Client] Listening on ", c.localAddress.String(), " for requests.")
	// Listen on localAddress for requests.
	listener, err := net.ListenTCP("tcp", c.localAddress)
	if err != nil {
		log.Fatal(errors.Wrap(err, "[Client] Error Listening: "))
	}
	c.listener = listener

	client, err := listener.AcceptTCP()
	if err != nil {
		log.Fatal(errors.Wrap(err, "[Client] Error Accepting TCP Connection: "))
	}
	c.client = client

	log.Println("[Client] Request received. Proxying data...")

	// server -> client
	go func() {
		for {
			// decrypt
			message, err := openpgp.ReadMessage(
				server,
				openpgp.EntityList{clientEntity, serverEntity},
				nil,
				nil,
			)
			if c.closed {
				return
			}
			if err != nil {
				log.Fatal(errors.Wrap(err, "[Client] Error receiving encrypted message."))
			}

			msg, err := ioutil.ReadAll(message.UnverifiedBody)
			if err != nil {
				log.Fatal(errors.Wrap(err, "[Client] Error reading message."))
			}

			_, err = client.Write(msg)
			if err != nil {
				log.Fatal(errors.Wrap(err, "[Client] Error writing to client."))
			}

		}
	}()

	// client -> server
	for {
		packet := make([]byte, 65535)
		n, err := client.Read(packet)
		if c.closed {
			return
		}
		if err != nil {
			log.Fatal(errors.Wrap(err, "[Client] Error reading packet from client."))
		}

		buffer := new(bytes.Buffer)
		encrypter, err := openpgp.Encrypt(
			buffer,
			openpgp.EntityList{serverEntity},
			nil,
			nil,
			nil,
		)
		if err != nil {
			log.Fatal(errors.Wrap(err, "[Client] Error creating encrypter."))
		}

		_, err = encrypter.Write(packet[:n])
		if err != nil {
			log.Fatal(errors.Wrap(err, "[Client] Error writing to encrypter."))
		}
		err = encrypter.Close()
		if err != nil {
			log.Fatal(errors.Wrap(err, "[Client] Error closing encrypter."))
		}

		message, err := ioutil.ReadAll(buffer)
		if err != nil {
			log.Fatal(errors.Wrap(err, "[Client] Error reading encrypted message to buffer."))
		}

		n, err = server.Write(message)
		if err != nil {
			log.Fatal(errors.Wrap(err, "[Client] Error writing message to server."))
		}
	}

}

func (c *Client) Close() {
	log.Println("[Client] Closing connections...")
	c.closed = true

	c.server.Close()
	c.client.Close()
	c.listener.Close()

}
