package main

import (
	"log"
	"sync"

	"github.com/nlopes/slack"
)

type SlackLink struct {
	Config *Config

	wg *sync.WaitGroup

	slack         *slack.Slack
	slackMessages chan slack.SlackEvent
}

func (s *SlackLink) Initialize() {
	var err error

	s.Config, err = ParseConfig("config.json")
	if err != nil {
		log.Fatalf("Failed to parse config: %v", err)
	}

	s.wg = new(sync.WaitGroup)

	s.slackMessages = make(chan slack.SlackEvent, 8)
	s.slack = slack.New(s.Config.Slack.Token)

	authTest, err := s.slack.AuthTest()
	if err != nil {
		panic(err)
	}

	log.Printf("Connected to Slack as %q", authTest.User)
}

func (s *SlackLink) Run() {
	s.wg.Add(3)
	go s.receiveChatLinkMessages()
	go s.receiveSlackMessages()
	go s.handleSlackMessages()

	s.wg.Wait()
}
