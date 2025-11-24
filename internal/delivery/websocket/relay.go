package websocket

import (
	"context"
	"encoding/json"
	"log"
	"strings"

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

				var envelope struct {
					Type string `json:"type"`
				}
				if err := json.Unmarshal([]byte(msg.Payload), &envelope); err == nil && envelope.Type != "" {
					switch envelope.Type {
					case "message.deleted":
						var deleted struct {
							Message struct {
								ID     string `json:"id"`
								RoomID string `json:"room_id"`
							} `json:"message"`
						}
						if err := json.Unmarshal([]byte(msg.Payload), &deleted); err != nil {
							log.Printf("websocket: decode message.deleted payload failed: %v", err)
							continue
						}
						roomID := strings.TrimSpace(deleted.Message.RoomID)
						if roomID == "" {
							continue
						}
						event, err := Event{Type: EventMessageDeleted, Data: map[string]string{
							"id":      deleted.Message.ID,
							"room_id": roomID,
						}}.Encode()
						if err != nil {
							log.Printf("websocket: encode message.deleted event failed: %v", err)
							continue
						}
						hub.BroadcastToRoom(roomID, event)
					default:
						log.Printf("websocket: unhandled room event type %q", envelope.Type)
					}
					continue
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
					UpdatedAt:   payload.UpdatedAt,
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
