# Protocol Package Documentation

The `protocol` package provides message serialization and deserialization for the P2P chat application. It handles JSON-based message format with validation and type safety.

## Message Structure

```go
type Message struct {
    Type      string `json:"type"`      // Message type: "chat", "join", "leave"
    From      string `json:"from"`      // Username/display name
    Text      string `json:"text"`      // Message content (max 1000 chars)
    Timestamp int64  `json:"timestamp"` // Unix milliseconds
}
```

## Core Functions

### Creating Messages

```go
// Create a new message with automatic timestamp
msg := NewMessage(TypeChat, "alice", "Hello world!")

// Or create manually
msg := Message{
    Type:      TypeChat,
    From:      "alice", 
    Text:      "Hello world!",
    Timestamp: time.Now().UnixMilli(),
}
```

### Serialization

```go
// Convert message to JSON bytes with trailing newline
data := Marshal(msg)
// Returns: {"type":"chat","from":"alice","text":"Hello world!","timestamp":1234567890}\n
```

### Deserialization

```go
// Parse JSON bytes into Message with validation
msg, err := Unmarshal(data)
if err != nil {
    // Handle validation errors
    log.Printf("Invalid message: %v", err)
}
```

## Message Types

Use the provided constants for message types:

```go
const (
    TypeChat  = "chat"   // Regular chat message
    TypeJoin  = "join"   // User joined the chat
    TypeLeave = "leave"  // User left the chat
)
```

## Validation Rules

The `Unmarshal` function enforces these rules:

- **Type**: Required, must be "chat", "join", or "leave"
- **From**: Required, cannot be empty
- **Text**: Optional, maximum 1000 characters
- **Timestamp**: Must be non-negative (0 is valid)

## Quick Validation

```go
// Check if message is valid without detailed errors
if msg.IsValid() {
    // Process valid message
}
```

## Error Handling

Common validation errors:

- `"message type is required"` - Missing or empty type field
- `"from field is required"` - Missing or empty from field  
- `"invalid message type"` - Type not in allowed values
- `"message text exceeds maximum length"` - Text > 1000 characters
- `"invalid timestamp"` - Negative timestamp
- `"invalid JSON format"` - Malformed JSON input

## Examples

### Basic Usage

```go
package main

import (
    "fmt"
    "your-project/pkg/protocol"
)

func main() {
    // Create and send a chat message
    msg := protocol.NewMessage(protocol.TypeChat, "alice", "Hello everyone!")
    
    // Serialize for transmission
    data := protocol.Marshal(msg)
    
    // Send data over network...
    
    // Receive and parse message
    received, err := protocol.Unmarshal(data)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }
    
    fmt.Printf("%s: %s\n", received.From, received.Text)
}
```

### Join/Leave Messages

```go
// User joining
joinMsg := protocol.NewMessage(protocol.TypeJoin, "bob", "")

// User leaving with message
leaveMsg := protocol.NewMessage(protocol.TypeLeave, "charlie", "Goodbye!")
```

## Testing

The package includes comprehensive tests covering:

- Marshal/unmarshal roundtrip integrity
- Validation edge cases (empty fields, oversized text, invalid JSON)
- All message types and error conditions
- Maximum text length boundaries

Run tests with:
```bash
go test ./pkg/protocol -v
```

## Thread Safety

All functions are stateless and thread-safe. The `Message` struct is immutable after creation, making it safe for concurrent use.

## Performance Notes

- JSON marshaling/unmarshaling is optimized for small message sizes
- Validation is performed on every unmarshal operation
- Maximum text length prevents memory exhaustion attacks