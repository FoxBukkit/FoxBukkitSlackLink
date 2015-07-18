package main

import (
	"time"

	"code.google.com/p/go-uuid/uuid"

	messages "github.com/foxelbox/foxbukkitslacklink/messages"
)

func ProtoCMOToCMO(cmo *messages.ChatMessageOut) *ChatMessageOut {
	out := &ChatMessageOut{
		ID:              cmo.GetId(),
		Server:          cmo.GetServer(),
		FinalizeContext: cmo.GetFinalizeContext(),
		Type:            cmo.GetType(),

		Contents: cmo.GetContents(),
	}

	context := cmo.GetContext()
	out.Context = protoUUIDToUUID(context)

	out.Timestamp = time.Unix(cmo.GetTimestamp(), 0)

	if from := cmo.GetFrom(); from != nil {
		out.From = &MinecraftPlayer{
			UUID: protoUUIDToUUID(from.GetUuid()),
			Name: from.GetName(),
		}
	}

	if target := cmo.GetTo(); target != nil {
		out.Target = &MessageTarget{
			Type:   target.GetType(),
			Filter: target.GetFilter(),
		}
	}

	return out
}

type MinecraftPlayer struct {
	UUID uuid.UUID
	Name string
}

type MessageTarget struct {
	Type   messages.TargetType
	Filter []string
}

type ChatMessageOut struct {
	ID              int64
	Server          string
	Context         uuid.UUID
	FinalizeContext bool
	Type            messages.MessageType

	Timestamp time.Time

	From   *MinecraftPlayer
	Target *MessageTarget

	Contents string
}
