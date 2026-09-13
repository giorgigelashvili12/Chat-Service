package chat

import (
	"time"
)

type Role string

const (
	RoleVisitor Role = "visitor"
	RoleAgent   Role = "agent"
	RoleBot     Role = "bot"
)

type RoomStatus string

const (
	StatusWaiting  RoomStatus = "waiting"
	StatusAssigned RoomStatus = "assigned"
	StatusBot      RoomStatus = "bot"
	StatusClosed   RoomStatus = "closed"
)

type EventType string

const (
	EventMessage EventType = "message"
	EventJoin    EventType = "join"
	EventLeave   EventType = "leave"
	EventAssign  EventType = "assign"
	EventSystem  EventType = "system"
	EventHistory EventType = "history"
)

type Message struct {
	ID        string    `json:"id"`
	RoomID    string    `json:"room_id"`
	From      Role      `json:"from"`
	Sender    string    `json:"sender"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

type Event struct {
	Type    EventType `json:"type"`
	RoomID  string    `json:"room_id,omitempty"`
	Message *Message  `json:"message,omitempty"`
	Payload any       `json:"payload,omitempty"`
}

type Room struct {
	ID           string     `json:"id"`
	VisitorName  string     `json:"visitor_name"`
	Status       RoomStatus `json:"status"`
	AssignedTo   string     `json:"assigned_to,omitempty"`
	Responder    Role       `json:"responder,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	AcceptedAt   *time.Time `json:"accepted_at,omitempty"`
	Messages     []Message  `json:"messages"`
}

type RoomSummary struct {
	ID          string     `json:"id"`
	VisitorName string     `json:"visitor_name"`
	Status      RoomStatus `json:"status"`
	AssignedTo  string     `json:"assigned_to,omitempty"`
	Responder   Role       `json:"responder,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	LastMessage string     `json:"last_message,omitempty"`
}
