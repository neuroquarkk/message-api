package handler

import "time"

type Message struct {
	ID        string
	Text      string
	UserID    string
	CreatedAt time.Time
}

type Reaction struct {
	ID        string
	MessageID string
	Type      string
	Score     int
	UserID    string
	CreatedAt time.Time
}
