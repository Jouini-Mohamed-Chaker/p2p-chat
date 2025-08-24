package webrtc

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/pion/webrtc/v3"
)

type Peer interface {
	// Creates and returns an SDP offer as a string
	CreateOffer() (string, error)

	// Sets the remote SDP answer
	SetRemoteAnswer(sdp string) error

	// Creates and returns an SDP answer as a string for the given offer
	CreateAnswer(offer string) (string, error)

	// Sets the remote SDP offer
	SetRemoteOffer(sdp string) error

	// Sends raw byte over the datachannel
	Send(data []byte) error

	// Registers a callback for incoming messages
	OnMessage(callback func([]byte))

	// Registers a callback for connection state change
	OnStateChange(callback func(string))

	// Closes the peer connection
	Close() error
}

// ICEServerConfig represents a TURN/STUN server configuration
type ICEServerConfig struct {
	URLs       interface{} `json:"urls"` // Can be string or []string
	Username   string      `json:"username,omitempty"`
	Credential string      `json:"credential,omitempty"`
}

// TURNCredentials represents the response from OpenRelay API
type TURNCredentials struct {
	ICEServers []ICEServerConfig `json:"iceServers"`
}

// RealPeer implements the peer interface using pion/webrtc
type RealPeer struct {
	pc          *webrtc.PeerConnection
	dataChannel *webrtc.DataChannel

	// Callbacks
	onMessage     func([]byte)
	onStateChange func(string)

	// Mutex to protect callback assignment
	mu sync.RWMutex
}

// getOpenRelayCredentials fetches TURN credentials from OpenRelay API
func getOpenRelayCredentials(apiKey string) ([]webrtc.ICEServer, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	url := fmt.Sprintf("https://jouini.metered.live/api/v1/turn/credentials?apiKey=%s", apiKey)
	
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch TURN credentials: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var credentials []ICEServerConfig
	if err := json.Unmarshal(body, &credentials); err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	// Convert to webrtc.ICEServer format
	var iceServers []webrtc.ICEServer
	for _, cred := range credentials {
		server := webrtc.ICEServer{}
		
		// Handle URLs field - can be string or []string
		switch urls := cred.URLs.(type) {
		case string:
			server.URLs = []string{urls}
		case []string:
			server.URLs = urls
		case []interface{}:
			// Convert []interface{} to []string
			var urlStrings []string
			for _, url := range urls {
				if urlStr, ok := url.(string); ok {
					urlStrings = append(urlStrings, urlStr)
				}
			}
			server.URLs = urlStrings
		default:
			log.Printf("Warning: Unknown URL type for ICE server: %T", urls)
			continue
		}
		
		if cred.Username != "" {
			server.Username = cred.Username
		}
		
		if cred.Credential != "" {
			server.Credential = cred.Credential
		}
		
		iceServers = append(iceServers, server)
	}

	return iceServers, nil
}

// getStaticOpenRelayServers returns hardcoded OpenRelay TURN servers (fallback)
func getStaticOpenRelayServers() []webrtc.ICEServer {
	// Get credentials from environment variables
	username := os.Getenv("OPENRELAY_USERNAME")
	credential := os.Getenv("OPENRELAY_CREDENTIAL")
	
	servers := []webrtc.ICEServer{
		{
			URLs: []string{"stun:stun.relay.metered.ca:80"},
		},
	}
	
	// Only add TURN servers if credentials are available
	if username != "" && credential != "" {
		turnServers := []webrtc.ICEServer{
			{
				URLs:       []string{"turn:standard.relay.metered.ca:80"},
				Username:   username,
				Credential: credential,
			},
			{
				URLs:       []string{"turn:standard.relay.metered.ca:80?transport=tcp"},
				Username:   username,
				Credential: credential,
			},
			{
				URLs:       []string{"turn:standard.relay.metered.ca:443"},
				Username:   username,
				Credential: credential,
			},
			{
				URLs:       []string{"turns:standard.relay.metered.ca:443?transport=tcp"},
				Username:   username,
				Credential: credential,
			},
		}
		servers = append(servers, turnServers...)
	} else {
		log.Println("Warning: TURN credentials not found in environment variables, falling back to STUN only")
	}
	
	return servers
}

