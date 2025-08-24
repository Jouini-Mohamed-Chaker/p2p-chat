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

	// Room creation/joining UI
	roomCreationContainer *fyne.Container
	roomJoiningContainer  *fyne.Container
	roomCodeEntry         *widget.Entry
	answerCodeEntry       *widget.Entry

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

	// Room code entry (for joining)
	ca.roomCodeEntry = widget.NewEntry()
	ca.roomCodeEntry.SetPlaceHolder("Paste room code here...")
	ca.roomCodeEntry.MultiLine = true

	// Answer code entry (for room creator)
	ca.answerCodeEntry = widget.NewEntry()
	ca.answerCodeEntry.SetPlaceHolder("Paste answer code here...")
	ca.answerCodeEntry.MultiLine = true
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

		// Ensure UI updates happen on the main thread
		fyne.Do(func() {
			ca.addMessage(displayText)
		})
	})

	ca.client.OnConnected(func() {
		// Ensure UI updates happen on the main thread
		fyne.Do(func() {
			ca.statusLabel.SetText("Connected! You can now chat.")
			ca.showChatView()
		})
	})

	ca.client.OnDisconnected(func() {
		// Ensure UI updates happen on the main thread
		fyne.Do(func() {
			ca.statusLabel.SetText("Disconnected from peer")
			ca.addMessage("*** Connection lost")
		})
	})

	ca.client.OnError(func(err error) {
		// Ensure UI updates happen on the main thread
		fyne.Do(func() {
			ca.addMessage(fmt.Sprintf("*** Error: %v", err))
		})
		log.Printf("Client error: %v", err)
	})
}

// showConnectionView displays the connection options (create or join room)
func (ca *ChatApp) showConnectionView() {
	createBtn := widget.NewButton("Create Room", ca.showCreateRoomView)
	joinBtn := widget.NewButton("Join Room", ca.showJoinRoomView)

	ca.connectContainer = container.NewVBox(
		widget.NewCard("Connection", fmt.Sprintf("Hello, %s!", ca.username), container.NewVBox(
			widget.NewLabel("Choose an option:"),
			createBtn,
			joinBtn,
		)),
		ca.statusLabel,
	)

	ca.window.SetContent(ca.connectContainer)
}

// showCreateRoomView shows the room creation interface
func (ca *ChatApp) showCreateRoomView() {
	// Create room immediately
	roomCode, err := ca.client.CreateRoom()
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to create room: %v", err), ca.window)
		return
	}

	// Create a selectable entry for the room code
	roomCodeDisplay := widget.NewEntry()
	roomCodeDisplay.SetText(roomCode)
	roomCodeDisplay.MultiLine = true
	roomCodeDisplay.Wrapping = fyne.TextWrapWord

	copyBtn := widget.NewButton("Copy Room Code", func() {
		ca.window.Clipboard().SetContent(roomCode)
		ca.statusLabel.SetText("Room code copied to clipboard!")
	})

	backBtn := widget.NewButton("Back", func() {
		// Cancel the room creation
		if ca.client != nil {
			ca.client.Disconnect()
		}
		ca.showConnectionView()
	})

	nextBtn := widget.NewButton("Continue", func() {
		ca.showWaitingForAnswerView()
	})

	ca.roomCreationContainer = container.NewVBox(
		widget.NewCard("Room Created", "Share this code with your friend", container.NewVBox(
			widget.NewLabel("Room Code:"),
			roomCodeDisplay,
			copyBtn,
			widget.NewSeparator(),
			widget.NewLabel("Click Continue after sharing the code"),
			container.NewHBox(backBtn, nextBtn),
		)),
		ca.statusLabel,
	)

	ca.statusLabel.SetText("Room created! Share the room code with your friend.")
	ca.window.SetContent(ca.roomCreationContainer)
}

// showWaitingForAnswerView shows interface waiting for answer code
func (ca *ChatApp) showWaitingForAnswerView() {
	backBtn := widget.NewButton("Back", func() {
		ca.showCreateRoomView()
	})

	connectBtn := widget.NewButton("Connect", func() {
		answerCode := strings.TrimSpace(ca.answerCodeEntry.Text)
		if answerCode == "" {
			dialog.ShowError(fmt.Errorf("answer code cannot be empty"), ca.window)
			return
		}
		ca.acceptAnswer(answerCode)
	})

	// Clear previous answer code
	ca.answerCodeEntry.SetText("")

	waitingContainer := container.NewVBox(
		widget.NewCard("Waiting for Friend", "Enter the answer code from your friend", container.NewVBox(
			widget.NewLabel("Answer Code:"),
			ca.answerCodeEntry,
			container.NewHBox(backBtn, connectBtn),
		)),
		ca.statusLabel,
	)

	ca.statusLabel.SetText("Waiting for answer code from your friend...")
	ca.window.SetContent(waitingContainer)
}

// showJoinRoomView shows the room joining interface
func (ca *ChatApp) showJoinRoomView() {
	backBtn := widget.NewButton("Back", func() {
		ca.showConnectionView()
	})

	joinBtn := widget.NewButton("Join Room", func() {
		roomCode := strings.TrimSpace(ca.roomCodeEntry.Text)
		if roomCode == "" {
			dialog.ShowError(fmt.Errorf("room code cannot be empty"), ca.window)
			return
		}
		ca.joinRoom(roomCode)
	})

	// Clear previous room code
	ca.roomCodeEntry.SetText("")

	ca.roomJoiningContainer = container.NewVBox(
		widget.NewCard("Join Room", "Enter the room code from your friend", container.NewVBox(
			widget.NewLabel("Room Code:"),
			ca.roomCodeEntry,
			container.NewHBox(backBtn, joinBtn),
		)),
		ca.statusLabel,
	)

	ca.statusLabel.SetText("Enter the room code to join...")
	ca.window.SetContent(ca.roomJoiningContainer)
}

// joinRoom joins an existing room
func (ca *ChatApp) joinRoom(roomCode string) {
	answerCode, err := ca.client.JoinRoom(roomCode)
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to join room: %v", err), ca.window)
		return
	}

	ca.showAnswerCodeView(answerCode)
}

// showAnswerCodeView shows the answer code that needs to be shared
func (ca *ChatApp) showAnswerCodeView(answerCode string) {
	// Create a selectable entry for the answer code
	answerCodeDisplay := widget.NewEntry()
	answerCodeDisplay.SetText(answerCode)
	answerCodeDisplay.MultiLine = true
	answerCodeDisplay.Wrapping = fyne.TextWrapWord

	copyBtn := widget.NewButton("Copy Answer Code", func() {
		ca.window.Clipboard().SetContent(answerCode)
		ca.statusLabel.SetText("Answer code copied to clipboard!")
	})

	backBtn := widget.NewButton("Back", func() {
		// Cancel the join attempt
		if ca.client != nil {
			ca.client.Disconnect()
		}
		ca.showJoinRoomView()
	})

	answerContainer := container.NewVBox(
		widget.NewCard("Share Answer Code", "Send this code to the room creator", container.NewVBox(
			widget.NewLabel("Answer Code:"),
			answerCodeDisplay,
			copyBtn,
			widget.NewSeparator(),
			widget.NewLabel("Waiting for connection..."),
			backBtn,
		)),
		ca.statusLabel,
	)

	ca.statusLabel.SetText("Share this answer code with the room creator and wait for connection...")
	ca.window.SetContent(answerContainer)
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
	
	// Go back to connection view
	ca.showConnectionView()
}

// Close handles application cleanup
func (ca *ChatApp) Close() {
	if ca.client != nil {
		ca.client.Disconnect()
	}
}