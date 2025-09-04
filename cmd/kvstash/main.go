// Copyright (c) 2024 Abhishek2095
// SPDX-License-Identifier: MIT

// Package main implements the kv-stash server executable.
package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Abhishek2095/kv-stash/internal/obs"
	"github.com/Abhishek2095/kv-stash/internal/server"
)

const (
	defaultConfigPath = "config.yaml"
	defaultAddr       = ":6380"
	shutdownTimeout   = 30 * time.Second
)

func main() {
	var (
		configPath = flag.String("config", defaultConfigPath, "Path to configuration file")
		addr       = flag.String("addr", defaultAddr, "Server listen address")
		version    = flag.Bool("version", false, "Show version information")
		debug      = flag.Bool("debug", false, "Enable debug logging")
	)
	flag.Parse()

	// Initialize logger
	logger := obs.NewLogger(*debug)

	if *version {
		logger.Info("kv-stash version", "version", getVersion())
		os.Exit(0)
	}
	logger.Info("Starting kv-stash server",
		"version", getVersion(),
		"config", *configPath,
		"addr", *addr)

	// Load configuration
	cfg, err := server.LoadConfig(*configPath)
	if err != nil {
		logger.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Override address if provided via flag
	if *addr != defaultAddr {
		cfg.Server.ListenAddr = *addr
	}

	// Create and start server
	srv, err := server.New(cfg, logger)
	if err != nil {
		logger.Error("Failed to create server", "error", err)
		os.Exit(1)
	}

	// Start server in a goroutine
	errCh := make(chan error, 1)
	go func() {
		logger.Info("Server starting", "addr", cfg.Server.ListenAddr)
		errCh <- srv.ListenAndServe()
	}()

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		if err != nil {
			logger.Error("Server error", "error", err)
			os.Exit(1)
		}
	case sig := <-sigCh:
		logger.Info("Received shutdown signal", "signal", sig)
	}

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	logger.Info("Shutting down server gracefully")
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Failed to shutdown server gracefully", "error", err)
		cancel()
		return
	}

	logger.Info("Server shutdown completed")
}

func getVersion() string {
	// This will be populated by build flags
	return "dev"
}
