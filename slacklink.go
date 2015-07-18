package main

import (
	"log"
	"sync"
)

type SlackLink struct {
	Config *Config

	wg *sync.WaitGroup

	chatLinkMessages chan *ChatMessageOut
}

func (s *SlackLink) Initialize() {
	var err error

	s.Config, err = ParseConfig("config.json")
	if err != nil {
		log.Fatalf("Failed to parse config: %v", err)
	}

	s.wg = new(sync.WaitGroup)

	s.chatLinkMessages = make(chan *ChatMessageOut, 8)
}

func (s *SlackLink) Run() {
	s.wg.Add(1)
	go s.receiveChatLinkMessages()

	s.wg.Wait()
}
