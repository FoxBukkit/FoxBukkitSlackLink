package main

import (
	"html"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nlopes/slack"
	"gopkg.in/redis.v3"

	messages "github.com/foxelbox/foxbukkitslacklink/messages"
)

func (s *SlackLink) handleSlackMessage(msg *slack.MessageEvent) {
	info := s.slack.GetInfo()

	if msg.User == "" || msg.User == info.User.ID {
		return
	}

	msg.Text = html.UnescapeString(msg.Text)

	if strings.HasPrefix(msg.Text, ".") {
		// Always handle commands, regardless of channel
		s.forwardSlackMessageToChatLink(msg, false)
		return
	}

	channel := info.GetChannelByID(msg.Channel)

	if channel == nil {
		// We don't know about this channel.
		return
	}

	switch channel.Name {
	case "minecraft":
	case "minecraft-ops":
		msg.Text = "#" + msg.Text
	default:
		return
	}

	s.forwardSlackMessageToChatLink(msg, true)
}

func (s *SlackLink) forwardSlackMessageToChatLink(msg *slack.MessageEvent, specialAcknowledgement bool) {
	if strings.HasPrefix(msg.Text, ".") {
		msg.Text = "/" + msg.Text[1:]
	}

	minecraftAccount := s.getMinecraftFromSlack(msg.User)
	if minecraftAccount == nil {
		// They aren't associated with an account. Ignore.
		return
	}

	muuid, _ := uuid.NewRandom()

	cmi := &ChatMessageIn{
		Server:  "Slack",
		Context: muuid,
		Type:    messages.MessageType_TEXT,

		From: minecraftAccount,

		Timestamp: parseSlackTimestamp(msg.Timestamp),

		Contents: msg.Text,
	}

	s.addContextAssociation(cmi.Context, msg.Channel)
	if specialAcknowledgement {
		cleanedMessage := cmi.Contents
		if strings.HasPrefix(cleanedMessage, "#") {
			cleanedMessage = cleanedMessage[1:]
		}

		ref := slack.NewRefToMessage(msg.Channel, msg.Timestamp)
		s.addSpecialAcknowledgementContext(cmi.Context, &ref, cleanedMessage)
	}

	s.chatLinkOut <- CMIToProtoCMI(cmi)
}

func (s *SlackLink) handlePresenceChange(ev *slack.PresenceChangeEvent) {
	mcID, err := s.redis.HGet("slacklinks:slack-to-mc", ev.User).Result()
	if err == redis.Nil {
		return
	} else if err != nil {
		panic(err)
	}

	if ev.Presence == "active" {
		s.redis.SAdd("playersOnline:Slack", mcID)
	} else {
		s.redis.SRem("playersOnline:Slack", mcID)
	}
}

func (s *SlackLink) handleSlackMessages() {
	defer s.wg.Done()

	for msg := range s.slack.IncomingEvents {
		switch data := msg.Data.(type) {
		case *slack.MessageEvent:
			s.handleSlackMessage(data)
		case *slack.PresenceChangeEvent:
			s.handlePresenceChange(data)
		case slack.HelloEvent:
			s.refreshPresenceInfo()
		default:
			//log.Printf("Unhandled message: %T", data)
		}
	}
}

func (s *SlackLink) refreshPresenceInfo() {
	users := s.slack.GetInfo().Users

	slackToMC, err := s.redis.HGetAllMap("slacklinks:slack-to-mc").Result()
	if err != nil {
		panic(err)
	}

	mcIDs := make([]string, 0, len(slackToMC))
	for _, user := range users {
		mcID, ok := slackToMC[user.ID]
		if !ok {
			continue
		}

		if user.Presence != "active" {
			continue
		}

		mcIDs = append(mcIDs, mcID)
	}

	s.redis.Del("playersOnline:Slack")
	s.redis.SAdd("playersOnline:Slack", mcIDs...)
}

func (s *SlackLink) receiveSlackMessages() {
	defer s.wg.Done()

	_, _, err := s.slack.StartRTM()
	if err != nil {
		panic(err)
	}

	s.slackClient.SetUserAsActive()

	s.slack.ManageConnection()

	log.Printf("WARNING: Slack WebSocket died.")
	os.Exit(2)
}

func (s *SlackLink) sendSlackMessages() {
	defer s.wg.Done()

	for msg := range s.slackOut {
		params := slack.NewPostMessageParameters()

		params.Markdown = !msg.DisableMarkdown

		if msg.As != nil {
			params.AsUser = false
			params.Username = msg.As.Name
			params.IconURL = "https://minotar.net/avatar/" + url.QueryEscape(strings.ToLower(msg.As.Name)) + "/48.png"
		} else {
			params.AsUser = true
		}

		s.slack.PostMessage(msg.To, msg.Message, params)
	}
}

type SlackMessage struct {
	To      string
	Message string

	As              *MinecraftPlayer
	DisableMarkdown bool
}

func parseSlackTimestamp(ts string) time.Time {
	f, _ := strconv.ParseFloat(ts, 64)

	tsInt := int64(f)
	leftover := f - float64(tsInt)
	leftoverNsec := int64((float64(time.Second) * leftover) / float64(time.Nanosecond))

	return time.Unix(tsInt, leftoverNsec)
}
