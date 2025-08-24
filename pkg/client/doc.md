# ChatClient Package Documentation

## Overview

The `client` package provides a high-level interface for peer-to-peer chat applications using WebRTC. It handles connection establishment, message sending/receiving, and room management through a simple API.

## Installation

```go
import "github.com/Jouini-Mohamed-Chaker/p2p-chat/pkg/client"
```

## Quick Start

```go
// Create a new chat client
client, err := client.NewChatClient("your-username")
if err != nil {
    log.Fatal(err)
}
defer client.Disconnect()

// Set up event handlers
client.OnMessage(func(msg protocol.Message) {
    fmt.Printf("%s: %s\n", msg.From, msg.Text)
})

client.OnConnected(func() {
    fmt.Println("Connected to chat!")
})

// Create or join a room (see examples below)
```

## Core Concepts

### Connection Flow
1. **Room Creator**: Creates a room and gets a room code to share
2. **Room Joiner**: Uses the room code to join and generates an answer code
3. **Room Creator**: Accepts the answer code to complete the connection

### Message Types
- `TypeChat`: Regular chat messages
- `TypeJoin`: Notification when someone joins
- `TypeLeave`: Notification when someone leaves

## API Reference

### Constructor

#### `NewChatClient(username string) (*ChatClient, error)`
Creates a new chat client instance.

**Parameters:**
- `username`: The display name for this user (cannot be empty)

**Returns:**
- `*ChatClient`: The client instance
- `error`: Error if creation fails

**Example:**
```go
client, err := client.NewChatClient("Alice")
if err != nil {
    log.Fatal("Failed to create client:", err)
}
```

### Room Management

#### `CreateRoom() (string, error)`
Creates a new chat room and returns a room code to share with others.

**Returns:**
- `string`: Room code to share with the person who wants to join
- `error`: Error if room creation fails

**Example:**
```go
roomCode, err := client.CreateRoom()
if err != nil {
    log.Fatal("Failed to create room:", err)
}
fmt.Println("Share this room code:", roomCode)
```

#### `JoinRoom(roomCode string) (string, error)`
Joins an existing room using a room code and returns an answer code.

**Parameters:**
- `roomCode`: The room code received from the room creator

**Returns:**
- `string`: Answer code to send back to the room creator
- `error`: Error if joining fails

**Example:**
```go
answerCode, err := client.JoinRoom(roomCodeFromFriend)
if err != nil {
    log.Fatal("Failed to join room:", err)
}
fmt.Println("Send this answer code back:", answerCode)
```

#### `AcceptAnswer(answerCode string) error`
Accepts an answer code from someone joining your room (room creator only).

**Parameters:**
- `answerCode`: The answer code received from the person joining

**Returns:**
- `error`: Error if accepting the answer fails

**Example:**
```go
err := client.AcceptAnswer(answerCodeFromJoiner)
if err != nil {
    log.Fatal("Failed to accept answer:", err)
}
```

### Messaging

#### `SendMessage(text string) error`
Sends a chat message to the connected peer.

**Parameters:**
- `text`: The message text (cannot be empty)

**Returns:**
- `error`: Error if sending fails

**Example:**
```go
err := client.SendMessage("Hello, world!")
if err != nil {
    log.Printf("Failed to send message: %v", err)
}
```

### Event Handlers

#### `OnMessage(callback func(protocol.Message))`
Sets a callback for receiving messages.

**Parameters:**
- `callback`: Function called when a message is received

**Message Structure:**
```go
type Message struct {
    Type protocol.MessageType // TypeChat, TypeJoin, TypeLeave
    From string              // Sender's username
    Text string              // Message content
    // ... other fields
}
```

**Example:**
```go
client.OnMessage(func(msg protocol.Message) {
    switch msg.Type {
    case protocol.TypeChat:
        fmt.Printf("%s: %s\n", msg.From, msg.Text)
    case protocol.TypeJoin:
        fmt.Printf("%s joined the chat\n", msg.From)
    case protocol.TypeLeave:
        fmt.Printf("%s left the chat\n", msg.From)
    }
})
```

#### `OnConnected(callback func())`
Sets a callback for when connection is established.

**Example:**
```go
client.OnConnected(func() {
    fmt.Println("Successfully connected to peer!")
})
```

#### `OnDisconnected(callback func())`
Sets a callback for when connection is lost.

**Example:**
```go
client.OnDisconnected(func() {
    fmt.Println("Disconnected from peer")
})
```

#### `OnError(callback func(error))`
Sets a callback for error events.

