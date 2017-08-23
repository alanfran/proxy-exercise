## TCP Proxy

This package implements a TCP proxy that uses PGP encryption.

[Documentation](https://godoc.org/github.com/alanfran/proxy-exercise)

To see this in action run:

`go run server/main.go`

and then

`go run client/main.go`

and point your browser to `localhost:9003`.

You will be able to browse www.imdb.com through the proxy.

### Example Server

The server supports the following flags:

`-l interface:port` - Sets the server's listen address.

`-o address:port` - Sets the address of the remote server. This defaults to `www.imdb.com:80`.

### Example Client

The client supports the following flags:

`-l interface:port` - Sets the client's listen address.

`-o address:port` - Sets the address of the proxy server.