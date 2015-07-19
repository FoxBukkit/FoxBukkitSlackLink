package main

import (
	"log"

	"github.com/golang/protobuf/proto"
	zmq "github.com/pebbe/zmq4"

	messages "github.com/foxelbox/foxbukkitslacklink/messages"
)

func (s *SlackLink) handleChatLinkMessage(msg *ChatMessageOut) {
	if msg.Type == messages.MessageType_TEXT {
		messageContents := sanitizeMinecraftHTML(msg.Contents)
		if msg.Server != "" && msg.Server != "Slack" {
			messageContents = "[" + msg.Server + "] " + messageContents
		}

		s.appendContextBuffer(msg.Context, messageContents)
	}

	if msg.FinalizeContext {
		toChannels := s.getDestinationsForMessage(msg)

		if toChannels != nil {
			buf := s.getContextBuffer(msg.Context)

			if buf != nil {
				messageContents := buf.String()

				for _, channel := range toChannels {
					s.slackOut <- &SlackMessage{
						To:              channel,
						Message:         messageContents,
						DisableMarkdown: true,

						As: msg.From,
					}
				}
			}
		}

		s.removeContextAssociation(msg.Context)
	}
}

func (s *SlackLink) getDestinationsForMessage(msg *ChatMessageOut) []string {
	contextString := msg.Context.String()
	s.contextChannelsLock.Lock()
	if responseChannel, ok := s.contextChannels[contextString]; ok {
		s.contextChannelsLock.Unlock()

		return []string{responseChannel}
	}
	s.contextChannelsLock.Unlock()

	if msg.Target == nil || msg.Target.Type == messages.TargetType_ALL {
		return []string{"#minecraft"}
	}

	if msg.Target.Type == messages.TargetType_PERMISSION {
		for _, filter := range msg.Target.Filter {
			if filter == "foxbukkit.opchat" {
				return []string{"#minecraft-ops"}
			}
		}
	}

	return nil
}

func (s *SlackLink) receiveChatLinkMessages() {
	defer s.wg.Done()

	chatLinkMessages, err := zmq.NewSocket(zmq.SUB)
	if err != nil {
		panic(err)
	}
	defer chatLinkMessages.Close()

	err = chatLinkMessages.SetSubscribe("CMO")
	if err != nil {
		panic(err)
	}

	err = ApplyZeroMQConfigs(chatLinkMessages, s.Config.ZeroMQ.BrokerToServer)
	if err != nil {
		panic(err)
	}

	var chatMessageData []byte

	log.Printf("Ready to read messages from ChatLink.")
	for {
		topic, err := chatLinkMessages.Recv(0)
		if err != nil {
			panic(err)
		}

		if topic != "CMO" {
			continue
		}

		chatMessageData, err = chatLinkMessages.RecvBytes(0)
		if err != nil {
			panic(err)
		}

		chatMessage := new(messages.ChatMessageOut)

		err = proto.Unmarshal(chatMessageData, chatMessage)
		if err != nil {
			panic(err)
		}

		readyMessage := ProtoCMOToCMO(chatMessage)
		s.handleChatLinkMessage(readyMessage)
	}
}

func (s *SlackLink) sendChatLinkMessages() {
	defer s.wg.Done()

	chatLinkOut, err := zmq.NewSocket(zmq.PUSH)
	if err != nil {
		panic(err)
	}
	defer chatLinkOut.Close()

	err = ApplyZeroMQConfigs(chatLinkOut, s.Config.ZeroMQ.ServerToBroker)
	if err != nil {
		panic(err)
	}

	for msg := range s.chatLinkOut {
		data, err := proto.Marshal(msg)
		if err != nil {
			panic(err)
		}

		_, err = chatLinkOut.SendBytes(data, zmq.DONTWAIT)
		if err != nil {
			panic(err)
		}
	}
}
