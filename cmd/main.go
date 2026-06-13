package main

import (
	"log"

	"github.com/stellhub/stellar"
	"github.com/stellhub/stellatlas-service/internal/server"
)

func main() {
	if err := stellar.Run(stellar.WithStarter(server.NewStarter())); err != nil {
		log.Fatal(err)
	}
}
