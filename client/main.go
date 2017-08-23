package main

import (
	"flag"
	"log"

	"github.com/alanfran/proxy-exercise"
)

var (
	localAddress = flag.String("l", ":9003", "local address")
	proxyAddress = flag.String("o", "localhost:9001", "proxy server address")
)

func main() {
	client, err := proxy.NewClient(*localAddress, *proxyAddress)
	if err != nil {
		log.Fatal(err)
	}

	client.Run()
}
