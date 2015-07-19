package main

import (
	"bytes"
	"log"
	"sync"

	"code.google.com/p/go-uuid/uuid"

	"github.com/nlopes/slack"
	"gopkg.in/redis.v3"

	messages "github.com/foxelbox/foxbukkitslacklink/messages"
)

type SlackLink struct {
	Config *Config

	wg *sync.WaitGroup

	slack         *slack.Slack
	slackMessages chan slack.SlackEvent

	redis *redis.Client

	chatLinkOut chan *messages.ChatMessageIn
	slackOut    chan *SlackMessage

	contextChannels     map[string]string
	contextChannelsLock sync.Mutex

	contextBuffers     map[string]*bytes.Buffer
	contextBuffersLock sync.Mutex
}

func (s *SlackLink) Initialize() {
	s.slackMessages = make(chan slack.SlackEvent)
	s.chatLinkOut = make(chan *messages.ChatMessageIn)
	s.slackOut = make(chan *SlackMessage)

	s.contextChannels = make(map[string]string)
	s.contextBuffers = make(map[string]*bytes.Buffer)

	var err error

	s.Config, err = ParseConfig("config.json")
	if err != nil {
		log.Fatalf("Failed to parse config: %v", err)
	}

	s.wg = new(sync.WaitGroup)

	s.slack = slack.New(s.Config.Slack.Token)

	authTest, err := s.slack.AuthTest()
	if err != nil {
		panic(err)
	}

	log.Printf("Connected to Slack as %q", authTest.User)

	s.redis = redis.NewClient(&redis.Options{
		Addr:     s.Config.Redis.Address,
		Password: s.Config.Redis.Password,
		DB:       s.Config.Redis.DB,
	})

	_, err = s.redis.Ping().Result()
	if err != nil {
		panic(err)
	}
}

func (s *SlackLink) Run() {
	s.wg.Add(5)
	go s.receiveChatLinkMessages()
	go s.sendChatLinkMessages()
	go s.receiveSlackMessages()
	go s.sendSlackMessages()
	go s.handleSlackMessages()

	s.wg.Wait()
}

func (s *SlackLink) SendToChatLink(msg *ChatMessageIn) {
	cmi := CMIToProtoCMI(msg)
	s.chatLinkOut <- cmi
}

func (s *SlackLink) SendToSlack(msg *SlackMessage) {
	s.slackOut <- msg
}

func (s *SlackLink) addContextAssociation(context uuid.UUID, channelID string) {
	s.contextChannelsLock.Lock()
	s.contextChannels[context.String()] = channelID
	s.contextChannelsLock.Unlock()
	s.contextBuffersLock.Lock()
	s.contextBuffers[context.String()] = new(bytes.Buffer)
	s.contextBuffersLock.Unlock()
}

func (s *SlackLink) removeContextAssociation(context uuid.UUID) {
	s.contextChannelsLock.Lock()
	delete(s.contextChannels, context.String())
	s.contextChannelsLock.Unlock()
	s.contextBuffersLock.Lock()
	delete(s.contextBuffers, context.String())
	s.contextBuffersLock.Unlock()
}

func (s *SlackLink) getContextBuffer(context uuid.UUID) *bytes.Buffer {
	s.contextBuffersLock.Lock()
	defer s.contextBuffersLock.Unlock()
	buf, ok := s.contextBuffers[context.String()]
	if !ok {
		buf = new(bytes.Buffer)
		s.contextBuffers[context.String()] = buf
	}

	return buf
}

func (s *SlackLink) appendContextBuffer(context uuid.UUID, str string) bool {
	buf := s.getContextBuffer(context)
	if buf == nil {
		return false
	}

	if buf.Len() > 0 {
		buf.WriteByte('\n')
	}

	buf.WriteString(str)
	return true
}
