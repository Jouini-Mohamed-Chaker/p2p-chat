# SDP Signaling Codec Documentation

A Go package for efficiently encoding and decoding WebRTC SDP (Session Description Protocol) data for sharing via URLs, QR codes, or other text-based mediums.

## Overview

This package provides a robust codec that:
- **Compresses** SDP data using gzip compression
- **Encodes** compressed data to base64url format (URL-safe)
- **Validates** input and output data integrity
- **Handles** both raw SDP and JSON-wrapped SDP formats
- **Supports** any text data, not just SDP

The encoding process: `Raw SDP → Gzip Compression → Base64URL Encoding → Shareable String`

## Usage

```go
package main

import (
    "fmt"
    "your-module/signaling"
)

func main() {
    // Your SDP data
    sdp := `v=0
o=- 123456 789012 IN IP4 0.0.0.0
s=-
t=0 0
m=application 9 UDP/DTLS/SCTP webrtc-datachannel
c=IN IP4 0.0.0.0
a=ice-ufrag:test
a=ice-pwd:testpassword`

    // Encode for sharing
    encoded, err := signaling.Encode(sdp)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Encoded: %s\n", encoded)

    // Decode back to original
    decoded, err := signaling.Decode(encoded)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Decoded matches: %v\n", decoded == sdp)
}
```

## API Reference

### Core Functions

#### `Encode(sdp string) (string, error)`

Encodes raw SDP string into a compressed, base64url-encoded format suitable for sharing.

**Parameters:**
- `sdp`: Raw SDP string or any text data

**Returns:**
- `string`: Base64url-encoded compressed data (without padding)
- `error`: Error if encoding fails

**Errors:**
- Empty SDP input
- SDP exceeds maximum size (4MB)
- Compression failure

**Example:**
```go
encoded, err := signaling.Encode(sdpString)
if err != nil {
    log.Fatal("Encoding failed:", err)
}
// encoded is now ready for URL sharing or QR codes
```

#### `Decode(encoded string) (string, error)`

Decodes a base64url-encoded string back to the original SDP format.

**Parameters:**
- `encoded`: Base64url-encoded string (from `Encode()`)

**Returns:**
- `string`: Original SDP or text data
- `error`: Error if decoding fails

**Errors:**
- Empty encoded string
- Invalid base64url characters
- Corrupted or invalid gzip data
- Decompressed data exceeds size limits
- Non-printable characters in result

**Example:**
```go
original, err := signaling.Decode(encodedString)
if err != nil {
    log.Fatal("Decoding failed:", err)
}
// original contains the restored SDP
```

### Utility Functions

#### `EstimateCompressionRatio() float64`

Returns the estimated compression ratio for typical SDP data (approximately 0.75).

```go
ratio := signaling.EstimateCompressionRatio()
// ratio ≈ 0.75 (75% of original size after encoding)
```

#### `EstimateEncodedSize(sdpLength int) int`

Estimates the final encoded size for a given input length.

```go
estimatedSize := signaling.EstimateEncodedSize(len(sdpData))
fmt.Printf("Expected encoded size: %d characters\n", estimatedSize)
```

### Constants

```go
const (
    MaxSDPSize       = 4 * 1024 * 1024  // 4MB maximum input size
    MinEncodedLength = 10               // Minimum valid encoded string length
)
```

## Features

### Compression Efficiency

The package uses gzip compression with best compression level, typically achieving:
- **40-60% compression** for realistic SDP data
- **Better compression** for repetitive content
- **Minimal overhead** for small data (though base64 encoding adds ~33% overhead)

### URL-Safe Encoding

- Uses base64url encoding (RFC 4648 Section 5)
- Safe for URLs, QR codes, and copy-paste scenarios
- No padding characters (`=`) in output
- Character set: `A-Z`, `a-z`, `0-9`, `-`, `_`

### Input Validation

- Checks for empty inputs
- Validates size limits (4MB maximum)
- Ensures base64url character validity
- Verifies decompressed data integrity
- Checks for printable text output

### Supported Data Types

While optimized for SDP data, the codec supports:
- Raw SDP strings
- JSON-wrapped SDP (common WebRTC format)
- Any UTF-8 text data
- Unicode content

## Error Handling

The package provides detailed error messages for debugging:

```go
encoded, err := signaling.Encode(sdp)
if err != nil {
    switch {
    case strings.Contains(err.Error(), "empty"):
        // Handle empty input
    case strings.Contains(err.Error(), "too large"):
        // Handle size limit exceeded
    case strings.Contains(err.Error(), "compress"):
        // Handle compression failure
    }
}
```

Common error scenarios:
- **Empty input**: `"SDP cannot be empty"`
- **Size limit**: `"SDP too large: X bytes (max 4194304)"`
- **Invalid encoding**: `"invalid base64url characters in encoded string"`
- **Corruption**: `"failed to decompress data"`

## Performance Considerations

### Memory Usage
- Processes data in memory (no streaming)
- Peak usage: ~3x input size during compression
- Suitable for typical SDP sizes (few KB to few MB)

### Speed
- Fast compression/decompression for typical SDP sizes
- Base64 encoding/decoding is very fast
- Minimal CPU overhead for small to medium datasets

### Concurrent Safety
- All functions are safe for concurrent use
- No shared state or global variables
- Multiple goroutines can encode/decode simultaneously

## Use Cases

### WebRTC Signaling
```go
// Encode offer for sharing
offer := rtcConnection.CreateOffer()
offerSDP, _ := json.Marshal(offer)
encoded, _ := signaling.Encode(string(offerSDP))

// Share via URL: https://example.com/connect?offer=<encoded>
```

### QR Code Generation
```go
encoded, _ := signaling.Encode(sdpData)
// Generate QR code containing the encoded string
// Users scan QR code to get connection info
```

### Copy-Paste Scenarios
```go
// Create shareable text block
encoded, _ := signaling.Encode(connectionInfo)
fmt.Printf("Share this code: %s\n", encoded)
```

## Testing

The package includes comprehensive tests covering:
- Round-trip encoding/decoding
- Edge cases (empty, large, malformed data)
- Compression efficiency validation
- Concurrent access safety
- Real SDP data scenarios
- Unicode and special character handling

Run tests:
```bash
go test ./pkg/signaling -v
```

## Size Optimization Tips

1. **Remove unnecessary SDP lines** before encoding
2. **Use terse attribute names** where possible
3. **Avoid excessive ICE candidates** in initial offers
4. **Consider truncating long session names/descriptions**

## Limitations

- **Maximum size**: 4MB input limit
- **Text only**: Binary data not supported
- **No streaming**: Entire input processed in memory
- **Base64 overhead**: ~33% size increase from base64 encoding
- **Small data penalty**: Gzip headers may increase very small inputs