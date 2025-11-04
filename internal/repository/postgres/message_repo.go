package postgres

import "context"

// MessageRepository implements persistence for chat messages.
type MessageRepository struct{}

// NewMessageRepository constructs a new MessageRepository.
func NewMessageRepository() *MessageRepository {
	return &MessageRepository{}
}

// Placeholder methods until implementation is added.
func (r *MessageRepository) Create(ctx context.Context, params interface{}) (interface{}, error) {
	return nil, nil
}
