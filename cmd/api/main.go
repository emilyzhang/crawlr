package main

import (
	"flag"
	"fmt"

	"github.com/emilyzhang/crawlr/api"
)

func main() {
	// Get configuration.
	dbDSN := flag.String("dsn", "", "connection data source name")
	flag.Parse()

	// Create api server and run it.
	s, err := api.New(*dbDSN)
	if err != nil {
		fmt.Println("Unable to start API server.")
		panic(err)
	}
	s.Start()
}
