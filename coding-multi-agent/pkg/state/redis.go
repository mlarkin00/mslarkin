package state

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mslarkin/coding-multi-agent/pkg/agent"
	"github.com/mslarkin/coding-multi-agent/pkg/config"
	"github.com/redis/go-redis/v9"
)

type Service struct {
	client *redis.Client
}

func NewService(cfg config.RedisConfig) *Service {
	client := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
	})
	return &Service{client: client}
}

func (s *Service) AppendMessage(ctx context.Context, sessionID string, msg agent.Message) error {
	key := fmt.Sprintf("session:%s:history", sessionID)
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	// Push to list
	if err := s.client.RPush(ctx, key, data).Err(); err != nil {
		return err
	}
	// Set expiry for 24h
	s.client.Expire(ctx, key, 24*time.Hour)
	return nil
}

func (s *Service) GetHistory(ctx context.Context, sessionID string) ([]agent.Message, error) {
	key := fmt.Sprintf("session:%s:history", sessionID)
	data, err := s.client.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		return nil, err
	}

	var messages []agent.Message
	for _, item := range data {
		var msg agent.Message
		if err := json.Unmarshal([]byte(item), &msg); err != nil {
			continue // skip bad data
		}
		messages = append(messages, msg)
	}
	return messages, nil
}

func (s *Service) SaveArtifact(ctx context.Context, sessionID, artifactID, content string) error {
	key := fmt.Sprintf("session:%s:artifacts:%s", sessionID, artifactID)
	if err := s.client.Set(ctx, key, content, 24*time.Hour).Err(); err != nil {
		return err
	}
	return nil
}

func (s *Service) GetArtifact(ctx context.Context, sessionID, artifactID string) (string, error) {
	key := fmt.Sprintf("session:%s:artifacts:%s", sessionID, artifactID)
	return s.client.Get(ctx, key).Result()
}
