package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Handler struct {
	pconn    *pgxpool.Pool
	rconn    *redis.Client
	cacheKey string
}

func New(pconn *pgxpool.Pool, rconn *redis.Client) *Handler {
	return &Handler{
		pconn:    pconn,
		rconn:    rconn,
		cacheKey: "cache:messages",
	}
}

func (h *Handler) CreateMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var data CreateMessageReq

	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := data.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	query := `
		INSERT INTO message (message_text, user_id)
		VALUES ($1, $2)
		RETURNING id, created_at`

	var msg Message
	msg.Text = data.MessageText
	msg.UserID = data.UserId

	err := h.pconn.QueryRow(
		ctx,
		query, msg.Text, msg.UserID,
	).Scan(&msg.ID, &msg.CreatedAt)
	if err != nil {
		log.Printf("failed to insert message: %v\n", err)
		http.Error(w, "failed to insert message", http.StatusInternalServerError)
		return
	}

	msgJson, err := json.Marshal(msg)
	if err == nil {
		pipe := h.rconn.TxPipeline()

		pipe.LPush(ctx, h.cacheKey, msgJson)
		pipe.LTrim(ctx, h.cacheKey, 0, 9)

		if _, err := pipe.Exec(ctx); err != nil {
			log.Printf("failed to update redis cache: %v\n", err)
		}
	} else {
		log.Printf("failed to marshal message for cache: %v\n", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(msg)
}

func (h *Handler) ListMessages(w http.ResponseWriter, r *http.Request) {
	page := 1
	pageStr := r.URL.Query().Get("page")
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 10
	offset := (page - 1) * limit

	query := `
		SELECT id, message_text, user_id, created_at
		FROM message
		ORDER BY created_at DESC, id DESC
		LIMIT $1 OFFSET $2`

	rows, err := h.pconn.Query(r.Context(), query, limit, offset)
	if err != nil {
		log.Printf("failed to query message: %v\n", err)
		http.Error(w, "failed to query messages", http.StatusInternalServerError)
		return
	}

	var messages []Message
	var messageIds []string

	msgIdx := make(map[string]*Message)

	for rows.Next() {
		var m Message
		err := rows.Scan(&m.ID, &m.Text, &m.UserID, &m.CreatedAt)
		if err != nil {
			rows.Close()
			log.Printf("failed to scan message: %v\n", err)
			http.Error(w, "failed to scan message", http.StatusInternalServerError)
			return
		}
		m.Reactions = []Reaction{}
		messages = append(messages, m)
	}
	rows.Close()

	if len(messages) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Message{})
		return
	}

	for i := range messages {
		messageIds = append(messageIds, messages[i].ID)
		msgIdx[messages[i].ID] = &messages[i]
	}

	rxQuery := `
		SELECT id, message_id, user_id, type, score, created_at
		FROM reactions
		WHERE message_id = ANY($1)`

	rxRows, err := h.pconn.Query(r.Context(), rxQuery, messageIds)
	if err != nil {
		log.Printf("failed to query reactions: %v\n", err)
		http.Error(w, "failed to query reactions", http.StatusInternalServerError)
		return
	}
	defer rxRows.Close()

	for rxRows.Next() {
		var rx Reaction
		if err := rxRows.Scan(
			&rx.ID, &rx.MessageID, &rx.UserID,
			&rx.Type, &rx.Score, &rx.CreatedAt,
		); err != nil {
			http.Error(w, "failed to scan reaction", http.StatusInternalServerError)
			return
		}

		if msg, exists := msgIdx[rx.MessageID]; exists {
			msg.Reactions = append(msg.Reactions, rx)
			msg.TotalReactionCount += rx.Score
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

func (h *Handler) CreateReaction(w http.ResponseWriter, r *http.Request) {
	messageId := r.PathValue("messageId")
	var data CreateReactionReq

	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := data.Validate(messageId); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	scoreVal := 1
	if data.Score != nil {
		scoreVal = *data.Score
	}

	query := `
		INSERT INTO reactions (message_id, user_id, type, score)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (message_id, user_id, type)
		DO UPDATE SET score = reactions.score + EXCLUDED.score
		RETURNING id, score, created_at`

	var react Reaction
	react.MessageID = messageId
	react.UserID = data.UserId
	react.Type = data.Type

	err := h.pconn.QueryRow(
		r.Context(),
		query, react.MessageID, react.UserID, react.Type, scoreVal,
	).Scan(&react.ID, &react.Score, &react.CreatedAt)
	if err != nil {
		log.Printf("failed to insert reaction: %v\n", err)
		http.Error(w, "failed to insert reaction", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(react)
}
