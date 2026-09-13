package application

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/giorgigelashvili12/Chat-Service/chat"
	"github.com/giorgigelashvili12/Chat-Service/handlers"
)

func loadRoutes(hub *chat.Hub) http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	chatHandler := handlers.NewChatHandler(hub)

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"service": "Chat-Service",
			"endpoints": map[string]string{
				"config_get":    "GET /api/config",
				"config_update": "PUT /api/config",
				"create_room":   "POST /api/rooms",
				"list_rooms":    "GET /api/rooms",
				"get_room":      "GET /api/rooms/{roomID}",
				"accept_room":   "POST /api/rooms/{roomID}/accept",
				"websocket":     "GET /ws/chat?room_id={id}&role=visitor|agent&name={name}",
			},
		})
	})

	router.Route("/api", func(r chi.Router) {
		r.Get("/config", chatHandler.GetConfig)
		r.Put("/config", chatHandler.UpdateConfig)

		r.Route("/rooms", func(r chi.Router) {
			r.Post("/", chatHandler.CreateRoom)
			r.Get("/", chatHandler.ListRooms)
			r.Get("/{roomID}", chatHandler.GetRoom)
			r.Post("/{roomID}/accept", chatHandler.AcceptRoom)
		})
	})

	router.Get("/ws/chat", chatHandler.ServeWebSocket)

	return router
}
