package ui

import (
	"fmt"
	"log"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/Jouini-Mohamed-Chaker/p2p-chat/pkg/client"
	"github.com/Jouini-Mohamed-Chaker/p2p-chat/pkg/protocol"
)

// ChatApp represents the main chat application UI
type ChatApp struct {
	app      fyne.App
	window   fyne.Window
	client   *client.ChatClient
	username string

	// UI components
	usernameEntry    *widget.Entry
	connectContainer *fyne.Container
	chatContainer    *fyne.Container
	messageList      *widget.List
	messageEntry     *widget.Entry
	statusLabel      *widget.Label
	roomCodeLabel    *widget.Label

	// Data
	messages []string
}

// NewChatApp creates a new chat application
func NewChatApp() *ChatApp {
	a := app.New()
	w := a.NewWindow("P2P Chat")
	w.Resize(fyne.NewSize(600, 500))

	return &ChatApp{
		app:      a,
		window:   w,
		messages: make([]string, 0),
	}
}

// Run starts the application
func (ca *ChatApp) Run() {
	ca.setupUI()
	ca.window.ShowAndRun()
}

// setupUI creates and arranges the UI components
func (ca *ChatApp) setupUI() {
	// Create components
	ca.createComponents()

	// Initial view - username input
	ca.showUsernameView()
}

// createComponents initializes all UI components
func (ca *ChatApp) createComponents() {
	// Username entry
	ca.usernameEntry = widget.NewEntry()
	ca.usernameEntry.SetPlaceHolder("Enter your username...")

	// Status label
	ca.statusLabel = widget.NewLabel("Enter your username to get started")

	// Room code label
	ca.roomCodeLabel = widget.NewLabel("")

	// Message list
	ca.messageList = widget.NewList(
		func() int {
			return len(ca.messages)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			if id < len(ca.messages) {
				label.SetText(ca.messages[id])
			}
		},
	)

	// Message entry
	ca.messageEntry = widget.NewEntry()
	ca.messageEntry.SetPlaceHolder("Type your message...")
	ca.messageEntry.OnSubmitted = func(text string) {
		ca.sendMessage(text)
	}
}

// showUsernameView displays the username input screen
func (ca *ChatApp) showUsernameView() {
	usernameBtn := widget.NewButton("Continue", func() {
		username := strings.TrimSpace(ca.usernameEntry.Text)
		if username == "" {
			dialog.ShowError(fmt.Errorf("username cannot be empty"), ca.window)
			return
		}
		ca.createClient(username)
	})

	content := container.NewVBox(
		widget.NewCard("Welcome to P2P Chat", "", container.NewVBox(
			ca.usernameEntry,
			usernameBtn,
		)),
		ca.statusLabel,
	)

	ca.window.SetContent(content)
}

// createClient creates a new chat client and shows connection options
func (ca *ChatApp) createClient(username string) {
	var err error
	ca.client, err = client.NewChatClient(username)
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to create client: %v", err), ca.window)
		return
	}

	ca.username = username
	ca.setupClientEventHandlers()
	ca.showConnectionView()
}

// setupClientEventHandlers sets up event handlers for the chat client
func (ca *ChatApp) setupClientEventHandlers() {
	ca.client.OnMessage(func(msg protocol.Message) {
		var displayText string
		switch msg.Type {
		case protocol.TypeChat:
			displayText = fmt.Sprintf("%s: %s", msg.From, msg.Text)
		case protocol.TypeJoin:
			displayText = fmt.Sprintf("*** %s joined the chat", msg.From)
		case protocol.TypeLeave:
			displayText = fmt.Sprintf("*** %s left the chat", msg.From)
		default:
			displayText = fmt.Sprintf("%s: %s", msg.From, msg.Text)
		}

		ca.addMessage(displayText)
	})

	ca.client.OnConnected(func() {
		ca.statusLabel.SetText("Connected! You can now chat.")
		ca.showChatView()
	})

	ca.client.OnDisconnected(func() {
		ca.statusLabel.SetText("Disconnected from peer")
		ca.addMessage("*** Connection lost")
	})

	ca.client.OnError(func(err error) {
		ca.addMessage(fmt.Sprintf("*** Error: %v", err))
		log.Printf("Client error: %v", err)
	})
}

// showConnectionView displays the connection options (create or join room)
func (ca *ChatApp) showConnectionView() {
	createBtn := widget.NewButton("Create Room", ca.createRoom)
	joinBtn := widget.NewButton("Join Room", ca.showJoinRoomDialog)

	ca.connectContainer = container.NewVBox(
		widget.NewCard("Connection", fmt.Sprintf("Hello, %s!", ca.username), container.NewVBox(
			widget.NewLabel("Choose an option:"),
			createBtn,
			joinBtn,
		)),
		ca.roomCodeLabel,
		ca.statusLabel,
	)

	ca.window.SetContent(ca.connectContainer)
}

