package websocket

import (
	"chatterstack/internal/domain/models"
	"time"
)

type sendMessagePayload struct {
	RoomID      string              `json:"room_id"`
	Content     string              `json:"content"`
	Attachments []attachmentPayload `json:"attachments,omitempty"`
}

type attachmentPayload struct {
	ID        string `json:"id,omitempty"`
	MessageID string `json:"message_id,omitempty"`
	URL       string `json:"url"`
	MimeType  string `json:"mime_type,omitempty"`
	SizeBytes int64  `json:"size_bytes,omitempty"`
}

type receiveMessagePayload struct {
	ID          string               `json:"id"`
	RoomID      string               `json:"room_id"`
	SenderID    string               `json:"sender_id"`
	Content     string               `json:"content"`
	Attachments []attachmentPayload  `json:"attachments,omitempty"`
	Status      models.MessageStatus `json:"status"`
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
}

type typingEventPayload struct {
	RoomID string `json:"room_id,omitempty"`
}

type typingBroadcastPayload struct {
	Username string `json:"username"`
	RoomID   string `json:"room_id,omitempty"`
}

func toDomainAttachments(list []attachmentPayload) []models.Attachment {
	if len(list) == 0 {
		return nil
	}

	out := make([]models.Attachment, 0, len(list))
	for _, att := range list {
		out = append(out, models.Attachment{
			ID:        att.ID,
			MessageID: att.MessageID,
			URL:       att.URL,
			MimeType:  att.MimeType,
			SizeBytes: att.SizeBytes,
		})
	}
	return out
}

func toPayloadAttachments(list []models.Attachment) []attachmentPayload {
	if len(list) == 0 {
		return nil
	}

	out := make([]attachmentPayload, 0, len(list))
	for _, att := range list {
		out = append(out, attachmentPayload{
			ID:        att.ID,
			MessageID: att.MessageID,
			URL:       att.URL,
			MimeType:  att.MimeType,
			SizeBytes: att.SizeBytes,
		})
	}
	return out
}
