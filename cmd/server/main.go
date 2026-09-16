package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"

	"message-api/internal/config"
	"message-api/internal/db"
	"message-api/internal/handler"
	"message-api/internal/middleware"
)

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT, syscall.SIGABRT,
	)
	defer stop()

	cfg := config.Load()
	pconn, err := db.PConn(ctx, cfg.POSTGRES_URL)
	if err != nil {
		log.Fatalf("failed to create postgres connection: %v\n", err)
	}
	defer pconn.Close()
	log.Println("postgres connected")

	hdl := handler.New(pconn)
	mux := http.NewServeMux()

	mux.HandleFunc("POST /messages", hdl.CreateMessage)
	mux.HandleFunc("GET /messages", hdl.ListMessages)
	mux.HandleFunc("POST /messages/{messageId}/reactions", hdl.CreateReaction)

	srv := &http.Server{
		Addr:    ":" + cfg.PORT,
		Handler: middleware.Logger(mux),
	}

	go func() {
		log.Println("server starting on", srv.Addr)
		if err := srv.ListenAndServe(); err != nil {
			if err != http.ErrServerClosed {
				log.Fatalf("failed to serve: %v\n", err)
			}
		}
	}()

	<-ctx.Done()
	if err := srv.Shutdown(context.TODO()); err != nil {
		log.Printf("failed to shutdown server: %v\n", err)
	}
}
