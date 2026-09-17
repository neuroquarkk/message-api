package handler

import (
	"errors"
	"strings"
	"uuid"
)

type CreateMessageReq struct {
	MessageText string `json:"message_text"`
	UserId      string `json:"user_id"`
}

func (req *CreateMessageReq) Validate() error {
	req.MessageText = strings.TrimSpace(req.MessageText)
	req.UserId = strings.TrimSpace(req.UserId)

	if req.MessageText == "" {
		return errors.New("message_text is required")
	}

	if len(req.MessageText) > 5000 {
		return errors.New("message_text exceeds maximum length")
	}

	if req.UserId == "" {
		return errors.New("user_id is required")
	}

	if len(req.UserId) > 255 {
		return errors.New("user_id exceeds 255 characters")
	}

	return nil
}

type CreateReactionReq struct {
	UserId string `json:"user_id"`
	Type   string `json:"type"`
	Score  *int   `json:"score"`
}

func (req *CreateReactionReq) Validate(messageId string) error {
	if _, err := uuid.Parse(messageId); err != nil {
		return errors.New("invalid messageId format")
	}

	if messageId == "" || len(messageId) != 36 {
		return errors.New("invalid messageID format")
	}

	req.UserId = strings.TrimSpace(req.UserId)
	req.Type = strings.TrimSpace(req.Type)

	if req.UserId == "" {
		return errors.New("user_id is required")
	}
	if len(req.UserId) > 255 {
		return errors.New("user_id exceeds 255 characters")
	}
	if req.Type == "" {
		return errors.New("reaction type is required")
	}
	if len(req.Type) > 100 {
		return errors.New("reaction type exceeds 100 characters")
	}
	if req.Score != nil && *req.Score < 1 {
		return errors.New("score must be at least 1")
	}
	return nil
}
