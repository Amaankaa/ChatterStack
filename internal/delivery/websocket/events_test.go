package websocket

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEventEncode(t *testing.T) {
	t.Helper()

	input := Event{
		Type: EventSendMessage,
		Data: map[string]string{"text": "hello"},
	}

	payload, err := input.Encode()
	require.NoError(t, err)

	var decoded map[string]interface{}
	require.NoError(t, json.Unmarshal(payload, &decoded))
	require.Equal(t, string(EventSendMessage), decoded["event"])
	require.Equal(t, "hello", decoded["data"].(map[string]interface{})["text"])
}

func TestEventEncodeError(t *testing.T) {
	t.Helper()

	input := Event{
		Type: EventSendMessage,
		Data: make(chan int),
	}

	_, err := input.Encode()
	require.Error(t, err)
}
