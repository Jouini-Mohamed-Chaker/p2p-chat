package client

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Jouini-Mohamed-Chaker/p2p-chat/pkg/protocol"
	"github.com/Jouini-Mohamed-Chaker/p2p-chat/pkg/signaling"
	"github.com/Jouini-Mohamed-Chaker/p2p-chat/pkg/webrtc"
)

// Provides a high-level interface for the chat application
type ChatClient struct {
	peer     webrtc.Peer
	username string
	roomCode string

	// Connection state
	isConnected bool
	mu          sync.RWMutex

	// Event callbacks
	onMessage      func(protocol.Message)
	onConnected    func()
	onDisconnected func()
	onError        func(error)
}

// Created a new chat client instance
func NewChatClient(username string) (*ChatClient, error) {
	if username == "" {
		return nil, fmt.Errorf("username cannot be empty")
	}

	peer, err := webrtc.NewRealPeer()
	if err != nil {
		return nil, fmt.Errorf("failed to create peer: %w", err)
	}

	client := &ChatClient{
		peer:     peer,
		username: username,
	}

	// Set up peer event handlers
	client.setupPeerHandlers()

	return client, nil
}

func (c *ChatClient) CreateRoom() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isConnected {
		return "", fmt.Errorf("already connected to a room")
	}

	// Create WebRTC offer
	offer, err := c.peer.CreateOffer()
	if err != nil {
		return "", fmt.Errorf("failed to create offer: %w", err)
	}

	// Encode the offer for sharing
	roomCode, err := signaling.Encode(offer)
	if err != nil {
		return "", fmt.Errorf("failed to encode offer: %w", err)
	}

	c.roomCode = roomCode
	log.Printf("Created room with code: %s", roomCode[:min(10, len(roomCode))]+"...")

	return roomCode, nil
}

// Join an existing room using a room code and returns the answer code
func (c *ChatClient) JoinRoom(roomCode string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isConnected {
		return "", fmt.Errorf("already connected to a room")
	}

	if roomCode == "" {
		return "", fmt.Errorf("room code cannot be empty")
	}

	// Decode the room code to get the offer
	offer, err := signaling.Decode(roomCode)
	if err != nil {
		return "", fmt.Errorf("invalid room code: %w", err)
	}

	// Create answer for the offer
	answer, err := c.peer.CreateAnswer(offer)
	if err != nil {
		return "", fmt.Errorf("failed to create answer: %w", err)
	}

	// Encode the answer for sharing
	encodedAnswer, err := signaling.Encode(answer)
	if err != nil {
		return "", fmt.Errorf("failed to encode answer: %w", err)
	}

	c.roomCode = roomCode
	log.Printf("Created answer for room. Answer code: %s", encodedAnswer[:min(10, len(encodedAnswer))]+"...")

	return encodedAnswer, nil
}

// Processes an answer from someone joining the room (room creator only)
func (c *ChatClient) AcceptAnswer(answerCode string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if answerCode == "" {
		return fmt.Errorf("answer code cannot be empty")
	}

	// Decode the answer
	answer, err := signaling.Decode(answerCode)
	if err != nil {
		return fmt.Errorf("invalid answer code: %w", err)
	}

	// Set the remote answer
	if err := c.peer.SetRemoteAnswer(answer); err != nil {
		return fmt.Errorf("failed to set remote answer: %w", err)
	}

	log.Printf("Accepted answer from peer")
	return nil
}

func (c *ChatClient) SendMessage(text string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.isConnected {
		return fmt.Errorf("not connected to any room")
	}

	if text == "" {
		return fmt.Errorf("message text cannot be empty")
	}

	// Create protocol message
	msg := protocol.NewMessage(protocol.TypeChat, c.username, text)

	// Marshal to bytes
	data := protocol.Marshal(msg)

	// Send over WebRTC data channel
	if err := c.peer.Send(data); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	log.Printf("Sent message: %s", text)
	return nil
}

