package websocket

import (
	"context"
	"encoding/json"
	"log"

	"chatterstack/internal/domain/models"

	"github.com/redis/go-redis/v9"
)

const roomMessagePattern = "rooms:*:messages"

// PatternSubscriber abstracts Redis pattern subscription capabilities.
type PatternSubscriber interface {
	PatternSubscribe(ctx context.Context, patterns ...string) (<-chan *redis.Message, func() error, error)
}

// StartMessageRelay connects the websocket hub to Redis pub/sub announcements.
func StartMessageRelay(ctx context.Context, hub *Hub, subscriber PatternSubscriber) {
	go func() {
		ch, closeFn, err := subscriber.PatternSubscribe(ctx, roomMessagePattern)
		if err != nil {
			log.Printf("websocket: pattern subscribe failed: %v", err)
			return
		}
		defer func() {
			if closeFn != nil {
				if err := closeFn(); err != nil {
					log.Printf("websocket: pattern unsubscribe failed: %v", err)
				}
			}
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}

				var payload models.Message
				if err := json.Unmarshal([]byte(msg.Payload), &payload); err != nil {
					log.Printf("websocket: invalid redis message payload: %v", err)
					continue
				}
				if payload.RoomID == "" {
					continue
				}

				event, err := Event{Type: EventReceiveMessage, Data: receiveMessagePayload{
					ID:          payload.ID,
					RoomID:      payload.RoomID,
					SenderID:    payload.SenderID,
					Content:     payload.Content,
					Attachments: toPayloadAttachments(payload.Attachments),
					Status:      payload.Status,
					CreatedAt:   payload.CreatedAt,
				}}.Encode()
				if err != nil {
					log.Printf("websocket: encode redis relay event failed: %v", err)
					continue
				}

				hub.BroadcastToRoom(payload.RoomID, event)
			}
		}
	}()
}
