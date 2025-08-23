package signaling

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

const (
	// MaxSDPSize defines the maximum size we'll accept for an SDP (4MB)
	MaxSDPSize = 4 * 1024 * 1024
	
	// MinEncodedLength is the minimum length for a valid encoded SDP
	MinEncodedLength = 10
)

// Encode takes a raw SDP string, compresses it with gzip, and encodes it to base64url
// Returns a short shareable string suitable for copy/paste or QR codes
func Encode(sdp string) (string, error) {
	if sdp == "" {
		return "", fmt.Errorf("SDP cannot be empty")
	}
	
	if len(sdp) > MaxSDPSize {
		return "", fmt.Errorf("SDP too large: %d bytes (max %d)", len(sdp), MaxSDPSize)
	}
	
	// Compress with gzip
	compressed, err := compressString(sdp)
	if err != nil {
		return "", fmt.Errorf("failed to compress SDP: %w", err)
	}
	
	// Encode to base64url (URL-safe base64)
	encoded := base64.URLEncoding.EncodeToString(compressed)
	
	// Remove padding for shorter URLs (we'll add it back when decoding)
	encoded = strings.TrimRight(encoded, "=")
	
	return encoded, nil
}

// Decode takes a base64url encoded string and returns the original SDP
// Reverses the process: base64url decode -> gzip decompress -> original SDP
func Decode(encoded string) (string, error) {
	if encoded == "" {
		return "", fmt.Errorf("encoded string cannot be empty")
	}
	
	if len(encoded) < MinEncodedLength {
		return "", fmt.Errorf("encoded string too short: %d characters (min %d)", len(encoded), MinEncodedLength)
	}
	
	// Validate that it looks like base64url
	if !isValidBase64URL(encoded) {
		return "", fmt.Errorf("invalid base64url characters in encoded string")
	}
	
	// Add padding back if needed for base64 decoding
	encoded = addBase64Padding(encoded)
	
	// Decode from base64url
	compressed, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64url: %w", err)
	}
	
	// Decompress with gzip
	sdp, err := decompressBytes(compressed)
	if err != nil {
		return "", fmt.Errorf("failed to decompress data: %w", err)
	}
	
	// Basic validation of the result
	if len(sdp) > MaxSDPSize {
		return "", fmt.Errorf("decompressed SDP too large: %d bytes (max %d)", len(sdp), MaxSDPSize)
	}
	
	// Basic validation - should look like SDP or JSON containing SDP
	// We're lenient here since this codec can be used for any text, not just SDP
	if len(sdp) > 0 && !isPrintableText(sdp) {
		return "", fmt.Errorf("result contains non-printable characters")
	}
	
	return sdp, nil
}

// compressString compresses a string using gzip
func compressString(data string) ([]byte, error) {
	var buf bytes.Buffer
	
	// Create gzip writer with best compression
	gzWriter, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	
	// Write the data
	_, err = gzWriter.Write([]byte(data))
	if err != nil {
		gzWriter.Close()
		return nil, err
	}
	
	// Close the writer to flush all data
	err = gzWriter.Close()
	if err != nil {
		return nil, err
	}
	
	return buf.Bytes(), nil
}

// decompressBytes decompresses gzip data and returns as string
func decompressBytes(data []byte) (string, error) {
	// Create a reader from the compressed data
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	defer reader.Close()
	
	// Read all decompressed data
	decompressed, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	
	return string(decompressed), nil
}

// isValidBase64URL checks if a string contains only valid base64url characters
func isValidBase64URL(s string) bool {
	if s == "" {
		return false
	}
	
	// Base64url alphabet: A-Z, a-z, 0-9, -, _
	for _, char := range s {
		if !((char >= 'A' && char <= 'Z') ||
			(char >= 'a' && char <= 'z') ||
			(char >= '0' && char <= '9') ||
			char == '-' || char == '_') {
			return false
		}
	}
	
	return true
}

// addBase64Padding adds the necessary padding to a base64url string for decoding
func addBase64Padding(s string) string {
	// Base64 strings must be a multiple of 4 characters
	switch len(s) % 4 {
	case 2:
		return s + "=="
	case 3:
		return s + "="
	default:
		return s
	}
}

// EstimateCompressionRatio returns the estimated compression ratio for typical data
// This is useful for UI to show expected encoded length
func EstimateCompressionRatio() float64 {
	// For small data (<500 chars), compression may not help much due to gzip overhead
	// For larger data, typical compression is 40-60%
	// Base64 encoding adds ~33% overhead
	// Net result varies by size, but we use a conservative estimate
	return 0.75
}

// isPrintableText checks if the string contains only printable characters
func isPrintableText(s string) bool {
	for _, r := range s {
		// Allow printable ASCII, unicode letters/digits, and common whitespace
		if !(r >= 32 && r <= 126) && r != '\t' && r != '\n' && r != '\r' {
			// Allow unicode letters and digits
			if !((r >= 0x80 && r <= 0x10FFFF) || r == '\u00A0') {
				return false
			}
		}
	}
	return true
}

// EstimateEncodedSize estimates the encoded size for a given SDP length
func EstimateEncodedSize(sdpLength int) int {
	return int(float64(sdpLength) * EstimateCompressionRatio())
}