// NewRealPeer creates a new RealPeer with OpenRelay TURN configuration
func NewRealPeer() (*RealPeer, error) {
	var iceServers []webrtc.ICEServer
	var err error

	// Try to get API key from environment
	apiKey := os.Getenv("OPENRELAY_API_KEY")
	
	if apiKey != "" {
		// Attempt to fetch dynamic credentials
		log.Println("Fetching TURN credentials from OpenRelay API...")
		iceServers, err = getOpenRelayCredentials(apiKey)
		if err != nil {
			log.Printf("Failed to fetch dynamic TURN credentials: %v", err)
			log.Println("Falling back to static configuration...")
			iceServers = getStaticOpenRelayServers()
		} else {
			log.Printf("Successfully fetched %d ICE servers from API", len(iceServers))
		}
	} else {
		log.Println("No API key found, using static TURN configuration...")
		iceServers = getStaticOpenRelayServers()
	}

	// Add Google STUN as backup
	iceServers = append(iceServers, webrtc.ICEServer{
		URLs: []string{"stun:stun.l.google.com:19302"},
	})

	// Configure ICE servers
	config := webrtc.Configuration{
		ICEServers: iceServers,
	}

	// Log the ICE servers being used (without credentials for security)
	for i, server := range config.ICEServers {
		if server.Username != "" {
			log.Printf("ICE Server %d: %v (with auth)", i, server.URLs)
		} else {
			log.Printf("ICE Server %d: %v", i, server.URLs)
		}
	}

	// Create a peer connection
	pc, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return nil, err
	}

	peer := &RealPeer{
		pc: pc,
	}

	// Set up connection state change handler
	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("Connection state changed: %s", state.String())
		peer.mu.RLock()
		callback := peer.onStateChange
		peer.mu.RUnlock()

		if callback != nil {
			callback(state.String())
		}
	})

	// Setup ICE connection state change handler for additional logging
	pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("ICE connection state changed: %s", state.String())
	})

	// Log ICE candidates for debugging
	pc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate != nil {
			log.Printf("New ICE candidate: %s", candidate.String())
		}
	})

	return peer, nil
}

// Creates and return an SDP offer as a string
func (p *RealPeer) CreateOffer() (string, error) {
	// Create the data channel first (as the offerer)
	if err := p.CreateDataChannel(); err != nil {
		return "", err
	}

	// Create offer
	offer, err := p.pc.CreateOffer(nil)
	if err != nil {
		return "", err
	}

	// Set local description
	if err := p.pc.SetLocalDescription(offer); err != nil {
		return "", err
	}

	// Wait for ICE gathering to complete
	gatherComplete := webrtc.GatheringCompletePromise(p.pc)
	<-gatherComplete

	// return the complete SDP as JSON string
	return p.sdpToString(p.pc.LocalDescription())
}

// Sets the remote SDP answer
func (p *RealPeer) SetRemoteAnswer(sdp string) error {
	sessionDesc, err := p.stringToSDP(sdp)
	if err != nil {
		return err
	}

	return p.pc.SetRemoteDescription(*sessionDesc)
}

// Creates and returns an SDP answer as a string for the given offer
func (p *RealPeer) CreateAnswer(offer string) (string, error) {
	// Set the remote offer first
	if err := p.SetRemoteOffer(offer); err != nil {
		return "", err
	}

	// Create answer
	answer, err := p.pc.CreateAnswer(nil)
	if err != nil {
		return "", err
	}

	// Set local description
	if err := p.pc.SetLocalDescription(answer); err != nil {
		return "", err
	}

	// Wait for ICE gathering to complete
	gatherComplete := webrtc.GatheringCompletePromise(p.pc)
	<-gatherComplete

	// Return the complete SDP as JSON string
	return p.sdpToString(p.pc.LocalDescription())
}

