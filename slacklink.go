package main

import (
	"bytes"
	"log"
	"sync"

	"github.com/google/uuid"

	"github.com/nlopes/slack"
	"gopkg.in/redis.v3"

	messages "github.com/foxelbox/foxbukkitslacklink/messages"
)

type SlackLink struct {
	Config *Config

	wg *sync.WaitGroup

	slack         *slack.RTM
	slackClient   *slack.Client

	redis *redis.Client

	chatLinkOut chan *messages.ChatMessageIn
	slackOut    chan *SlackMessage

	contextChannels     map[string]string
	contextChannelsLock sync.Mutex

	contextBuffers     map[string]*bytes.Buffer
	contextBuffersLock sync.Mutex

	specialAcknowledgementContexts map[string]struct {
		MustContain string
		Ref         *slack.ItemRef
	}
	specialAcknowledgementContextsLock sync.Mutex
}

func (s *SlackLink) Initialize() {
	s.chatLinkOut = make(chan *messages.ChatMessageIn)
	s.slackOut = make(chan *SlackMessage)

	s.contextChannels = make(map[string]string)
	s.contextBuffers = make(map[string]*bytes.Buffer)
	s.specialAcknowledgementContexts = make(map[string]struct {
		MustContain string
		Ref         *slack.ItemRef
	})

	var err error

	s.Config, err = ParseConfig("config.json")
	if err != nil {
		log.Fatalf("Failed to parse config: %v", err)
	}

	s.wg = new(sync.WaitGroup)

	s.slackClient = slack.New(s.Config.Slack.Token)

	authTest, err := s.slackClient.AuthTest()
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
	s.contextBuffers[context.String()] = bytes.NewBuffer(nil)
	s.contextBuffersLock.Unlock()
}

func (s *SlackLink) removeContextAssociation(context uuid.UUID) {
	s.contextChannelsLock.Lock()
	delete(s.contextChannels, context.String())
	s.contextChannelsLock.Unlock()
	s.contextBuffersLock.Lock()
	delete(s.contextBuffers, context.String())
	s.contextBuffersLock.Unlock()
	s.removeSpecialAcknowledgementContext(context)
}

func (s *SlackLink) addSpecialAcknowledgementContext(context uuid.UUID, ref *slack.ItemRef, mustContain string) {
	s.specialAcknowledgementContextsLock.Lock()
	defer s.specialAcknowledgementContextsLock.Unlock()
	s.specialAcknowledgementContexts[context.String()] = struct {
		MustContain string
		Ref         *slack.ItemRef
	}{
		MustContain: mustContain,
		Ref:         ref,
	}
}

func (s *SlackLink) getSpecialAcknowledgement(context uuid.UUID) (*slack.ItemRef, string, bool) {
	s.specialAcknowledgementContextsLock.Lock()
	defer s.specialAcknowledgementContextsLock.Unlock()
	entry, ok := s.specialAcknowledgementContexts[context.String()]
	if !ok {
		return nil, "", false
	}

	return entry.Ref, entry.MustContain, true
}

func (s *SlackLink) removeSpecialAcknowledgementContext(context uuid.UUID) {
	s.specialAcknowledgementContextsLock.Lock()
	defer s.specialAcknowledgementContextsLock.Unlock()
	delete(s.specialAcknowledgementContexts, context.String())
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
