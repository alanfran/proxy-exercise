package main

import (
	"log"

	"github.com/alanfran/proxy-exercise"
)

func main() {
	client, err := proxy.NewClient(":9003", "localhost:9001")
	if err != nil {
		log.Fatal(err)
	}

	err = client.Run()
	if err != nil {
		log.Fatal(err)
	}
}
