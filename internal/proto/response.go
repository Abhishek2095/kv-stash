package proto

import (
	"fmt"
	"io"
)

// Response represents a RESP response
type Response struct {
	Type ResponseType
	Data any
}

// ResponseType represents the type of RESP response
type ResponseType int

const (
	// SimpleString represents a RESP simple string response type
	SimpleString ResponseType = iota
	// Error represents a RESP error response type
	Error
	// Integer represents a RESP integer response type
	Integer
	// BulkString represents a RESP bulk string response type
	BulkString
	// Array represents a RESP array response type
	Array
	// NullBulkString represents a RESP null bulk string response type
	NullBulkString
)

// WriteResponse writes a RESP response to the writer
func WriteResponse(w io.Writer, resp *Response) error {
	switch resp.Type {
	case SimpleString:
		return writeSimpleString(w, resp.Data.(string))
	case Error:
		return writeError(w, resp.Data.(string))
	case Integer:
		return writeInteger(w, resp.Data.(int64))
	case BulkString:
		return writeBulkString(w, resp.Data.(string))
	case NullBulkString:
		return writeNullBulkString(w)
	case Array:
		if resp.Data == nil {
			return writeArray(w, nil)
		}
		return writeArray(w, resp.Data.([]any))
	default:
		return fmt.Errorf("unknown response type: %d", resp.Type)
	}
}

// writeSimpleString writes a simple string response
func writeSimpleString(w io.Writer, s string) error {
	_, err := fmt.Fprintf(w, "+%s\r\n", s)
	return err
}

// writeError writes an error response
func writeError(w io.Writer, s string) error {
	_, err := fmt.Fprintf(w, "-%s\r\n", s)
	return err
}

// writeInteger writes an integer response
func writeInteger(w io.Writer, i int64) error {
	_, err := fmt.Fprintf(w, ":%d\r\n", i)
	return err
}

// writeBulkString writes a bulk string response
func writeBulkString(w io.Writer, s string) error {
	_, err := fmt.Fprintf(w, "$%d\r\n%s\r\n", len(s), s)
	return err
}

// writeNullBulkString writes a null bulk string response
func writeNullBulkString(w io.Writer) error {
	_, err := w.Write([]byte("$-1\r\n"))
	return err
}

// writeArray writes an array response
func writeArray(w io.Writer, arr []any) error {
	if arr == nil {
		_, err := w.Write([]byte("*-1\r\n"))
		return err
	}

	// Write array length
	if _, err := fmt.Fprintf(w, "*%d\r\n", len(arr)); err != nil {
		return err
	}

	// Write each element
	for _, elem := range arr {
		var resp *Response
		switch v := elem.(type) {
		case string:
			resp = &Response{Type: BulkString, Data: v}
		case int64:
			resp = &Response{Type: Integer, Data: v}
		case int:
			resp = &Response{Type: Integer, Data: int64(v)}
		case nil:
			resp = &Response{Type: NullBulkString}
		default:
			resp = &Response{Type: BulkString, Data: fmt.Sprintf("%v", v)}
		}

		if err := WriteResponse(w, resp); err != nil {
			return err
		}
	}

	return nil
}

// NewSimpleString creates a simple string response
func NewSimpleString(s string) *Response {
	return &Response{Type: SimpleString, Data: s}
}

// NewError creates an error response
func NewError(s string) *Response {
	return &Response{Type: Error, Data: s}
}

// NewInteger creates an integer response
func NewInteger(i int64) *Response {
	return &Response{Type: Integer, Data: i}
}

// NewBulkString creates a bulk string response
func NewBulkString(s string) *Response {
	return &Response{Type: BulkString, Data: s}
}

// NewNullBulkString creates a null bulk string response
func NewNullBulkString() *Response {
	return &Response{Type: NullBulkString}
}

// NewArray creates an array response
func NewArray(arr []any) *Response {
	return &Response{Type: Array, Data: arr}
}
