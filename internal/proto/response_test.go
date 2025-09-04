package proto_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/Abhishek2095/kv-stash/internal/proto"
)

func TestWriteResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		response *proto.Response
		expected string
		wantErr  bool
	}{
		{
			name: "Simple string response",
			response: &proto.Response{
				Type: proto.SimpleString,
				Data: "OK",
			},
			expected: "+OK\r\n",
		},
		{
			name: "Error response",
			response: &proto.Response{
				Type: proto.Error,
				Data: "ERR unknown command",
			},
			expected: "-ERR unknown command\r\n",
		},
		{
			name: "Integer response",
			response: &proto.Response{
				Type: proto.Integer,
				Data: int64(42),
			},
			expected: ":42\r\n",
		},
		{
			name: "Negative integer response",
			response: &proto.Response{
				Type: proto.Integer,
				Data: int64(-1),
			},
			expected: ":-1\r\n",
		},
		{
			name: "Bulk string response",
			response: &proto.Response{
				Type: proto.BulkString,
				Data: "hello",
			},
			expected: "$5\r\nhello\r\n",
		},
		{
			name: "Empty bulk string response",
			response: &proto.Response{
				Type: proto.BulkString,
				Data: "",
			},
			expected: "$0\r\n\r\n",
		},
		{
			name: "Null bulk string response",
			response: &proto.Response{
				Type: proto.NullBulkString,
				Data: nil,
			},
			expected: "$-1\r\n",
		},
		{
			name: "Array response with strings",
			response: &proto.Response{
				Type: proto.Array,
				Data: []any{"hello", "world"},
			},
			expected: "*2\r\n$5\r\nhello\r\n$5\r\nworld\r\n",
		},
		{
			name: "Array response with mixed types",
			response: &proto.Response{
				Type: proto.Array,
				Data: []any{"hello", int64(42), nil},
			},
			expected: "*3\r\n$5\r\nhello\r\n:42\r\n$-1\r\n",
		},
		{
			name: "Empty array response",
			response: &proto.Response{
				Type: proto.Array,
				Data: []any{},
			},
			expected: "*0\r\n",
		},
		{
			name: "Null array response",
			response: &proto.Response{
				Type: proto.Array,
				Data: nil,
			},
			expected: "*-1\r\n",
		},
		{
			name: "Array with int type",
			response: &proto.Response{
				Type: proto.Array,
				Data: []any{int(123)},
			},
			expected: "*1\r\n:123\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			err := proto.WriteResponse(&buf, tt.response)

			if tt.wantErr {
				if err == nil {
					t.Errorf("WriteResponse() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("WriteResponse() error = %v", err)
				return
			}

			result := buf.String()
			if result != tt.expected {
				t.Errorf("WriteResponse() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestResponseConstructors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		response *proto.Response
		expected proto.ResponseType
		data     any
	}{
		{
			name:     "NewSimpleString",
			response: proto.NewSimpleString("OK"),
			expected: proto.SimpleString,
			data:     "OK",
		},
		{
			name:     "NewError",
			response: proto.NewError("ERR invalid"),
			expected: proto.Error,
			data:     "ERR invalid",
		},
		{
			name:     "NewInteger",
			response: proto.NewInteger(123),
			expected: proto.Integer,
			data:     int64(123),
		},
		{
			name:     "NewBulkString",
			response: proto.NewBulkString("hello"),
			expected: proto.BulkString,
			data:     "hello",
		},
		{
			name:     "NewNullBulkString",
			response: proto.NewNullBulkString(),
			expected: proto.NullBulkString,
			data:     nil,
		},
		{
			name:     "NewArray",
			response: proto.NewArray([]any{"a", "b"}),
			expected: proto.Array,
			data:     []any{"a", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.response.Type != tt.expected {
				t.Errorf("Response type = %v, want %v", tt.response.Type, tt.expected)
			}

			if tt.data != nil {
				// Special handling for slices
				if expectedSlice, ok := tt.data.([]any); ok {
					actualSlice, ok := tt.response.Data.([]any)
					if !ok {
						t.Errorf("Response data type mismatch")
						return
					}
					if len(actualSlice) != len(expectedSlice) {
						t.Errorf("Response data slice length = %v, want %v", len(actualSlice), len(expectedSlice))
						return
					}
					for i, v := range expectedSlice {
						if actualSlice[i] != v {
							t.Errorf("Response data slice[%d] = %v, want %v", i, actualSlice[i], v)
						}
					}
				} else {
					// For non-slice types, compare directly
					if fmt.Sprintf("%v", tt.response.Data) != fmt.Sprintf("%v", tt.data) {
						t.Errorf("Response data = %v, want %v", tt.response.Data, tt.data)
					}
				}
			}
		})
	}
}

func TestWriteResponseErrors(t *testing.T) {
	t.Parallel()

	// Test with unknown response type
	response := &proto.Response{
		Type: proto.ResponseType(999), // Invalid type
		Data: "test",
	}

	var buf bytes.Buffer
	err := proto.WriteResponse(&buf, response)
	if err == nil {
		t.Errorf("WriteResponse() with invalid type expected error, got nil")
	}

	expectedErrMsg := "unknown response type"
	if !strings.Contains(err.Error(), expectedErrMsg) {
		t.Errorf("WriteResponse() error = %v, want error containing %q", err, expectedErrMsg)
	}
}

func TestComplexArrayResponse(t *testing.T) {
	t.Parallel()

	// Test nested array-like structures
	response := proto.NewArray([]any{
		"first",
		int64(42),
		nil,
		"last",
	})

	var buf bytes.Buffer
	err := proto.WriteResponse(&buf, response)
	if err != nil {
		t.Errorf("WriteResponse() error = %v", err)
		return
	}

	expected := "*4\r\n$5\r\nfirst\r\n:42\r\n$-1\r\n$4\r\nlast\r\n"
	result := buf.String()
	if result != expected {
		t.Errorf("WriteResponse() = %q, want %q", result, expected)
	}
}

func TestBulkStringWithSpecialCharacters(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		data string
	}{
		{"Bulk string with CRLF", "hello\r\nworld"},
		{"Bulk string with null byte", "hello\x00world"},
		{"Bulk string with unicode", "hello世界"},
		{"Bulk string with only CRLF", "\r\n"},
		{"Bulk string with tabs", "hello\tworld"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			response := proto.NewBulkString(tc.data)
			var buf bytes.Buffer
			err := proto.WriteResponse(&buf, response)
			if err != nil {
				t.Errorf("WriteResponse() error = %v", err)
				return
			}

			expectedPrefix := fmt.Sprintf("$%d\r\n", len(tc.data))
			expectedSuffix := "\r\n"
			expected := expectedPrefix + tc.data + expectedSuffix

			result := buf.String()
			if result != expected {
				t.Errorf("WriteResponse() = %q, want %q", result, expected)
			}
		})
	}
}
