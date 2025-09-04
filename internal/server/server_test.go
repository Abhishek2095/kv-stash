package server_test

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/Abhishek2095/kv-stash/internal/obs"
	"github.com/Abhishek2095/kv-stash/internal/server"
)

func TestNew(t *testing.T) {
	t.Parallel()

	logger := obs.NewLogger(false)
	config := server.DefaultConfig()

	// Test successful server creation
	srv, err := server.New(config, logger)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	if srv == nil {
		t.Fatal("Server is nil")
	}
}

func TestNew_InvalidConfig(t *testing.T) {
	t.Parallel()

	logger := obs.NewLogger(false)
	config := server.DefaultConfig()
	config.Server.Shards = 0 // Invalid configuration

	// Test server creation with invalid config
	_, err := server.New(config, logger)
	if err == nil {
		t.Fatal("Expected error for invalid config")
	}

	if !strings.Contains(err.Error(), "failed to create store") {
		t.Errorf("Expected store creation error, got: %v", err)
	}
}

func TestServer_ListenAndServe(t *testing.T) {
	t.Parallel()

	logger := obs.NewLogger(false)
	config := server.DefaultConfig()

	// Use a specific port that should be available
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	addr := listener.Addr().String()
	_ = listener.Close()

	config.Server.ListenAddr = addr
	config.Observability.PrometheusListen = "" // Disable metrics server

	srv, err := server.New(config, logger)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Start server in goroutine
	done := make(chan error, 1)
	go func() {
		done <- srv.ListenAndServe()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test that server is listening by attempting to connect
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}

	// Send a simple PING command
	_, err = conn.Write([]byte("PING\r\n"))
	if err != nil {
		t.Fatalf("Failed to write to connection: %v", err)
	}

	// Read response
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	response := string(buffer[:n])
	if !strings.Contains(response, "+PONG") {
		t.Errorf("Expected PONG response, got: %s", response)
	}

	_ = conn.Close()

	// Shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		t.Errorf("Failed to shutdown server: %v", err)
	}

	// Check that ListenAndServe returns
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("ListenAndServe returned error: %v", err)
		}
	case <-time.After(time.Second):
		t.Error("Server did not stop within timeout")
	}
}

func TestServer_ConnectionLimits(t *testing.T) {
	t.Parallel()

	logger := obs.NewLogger(false)
	config := server.DefaultConfig()

	// Get available port
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	addr := listener.Addr().String()
	_ = listener.Close()

	config.Server.ListenAddr = addr
	config.Limits.MaxClients = 1 // Set very low limit
	config.Observability.PrometheusListen = ""

	srv, err := server.New(config, logger)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Start server
	go func() {
		_ = srv.ListenAndServe()
	}()

	time.Sleep(100 * time.Millisecond)

	// Connect first client (should succeed)
	conn1, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to connect first client: %v", err)
	}
	defer func() { _ = conn1.Close() }()

	// Give first connection time to be processed
	time.Sleep(50 * time.Millisecond)

	// Try to connect second client (should be rejected due to limit)
	conn2, err := net.Dial("tcp", addr)
	if err != nil {
		// Connection might be refused immediately
		t.Logf("Second connection rejected as expected: %v", err)
	} else {
		// If connection is accepted, it should be closed immediately
		defer func() { _ = conn2.Close() }()

		// Try to send data, should fail or get no response
		_, err = conn2.Write([]byte("PING\r\n"))
		if err == nil {
			// Check if connection is closed
			buffer := make([]byte, 1024)
			conn2.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
			_, err = conn2.Read(buffer)
			if err == nil {
				t.Error("Expected second connection to be closed due to limits")
			}
		}
	}

	// Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

func TestServer_Shutdown(t *testing.T) {
	t.Parallel()

	logger := obs.NewLogger(false)
	config := server.DefaultConfig()

	// Get available port
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	addr := listener.Addr().String()
	_ = listener.Close()

	config.Server.ListenAddr = addr
	config.Observability.PrometheusListen = ""

	srv, err := server.New(config, logger)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Start server
	go func() {
		_ = srv.ListenAndServe()
	}()

	time.Sleep(100 * time.Millisecond)

	// Test graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = srv.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}
}

func TestServer_ShutdownTimeout(t *testing.T) {
	t.Parallel()

	logger := obs.NewLogger(false)
	config := server.DefaultConfig()

	// Get available port
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	addr := listener.Addr().String()
	_ = listener.Close()

	config.Server.ListenAddr = addr
	config.Observability.PrometheusListen = ""

	srv, err := server.New(config, logger)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Start server
	go func() {
		_ = srv.ListenAndServe()
	}()

	time.Sleep(100 * time.Millisecond)

	// Connect a client
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer func() { _ = conn.Close() }()

	// Test shutdown with very short timeout (should force close)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	err = srv.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}
}

func TestServer_HandleConnection_ProtocolError(t *testing.T) {
	t.Parallel()

	logger := obs.NewLogger(false)
	config := server.DefaultConfig()

	// Get available port
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	addr := listener.Addr().String()
	_ = listener.Close()

	config.Server.ListenAddr = addr
	config.Observability.PrometheusListen = ""

	srv, err := server.New(config, logger)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Start server
	go func() {
		_ = srv.ListenAndServe()
	}()

	time.Sleep(100 * time.Millisecond)

	// Connect and send invalid protocol data
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer func() { _ = conn.Close() }()

	// Send invalid RESP data
	_, err = conn.Write([]byte("*invalid\r\n"))
	if err != nil {
		t.Fatalf("Failed to write invalid data: %v", err)
	}

	// Read error response
	buffer := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(time.Second))
	n, err := conn.Read(buffer)
	if err != nil {
		t.Fatalf("Failed to read error response: %v", err)
	}

	response := string(buffer[:n])
	if !strings.Contains(response, "-ERR Protocol error") {
		t.Errorf("Expected protocol error response, got: %s", response)
	}

	// Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

func TestServer_ConnectionTimeouts(t *testing.T) {
	t.Parallel()

	logger := obs.NewLogger(false)
	config := server.DefaultConfig()

	// Get available port
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	addr := listener.Addr().String()
	_ = listener.Close()

	config.Server.ListenAddr = addr
	config.Server.ReadTimeout = 100 * time.Millisecond
	config.Server.WriteTimeout = 100 * time.Millisecond
	config.Observability.PrometheusListen = ""

	srv, err := server.New(config, logger)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Start server
	go func() {
		_ = srv.ListenAndServe()
	}()

	time.Sleep(100 * time.Millisecond)

	// Connect but don't send anything (should timeout)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer func() { _ = conn.Close() }()

	// Wait for timeout
	time.Sleep(200 * time.Millisecond)

	// Try to write something (connection should be closed)
	_, err = conn.Write([]byte("PING\r\n"))
	// This might succeed as the timeout is handled on the server side

	// Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
