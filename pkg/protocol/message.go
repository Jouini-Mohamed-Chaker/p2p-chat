package protocol
/*
Package: pkg/protocol (Start Here - It's the Foundation)
File: pkg/protocol/message.go
TODO List:

 Define a simple Message struct with these fields:

Type string (values: "chat", "join", "leave")
From string (username/display name)
Text string (message content)
Timestamp int64 (unix milliseconds)


 Create Marshal(msg Message) []byte function that converts Message to JSON + newline
 Create Unmarshal(data []byte) (Message, error) function that parses JSON
 Add basic validation in Unmarshal (check required fields, max text length of 1000 chars)
 Write unit tests for marshal/unmarshal roundtrip
 Test edge cases: empty message, too long text, invalid JSON
 */