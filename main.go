package main

import (
	"log"
	"sync"
)

// globalWG is the WaitGroup used for the primary Goroutines so main() will only
// exit gracefully.
var globalWG = new(sync.WaitGroup)

func main() {
	log.SetFlags(log.Lshortfile)

	slackLink := new(SlackLink)
	slackLink.Initialize()
	slackLink.Run()
}
