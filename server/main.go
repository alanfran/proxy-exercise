package main

import (
	"flag"

	"github.com/alanfran/proxy-exercise"
)

var (
	localAddress  = flag.String("l", ":9001", "local address")
	remoteAddress = flag.String("o", "www.imdb.com:80", "remote address")
)

func main() {
	flag.Parse()

	proxy, err := proxy.NewProxy(*localAddress, *remoteAddress)
	if err != nil {
		panic(err)
	}

	proxy.Run()
}
