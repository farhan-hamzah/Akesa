package notification

import (
	"context"
	"log"
)

type PushClient interface {
	SendPush(ctx context.Context, tokens []string, title, message string, data map[string]string) error
}

type LoggerPushClient struct{}

func NewLoggerPushClient() *LoggerPushClient {
	return &LoggerPushClient{}
}

func (c *LoggerPushClient) SendPush(ctx context.Context, tokens []string, title, message string, data map[string]string) error {
	if len(tokens) == 0 {
		return nil
	}

	log.Printf("[PushGateway] Delivering notification to %d device(s): title=%q message=%q data=%v",
		len(tokens), title, message, data)
	for i, t := range tokens {
		masked := t
		if len(masked) > 12 {
			masked = masked[:8] + "..." + masked[len(masked)-4:]
		}
		log.Printf("[PushGateway]   -> target[%d]: token=%s", i, masked)
	}
	return nil
}
