package chat

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"github.com/giorgigelashvili12/Chat-Service/config"
)

type Client struct {
	RoomID string
	Role   Role
	Name   string
	Send   chan Event
	hub    *Hub
}

type Hub struct {
	mu      sync.RWMutex
	rooms   map[string]*Room
	clients map[string]map[*Client]struct{}
	timers  map[string]*time.Timer
}

func NewHub() *Hub {
	return &Hub{
		rooms:   make(map[string]*Room),
		clients: make(map[string]map[*Client]struct{}),
		timers:  make(map[string]*time.Timer),
	}
}

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (h *Hub) CreateRoom(visitorName string) *Room {
	h.mu.Lock()
	defer h.mu.Unlock()

	if visitorName == "" {
		visitorName = "Visitor"
	}

	room := &Room{
		ID:          newID(),
		VisitorName: visitorName,
		Status:      StatusWaiting,
		CreatedAt:   time.Now(),
		Messages:    []Message{},
	}
	h.rooms[room.ID] = room
	h.clients[room.ID] = make(map[*Client]struct{})

	h.startAcceptTimerLocked(room.ID)
	return cloneRoom(room)
}

func (h *Hub) ListRooms() []RoomSummary {
	h.mu.RLock()
	defer h.mu.RUnlock()

	summaries := make([]RoomSummary, 0, len(h.rooms))
	for _, room := range h.rooms {
		summary := RoomSummary{
			ID:          room.ID,
			VisitorName: room.VisitorName,
			Status:      room.Status,
			AssignedTo:  room.AssignedTo,
			Responder:   room.Responder,
			CreatedAt:   room.CreatedAt,
		}
		if n := len(room.Messages); n > 0 {
			summary.LastMessage = room.Messages[n-1].Content
		}
		summaries = append(summaries, summary)
	}
	return summaries
}

func (h *Hub) GetRoom(roomID string) (*Room, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	room, ok := h.rooms[roomID]
	if !ok {
		return nil, false
	}
	return cloneRoom(room), true
}

func (h *Hub) AcceptRoom(roomID, agentName string) (*Room, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	room, ok := h.rooms[roomID]
	if !ok {
		return nil, ErrRoomNotFound
	}
	if room.Status == StatusClosed {
		return nil, ErrRoomClosed
	}
	if room.Status != StatusWaiting {
		return nil, ErrRoomAlreadyAssigned
	}

	h.clearAcceptTimerLocked(roomID)

	now := time.Now()
	room.Status = StatusAssigned
	room.AssignedTo = agentName
	room.Responder = RoleAgent
	room.AcceptedAt = &now

	msg := h.appendMessageLocked(room, RoleAgent, agentName, agentName+" has joined the chat.")
	h.broadcastLocked(roomID, Event{Type: EventAssign, RoomID: roomID, Payload: map[string]string{
		"responder": string(RoleAgent),
		"agent":     agentName,
	}})
	h.broadcastLocked(roomID, Event{Type: EventMessage, RoomID: roomID, Message: &msg})

	return cloneRoom(room), nil
}

func (h *Hub) RegisterClient(roomID string, role Role, name string) (*Client, []Message, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	room, ok := h.rooms[roomID]
	if !ok {
		return nil, nil, ErrRoomNotFound
	}
	if room.Status == StatusClosed {
		return nil, nil, ErrRoomClosed
	}

	if role == RoleAgent && room.Status == StatusWaiting {
		h.mu.Unlock()
		_, err := h.AcceptRoom(roomID, name)
		h.mu.Lock()
		if err != nil && err != ErrRoomAlreadyAssigned {
			return nil, nil, err
		}
		room = h.rooms[roomID]
	}

	client := &Client{
		RoomID: roomID,
		Role:   role,
		Name:   name,
		Send:   make(chan Event, 32),
		hub:    h,
	}
	h.clients[roomID][client] = struct{}{}

	history := append([]Message(nil), room.Messages...)
	return client, history, nil
}

