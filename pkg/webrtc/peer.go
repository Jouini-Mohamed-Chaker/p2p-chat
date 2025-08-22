package webrtc

/*
Package: pkg/webrtc (The Core Connection)
File: pkg/webrtc/peer.go
TODO List:

 Define Peer interface with these methods:

CreateOffer() (string, error) - returns SDP as string
SetRemoteAnswer(sdp string) error
CreateAnswer(offer string) (string, error)
SetRemoteOffer(sdp string) error
Send(data []byte) error - send raw bytes over datachannel
OnMessage(callback func([]byte)) - register message handler
OnStateChange(callback func(string)) - register connection state handler
Close() error


 Create RealPeer struct that implements the interface using pion/webrtc
 In RealPeer constructor, configure ICE servers (just Google STUN for now: stun:stun.l.google.com:19302)
 Set up a single DataChannel named "chat" with ordered delivery
 Handle DataChannel onOpen, onMessage, onClose events
 Convert pion's complex types to simple strings/bytes in the interface
 Add basic error handling and logging (use log.Printf for now)

Implementation Notes:

Keep pion-specific code isolated inside RealPeer
The interface should be simple enough that you can write a fake implementation easily
Don't worry about TURN servers yet - just STUN for direct connections
*/

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/pion/webrtc/v3"
)

type Peer interface {
	// Creates and returns an SDP offer as a string
	CreateOffer() (string, error)

	// Sets the remote SDP answer
	SetRemoteAnswer(sdp string) error

	// Creates and returns an SDP answer as a string for the given offer
	CreateAnswer(offer string) (string , error)

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

// RealPeer implements the peer interface using pion/webrtc
type RealPeer struct {
	pc          *webrtc.PeerConnection
	dataChannel *webrtc.DataChannel

	// Callbacks
	onMessage func([]byte)
	onStateChange func(string)

	// Mutex to protect callback assignment
	mu sync.RWMutex
}

// Creates a new RealPeer with basic STUN config
func NewRealPeer() (*RealPeer, error){
	// Configure ICE servers with Google STUN
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
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
	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState){
		log.Printf("Connection state changed: %s", state.String())
		peer.mu.RLock()
		callback := peer.onStateChange
		peer.mu.RUnlock()

		if callback != nil {
			callback(state.String())
		}
	})

	// Setup ICE connection state change handler for additional logging
	pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState){
		log.Printf("ICE connection state changed: %s", state.String())
	})

	return peer, nil
}

// Creates and return an SDP offer as a string
func (p *RealPeer) CreateOffer() (string, error){
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
func (p *RealPeer) SetRemoteOffer(sdp string) error{
	sessionDesc, err:= p.stringToSDP(sdp)
	if err != nil {
		return err
	}

	// Set remote description
	if err := p.pc.SetRemoteDescription(*sessionDesc) ; err != nil {
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
func (p *RealPeer) OnMessage(callback func([]byte)){
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onMessage = callback
}

// Registers a callback for connection state change
func (p *RealPeer) OnStateChange(callback func(string)){
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onStateChange = callback
}

// Closes the peer connection
func (p *RealPeer) Close() error {
	if p.dataChannel != nil {
		if err := p.dataChannel.Close() ; err != nil {
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
func (p *RealPeer) CreateDataChannel()error {
	// Configure data channel with oredered delivery 
	dcConfig := &webrtc.DataChannelInit{
		Ordered: &[]bool{true}[0],
	}
	
	// Create data channel
	dc , err := p.pc.CreateDataChannel("chat", dcConfig)
	if err != nil {
		return err
	}

	p.dataChannel = dc
	p.setupDataChannelHandlers()

	return nil
}

// Sets up event handlers for data channel
func (p *RealPeer) setupDataChannelHandlers(){
	p.dataChannel.OnOpen(func() {
		log.Printf("Data channel opened")
	})

	p.dataChannel.OnMessage(func(msg webrtc.DataChannelMessage){
		log.Printf("Received message: %s", string(msg.Data))

		p.mu.RLock()
		callback := p.onMessage
		p.mu.RUnlock()


		if callback != nil {
			callback(msg.Data)
		}
	})

	p.dataChannel.OnClose(func(){
		log.Printf("Data channel closed")
	})

	p.dataChannel.OnError(func (err error) {
		log.Printf("Data channel error: %v", err)
	})
}

// Converts a SessionDescription to a JSON string
func (p *RealPeer) sdpToString(desc *webrtc.SessionDescription) (string, error){
	if desc == nil {
		return "", webrtc.ErrSessionDescriptionNoFingerprint
	}

	descMap := map[string] interface{}{
		"type": desc.Type.String(),
		"sdp": desc.SDP,
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