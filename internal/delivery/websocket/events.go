package websocket

// EventType enumerates supported websocket events.
type EventType string

const (
	EventSendMessage    EventType = "send_message"
	EventReceiveMessage EventType = "receive_message"
	EventTypingStart    EventType = "typing_start"
	EventTypingUpdate   EventType = "typing_update"
	EventMessageRead    EventType = "message_read"
	EventMessageStatus  EventType = "message_status_update"
	EventConnect        EventType = "connect"
)

// Event represents a websocket payload exchanged between client and server.
type Event struct {
	Type EventType   `json:"event"`
	Data interface{} `json:"data"`
}
