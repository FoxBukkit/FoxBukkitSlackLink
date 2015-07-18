package main

import (
	"log"

	"github.com/golang/protobuf/proto"
	zmq "github.com/pebbe/zmq4"

	messages "github.com/foxelbox/foxbukkitslacklink/messages"
)

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

	log.Printf("Waiting for messages from ChatLink...")
	for {
		topic, err := chatLinkMessages.Recv(0)
		if err != nil {
			panic(err)
		}

		if topic != "CMO" {
			continue
		}

		chatMessageData, err := chatLinkMessages.RecvBytes(0)
		if err != nil {
			panic(err)
		}

		chatMessage := new(messages.ChatMessageOut)

		err = proto.Unmarshal(chatMessageData, chatMessage)
		if err != nil {
			panic(err)
		}

		readyMessage := ProtoCMOToCMO(chatMessage)

		s.chatLinkMessages <- readyMessage
	}
}
