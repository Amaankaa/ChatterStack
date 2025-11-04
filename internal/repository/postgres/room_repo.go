package postgres

import "context"

// RoomRepository implements persistence for chat rooms.
type RoomRepository struct{}

// NewRoomRepository constructs a new RoomRepository.
func NewRoomRepository() *RoomRepository {
	return &RoomRepository{}
}

// Placeholder methods until implementation is added.
func (r *RoomRepository) Create(ctx context.Context, params interface{}) (interface{}, error) {
	return nil, nil
}
