package main

import (
	"flag"
	"fmt"

	"github.com/emilyzhang/crawlr/graphcrawler"
)

func main() {
	// Get configuration.
	dbDSN := flag.String("dsn", "", "connection data source name")
	maxWorkers := flag.Int("max-workers", 20, "maximum number of workers")
	flag.Parse()

	// Create graph crawler worker and run it.
	w, err := graphcrawler.New(*dbDSN, *maxWorkers)
	if err != nil {
		fmt.Println("Unable to start crawler.")
		panic(err)
	}
	w.Start()
}
