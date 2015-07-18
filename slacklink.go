package main

import "log"

type SlackLink struct {
	Config *Config
}

func (s *SlackLink) Initialize() {
	var err error

	s.Config, err = ParseConfig("config.json")
	if err != nil {
		log.Fatalf("Failed to parse config: %v", err)
	}
}

func (s *SlackLink) Run() {

}
