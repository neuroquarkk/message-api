package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	pconn *pgxpool.Pool
}

func New(pconn *pgxpool.Pool) *Handler {
	return &Handler{pconn: pconn}
}

func (h *Handler) CreateMessage(w http.ResponseWriter, r *http.Request) {
	var data struct {
		MessageText string `json:"message_text"`
		UserId      string `json:"user_id"`
	}

	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
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
		r.Context(),
		query, msg.Text, msg.UserID,
	).Scan(&msg.ID, &msg.CreatedAt)
	if err != nil {
		log.Printf("failed to insert message: %v\n", err)
		http.Error(w, "failed to insert message", http.StatusInternalServerError)
		return
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
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := h.pconn.Query(r.Context(), query, limit, offset)
	if err != nil {
		log.Printf("failed to query message: %v\n", err)
		http.Error(w, "failed to query messages", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var m Message
		err := rows.Scan(&m.ID, &m.Text, &m.UserID, &m.CreatedAt)
		if err != nil {
			log.Printf("failed to scan message: %v\n", err)
			http.Error(w, "failed to scan message", http.StatusInternalServerError)
			return
		}
		messages = append(messages, m)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}
