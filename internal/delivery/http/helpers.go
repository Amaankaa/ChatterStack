package http

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"

	"chatterstack/internal/domain/auth"
	"chatterstack/internal/domain/messages"
	"chatterstack/internal/domain/models"
	"chatterstack/internal/domain/rooms"
	"chatterstack/internal/domain/users"
	"chatterstack/pkg/middleware"
)

type userPayload struct {
	ID         string            `json:"id"`
	Username   string            `json:"username"`
	Email      string            `json:"email"`
	Status     models.UserStatus `json:"status"`
	LastSeenAt *time.Time        `json:"last_seen_at,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
}

type tokenPayload struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type attachmentPayload struct {
	ID        string `json:"id,omitempty"`
	MessageID string `json:"message_id,omitempty"`
	URL       string `json:"url"`
	MimeType  string `json:"mime_type,omitempty"`
	SizeBytes int64  `json:"size_bytes,omitempty"`
}

type messagePayload struct {
	ID          string               `json:"id"`
	RoomID      string               `json:"room_id"`
	SenderID    string               `json:"sender_id"`
	Content     string               `json:"content"`
	Attachments []attachmentPayload  `json:"attachments,omitempty"`
	Status      models.MessageStatus `json:"status"`
	CreatedAt   time.Time            `json:"created_at"`
}

type roomPayload struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	IsGroup   bool      `json:"is_group"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type roomMemberPayload struct {
	ID     string          `json:"id,omitempty"`
	RoomID string          `json:"room_id,omitempty"`
	UserID string          `json:"user_id"`
	Role   models.RoomRole `json:"role"`
}

func toUserPayload(u *models.User) userPayload {
	var lastSeen *time.Time
	if !u.LastSeenAt.IsZero() {
		ts := u.LastSeenAt
		lastSeen = &ts
	}

	return userPayload{
		ID:         u.ID,
		Username:   u.Username,
		Email:      u.Email,
		Status:     u.Status,
		LastSeenAt: lastSeen,
		CreatedAt:  u.CreatedAt,
	}
}

func respondBadRequest(c *gin.Context, message string) {
	respondJSONError(c, http.StatusBadRequest, message)
}

func respondJSONError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}

func respondInternalServerError(c *gin.Context, err error) {
	log.Printf("http: internal error: %v", err)
	respondJSONError(c, http.StatusInternalServerError, "internal server error")
}

func mapAuthError(err error) (int, string) {
	switch {
	case errors.Is(err, auth.ErrMissingUsername),
		errors.Is(err, auth.ErrMissingEmail),
		errors.Is(err, auth.ErrMissingPassword):
		return http.StatusBadRequest, err.Error()
	case errors.Is(err, auth.ErrEmailAlreadyUsed):
		return http.StatusConflict, err.Error()
	case errors.Is(err, auth.ErrInvalidCredentials),
		errors.Is(err, auth.ErrInvalidRefreshToken):
		return http.StatusUnauthorized, err.Error()
	default:
		return http.StatusInternalServerError, "internal server error"
	}
}

func mapMessageError(err error) (int, string) {
	switch {
	case errors.Is(err, messages.ErrInvalidRoomID),
		errors.Is(err, messages.ErrInvalidSenderID),
		errors.Is(err, messages.ErrInvalidContent),
		errors.Is(err, messages.ErrInvalidMessageID),
		errors.Is(err, messages.ErrInvalidUserID),
		errors.Is(err, messages.ErrInvalidSearch):
		return http.StatusBadRequest, err.Error()
	default:
		return http.StatusInternalServerError, "internal server error"
	}
}

func mapRoomError(err error) (int, string) {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503":
			// Foreign key violation indicates a referenced entity is missing.
			if strings.Contains(pgErr.ConstraintName, "user") {
				return http.StatusNotFound, "rooms: user not found"
			}
			if strings.Contains(pgErr.ConstraintName, "room") {
				return http.StatusNotFound, "rooms: room not found"
			}
			return http.StatusNotFound, "rooms: related entity not found"
		case "23505":
			return http.StatusConflict, "rooms: duplicate record"
		}
	}

	switch {
	case errors.Is(err, rooms.ErrInvalidRoomName),
		errors.Is(err, rooms.ErrInvalidCreatorID),
		errors.Is(err, rooms.ErrInvalidMemberUser),
		errors.Is(err, rooms.ErrInvalidUserID),
		errors.Is(err, rooms.ErrDirectMessageSelf):
		return http.StatusBadRequest, err.Error()
	case errors.Is(err, rooms.ErrRoomNotFound):
		return http.StatusNotFound, err.Error()
	case errors.Is(err, rooms.ErrDeleteNotAllowed):
		return http.StatusForbidden, err.Error()
	default:
		if err != nil && strings.HasPrefix(err.Error(), "rooms:") {
			return http.StatusBadRequest, err.Error()
		}
		return http.StatusInternalServerError, "internal server error"
	}
}

func mapUserError(err error) (int, string) {
	switch {
	case errors.Is(err, users.ErrUserNotFound):
		return http.StatusNotFound, err.Error()
	default:
		if err != nil && strings.HasPrefix(err.Error(), "users:") {
			return http.StatusBadRequest, err.Error()
		}
		return http.StatusInternalServerError, "internal server error"
	}
}

func toAttachmentPayloads(list []models.Attachment) []attachmentPayload {
	if len(list) == 0 {
		return nil
	}
	res := make([]attachmentPayload, 0, len(list))
	for _, att := range list {
		res = append(res, attachmentPayload{
			ID:        att.ID,
			MessageID: att.MessageID,
			URL:       att.URL,
			MimeType:  att.MimeType,
			SizeBytes: att.SizeBytes,
		})
	}
	return res
}

func toMessagePayload(msg models.Message) messagePayload {
	return messagePayload{
		ID:          msg.ID,
		RoomID:      msg.RoomID,
		SenderID:    msg.SenderID,
		Content:     msg.Content,
		Attachments: toAttachmentPayloads(msg.Attachments),
		Status:      msg.Status,
		CreatedAt:   msg.CreatedAt,
	}
}

func toMessagePayloads(list []models.Message) []messagePayload {
	if len(list) == 0 {
		return []messagePayload{}
	}
	res := make([]messagePayload, 0, len(list))
	for _, msg := range list {
		res = append(res, toMessagePayload(msg))
	}
	return res
}

func toRoomPayload(room *models.Room) roomPayload {
	return roomPayload{
		ID:        room.ID,
		Name:      room.Name,
		IsGroup:   room.IsGroup,
		CreatedBy: room.CreatedBy,
		CreatedAt: room.CreatedAt,
	}
}

func toRoomMemberPayloads(members []models.RoomMember) []roomMemberPayload {
	if len(members) == 0 {
		return []roomMemberPayload{}
	}
	res := make([]roomMemberPayload, 0, len(members))
	for _, member := range members {
		res = append(res, roomMemberPayload{
			ID:     member.ID,
			RoomID: member.RoomID,
			UserID: member.UserID,
			Role:   member.Role,
		})
	}
	return res
}

func toRoomPayloads(list []models.Room) []roomPayload {
	if len(list) == 0 {
		return []roomPayload{}
	}
	res := make([]roomPayload, 0, len(list))
	for i := range list {
		res = append(res, toRoomPayload(&list[i]))
	}
	return res
}

func currentUserID(c *gin.Context) (string, bool) {
	value, ok := c.Get(middleware.UserIDContextKey)
	if !ok {
		return "", false
	}
	userID, ok := value.(string)
	if !ok || strings.TrimSpace(userID) == "" {
		return "", false
	}
	return userID, true
}
