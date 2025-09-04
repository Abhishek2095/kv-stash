// Copyright (c) 2024 Abhishek2095
// SPDX-License-Identifier: MIT

package server

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Abhishek2095/kv-stash/internal/obs"
	"github.com/Abhishek2095/kv-stash/internal/proto"
	"github.com/Abhishek2095/kv-stash/internal/store"
)

// Server represents the main kv-stash server
type Server struct {
	config    *Config
	logger    *obs.Logger
	listener  net.Listener
	store     *store.Store
	metrics   *obs.Metrics
	startTime time.Time

	// Connection management
	connections sync.Map
	connCount   int64

	// Shutdown
	shutdown chan struct{}
	done     chan struct{}
	wg       sync.WaitGroup
}

// New creates a new server instance
func New(config *Config, logger *obs.Logger) (*Server, error) {
	// Create the store
	storeInstance, err := store.New(&store.Config{
		Shards:         config.Server.Shards,
		MaxMemoryBytes: config.Storage.MaxMemoryBytes,
		EvictionPolicy: config.Storage.EvictionPolicy,
	}, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create store: %w", err)
	}

	// Create metrics
	metrics := obs.NewMetrics()

	// Start metrics server
	if config.Observability.PrometheusListen != "" {
		go func() {
			if err := metrics.StartMetricsServer(config.Observability.PrometheusListen, logger); err != nil {
				logger.Error("Failed to start metrics server", "error", err)
			}
		}()
	}

	return &Server{
		config:    config,
		logger:    logger,
		store:     storeInstance,
		metrics:   metrics,
		startTime: time.Now(),
		shutdown:  make(chan struct{}),
		done:      make(chan struct{}),
	}, nil
}

// ListenAndServe starts the server and listens for connections
func (s *Server) ListenAndServe() error {
	listener, err := net.Listen("tcp", s.config.Server.ListenAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.config.Server.ListenAddr, err)
	}
	s.listener = listener

	s.logger.Info("Server listening", "addr", s.config.Server.ListenAddr)

	// Accept connections
	for {
		select {
		case <-s.shutdown:
			return nil
		default:
		}

		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-s.shutdown:
				return nil
			default:
				s.logger.Error("Failed to accept connection", "error", err)
				continue
			}
		}

		// Check connection limits
		if atomic.LoadInt64(&s.connCount) >= int64(s.config.Limits.MaxClients) {
			s.logger.Warn("Connection limit reached, closing new connection")
			conn.Close()
			continue
		}

		// Handle connection
		go s.handleConnection(conn)
	}
}

// handleConnection handles a single client connection
func (s *Server) handleConnection(conn net.Conn) {
	atomic.AddInt64(&s.connCount, 1)
	s.metrics.IncConnections()
	defer func() {
		atomic.AddInt64(&s.connCount, -1)
		s.metrics.DecConnections()
		conn.Close()
	}()

	s.wg.Add(1)
	defer s.wg.Done()

	// Set connection timeouts
	if s.config.Server.ReadTimeout > 0 {
		conn.SetReadDeadline(time.Now().Add(s.config.Server.ReadTimeout))
	}
	if s.config.Server.WriteTimeout > 0 {
		conn.SetWriteDeadline(time.Now().Add(s.config.Server.WriteTimeout))
	}

	clientID := fmt.Sprintf("%s", conn.RemoteAddr())
	s.connections.Store(clientID, conn)
	defer s.connections.Delete(clientID)

	logger := s.logger.WithFields("client", clientID)
	logger.Debug("Client connected")

	// Create RESP parser and handler
	parser := proto.NewParser(conn)
	handler := NewHandler(s.store, s.config, logger)

	// Main request loop
	for {
		select {
		case <-s.shutdown:
			return
		default:
		}

		// Update read deadline
		if s.config.Server.ReadTimeout > 0 {
			conn.SetReadDeadline(time.Now().Add(s.config.Server.ReadTimeout))
		}

		// Parse command
		cmd, err := parser.ParseCommand()
		if err != nil {
			if err.Error() == "EOF" {
				logger.Debug("Client disconnected")
				return
			}
			logger.Debug("Parse error", "error", err)
			// Send error response for protocol errors
			proto.WriteResponse(conn, proto.NewError("ERR Protocol error: "+err.Error()))
			return
		}

		// Update write deadline
		if s.config.Server.WriteTimeout > 0 {
			conn.SetWriteDeadline(time.Now().Add(s.config.Server.WriteTimeout))
		}

		// Handle command with metrics
		s.metrics.IncCommandsInFlight()
		start := time.Now()

		response := handler.HandleCommand(cmd)

		duration := time.Since(start)
		success := response.Type != proto.Error
		s.metrics.RecordCommand(cmd.Name, duration, success)
		s.metrics.DecCommandsInFlight()

		// Update metrics
		s.metrics.SetKeys(s.store.DBSize())
		s.metrics.SetUptime(time.Since(s.startTime))

		// Note: Expired keys are tracked automatically in the store

		// Send response
		if err := proto.WriteResponse(conn, response); err != nil {
			logger.Debug("Write error", "error", err)
			return
		}
	}
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Starting graceful shutdown")

	// Signal shutdown
	close(s.shutdown)

	// Close listener
	if s.listener != nil {
		s.listener.Close()
	}

	// Wait for connections to finish or timeout
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.logger.Info("All connections closed gracefully")
	case <-ctx.Done():
		s.logger.Warn("Shutdown timeout reached, forcing close")
		// Force close all connections
		s.connections.Range(func(key, value any) bool {
			if conn, ok := value.(net.Conn); ok {
				conn.Close()
			}
			return true
		})
	}

	close(s.done)
	return nil
}
