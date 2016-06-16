package main

import (
	"log"

	"github.com/google/uuid"
	"gopkg.in/redis.v3"
)

func (s *SlackLink) getMinecraftFromSlack(userID string) *MinecraftPlayer {
	minecraftID, err := s.redis.HGet("slacklinks:slack-to-mc", userID).Result()
	if err == redis.Nil {
		return nil
	} else if err != nil {
		log.Printf("error in getMinecraftFromSlack: %v", err)
		return nil
	}

	id, err := uuid.Parse(minecraftID)
	if err != nil { // Invalid ID
		return nil
	}

	return s.getMinecraftPlayer(id)
}

func (s *SlackLink) getSlackFromMinecraft(user *MinecraftPlayer) string {
	slackID, err := s.redis.HGet("slacklinks:mc-to-slack", user.UUID.String()).Result()
	if err == redis.Nil {
		return ""
	} else if err != nil {
		log.Printf("error in getSlackFromMinecraft: %v", err)
		return ""
	}

	return slackID
}

func (s *SlackLink) getMinecraftPlayer(id uuid.UUID) *MinecraftPlayer {
	name, err := s.redis.HGet("playerUUIDToName", id.String()).Result()
	if err == redis.Nil {
		return &MinecraftPlayer{
			UUID: id,
		}
	} else if err != nil {
		log.Printf("error in getMinecraftPlayer: %v", err)
		return nil
	}

	return &MinecraftPlayer{
		UUID: id,
		Name: name,
	}
}
