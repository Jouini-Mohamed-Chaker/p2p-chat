package testutil
/*
Package: testutil (Testing Support)
File: testutil/fake_peer.go
TODO List:

 Create FakePeer struct that implements the webrtc.Peer interface
 Use channels internally to simulate async behavior:

messageChannel chan []byte for received messages
stateChannel chan string for state changes


 Store callbacks in struct fields, call them when channels receive data
 Implement CreateOffer() to return a fake SDP string like "fake-offer-123"
 Implement Send() to write to an internal buffer you can inspect in tests
 Add helper methods like SimulateMessage(data []byte) to trigger callbacks
 Add GetSentMessages() [][]byte to inspect what was sent
 Make it deterministic (no random UUIDs, use counters)

Why This Matters:

Unit tests shouldn't depend on real network connections
You can simulate network failures, delays, message ordering
Tests run fast and are reliable
*/