// Closes the connection and cleans up resources
func (c *ChatClient) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	wasConnected := c.isConnected

	if c.isConnected {
		// Send leave message before disconnecting
		leaveMsg := protocol.NewMessage(protocol.TypeLeave, c.username, "")
		data := protocol.Marshal(leaveMsg)
		
		// Try to send leave message, but don't fail if it doesn't work
		if err := c.peer.Send(data); err != nil {
			log.Printf("Warning: Failed to send leave message: %v", err)
		}

		c.isConnected = false
	}

	c.roomCode = ""

	// Close the peer connection
	err := c.peer.Close()

	// Notify disconnection after closing (only if we were connected)
	if wasConnected && c.onDisconnected != nil {
		// Use a small delay to ensure the close operation completes
		go func() {
			time.Sleep(100 * time.Millisecond)
			c.onDisconnected()
		}()
	}

	return err
}

// Event handlers for setters
func (c *ChatClient) OnMessage(callback func(protocol.Message)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onMessage = callback
}

func (c *ChatClient) OnConnected(callback func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onConnected = callback
}

func (c *ChatClient) OnDisconnected(callback func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onDisconnected = callback
}

func (c *ChatClient) OnError(callback func(error)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onError = callback
}

// GetUsername returns the current username
func (c *ChatClient) GetUsername() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.username
}

// IsConnected returns whether the client is connected to a room
func (c *ChatClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isConnected
}

// GetRoomCode returns the current room code (if any)
func (c *ChatClient) GetRoomCode() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.roomCode
}

// GetConnectionInstructions returns user-friendly instructions for the copy/paste flow
func (c *ChatClient) GetConnectionInstructions() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.roomCode != "" {
		return `Connection Instructions:
1. You created a room - share your room code with the other person
2. They will join your room and give you an "answer code"  
3. Paste their answer code using AcceptAnswer() to complete connection`
	}

	return `Connection Instructions:
1. Get a room code from someone else
2. Use JoinRoom() with their code - you'll get an "answer code"
3. Send your answer code back to them
4. Connection will establish automatically once they accept your answer`
}

// ConnectionStatus returns a user-friendly connection status
func (c *ChatClient) ConnectionStatus() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.isConnected {
		return "Connected - ready to chat!"
	}

	if c.roomCode != "" {
		return "Room created - waiting for connection..."
	}

	return "Not connected"
}

func (c *ChatClient) setupPeerHandlers() {
	// Handle incoming messages
	c.peer.OnMessage(func(data []byte) {
		msg, err := protocol.Unmarshal(data)
		if err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			c.mu.RLock()
			errorCallback := c.onError
			c.mu.RUnlock()
			
			if errorCallback != nil {
				go errorCallback(fmt.Errorf("invalid message received: %w", err))
			}
			return
		}
		log.Printf("Received message: %s from %s", msg.Text, msg.From)

		// Handle special message types
		switch msg.Type {
		case protocol.TypeJoin:
			log.Printf("%s joined the chat", msg.From)
		case protocol.TypeLeave:
			log.Printf("%s left the chat", msg.From)
		}

		// Notify callback
		c.mu.RLock()
		callback := c.onMessage
		c.mu.RUnlock()

		if callback != nil {
			go callback(msg)
		}
	})

	// Handle connection state change
	c.peer.OnStateChange(func(state string) {
		log.Printf("Connection state: %s", state)

		c.mu.Lock()
		wasConnected := c.isConnected
		
		// Update connection state based on WebRTC state
		switch state {
		case "connected":
			c.isConnected = true
		case "disconnected", "failed", "closed":
			c.isConnected = false
		default:
			// For other states like "connecting", keep current state
		}
		
		connectedCallback := c.onConnected
		disconnectedCallback := c.onDisconnected
		c.mu.Unlock()

		// Notify about state changes
		if c.isConnected && !wasConnected {
			// Just connected
			log.Printf("Successfully connected to peer")

			// Send join message
			joinMsg := protocol.NewMessage(protocol.TypeJoin, c.username, "")
			data := protocol.Marshal(joinMsg)
			
			// Try to send join message, but don't fail if it doesn't work immediately
			go func() {
				// Small delay to ensure data channel is fully ready
				time.Sleep(100 * time.Millisecond)
				if err := c.peer.Send(data); err != nil {
					log.Printf("Warning: Failed to send join message: %v", err)
				}
			}()

			if connectedCallback != nil {
				go connectedCallback()
			}
		} else if !c.isConnected && wasConnected {
			// Just disconnected
			log.Printf("Disconnected from peer")
			if disconnectedCallback != nil {
				go disconnectedCallback()
			}
		}
	})
}

// Helper function for Go 1.20 compatibility (min function)
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}