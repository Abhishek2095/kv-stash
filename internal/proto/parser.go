// Package proto implements the RESP2 protocol parser and response utilities for Redis-compatible communication.
package proto

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Command represents a parsed RESP command
type Command struct {
	Name string
	Args []string
}

// Parser handles RESP2 protocol parsing
type Parser struct {
	reader *bufio.Reader
}

// NewParser creates a new RESP parser
func NewParser(r io.Reader) *Parser {
	return &Parser{
		reader: bufio.NewReader(r),
	}
}

// ParseCommand parses a single RESP command
func (p *Parser) ParseCommand() (*Command, error) {
	line, err := p.readLine()
	if err != nil {
		return nil, err
	}

	if len(line) == 0 {
		return nil, errors.New("empty command")
	}

	switch line[0] {
	case '*':
		return p.parseArray(line)
	case '+', '-', ':', '$':
		// Single line commands (inline)
		return p.parseInline(line)
	default:
		// Inline command format
		return p.parseInlineString(line)
	}
}

// parseArray parses an array command (standard RESP format)
func (p *Parser) parseArray(line string) (*Command, error) {
	// Parse array length
	countStr := line[1:]
	count, err := strconv.Atoi(countStr)
	if err != nil {
		return nil, fmt.Errorf("invalid array length: %s", countStr)
	}

	if count < 0 {
		return nil, fmt.Errorf("negative array length: %d", count)
	}

	if count == 0 {
		return &Command{Name: "", Args: []string{}}, nil
	}

	// Read array elements
	elements := make([]string, count)
	for i := range count {
		element, err := p.parseElement()
		if err != nil {
			return nil, fmt.Errorf("failed to parse array element %d: %w", i, err)
		}
		elements[i] = element
	}

	if len(elements) == 0 {
		return &Command{Name: "", Args: []string{}}, nil
	}

	return &Command{
		Name: strings.ToUpper(elements[0]),
		Args: elements[1:],
	}, nil
}

// parseElement parses a single RESP element
func (p *Parser) parseElement() (string, error) {
	line, err := p.readLine()
	if err != nil {
		return "", err
	}

	if len(line) == 0 {
		return "", errors.New("empty element")
	}

	switch line[0] {
	case '$':
		return p.parseBulkString(line)
	case '+':
		return line[1:], nil
	case '-':
		return line[1:], nil
	case ':':
		return line[1:], nil
	default:
		return line, nil
	}
}

// parseBulkString parses a bulk string
func (p *Parser) parseBulkString(line string) (string, error) {
	lengthStr := line[1:]
	length, err := strconv.Atoi(lengthStr)
	if err != nil {
		return "", fmt.Errorf("invalid bulk string length: %s", lengthStr)
	}

	if length < 0 {
		return "", nil // null bulk string
	}

	if length == 0 {
		// Read empty line
		_, err := p.readLine()
		if err != nil {
			return "", err
		}
		return "", nil
	}

	// Read the bulk string data
	data := make([]byte, length)
	if _, err := io.ReadFull(p.reader, data); err != nil {
		return "", fmt.Errorf("failed to read bulk string data: %w", err)
	}

	// Read trailing CRLF
	if _, err := p.readLine(); err != nil {
		return "", fmt.Errorf("failed to read bulk string trailing CRLF: %w", err)
	}

	return string(data), nil
}

// parseInline parses a single-line RESP element
func (p *Parser) parseInline(line string) (*Command, error) {
	switch line[0] {
	case '+':
		return &Command{Name: "RESPONSE", Args: []string{line[1:]}}, nil
	case '-':
		return &Command{Name: "ERROR", Args: []string{line[1:]}}, nil
	case ':':
		return &Command{Name: "INTEGER", Args: []string{line[1:]}}, nil
	default:
		return p.parseInlineString(line)
	}
}

// parseInlineString parses an inline string command
func (p *Parser) parseInlineString(line string) (*Command, error) {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return &Command{Name: "", Args: []string{}}, nil
	}

	return &Command{
		Name: strings.ToUpper(parts[0]),
		Args: parts[1:],
	}, nil
}

// readLine reads a line ending with CRLF
func (p *Parser) readLine() (string, error) {
	line, err := p.reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	// Remove CRLF
	if len(line) >= 2 && line[len(line)-2:] == "\r\n" {
		return line[:len(line)-2], nil
	}

	// Handle LF only
	if len(line) >= 1 && line[len(line)-1:] == "\n" {
		return line[:len(line)-1], nil
	}

	return line, nil
}
