package protocol

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// MessageTestSuite groups related tests together
type MessageTestSuite struct {
	suite.Suite
}

func TestMessageSuite(t *testing.T) {
	suite.Run(t, new(MessageTestSuite))
}

// Test NewMessage constructor
func (suite *MessageTestSuite) TestNewMessage() {
	t := suite.T()
	
	before := time.Now().UnixMilli()
	msg := NewMessage(TypeChat, "alice", "hello world")
	after := time.Now().UnixMilli()
	
	assert.Equal(t, TypeChat, msg.Type)
	assert.Equal(t, "alice", msg.From)
	assert.Equal(t, "hello world", msg.Text)
	assert.GreaterOrEqual(t, msg.Timestamp, before)
	assert.LessOrEqual(t, msg.Timestamp, after)
}

// Test Marshal function
func (suite *MessageTestSuite) TestMarshal() {
	t := suite.T()
	
	testCases := []struct {
		name     string
		message  Message
		expected string
	}{
		{
			name: "chat message",
			message: Message{
				Type:      TypeChat,
				From:      "alice",
				Text:      "hello",
				Timestamp: 1234567890,
			},
			expected: `{"type":"chat","from":"alice","text":"hello","timestamp":1234567890}` + "\n",
		},
		{
			name: "join message",
			message: Message{
				Type:      TypeJoin,
				From:      "bob",
				Text:      "",
				Timestamp: 1234567890,
			},
			expected: `{"type":"join","from":"bob","text":"","timestamp":1234567890}` + "\n",
		},
		{
			name: "message with special characters",
			message: Message{
				Type:      TypeChat,
				From:      "user",
				Text:      "hello \"world\" \n\t",
				Timestamp: 1234567890,
			},
			expected: `{"type":"chat","from":"user","text":"hello \"world\" \n\t","timestamp":1234567890}` + "\n",
		},
		{
			name: "empty message",
			message: Message{},
			expected: `{"type":"","from":"","text":"","timestamp":0}` + "\n",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Marshal(tc.message)
			assert.Equal(t, tc.expected, string(result))
		})
	}
}

// Test Unmarshal function - valid cases
func (suite *MessageTestSuite) TestUnmarshalValid() {
	t := suite.T()
	
	testCases := []struct {
		name     string
		input    string
		expected Message
	}{
		{
			name:  "chat message",
			input: `{"type":"chat","from":"alice","text":"hello","timestamp":1234567890}`,
			expected: Message{
				Type:      TypeChat,
				From:      "alice",
				Text:      "hello",
				Timestamp: 1234567890,
			},
		},
		{
			name:  "with trailing newline",
			input: `{"type":"join","from":"bob","text":"","timestamp":1234567890}` + "\n",
			expected: Message{
				Type:      TypeJoin,
				From:      "bob",
				Text:      "",
				Timestamp: 1234567890,
			},
		},
		{
			name:  "leave message",
			input: `{"type":"leave","from":"charlie","text":"goodbye","timestamp":1234567890}`,
			expected: Message{
				Type:      TypeLeave,
				From:      "charlie",
				Text:      "goodbye",
				Timestamp: 1234567890,
			},
		},
		{
			name:  "maximum text length",
			input: `{"type":"chat","from":"user","text":"` + strings.Repeat("a", MaxTextLength) + `","timestamp":1234567890}`,
			expected: Message{
				Type:      TypeChat,
				From:      "user",
				Text:      strings.Repeat("a", MaxTextLength),
				Timestamp: 1234567890,
			},
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := Unmarshal([]byte(tc.input))
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// Test Unmarshal function - invalid cases
func (suite *MessageTestSuite) TestUnmarshalInvalid() {
	t := suite.T()
	
	testCases := []struct {
		name        string
		input       string
		expectedErr string
	}{
		{
			name:        "invalid JSON",
			input:       `{"type":"chat","from":"alice"`,
			expectedErr: "invalid JSON format",
		},
		{
			name:        "completely invalid JSON",
			input:       `not json at all`,
			expectedErr: "invalid JSON format",
		},
		{
			name:        "empty JSON object",
			input:       `{}`,
			expectedErr: "message type is required",
		},
		{
			name:        "missing type",
			input:       `{"from":"alice","text":"hello","timestamp":1234567890}`,
			expectedErr: "message type is required",
		},
		{
			name:        "missing from",
			input:       `{"type":"chat","text":"hello","timestamp":1234567890}`,
			expectedErr: "from field is required",
		},
		{
			name:        "invalid message type",
			input:       `{"type":"invalid","from":"alice","text":"hello","timestamp":1234567890}`,
			expectedErr: "invalid message type",
		},
		{
			name:        "text too long",
			input:       `{"type":"chat","from":"alice","text":"` + strings.Repeat("a", MaxTextLength+1) + `","timestamp":1234567890}`,
			expectedErr: "message text exceeds maximum length",
		},
		{
			name:        "negative timestamp",
			input:       `{"type":"chat","from":"alice","text":"hello","timestamp":-1}`,
			expectedErr: "invalid timestamp",
		},
		{
			name:        "empty string input",
			input:       "",
			expectedErr: "invalid JSON format",
		},
		{
			name:        "null JSON",
			input:       "null",
			expectedErr: "message type is required",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := Unmarshal([]byte(tc.input))
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectedErr)
			assert.Equal(t, Message{}, result) // Should return zero value on error
		})
	}
}

// Test Marshal/Unmarshal roundtrip
func (suite *MessageTestSuite) TestMarshalUnmarshalRoundtrip() {
	t := suite.T()
	
	testCases := []Message{
		{
			Type:      TypeChat,
			From:      "alice",
			Text:      "hello world",
			Timestamp: 1234567890,
		},
		{
			Type:      TypeJoin,
			From:      "bob",
			Text:      "",
			Timestamp: time.Now().UnixMilli(),
		},
		{
			Type:      TypeLeave,
			From:      "charlie",
			Text:      "goodbye everyone!",
			Timestamp: 9876543210,
		},
		{
			Type:      TypeChat,
			From:      "user",
			Text:      strings.Repeat("test ", 100), // Long but valid text
			Timestamp: 1111111111,
		},
	}
	
	for i, original := range testCases {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			// Marshal the message
			data := Marshal(original)
			assert.NotEmpty(t, data)
			assert.True(t, strings.HasSuffix(string(data), "\n"))
			
			// Unmarshal it back
			result, err := Unmarshal(data)
			require.NoError(t, err)
			
			// Should be identical
			assert.Equal(t, original, result, "roundtrip failed for test case %d", i)
		})
	}
}

