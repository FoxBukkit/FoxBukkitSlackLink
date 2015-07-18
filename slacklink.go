package main

import (
	"log"
	"sync"
)

type SlackLink struct {
	Config *Config

	wg *sync.WaitGroup
}

func (s *SlackLink) Initialize() {
	var err error

	s.Config, err = ParseConfig("config.json")
	if err != nil {
		log.Fatalf("Failed to parse config: %v", err)
	}

	s.wg = new(sync.WaitGroup)
}

func (s *SlackLink) Run() {
	s.wg.Add(1)
	go s.receiveChatLinkMessages()

	s.wg.Wait()
}
