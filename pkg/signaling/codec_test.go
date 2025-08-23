package signaling

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Real SDP examples for testing
const (
	// Minimal valid SDP
	minimalSDP = `v=0
o=- 123456 789012 IN IP4 0.0.0.0
s=-
t=0 0
m=application 9 UDP/DTLS/SCTP webrtc-datachannel
c=IN IP4 0.0.0.0
a=ice-ufrag:test
a=ice-pwd:testpassword
a=fingerprint:sha-256 AB:CD:EF:12:34:56:78:90:AB:CD:EF:12:34:56:78:90:AB:CD:EF:12:34:56:78:90:AB:CD:EF:12:34:56
a=setup:active
a=mid:0
a=sctp-port:5000
a=max-message-size:262144`

	// More realistic SDP with ICE candidates
	realisticSDP = `v=0
o=- 4611731400430051336 2 IN IP4 127.0.0.1
s=-
t=0 0
a=group:BUNDLE 0
a=extmap-allow-mixed
a=msid-semantic: WMS
m=application 9 UDP/DTLS/SCTP webrtc-datachannel
c=IN IP4 0.0.0.0
a=ice-ufrag:4ZcD
a=ice-pwd:2/1muCWoOi3uEOanAa2d3e
a=ice-options:trickle
a=fingerprint:sha-256 00:11:22:33:44:55:66:77:88:99:AA:BB:CC:DD:EE:FF:00:11:22:33:44:55:66:77:88:99:AA:BB:CC:DD:EE:FF
a=setup:active
a=mid:0
a=sctp-port:5000
a=max-message-size:262144
a=candidate:842163049 1 udp 1677729535 192.168.1.100 54400 typ srflx raddr 192.168.1.100 rport 54400
a=candidate:842163049 1 udp 2113667326 10.0.0.1 54400 typ host
a=candidate:1467250027 1 tcp 1518280447 192.168.1.100 56143 typ srflx raddr 192.168.1.100 rport 56143 tcptype active
a=candidate:1467250027 1 tcp 2113667326 10.0.0.1 56143 typ host tcptype active`

	// JSON-wrapped SDP (like what our WebRTC package produces)
	jsonWrappedSDP = `{"type":"offer","sdp":"v=0\r\no=- 4611731400430051336 2 IN IP4 127.0.0.1\r\ns=-\r\nt=0 0\r\na=group:BUNDLE 0\r\nm=application 9 UDP/DTLS/SCTP webrtc-datachannel\r\nc=IN IP4 0.0.0.0\r\na=ice-ufrag:4ZcD\r\na=ice-pwd:2/1muCWoOi3uEOanAa2d3e\r\na=fingerprint:sha-256 00:11:22:33:44:55:66:77:88:99:AA:BB:CC:DD:EE:FF:00:11:22:33:44:55:66:77:88:99:AA:BB:CC:DD:EE:FF\r\na=setup:active\r\na=mid:0\r\na=sctp-port:5000\r\na=max-message-size:262144"}`
)

func TestEncode(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "minimal SDP",
			input:   minimalSDP,
			wantErr: false,
		},
		{
			name:    "realistic SDP",
			input:   realisticSDP,
			wantErr: false,
		},
		{
			name:    "JSON wrapped SDP",
			input:   jsonWrappedSDP,
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
			errMsg:  "SDP cannot be empty",
		},
		{
			name:    "very large SDP",
			input:   strings.Repeat("a", MaxSDPSize+1),
			wantErr: true,
			errMsg:  "SDP too large",
		},
		{
			name:    "simple text",
			input:   "Hello, World!",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := Encode(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, encoded)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, encoded)

				// Encoded string should be base64url (no padding, URL-safe chars)
				assert.True(t, isValidBase64URL(encoded), "encoded string should be valid base64url")
				
				// Should not contain base64 padding
				assert.NotContains(t, encoded, "=")
				
				// Should be significantly shorter than original for large enough data
				if len(tt.input) > 500 {
					compressionRatio := float64(len(encoded)) / float64(len(tt.input))
					assert.Less(t, compressionRatio, 1.2, "should not expand too much even with overhead")
				}
			}
		})
	}
}

func TestDecode(t *testing.T) {
	// First encode some test data to get valid encoded strings
	validEncoded1, err := Encode(minimalSDP)
	require.NoError(t, err)

	validEncoded2, err := Encode(realisticSDP)
	require.NoError(t, err)

	validEncoded3, err := Encode(jsonWrappedSDP)
	require.NoError(t, err)

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "minimal SDP roundtrip",
			input:   validEncoded1,
			want:    minimalSDP,
			wantErr: false,
		},
		{
			name:    "realistic SDP roundtrip",
			input:   validEncoded2,
			want:    realisticSDP,
			wantErr: false,
		},
		{
			name:    "JSON wrapped SDP roundtrip",
			input:   validEncoded3,
			want:    jsonWrappedSDP,
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
			errMsg:  "encoded string cannot be empty",
		},
		{
			name:    "too short",
			input:   "abc",
			wantErr: true,
			errMsg:  "encoded string too short",
		},
		{
			name:    "invalid base64url chars",
			input:   "invalid+chars/here=",
			wantErr: true,
			errMsg:  "invalid base64url characters",
		},
		{
			name:    "valid base64url but invalid gzip",
			input:   "dGhpcyBpcyBub3QgZ3ppcCBkYXRh",
			wantErr: true,
			errMsg:  "failed to decompress",
		},
		{
			name:    "malformed base64url",
			input:   "!!!invalid!!!",
			wantErr: true,
			errMsg:  "invalid base64url characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoded, err := Decode(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, decoded)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, decoded)
			}
		})
	}
}

