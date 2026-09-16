package handler

import "time"

type Message struct {
	ID                 string     `json:"id"`
	Text               string     `json:"text"`
	UserID             string     `json:"user_id"`
	CreatedAt          time.Time  `json:"created_at"`
	TotalReactionCount int        `json:"total_reaction_count"`
	Reactions          []Reaction `json:"reactions"`
}

type Reaction struct {
	ID        string    `json:"id"`
	MessageID string    `json:"message_id"`
	Type      string    `json:"type"`
	Score     int       `json:"score"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}