// Sets the remote SDP offer
func (p *RealPeer) SetRemoteOffer(sdp string) error {
	sessionDesc, err := p.stringToSDP(sdp)
	if err != nil {
		return err
	}

	// Set remote description
	if err := p.pc.SetRemoteDescription(*sessionDesc); err != nil {
		return err
	}

	// Set up data channel handler (as the answerer we receive the channel)
	p.pc.OnDataChannel(func(dc *webrtc.DataChannel) {
		log.Printf("Data Channel '%s' opened", dc.Label())
		p.dataChannel = dc
		p.setupDataChannelHandlers()
	})

	return nil
}

// Sends raw bytes over the datachannel
func (p *RealPeer) Send(data []byte) error {
	if p.dataChannel == nil {
		return webrtc.ErrDataChannelNotOpen
	}

	if p.dataChannel.ReadyState() != webrtc.DataChannelStateOpen {
		return webrtc.ErrDataChannelNotOpen
	}

	return p.dataChannel.Send(data)
}

// On message registers a callback for incoming messages
func (p *RealPeer) OnMessage(callback func([]byte)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onMessage = callback
}

// Registers a callback for connection state change
func (p *RealPeer) OnStateChange(callback func(string)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onStateChange = callback
}

// Closes the peer connection
func (p *RealPeer) Close() error {
	if p.dataChannel != nil {
		if err := p.dataChannel.Close(); err != nil {
			log.Printf("Error closing data channel: %v", err)
		}
	}

	if p.pc != nil {
		if err := p.pc.Close(); err != nil {
			log.Printf("Error closing peer connection: %v", err)
			return err
		}
	}

	return nil
}

// Creates the "chat" data channel with ordered delivery
func (p *RealPeer) CreateDataChannel() error {
	// Configure data channel with ordered delivery
	dcConfig := &webrtc.DataChannelInit{
		Ordered: &[]bool{true}[0],
	}

	// Create data channel
	dc, err := p.pc.CreateDataChannel("chat", dcConfig)
	if err != nil {
		return err
	}

	p.dataChannel = dc
	p.setupDataChannelHandlers()

	return nil
}

// Sets up event handlers for data channel
func (p *RealPeer) setupDataChannelHandlers() {
	p.dataChannel.OnOpen(func() {
		log.Printf("Data channel opened")
	})

	p.dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		log.Printf("Received message: %s", string(msg.Data))

		p.mu.RLock()
		callback := p.onMessage
		p.mu.RUnlock()

		if callback != nil {
			callback(msg.Data)
		}
	})

	p.dataChannel.OnClose(func() {
		log.Printf("Data channel closed")
	})

	p.dataChannel.OnError(func(err error) {
		log.Printf("Data channel error: %v", err)
	})
}

// Converts a SessionDescription to a JSON string
func (p *RealPeer) sdpToString(desc *webrtc.SessionDescription) (string, error) {
	if desc == nil {
		return "", webrtc.ErrSessionDescriptionNoFingerprint
	}

	descMap := map[string]interface{}{
		"type": desc.Type.String(),
		"sdp":  desc.SDP,
	}

	jsonBytes, err := json.Marshal(descMap)
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}

// stringToSDP converts a JSON string to a SessionDescription
func (p *RealPeer) stringToSDP(sdpStr string) (*webrtc.SessionDescription, error) {
	var descMap map[string]interface{}
	if err := json.Unmarshal([]byte(sdpStr), &descMap); err != nil {
		return nil, err
	}

	typeStr, ok := descMap["type"].(string)
	if !ok {
		return nil, webrtc.ErrSessionDescriptionNoFingerprint
	}

	sdp, ok := descMap["sdp"].(string)
	if !ok {
		return nil, webrtc.ErrSessionDescriptionNoFingerprint
	}
	// Convert string type to webrtc.SDPType
	var sdpType webrtc.SDPType
	switch typeStr {
	case "offer":
		sdpType = webrtc.SDPTypeOffer
	case "answer":
		sdpType = webrtc.SDPTypeAnswer
	default:
		return nil, webrtc.ErrSessionDescriptionNoFingerprint
	}

	return &webrtc.SessionDescription{
		Type: sdpType,
		SDP:  sdp,
	}, nil
}