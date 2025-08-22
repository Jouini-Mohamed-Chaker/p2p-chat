package webrtc

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRealPeer(t *testing.T) {
	peer, err := NewRealPeer()
	require.NoError(t, err)
	require.NotNil(t, peer)
	require.NotNil(t, peer.pc)
	
	// Clean up
	defer peer.Close()
}

func TestRealPeer_CreateOffer(t *testing.T) {
	peer, err := NewRealPeer()
	require.NoError(t, err)
	defer peer.Close()
	
	offer, err := peer.CreateOffer()
	require.NoError(t, err)
	require.NotEmpty(t, offer)
	
	// Verify offer is valid JSON with required fields
	var offerMap map[string]interface{}
	err = json.Unmarshal([]byte(offer), &offerMap)
	require.NoError(t, err)
	
	assert.Equal(t, "offer", offerMap["type"])
	assert.NotEmpty(t, offerMap["sdp"])
	
	// Verify SDP contains basic WebRTC components
	sdp := offerMap["sdp"].(string)
	assert.Contains(t, sdp, "v=0") // Version
	assert.Contains(t, sdp, "m=application") // Media line for datachannel
}

func TestRealPeer_CreateAnswer(t *testing.T) {
	// Create two peers
	offerer, err := NewRealPeer()
	require.NoError(t, err)
	defer offerer.Close()
	
	answerer, err := NewRealPeer()
	require.NoError(t, err)
	defer answerer.Close()
	
	// Create offer
	offer, err := offerer.CreateOffer()
	require.NoError(t, err)
	
	// Create answer
	answer, err := answerer.CreateAnswer(offer)
	require.NoError(t, err)
	require.NotEmpty(t, answer)
	
	// Verify answer is valid JSON with required fields
	var answerMap map[string]interface{}
	err = json.Unmarshal([]byte(answer), &answerMap)
	require.NoError(t, err)
	
	assert.Equal(t, "answer", answerMap["type"])
	assert.NotEmpty(t, answerMap["sdp"])
}

func TestRealPeer_SetRemoteAnswer(t *testing.T) {
	// Create two peers
	offerer, err := NewRealPeer()
	require.NoError(t, err)
	defer offerer.Close()
	
	answerer, err := NewRealPeer()
	require.NoError(t, err)
	defer answerer.Close()
	
	// Complete handshake
	offer, err := offerer.CreateOffer()
	require.NoError(t, err)
	
	answer, err := answerer.CreateAnswer(offer)
	require.NoError(t, err)
	
	// Set remote answer - should not error
	err = offerer.SetRemoteAnswer(answer)
	assert.NoError(t, err)
}

func TestRealPeer_SetRemoteOffer(t *testing.T) {
	offerer, err := NewRealPeer()
	require.NoError(t, err)
	defer offerer.Close()
	
	answerer, err := NewRealPeer()
	require.NoError(t, err)
	defer answerer.Close()
	
	offer, err := offerer.CreateOffer()
	require.NoError(t, err)
	
	// Set remote offer - should not error
	err = answerer.SetRemoteOffer(offer)
	assert.NoError(t, err)
}

func TestRealPeer_SendBeforeConnection(t *testing.T) {
	peer, err := NewRealPeer()
	require.NoError(t, err)
	defer peer.Close()
	
	// Try to send before datachannel is ready
	err = peer.Send([]byte("test message"))
	assert.Error(t, err)
}

func TestRealPeer_OnMessageCallback(t *testing.T) {
	peer, err := NewRealPeer()
	require.NoError(t, err)
	defer peer.Close()
	
	// Set message handler
	peer.OnMessage(func(data []byte) {
		// Callback logic would go here in real usage
	})
	
	// Verify callback was set (we can't easily trigger it without full connection)
	peer.mu.RLock()
	assert.NotNil(t, peer.onMessage)
	peer.mu.RUnlock()
}

func TestRealPeer_OnStateChangeCallback(t *testing.T) {
	peer, err := NewRealPeer()
	require.NoError(t, err)
	defer peer.Close()
	
	// Set state change handler
	peer.OnStateChange(func(state string) {
		// Callback logic would go here in real usage
	})
	
	// Verify callback was set
	peer.mu.RLock()
	assert.NotNil(t, peer.onStateChange)
	peer.mu.RUnlock()
}

