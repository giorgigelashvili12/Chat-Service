package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"

	"github.com/giorgigelashvili12/Chat-Service/chat"
	"github.com/giorgigelashvili12/Chat-Service/config"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type ChatHandler struct {
	Hub *chat.Hub
}

func NewChatHandler(hub *chat.Hub) *ChatHandler {
	return &ChatHandler{Hub: hub}
}

type createRoomRequest struct {
	VisitorName string `json:"visitor_name"`
}

type acceptRoomRequest struct {
	AgentName string `json:"agent_name"`
}

type configUpdateRequest struct {
	BotEnabled              bool   `json:"bot_enabled"`
	BotName                 string `json:"bot_name"`
	BotWelcomeMessage       string `json:"bot_welcome_message"`
	AgentAcceptTimeoutSec   int    `json:"agent_accept_timeout_seconds"`
	AutoAssignBotOnTimeout  bool   `json:"auto_assign_bot_on_timeout"`
	NoAgentAvailableMessage string `json:"no_agent_available_message"`
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func (h *ChatHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, config.Get())
}

func (h *ChatHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	var req configUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	updated := config.Update(config.SupportConfig{
		BotEnabled:              req.BotEnabled,
		BotName:                 req.BotName,
		BotWelcomeMessage:       req.BotWelcomeMessage,
		AgentAcceptTimeoutSec:   req.AgentAcceptTimeoutSec,
		AutoAssignBotOnTimeout:  req.AutoAssignBotOnTimeout,
		NoAgentAvailableMessage: req.NoAgentAvailableMessage,
	})
	writeJSON(w, http.StatusOK, updated)
}

func (h *ChatHandler) CreateRoom(w http.ResponseWriter, r *http.Request) {
	var req createRoomRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	room := h.Hub.CreateRoom(req.VisitorName)
	writeJSON(w, http.StatusCreated, room)
}

func (h *ChatHandler) ListRooms(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.Hub.ListRooms())
}

func (h *ChatHandler) GetRoom(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "roomID")
	room, ok := h.Hub.GetRoom(roomID)
	if !ok {
		writeError(w, http.StatusNotFound, "room not found")
		return
	}
	writeJSON(w, http.StatusOK, room)
}

func (h *ChatHandler) AcceptRoom(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "roomID")

	var req acceptRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.AgentName == "" {
		writeError(w, http.StatusBadRequest, "agent_name is required")
		return
	}

	room, err := h.Hub.AcceptRoom(roomID, req.AgentName)
	if err != nil {
		switch err {
		case chat.ErrRoomNotFound:
			writeError(w, http.StatusNotFound, err.Error())
		case chat.ErrRoomClosed:
			writeError(w, http.StatusGone, err.Error())
		case chat.ErrRoomAlreadyAssigned:
			writeError(w, http.StatusConflict, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, room)
}

type wsInbound struct {
	Type    chat.EventType `json:"type"`
	Content string         `json:"content"`
	Name    string         `json:"name"`
}

func (h *ChatHandler) ServeWebSocket(w http.ResponseWriter, r *http.Request) {
	roomID := r.URL.Query().Get("room_id")
	role := chat.Role(r.URL.Query().Get("role"))
	name := r.URL.Query().Get("name")

	if roomID == "" {
		writeError(w, http.StatusBadRequest, "room_id is required")
		return
	}
	if role != chat.RoleVisitor && role != chat.RoleAgent {
		writeError(w, http.StatusBadRequest, "role must be visitor or agent")
		return
	}
	if name == "" {
		if role == chat.RoleVisitor {
			name = "Visitor"
		} else {
			name = "Agent"
		}
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	client, history, err := h.Hub.RegisterClient(roomID, role, name)
	if err != nil {
		_ = conn.WriteJSON(map[string]string{"error": err.Error()})
		return
	}
	defer h.Hub.UnregisterClient(client)

	_ = conn.WriteJSON(chat.Event{
		Type:    chat.EventHistory,
		RoomID:  roomID,
		Payload: history,
	})

	go h.pumpOutbound(conn, client)

	for {
		var inbound wsInbound
		if err := conn.ReadJSON(&inbound); err != nil {
			return
		}

		switch inbound.Type {
		case chat.EventMessage:
			if inbound.Content == "" {
				continue
			}
			if err := h.Hub.HandleIncoming(client, inbound.Content); err != nil {
				_ = conn.WriteJSON(chat.Event{
					Type:    chat.EventSystem,
					RoomID:  roomID,
					Payload: map[string]string{"error": err.Error()},
				})
			}
		default:
			_ = conn.WriteJSON(chat.Event{
				Type:    chat.EventSystem,
				RoomID:  roomID,
				Payload: map[string]string{"error": "unsupported event type"},
			})
		}
	}
}

func (h *ChatHandler) pumpOutbound(conn *websocket.Conn, client *chat.Client) {
	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	for {
		select {
		case event, ok := <-client.Send:
			if !ok {
				return
			}
			if err := conn.WriteJSON(event); err != nil {
				return
			}
		case <-pingTicker.C:
			if err := conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(time.Second)); err != nil {
				return
			}
		}
	}
}