// createRoom creates a new chat room
func (ca *ChatApp) createRoom() {
	roomCode, err := ca.client.CreateRoom()
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to create room: %v", err), ca.window)
		return
	}

	ca.roomCodeLabel.SetText(fmt.Sprintf("Room Code: %s", roomCode))
	ca.statusLabel.SetText("Room created! Share the room code with your friend.")

	// Show dialog to get answer code
	ca.showAnswerCodeDialog()
}

// showJoinRoomDialog shows dialog to enter room code
func (ca *ChatApp) showJoinRoomDialog() {
	roomEntry := widget.NewEntry()
	roomEntry.SetPlaceHolder("Enter room code...")

	dialog.ShowForm("Join Room", "Join", "Cancel",
		[]*widget.FormItem{
			{Text: "Room Code:", Widget: roomEntry},
		},
		func(confirmed bool) {
			if !confirmed {
				return
			}
			roomCode := strings.TrimSpace(roomEntry.Text)
			if roomCode == "" {
				dialog.ShowError(fmt.Errorf("room code cannot be empty"), ca.window)
				return
			}
			ca.joinRoom(roomCode)
		}, ca.window)
}

// joinRoom joins an existing room
func (ca *ChatApp) joinRoom(roomCode string) {
	answerCode, err := ca.client.JoinRoom(roomCode)
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to join room: %v", err), ca.window)
		return
	}

	ca.statusLabel.SetText("Joined room! Send the answer code to the room creator.")

	// Show answer code to user
	dialog.ShowInformation("Answer Code", 
		fmt.Sprintf("Send this answer code to the room creator:\n\n%s", answerCode), 
		ca.window)
}

// showAnswerCodeDialog shows dialog to enter answer code (for room creator)
func (ca *ChatApp) showAnswerCodeDialog() {
	answerEntry := widget.NewEntry()
	answerEntry.SetPlaceHolder("Enter answer code from joiner...")

	dialog.ShowForm("Accept Answer", "Accept", "Cancel",
		[]*widget.FormItem{
			{Text: "Answer Code:", Widget: answerEntry},
		},
		func(confirmed bool) {
			if !confirmed {
				return
			}
			answerCode := strings.TrimSpace(answerEntry.Text)
			if answerCode == "" {
				dialog.ShowError(fmt.Errorf("answer code cannot be empty"), ca.window)
				return
			}
			ca.acceptAnswer(answerCode)
		}, ca.window)
}

// acceptAnswer accepts an answer code
func (ca *ChatApp) acceptAnswer(answerCode string) {
	err := ca.client.AcceptAnswer(answerCode)
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to accept answer: %v", err), ca.window)
		return
	}

	ca.statusLabel.SetText("Answer accepted! Establishing connection...")
}

// showChatView displays the main chat interface
func (ca *ChatApp) showChatView() {
	// Send button
	sendBtn := widget.NewButton("Send", func() {
		text := strings.TrimSpace(ca.messageEntry.Text)
		if text != "" {
			ca.sendMessage(text)
		}
	})

	// Message input area
	messageArea := container.NewBorder(nil, nil, nil, sendBtn, ca.messageEntry)

	// Disconnect button
	disconnectBtn := widget.NewButton("Disconnect", func() {
		ca.disconnect()
	})

	// Status area
	statusArea := container.NewBorder(nil, nil, nil, disconnectBtn, ca.statusLabel)

	// Main chat container
	ca.chatContainer = container.NewBorder(
		statusArea,    // top
		messageArea,   // bottom
		nil,          // left
		nil,          // right
		ca.messageList, // center
	)

	ca.window.SetContent(ca.chatContainer)

	// Focus on message entry
	ca.window.Canvas().Focus(ca.messageEntry)
}

// sendMessage sends a message to the peer
func (ca *ChatApp) sendMessage(text string) {
	if ca.client == nil || !ca.client.IsConnected() {
		dialog.ShowError(fmt.Errorf("not connected to any peer"), ca.window)
		return
	}

	err := ca.client.SendMessage(text)
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to send message: %v", err), ca.window)
		return
	}

	// Add our own message to the list
	ca.addMessage(fmt.Sprintf("You: %s", text))
	ca.messageEntry.SetText("")
}

// addMessage adds a message to the message list and scrolls to bottom
func (ca *ChatApp) addMessage(message string) {
	ca.messages = append(ca.messages, message)
	ca.messageList.Refresh()
	
	// Scroll to bottom
	if len(ca.messages) > 0 {
		ca.messageList.ScrollToBottom()
	}
}

// disconnect disconnects from the current session
func (ca *ChatApp) disconnect() {
	if ca.client != nil {
		err := ca.client.Disconnect()
		if err != nil {
			log.Printf("Error disconnecting: %v", err)
		}
		ca.client = nil
	}

	// Reset UI state
	ca.messages = make([]string, 0)
	ca.messageList.Refresh()
	ca.roomCodeLabel.SetText("")
	
	// Go back to connection view
	ca.showConnectionView()
}

// Close handles application cleanup
func (ca *ChatApp) Close() {
	if ca.client != nil {
		ca.client.Disconnect()
	}
}