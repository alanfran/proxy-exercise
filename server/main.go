package main

import (
	"flag"
	"log"

	"github.com/alanfran/proxy-exercise"
)

var (
	localAddress  = flag.String("l", ":9001", "local address")
	remoteAddress = flag.String("o", "www.imdb.com:80", "remote address")
)

func main() {
	flag.Parse()

	proxy := proxy.NewProxy(*localAddress, *remoteAddress)

	err := proxy.Run()
	if err != nil {
		log.Fatal(err)
	}
}