// Test IsValid method
func (suite *MessageTestSuite) TestIsValid() {
	t := suite.T()
	
	validMessages := []Message{
		{Type: TypeChat, From: "alice", Text: "hello", Timestamp: 1234567890},
		{Type: TypeJoin, From: "bob", Text: "", Timestamp: 0},
		{Type: TypeLeave, From: "charlie", Text: "bye", Timestamp: 9999999999},
	}
	
	invalidMessages := []Message{
		{Type: "", From: "alice", Text: "hello", Timestamp: 1234567890},           // missing type
		{Type: TypeChat, From: "", Text: "hello", Timestamp: 1234567890},          // missing from
		{Type: "invalid", From: "alice", Text: "hello", Timestamp: 1234567890},    // invalid type
		{Type: TypeChat, From: "alice", Text: strings.Repeat("a", 1001), Timestamp: 1234567890}, // text too long
		{Type: TypeChat, From: "alice", Text: "hello", Timestamp: -1},             // negative timestamp
	}
	
	for i, msg := range validMessages {
		t.Run(fmt.Sprintf("valid_%d", i), func(t *testing.T) {
			assert.True(t, msg.IsValid(), "valid message %d should pass validation", i)
		})
	}
	
	for i, msg := range invalidMessages {
		t.Run(fmt.Sprintf("invalid_%d", i), func(t *testing.T) {
			assert.False(t, msg.IsValid(), "invalid message %d should fail validation", i)
		})
	}
}

// Test String method
func (suite *MessageTestSuite) TestString() {
	t := suite.T()
	
	msg := Message{
		Type:      TypeChat,
		From:      "alice",
		Text:      "hello",
		Timestamp: 1234567890,
	}
	
	result := msg.String()
	expected := `{"type":"chat","from":"alice","text":"hello","timestamp":1234567890}` + "\n"
	
	assert.Equal(t, expected, result)
}

// Test edge cases and error conditions
func (suite *MessageTestSuite) TestEdgeCases() {
	t := suite.T()
	
	t.Run("unmarshal empty byte slice", func(t *testing.T) {
		_, err := Unmarshal([]byte{})
		assert.Error(t, err)
	})
	
	t.Run("unmarshal whitespace only", func(t *testing.T) {
		_, err := Unmarshal([]byte("   \n\t  "))
		assert.Error(t, err)
	})
	
	t.Run("text exactly at limit", func(t *testing.T) {
		msg := Message{
			Type:      TypeChat,
			From:      "user",
			Text:      strings.Repeat("a", MaxTextLength),
			Timestamp: 1234567890,
		}
		assert.True(t, msg.IsValid())
		
		data := Marshal(msg)
		result, err := Unmarshal(data)
		require.NoError(t, err)
		assert.Equal(t, msg, result)
	})
	
	t.Run("zero timestamp is valid", func(t *testing.T) {
		msg := Message{
			Type:      TypeJoin,
			From:      "user",
			Text:      "",
			Timestamp: 0,
		}
		assert.True(t, msg.IsValid())
	})
}