func (h *Hub) UnregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	roomClients, ok := h.clients[client.RoomID]
	if !ok {
		return
	}
	delete(roomClients, client)
	close(client.Send)
}

func (h *Hub) HandleIncoming(client *Client, content string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	room, ok := h.rooms[client.RoomID]
	if !ok {
		return ErrRoomNotFound
	}
	if room.Status == StatusClosed {
		return ErrRoomClosed
	}
	if client.Role == RoleAgent && room.Responder != RoleAgent {
		return ErrNotAssignedAgent
	}

	msg := h.appendMessageLocked(room, client.Role, client.Name, content)
	h.broadcastLocked(client.RoomID, Event{Type: EventMessage, RoomID: client.RoomID, Message: &msg})

	if client.Role == RoleVisitor && room.Responder == RoleBot {
		go h.sendBotReply(client.RoomID, content)
	}

	return nil
}

func (h *Hub) sendBotReply(roomID, visitorContent string) {
	time.Sleep(600 * time.Millisecond)

	cfg := config.Get()
	reply := botReply(visitorContent)

	h.mu.Lock()
	defer h.mu.Unlock()

	room, ok := h.rooms[roomID]
	if !ok || room.Responder != RoleBot || room.Status == StatusClosed {
		return
	}

	msg := h.appendMessageLocked(room, RoleBot, cfg.BotName, reply)
	h.broadcastLocked(roomID, Event{Type: EventMessage, RoomID: roomID, Message: &msg})
}

func (h *Hub) startAcceptTimerLocked(roomID string) {
	cfg := config.Get()
	if !cfg.BotEnabled || !cfg.AutoAssignBotOnTimeout {
		return
	}

	h.clearAcceptTimerLocked(roomID)

	timer := time.AfterFunc(cfg.AgentAcceptTimeout, func() {
		h.assignBot(roomID)
	})
	h.timers[roomID] = timer
}

func (h *Hub) assignBot(roomID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	room, ok := h.rooms[roomID]
	if !ok || room.Status != StatusWaiting {
		return
	}

	cfg := config.Get()
	now := time.Now()
	room.Status = StatusBot
	room.Responder = RoleBot
	room.AssignedTo = cfg.BotName
	room.AcceptedAt = &now

	h.broadcastLocked(roomID, Event{Type: EventAssign, RoomID: roomID, Payload: map[string]string{
		"responder": string(RoleBot),
		"agent":     cfg.BotName,
		"reason":    "timeout",
	}})

	systemMsg := h.appendMessageLocked(room, RoleBot, cfg.BotName, cfg.NoAgentAvailableMessage)
	h.broadcastLocked(roomID, Event{Type: EventMessage, RoomID: roomID, Message: &systemMsg})

	welcomeMsg := h.appendMessageLocked(room, RoleBot, cfg.BotName, cfg.BotWelcomeMessage)
	h.broadcastLocked(roomID, Event{Type: EventMessage, RoomID: roomID, Message: &welcomeMsg})
}

func (h *Hub) clearAcceptTimerLocked(roomID string) {
	if timer, ok := h.timers[roomID]; ok {
		timer.Stop()
		delete(h.timers, roomID)
	}
}

func (h *Hub) appendMessageLocked(room *Room, role Role, sender, content string) Message {
	msg := Message{
		ID:        newID(),
		RoomID:    room.ID,
		From:      role,
		Sender:    sender,
		Content:   content,
		Timestamp: time.Now(),
	}
	room.Messages = append(room.Messages, msg)
	return msg
}

func (h *Hub) broadcastLocked(roomID string, event Event) {
	for client := range h.clients[roomID] {
		select {
		case client.Send <- event:
		default:
		}
	}
}

func cloneRoom(room *Room) *Room {
	copyRoom := *room
	copyRoom.Messages = append([]Message(nil), room.Messages...)
	return &copyRoom
}
