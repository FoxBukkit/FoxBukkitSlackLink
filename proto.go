package main

import (
	"encoding/binary"

	"code.google.com/p/go-uuid/uuid"
	"github.com/golang/protobuf/proto"

	messages "github.com/foxelbox/foxbukkitslacklink/messages"
)

//go:generate protoc --go_out=messages messages.proto

func protoUUIDToUUID(u *messages.UUID) uuid.UUID {
	out := make(uuid.UUID, 16)

	binary.BigEndian.PutUint64(out, uint64(*u.Msb))
	binary.BigEndian.PutUint64(out[8:], uint64(*u.Lsb))

	return out
}

func uuidToProtoUUID(u uuid.UUID) *messages.UUID {
	return &messages.UUID{
		Msb: proto.Int64(int64(binary.BigEndian.Uint64(u[:8]))),
		Lsb: proto.Int64(int64(binary.BigEndian.Uint64(u[8:]))),
	}
}
