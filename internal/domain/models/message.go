package models

import "time"

// MessageStatus enumerates the lifecycle states of a message.
type MessageStatus string

const (
	// MessageStatusSent indicates the message has been persisted.
	MessageStatusSent MessageStatus = "SENT"
	// MessageStatusDelivered indicates the recipient is online and received it.
	MessageStatusDelivered MessageStatus = "DELIVERED"
	// MessageStatusRead indicates the recipient opened the message.
	MessageStatusRead MessageStatus = "READ"
)

// Message captures a single chat payload persisted to storage.
type Message struct {
	ID          string
	RoomID      string
	SenderID    string
	Content     string
	Attachments []Attachment
	Status      MessageStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Attachment represents supplemental payload metadata for a message.
type Attachment struct {
	ID        string
	MessageID string
	URL       string
	MimeType  string
	SizeBytes int64
}

// MessageReceipt tracks delivery/read acknowledgements for a message.
type MessageReceipt struct {
	ID        string
	MessageID string
	UserID    string
	Status    MessageStatus
	SeenAt    *time.Time
}
