package main

import "log"

func (s *SlackLink) handleSlackMessages() {
	defer s.wg.Done()

	for msg := range s.slackMessages {
		log.Printf("slack message: %#v", msg)
	}
}

func (s *SlackLink) receiveSlackMessages() {
	defer s.wg.Done()

	rtm, err := s.slack.StartRTM("", "https://slack.com")
	if err != nil {
		panic(err)
	}

	rtm.HandleIncomingEvents(s.slackMessages)
}