func TestRealPeer_Close(t *testing.T) {
	peer, err := NewRealPeer()
	require.NoError(t, err)
	
	// Create offer to initialize datachannel
	_, err = peer.CreateOffer()
	require.NoError(t, err)
	
	// Close should not error
	err = peer.Close()
	assert.NoError(t, err)
	
	// Second close should still not error
	err = peer.Close()
	assert.NoError(t, err)
}

func TestRealPeer_sdpToString(t *testing.T) {
	peer, err := NewRealPeer()
	require.NoError(t, err)
	defer peer.Close()
	
	// Create offer to get a real SessionDescription
	offer, err := peer.CreateOffer()
	require.NoError(t, err)
	
	// Verify it's valid JSON
	var offerMap map[string]interface{}
	err = json.Unmarshal([]byte(offer), &offerMap)
	require.NoError(t, err)
	
	assert.Equal(t, "offer", offerMap["type"])
	assert.NotEmpty(t, offerMap["sdp"])
	assert.IsType(t, "", offerMap["sdp"])
}

func TestRealPeer_stringToSDP(t *testing.T) {
	peer, err := NewRealPeer()
	require.NoError(t, err)
	defer peer.Close()
	
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid offer",
			input:   `{"type":"offer","sdp":"v=0\r\no=- 123 456 IN IP4 0.0.0.0\r\n"}`,
			wantErr: false,
		},
		{
			name:    "valid answer",
			input:   `{"type":"answer","sdp":"v=0\r\no=- 123 456 IN IP4 0.0.0.0\r\n"}`,
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			input:   `{invalid json}`,
			wantErr: true,
		},
		{
			name:    "missing type",
			input:   `{"sdp":"v=0\r\n"}`,
			wantErr: true,
		},
		{
			name:    "missing sdp",
			input:   `{"type":"offer"}`,
			wantErr: true,
		},
		{
			name:    "invalid type",
			input:   `{"type":"invalid","sdp":"v=0\r\n"}`,
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desc, err := peer.stringToSDP(tt.input)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, desc)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, desc)
				assert.Contains(t, strings.ToLower(tt.input), strings.ToLower(desc.Type.String()))
			}
		})
	}
}

func TestRealPeer_CallbackThreadSafety(t *testing.T) {
	peer, err := NewRealPeer()
	require.NoError(t, err)
	defer peer.Close()
	
	// Test concurrent callback registration
	done := make(chan bool, 2)
	
	go func() {
		for i := 0; i < 100; i++ {
			peer.OnMessage(func([]byte) {})
		}
		done <- true
	}()
	
	go func() {
		for i := 0; i < 100; i++ {
			peer.OnStateChange(func(string) {})
		}
		done <- true
	}()
	
	// Wait for both goroutines to complete
	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("Test timed out - possible deadlock")
	}
	
	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("Test timed out - possible deadlock")
	}
	
	// Verify callbacks were set
	peer.mu.RLock()
	assert.NotNil(t, peer.onMessage)
	assert.NotNil(t, peer.onStateChange)
	peer.mu.RUnlock()
}

func TestRealPeer_InterfaceCompliance(t *testing.T) {
	// This test ensures RealPeer implements the Peer interface
	var peer Peer
	realPeer, err := NewRealPeer()
	require.NoError(t, err)
	defer realPeer.Close()
	
	peer = realPeer
	assert.NotNil(t, peer)
	
	// Test that all interface methods are callable
	_, err = peer.CreateOffer()
	assert.NoError(t, err)
	
	// Other methods would require a full connection setup, 
	// so we just verify they exist and don't panic when called with invalid data
	err = peer.SetRemoteAnswer(`{"type":"answer","sdp":"invalid"}`)
	assert.Error(t, err) // Should error on invalid SDP, but not panic
	
	err = peer.Send([]byte("test"))
	assert.Error(t, err) // Should error when not connected
	
	peer.OnMessage(func([]byte) {})     // Should not panic
	peer.OnStateChange(func(string) {}) // Should not panic
	
	err = peer.Close()
	assert.NoError(t, err)
}