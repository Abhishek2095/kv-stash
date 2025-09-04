package proto_test

import (
	"strings"
	"testing"

	"github.com/Abhishek2095/kv-stash/internal/proto"
)

func TestParser_ParseCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected *proto.Command
		wantErr  bool
	}{
		{
			name:  "PING command array format",
			input: "*1\r\n$4\r\nPING\r\n",
			expected: &proto.Command{
				Name: "PING",
				Args: []string{},
			},
		},
		{
			name:  "SET command with arguments",
			input: "*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n",
			expected: &proto.Command{
				Name: "SET",
				Args: []string{"key", "value"},
			},
		},
		{
			name:  "Inline PING command",
			input: "PING\r\n",
			expected: &proto.Command{
				Name: "PING",
				Args: []string{},
			},
		},
		{
			name:  "Inline GET command",
			input: "GET mykey\r\n",
			expected: &proto.Command{
				Name: "GET",
				Args: []string{"mykey"},
			},
		},
		{
			name:  "Multiple argument command",
			input: "*4\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n$2\r\nEX\r\n",
			expected: &proto.Command{
				Name: "SET",
				Args: []string{"key", "value", "EX"},
			},
		},
		{
			name:  "Empty array command",
			input: "*0\r\n",
			expected: &proto.Command{
				Name: "",
				Args: []string{},
			},
		},
		{
			name:    "Invalid array length",
			input:   "*abc\r\n",
			wantErr: true,
		},
		{
			name:    "Negative array length",
			input:   "*-5\r\n",
			wantErr: true,
		},
		{
			name:    "Empty input",
			input:   "",
			wantErr: true,
		},
		{
			name:  "Simple string response",
			input: "+OK\r\n",
			expected: &proto.Command{
				Name: "RESPONSE",
				Args: []string{"OK"},
			},
		},
		{
			name:  "Error response",
			input: "-ERR unknown command\r\n",
			expected: &proto.Command{
				Name: "ERROR",
				Args: []string{"ERR unknown command"},
			},
		},
		{
			name:  "Integer response",
			input: ":42\r\n",
			expected: &proto.Command{
				Name: "INTEGER",
				Args: []string{"42"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			parser := proto.NewParser(strings.NewReader(tt.input))
			cmd, err := parser.ParseCommand()

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseCommand() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseCommand() error = %v", err)
				return
			}

			if cmd.Name != tt.expected.Name {
				t.Errorf("ParseCommand() name = %v, want %v", cmd.Name, tt.expected.Name)
			}

			if len(cmd.Args) != len(tt.expected.Args) {
				t.Errorf("ParseCommand() args length = %v, want %v", len(cmd.Args), len(tt.expected.Args))
				return
			}

			for i, arg := range cmd.Args {
				if arg != tt.expected.Args[i] {
					t.Errorf("ParseCommand() args[%d] = %v, want %v", i, arg, tt.expected.Args[i])
				}
			}
		})
	}
}

func TestParser_ParseBulkString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "Empty bulk string",
			input:    "$0\r\n\r\n",
			expected: "",
		},
		{
			name:     "Simple bulk string",
			input:    "$5\r\nhello\r\n",
			expected: "hello",
		},
		{
			name:     "Null bulk string",
			input:    "$-1\r\n",
			expected: "",
		},
		{
			name:    "Invalid bulk string length",
			input:   "$abc\r\n",
			wantErr: true,
		},
		{
			name:     "Bulk string with spaces",
			input:    "$11\r\nhello world\r\n",
			expected: "hello world",
		},
		{
			name:     "Bulk string with special characters",
			input:    "$12\r\nhello\r\nworld\r\n",
			expected: "hello\r\nworld",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a two-element array command (like "GET key")
			var commandInput string
			if tt.wantErr {
				commandInput = "*2\r\n$3\r\nGET\r\n" + tt.input
			} else {
				commandInput = "*2\r\n$3\r\nGET\r\n" + tt.input
			}

			reader := strings.NewReader(commandInput)
			parser := proto.NewParser(reader)

			cmd, err := parser.ParseCommand()

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseCommand() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseCommand() error = %v", err)
				return
			}

			// Should have GET command with one argument (the bulk string)
			if cmd.Name != "GET" {
				t.Errorf("ParseCommand() command name = %q, want GET", cmd.Name)
				return
			}

			// For bulk string tests, check the argument content
			if tt.expected != "" {
				if len(cmd.Args) == 0 {
					t.Errorf("ParseCommand() expected args with bulk string content")
					return
				}
				if cmd.Args[0] != tt.expected {
					t.Errorf("ParseCommand() bulk string = %q, want %q", cmd.Args[0], tt.expected)
				}
			} else {
				// For null bulk string and empty bulk string, expect empty string argument
				if len(cmd.Args) == 0 {
					t.Errorf("ParseCommand() expected one argument for bulk string, got none")
					return
				}
				if cmd.Args[0] != "" {
					t.Errorf("ParseCommand() expected empty string argument, got %q", cmd.Args[0])
				}
			}
		})
	}
}
