package main

import (
	"log"
	"sync"

	"github.com/golang/protobuf/proto"
	zmq "github.com/pebbe/zmq4"

	messages "github.com/foxelbox/foxbukkitslacklink/messages"
)

func receiveMessages(wg *sync.WaitGroup) {
	defer wg.Done()

	incomingMessages, err := zmq.NewSocket(zmq.SUB)
	if err != nil {
		panic(err)
	}
	defer incomingMessages.Close()

	err = incomingMessages.SetSubscribe("CMO")
	if err != nil {
		panic(err)
	}

	err = incomingMessages.Connect("tcp://127.0.0.1:5558")
	if err != nil {
		panic(err)
	}

	log.Printf("Waiting for incoming messages...")
	for {
		topic, err := incomingMessages.Recv(0)
		if err != nil {
			panic(err)
		}

		if topic != "CMO" {
			continue
		}

		chatMessageData, err := incomingMessages.RecvBytes(0)
		if err != nil {
			panic(err)
		}

		chatMessage := new(messages.ChatMessageOut)

		err = proto.Unmarshal(chatMessageData, chatMessage)
		if err != nil {
			panic(err)
		}

		log.Printf("Got chatMessage: %#v", chatMessage)
	}
}
