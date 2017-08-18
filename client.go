package proxy

import (
	"bytes"
	"io/ioutil"
	"log"
	"net"

	"github.com/johnmcconnell/nop"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/packet"
)

type Client struct {
	localAddress  *net.TCPAddr
	serverAddress *net.TCPAddr

	server *net.TCPConn
	client *net.TCPConn

	listener *net.TCPListener

	entity       *openpgp.Entity
	serverEntity *openpgp.Entity
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

func (c *Client) Run() error {
	// Connect to the proxy server.
	log.Println("Connecting to proxy server: ", c.serverAddress.String())
	server, err := net.DialTCP("tcp", nil, c.serverAddress)
	if err != nil {
		return err
	}

	// Generate keypair and send public key to proxy.
	clientEntity, err := openpgp.NewEntity("client", "", "", nil)
	if err != nil {
		return err
	}

	err = clientEntity.SerializePrivate(nop.NewWriter(), nil)
	if err != nil {
		return err
	}

	log.Println("Exchanging public keys...")
	err = clientEntity.Serialize(server)
	if err != nil {
		return err
	}

	// Await the server's public key.
	serverEntity, err := openpgp.ReadEntity(packet.NewReader(server))
	if err != nil {
		return err
	}

	log.Println("Listening on ", c.localAddress.String(), " for requests.")
	// Listen on localAddress for requests.
	listener, err := net.ListenTCP("tcp", c.localAddress)
	if err != nil {
		return err
	}

	client, err := listener.AcceptTCP()
	if err != nil {
		return err
	}

	log.Println("Request received. Proxying data...")

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
			if err != nil {
				log.Fatal(err)
			}

			msg, err := ioutil.ReadAll(message.UnverifiedBody)
			if err != nil {
				log.Fatal(err)
			}

			_, err = client.Write(msg)
			if err != nil {
				log.Fatal(err)
			}

		}
	}()

	// client -> server
	for {
		packet := make([]byte, 65535)
		_, err := client.Read(packet)
		if err != nil {
			log.Fatal(err)
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

		_, err = server.Write(message)
		if err != nil {
			log.Fatal(err)
		}
	}

}