func TestEncodeDecodeRoundtrip(t *testing.T) {
	testCases := []string{
		minimalSDP,
		realisticSDP,
		jsonWrappedSDP,
		"Simple text for testing",
		"Unicode test: 🚀 Hello 世界 🌟",
		strings.Repeat("Repetitive text for compression testing. ", 100),
	}

	for i, original := range testCases {
		t.Run(fmt.Sprintf("roundtrip_%d", i), func(t *testing.T) {
			// Encode
			encoded, err := Encode(original)
			require.NoError(t, err)
			require.NotEmpty(t, encoded)

			// Decode
			decoded, err := Decode(encoded)
			require.NoError(t, err)

			// Should match exactly
			assert.Equal(t, original, decoded)
		})
	}
}

func TestIsValidBase64URL(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"", false},
		{"abc123", true},
		{"ABC123", true},
		{"abc-123", true},
		{"abc_123", true},
		{"abc+123", false}, // + not allowed in base64url
		{"abc/123", false}, // / not allowed in base64url
		{"abc=123", false}, // = not allowed (padding removed)
		{"abc 123", false}, // space not allowed
		{"abc\n123", false}, // newline not allowed
		{"validBase64URLstring", true},
		{"valid-Base64URL_string", true},
		{"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isValidBase64URL(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAddBase64Padding(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"a", "a"},
		{"ab", "ab=="},
		{"abc", "abc="},
		{"abcd", "abcd"},
		{"abcde", "abcde"},
		{"abcdef", "abcdef=="},
		{"abcdefg", "abcdefg="},
		{"abcdefgh", "abcdefgh"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := addBase64Padding(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCompressionEfficiency(t *testing.T) {
	// Test that we actually achieve good compression on realistic SDP data
	testCases := []struct {
		name     string
		input    string
		maxRatio float64 // Maximum acceptable ratio (smaller = better compression)
	}{
		{
			name:     "realistic SDP",
			input:    realisticSDP,
			maxRatio: 0.85, // More realistic expectation
		},
		{
			name:     "JSON wrapped SDP",
			input:    jsonWrappedSDP,
			maxRatio: 1.1, // Small JSON may not compress well
		},
		{
			name:     "repetitive text",
			input:    strings.Repeat("This is repetitive text. ", 50),
			maxRatio: 0.2, // Should compress very well
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := Encode(tt.input)
			require.NoError(t, err)

			ratio := float64(len(encoded)) / float64(len(tt.input))
			assert.Less(t, ratio, tt.maxRatio,
				"compression ratio %.2f should be less than %.2f (encoded: %d, original: %d)",
				ratio, tt.maxRatio, len(encoded), len(tt.input))

			// Verify the estimate is reasonable for typical data (not super repetitive text)
			if !strings.Contains(tt.name, "repetitive") {
				estimated := EstimateEncodedSize(len(tt.input))
				actualDiff := float64(abs(len(encoded)-estimated)) / float64(len(encoded))
				assert.Less(t, actualDiff, 1.0, "size estimate should be within 100% of actual for typical data")
			}
		})
	}
}

func TestEstimateCompressionRatio(t *testing.T) {
	ratio := EstimateCompressionRatio()
	assert.Greater(t, ratio, 0.0)
	assert.Less(t, ratio, 1.0)
	assert.InDelta(t, 0.75, ratio, 0.2, "compression ratio should be around 75%")
}

func TestEstimateEncodedSize(t *testing.T) {
	tests := []struct {
		inputSize int
		want      int
	}{
		{0, 0},
		{100, 75},
		{1000, 750},
		{2000, 1500},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("size_%d", tt.inputSize), func(t *testing.T) {
			got := EstimateEncodedSize(tt.inputSize)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEdgeCases(t *testing.T) {
	t.Run("very small SDP", func(t *testing.T) {
		smallSDP := "v=0\no=- 1 1 IN IP4 0.0.0.0\ns=-\nt=0 0"
		encoded, err := Encode(smallSDP)
		require.NoError(t, err)

		decoded, err := Decode(encoded)
		require.NoError(t, err)
		assert.Equal(t, smallSDP, decoded)
	})

	t.Run("SDP with special characters", func(t *testing.T) {
		specialSDP := minimalSDP + "\na=special:chars!@#$%^&*()[]{}|\\:;\"'<>,.?/~`"
		encoded, err := Encode(specialSDP)
		require.NoError(t, err)

		decoded, err := Decode(encoded)
		require.NoError(t, err)
		assert.Equal(t, specialSDP, decoded)
	})

	t.Run("maximum size SDP", func(t *testing.T) {
		maxSDP := strings.Repeat("v=0\n", MaxSDPSize/4) // Stay under limit
		maxSDP = maxSDP[:MaxSDPSize-1] // Ensure exact limit
		maxSDP = "v=0\n" + maxSDP[4:] // Ensure it starts properly

		encoded, err := Encode(maxSDP)
		require.NoError(t, err)

		decoded, err := Decode(encoded)
		require.NoError(t, err)
		assert.Equal(t, maxSDP, decoded)
	})
}

func TestConcurrentAccess(t *testing.T) {
	// Test that encode/decode functions are safe for concurrent use
	const numGoroutines = 10
	const numIterations = 100

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			testData := realisticSDP + fmt.Sprintf("\na=goroutine-id:%d", id)

			for j := 0; j < numIterations; j++ {
				encoded, err := Encode(testData)
				assert.NoError(t, err)

				decoded, err := Decode(encoded)
				assert.NoError(t, err)
				assert.Equal(t, testData, decoded)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("Test timed out - possible deadlock or infinite loop")
		}
	}
}

// Helper function for absolute value
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}