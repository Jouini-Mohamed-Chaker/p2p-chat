package chat
/*
Application: cmd/chat/main.go (The Minimal CLI)
TODO List:

 Create simple CLI that asks: "Host (h) or Join (j)?"
 Host flow:

Create a RealPeer
Call CreateOffer()
Encode the SDP using signaling.Encode()
Print: "Share this code: [encoded-string]"
Wait for user to paste the answer
Call SetRemoteAnswer()
Start message loop


 Join flow:

Ask user to paste host's offer code
Decode using signaling.Decode()
Create RealPeer and call CreateAnswer()
Encode and print the answer for user to copy back
Call SetRemoteOffer()
Start message loop


 Message loop:

Set up OnMessage handler to print received messages
Read from stdin in a goroutine
Send each line as a protocol.Message with type "chat"
Handle Ctrl+C gracefully (close peer connection)



Keep It Simple:

No fancy UI, just terminal input/output
Use bufio.Scanner to read lines
Print connection state changes so you can see when it connects
Exit cleanly on connection failure
*/