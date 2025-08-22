package signaling
/* 
Package: pkg/signaling (SDP Exchange Helper)
File: pkg/signaling/codec.go
TODO List:

 Create Encode(sdp string) string function that:

Takes raw SDP string
Compresses it with gzip
Encodes to base64url (URL-safe base64)
Returns short shareable string


 Create Decode(encoded string) (string, error) function that reverses the process
 Add validation: check if input looks like valid base64
 Write unit tests with real SDP examples
 Handle errors gracefully (return empty string + error, don't panic)

Why This Package:

Raw SDPs are huge (2-4KB), this makes them copy/paste friendly
Later you can add QR code generation here
*/