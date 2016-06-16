package main

//go:generate protoc --go_out=messages messages.proto

import (
	"encoding/binary"
	"time"

	"github.com/google/uuid"
	"github.com/golang/protobuf/proto"

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

func CMIToProtoCMI(cmi *ChatMessageIn) *messages.ChatMessageIn {
	out := &messages.ChatMessageIn{
		Server:  proto.String(cmi.Server),
		Context: uuidToProtoUUID(cmi.Context),
		Type:    &cmi.Type,

		Timestamp: proto.Int64(cmi.Timestamp.Unix()),
	}

	if cmi.From != nil {
		out.From = &messages.UserInfo{
			Uuid: uuidToProtoUUID(cmi.From.UUID),
			Name: proto.String(cmi.From.Name),
		}
	}

	if len(cmi.Contents) > 0 {
		out.Contents = proto.String(cmi.Contents)
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

type ChatMessageIn struct {
	Server  string
	Context uuid.UUID
	Type    messages.MessageType

	From *MinecraftPlayer

	Timestamp time.Time

	Contents string
}

func protoUUIDToUUID(u *messages.UUID) uuid.UUID {
	var out uuid.UUID

	binary.BigEndian.PutUint64(out[:], uint64(u.GetMsb()))
	binary.BigEndian.PutUint64(out[8:], uint64(u.GetLsb()))

	return out
}

func uuidToProtoUUID(u uuid.UUID) *messages.UUID {
	return &messages.UUID{
		Msb: proto.Int64(int64(binary.BigEndian.Uint64(u[:8]))),
		Lsb: proto.Int64(int64(binary.BigEndian.Uint64(u[8:]))),
	}
}
