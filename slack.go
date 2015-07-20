package main

import (
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"code.google.com/p/go-uuid/uuid"
	"github.com/nlopes/slack"
	"gopkg.in/redis.v3"

	messages "github.com/foxelbox/foxbukkitslacklink/messages"
)

func (s *SlackLink) handleSlackMessage(msg *slack.MessageEvent) {
	info := s.slack.GetInfo()

	if msg.UserId == "" || msg.UserId == info.User.Id {
		return
	}

	if strings.HasPrefix(msg.Text, ".") {
		// Always handle commands, regardless of channel
		s.forwardSlackMessageToChatLink(msg)
		return
	}

	channel := info.GetChannelById(msg.ChannelId)

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

	s.forwardSlackMessageToChatLink(msg)
}

func (s *SlackLink) forwardSlackMessageToChatLink(msg *slack.MessageEvent) {
	if strings.HasPrefix(msg.Text, ".") {
		msg.Text = "/" + msg.Text[1:]
	}

	minecraftAccount := s.getMinecraftFromSlack(msg.UserId)
	if minecraftAccount == nil {
		// They aren't associated with an account. Ignore.
		return
	}

	cmi := &ChatMessageIn{
		Server:  "Slack",
		Context: uuid.NewRandom(),
		Type:    messages.MessageType_TEXT,

		From: minecraftAccount,

		Timestamp: parseSlackTimestamp(msg.Timestamp),

		Contents: msg.Text,
	}

	s.addContextAssociation(cmi.Context, msg.ChannelId)

	s.chatLinkOut <- CMIToProtoCMI(cmi)
}

func (s *SlackLink) handlePresenceChange(ev *slack.PresenceChangeEvent) {
	mcID, err := s.redis.HGet("slacklinks:slack-to-mc", ev.UserId).Result()
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

	for msg := range s.slackMessages {
		switch data := msg.Data.(type) {
		case *slack.MessageEvent:
			s.handleSlackMessage(data)
		case *slack.PresenceChangeEvent:
			s.handlePresenceChange(data)
		case slack.HelloEvent:
			s.refreshPresenceInfo()
		case *slack.SlackWSError:
			panic(data)
		default:
			log.Printf("Unhandled message: %T", data)
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
		mcID, ok := slackToMC[user.Id]
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

	rtm, err := s.slack.StartRTM("", "https://slack.com")
	if err != nil {
		panic(err)
	}

	rtm.SetUserAsActive()

	rtm.HandleIncomingEvents(s.slackMessages)
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
