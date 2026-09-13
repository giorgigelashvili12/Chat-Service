package application

import (
	"context"
	"fmt"
	"net/http"

	"github.com/giorgigelashvili12/Chat-Service/chat"
)

type App struct {
	router http.Handler
	hub    *chat.Hub
}

func New() *App {
	hub := chat.NewHub()
	return &App{
		hub:    hub,
		router: loadRoutes(hub),
	}
}

func (a *App) Start(ctx context.Context) error {
	server := &http.Server{
		Addr:    ":3000",
		Handler: a.router,
	}

	fmt.Println("Chat server listening on http://localhost:3000")

	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to run server: %w", err)
	}

	return nil
}