**Example:**
```go
client.OnError(func(err error) {
    log.Printf("Client error: %v", err)
})
```

### Status and Information

#### `GetUsername() string`
Returns the current username.

#### `IsConnected() bool`
Returns whether the client is connected to a peer.

#### `GetRoomCode() string`
Returns the current room code (if any).

#### `ConnectionStatus() string`
Returns a user-friendly connection status message.

#### `GetConnectionInstructions() string`
Returns detailed instructions for the connection process.

### Cleanup

#### `Disconnect() error`
Closes the connection and cleans up resources. Always call this when done.

**Returns:**
- `error`: Error if disconnection fails

## Complete Examples

### Example 1: Creating and Hosting a Room

```go
package main

import (
    "fmt"
    "log"
    "bufio"
    "os"
    
    "github.com/Jouini-Mohamed-Chaker/p2p-chat/pkg/client"
    "github.com/Jouini-Mohamed-Chaker/p2p-chat/pkg/protocol"
)

func main() {
    // Create client
    client, err := client.NewChatClient("Host")
    if err != nil {
        log.Fatal(err)
    }
    defer client.Disconnect()
    
    // Set up event handlers
    client.OnMessage(func(msg protocol.Message) {
        fmt.Printf("%s: %s\n", msg.From, msg.Text)
    })
    
    client.OnConnected(func() {
        fmt.Println("Peer connected! You can start chatting.")
    })
    
    // Create room
    roomCode, err := client.CreateRoom()
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Room created!")
    fmt.Println("Share this code with your friend:", roomCode)
    fmt.Println("Waiting for them to join...")
    
    // Wait for answer code
    fmt.Print("Enter the answer code from your friend: ")
    scanner := bufio.NewScanner(os.Stdin)
    scanner.Scan()
    answerCode := scanner.Text()
    
    // Accept the answer
    err = client.AcceptAnswer(answerCode)
    if err != nil {
        log.Fatal(err)
    }
    
    // Chat loop
    fmt.Println("Type messages (empty line to quit):")
    for scanner.Scan() {
        text := scanner.Text()
        if text == "" {
            break
        }
        
        err := client.SendMessage(text)
        if err != nil {
            log.Printf("Failed to send: %v", err)
        }
    }
}
```

### Example 2: Joining an Existing Room

```go
package main

import (
    "fmt"
    "log"
    "bufio"
    "os"
    
    "github.com/Jouini-Mohamed-Chaker/p2p-chat/pkg/client"
    "github.com/Jouini-Mohamed-Chaker/p2p-chat/pkg/protocol"
)

func main() {
    // Create client
    client, err := client.NewChatClient("Joiner")
    if err != nil {
        log.Fatal(err)
    }
    defer client.Disconnect()
    
    // Set up event handlers
    client.OnMessage(func(msg protocol.Message) {
        fmt.Printf("%s: %s\n", msg.From, msg.Text)
    })
    
    client.OnConnected(func() {
        fmt.Println("Connected! You can start chatting.")
    })
    
    // Get room code from user
    scanner := bufio.NewScanner(os.Stdin)
    fmt.Print("Enter the room code: ")
    scanner.Scan()
    roomCode := scanner.Text()
    
    // Join room
    answerCode, err := client.JoinRoom(roomCode)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Send this answer code to the room creator:", answerCode)
    fmt.Println("Connection will establish once they accept it...")
    
    // Chat loop
    fmt.Println("Type messages (empty line to quit):")
    for scanner.Scan() {
        text := scanner.Text()
        if text == "" {
            break
        }
        
        err := client.SendMessage(text)
        if err != nil {
            log.Printf("Failed to send: %v", err)
        }
    }
}
```

## Error Handling

The client returns descriptive errors for common issues:

- `"username cannot be empty"` - When creating client with empty username
- `"already connected to a room"` - When trying to create/join while already connected
- `"room code cannot be empty"` - When joining with empty room code
- `"not connected to any room"` - When trying to send message while disconnected
- `"message text cannot be empty"` - When trying to send empty message

Always check errors and handle them appropriately in your application.

## Thread Safety

The ChatClient is thread-safe. All methods can be called safely from multiple goroutines. Event callbacks are called in separate goroutines, so they won't block the main client operations.

## Dependencies

This package requires the following internal packages:
- `github.com/Jouini-Mohamed-Chaker/p2p-chat/pkg/protocol`
- `github.com/Jouini-Mohamed-Chaker/p2p-chat/pkg/signaling`
- `github.com/Jouini-Mohamed-Chaker/p2p-chat/pkg/webrtc`