package messages

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"chatterstack/internal/domain/models"
)

const (
	defaultListLimit = 50
	maxSearchLimit   = 100
)

var (
	ErrInvalidRoomID    = errors.New("messages: room id is required")
	ErrInvalidSenderID  = errors.New("messages: sender id is required")
	ErrInvalidContent   = errors.New("messages: content or attachments are required")
	ErrInvalidMessageID = errors.New("messages: message id is required")
	ErrInvalidUserID    = errors.New("messages: user id is required")
	ErrInvalidSearch    = errors.New("messages: search query is required")
)

// messageRepository outlines the persistence operations needed by the domain service.
type messageRepository interface {
	Create(ctx context.Context, msg *models.Message) error
	ListByRoom(ctx context.Context, roomID string, limit, offset int) ([]models.Message, error)
	ListAround(ctx context.Context, roomID, messageID string, before, after int) ([]models.Message, error)
	UpdateStatus(ctx context.Context, id string, status models.MessageStatus) error
	UpsertReceipt(ctx context.Context, receipt *models.MessageReceipt) error
	Search(ctx context.Context, userID, roomID, query string, limit int) ([]models.Message, error)
}

// publisher emits events to interested subscribers (e.g. via Redis pub/sub).
type publisher interface {
	Publish(ctx context.Context, channel string, payload []byte) error
}

type service struct {
	repo messageRepository
	pub  publisher
}

// NewService wires the message domain service with its dependencies.
func NewService(repo messageRepository, pub publisher) Service {
	return &service{repo: repo, pub: pub}
}

func (s *service) Send(ctx context.Context, payload SendMessageInput) (*models.Message, error) {
	if strings.TrimSpace(payload.RoomID) == "" {
		return nil, ErrInvalidRoomID
	}
	if strings.TrimSpace(payload.SenderID) == "" {
		return nil, ErrInvalidSenderID
	}
	if strings.TrimSpace(payload.Content) == "" && len(payload.Attachments) == 0 {
		return nil, ErrInvalidContent
	}

	msg := &models.Message{
		RoomID:      payload.RoomID,
		SenderID:    payload.SenderID,
		Content:     strings.TrimSpace(payload.Content),
		Attachments: payload.Attachments,
		Status:      models.MessageStatusSent,
	}

	if err := s.repo.Create(ctx, msg); err != nil {
		return nil, err
	}

	if s.pub != nil {
		if data, err := json.Marshal(msg); err == nil {
			if err := s.pub.Publish(ctx, roomChannel(msg.RoomID), data); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	return msg, nil
}

func (s *service) ListByRoom(ctx context.Context, roomID string, page, limit int) ([]models.Message, error) {
	if strings.TrimSpace(roomID) == "" {
		return nil, ErrInvalidRoomID
	}
	if limit <= 0 {
		limit = defaultListLimit
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit
	return s.repo.ListByRoom(ctx, roomID, limit, offset)
}

func (s *service) ListAround(ctx context.Context, roomID, messageID string, limit int) ([]models.Message, error) {
	roomID = strings.TrimSpace(roomID)
	if roomID == "" {
		return nil, ErrInvalidRoomID
	}
	messageID = strings.TrimSpace(messageID)
	if messageID == "" {
		return nil, ErrInvalidMessageID
	}
	if limit <= 0 {
		limit = defaultListLimit
	}

	before := limit / 2
	after := limit - before - 1
	if after < 0 {
		after = 0
	}

	return s.repo.ListAround(ctx, roomID, messageID, before, after)
}

func (s *service) MarkDelivered(ctx context.Context, messageID, userID string) error {
	if strings.TrimSpace(messageID) == "" {
		return ErrInvalidMessageID
	}
	if strings.TrimSpace(userID) == "" {
		return ErrInvalidUserID
	}

	receipt := &models.MessageReceipt{MessageID: messageID, UserID: userID, Status: models.MessageStatusDelivered}
	if err := s.repo.UpsertReceipt(ctx, receipt); err != nil {
		return err
	}
	return s.repo.UpdateStatus(ctx, messageID, models.MessageStatusDelivered)
}

func (s *service) MarkRead(ctx context.Context, messageID, userID string) error {
	if strings.TrimSpace(messageID) == "" {
		return ErrInvalidMessageID
	}
	if strings.TrimSpace(userID) == "" {
		return ErrInvalidUserID
	}

	seenAt := time.Now().UTC()
	receipt := &models.MessageReceipt{MessageID: messageID, UserID: userID, Status: models.MessageStatusRead, SeenAt: &seenAt}
	if err := s.repo.UpsertReceipt(ctx, receipt); err != nil {
		return err
	}
	return s.repo.UpdateStatus(ctx, messageID, models.MessageStatusRead)
}

func (s *service) Search(ctx context.Context, userID, roomID, query string, limit int) ([]models.Message, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, ErrInvalidUserID
	}

	query = strings.TrimSpace(query)
	if query == "" {
		return nil, ErrInvalidSearch
	}

	roomID = strings.TrimSpace(roomID)
	if limit <= 0 {
		limit = defaultListLimit
	}
	if limit > maxSearchLimit {
		limit = maxSearchLimit
	}

	return s.repo.Search(ctx, userID, roomID, query, limit)
}

func roomChannel(roomID string) string {
	return fmt.Sprintf("rooms:%s:messages", roomID)
}
