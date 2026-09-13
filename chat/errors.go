package chat

import "errors"

var (
	ErrRoomNotFound       = errors.New("room not found")
	ErrRoomClosed         = errors.New("room is closed")
	ErrRoomAlreadyAssigned = errors.New("room already has a responder")
	ErrNotAssignedAgent   = errors.New("agent is not assigned to this room")
)
