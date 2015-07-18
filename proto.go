package main

import (
	"code.google.com/p/go-uuid/uuid"

	messages "github.com/foxelbox/foxbukkitslacklink/messages"
)

//go:generate protoc --go_out=messages messages.proto

// protoToUUID converts a protobuf UUID (of two 64-bit integers) to a UUID that
// we can use.
func protoToUUID(m messages.UUID) uuid.UUID {
	return nil
}
