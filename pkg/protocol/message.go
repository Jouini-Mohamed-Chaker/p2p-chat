package protocol

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// Message represents a chat message in the protocol
type Message struct {
	Type      string `json:"type"`
	From      string `json:"from"`
	Text      string `json:"text"`
	Timestamp int64  `json:"timestamp"`
}

const (
	// Message types
	TypeChat  = "chat"
	TypeJoin  = "join"
	TypeLeave = "leave"

	// Validation constraints
	MaxTextLength = 1000
)

// NewMessage creates a new message with the current timestamp
func NewMessage(msgType, from, text string) Message {
	return Message{
		Type:      msgType,
		From:      from,
		Text:      text,
		Timestamp: time.Now().UnixMilli(),
	}
}

// Marshal converts a Message to JSON bytes with a trailing newline
func Marshal(msg Message) []byte {
	data, err := json.Marshal(msg)
	if err != nil {
		// This should rarely happen with a simple struct like Message
		// but we'll handle it gracefully by returning empty JSON object
		return []byte("{}\n")
	}
	
	// Add newline for protocol compatibility
	data = append(data, '\n')
	return data
}

// Unmarshal parses JSON data into a Message with validation
func Unmarshal(data []byte) (Message, error) {
	var msg Message
	
	// Remove trailing newline if present
	data = []byte(strings.TrimSuffix(string(data), "\n"))
	
	// Parse JSON
	if err := json.Unmarshal(data, &msg); err != nil {
		return Message{}, errors.New("invalid JSON format")
	}
	
	// Validate the message
	if err := validateMessage(msg); err != nil {
		return Message{}, err
	}
	
	return msg, nil
}

// validateMessage performs validation on message fields
func validateMessage(msg Message) error {
	// Check required fields
	if msg.Type == "" {
		return errors.New("message type is required")
	}
	
	if msg.From == "" {
		return errors.New("from field is required")
	}
	
	// Validate message type
	if msg.Type != TypeChat && msg.Type != TypeJoin && msg.Type != TypeLeave {
		return errors.New("invalid message type")
	}
	
	// Check text length constraint
	if len(msg.Text) > MaxTextLength {
		return errors.New("message text exceeds maximum length")
	}
	
	// Timestamp validation (should be positive)
	if msg.Timestamp < 0 {
		return errors.New("invalid timestamp")
	}
	
	return nil
}

// IsValid checks if a message is valid without returning specific error details
func (m Message) IsValid() bool {
	return validateMessage(m) == nil
}

// String returns a string representation of the message for debugging
func (m Message) String() string {
	return string(Marshal(m))
}