// Copyright (c) 2024 Abhishek2095
// SPDX-License-Identifier: MIT

package server

import (
	"strconv"
	"strings"
	"time"

	"github.com/Abhishek2095/kv-stash/internal/obs"
	"github.com/Abhishek2095/kv-stash/internal/proto"
	"github.com/Abhishek2095/kv-stash/internal/store"
)

// Handler handles RESP commands
type Handler struct {
	store  *store.Store
	config *Config
	logger *obs.Logger
}

// NewHandler creates a new command handler
func NewHandler(store *store.Store, config *Config, logger *obs.Logger) *Handler {
	return &Handler{
		store:  store,
		config: config,
		logger: logger,
	}
}

// HandleCommand processes a single command
func (h *Handler) HandleCommand(cmd *proto.Command) *proto.Response {
	h.logger.Debug("Handling command", "name", cmd.Name, "args", len(cmd.Args))

	switch cmd.Name {
	case "PING":
		return h.handlePing(cmd.Args)
	case "ECHO":
		return h.handleEcho(cmd.Args)
	case "INFO":
		return h.handleInfo(cmd.Args)
	case "GET":
		return h.handleGet(cmd.Args)
	case "SET":
		return h.handleSet(cmd.Args)
	case "DEL":
		return h.handleDel(cmd.Args)
	case "EXISTS":
		return h.handleExists(cmd.Args)
	case "EXPIRE":
		return h.handleExpire(cmd.Args)
	case "TTL":
		return h.handleTTL(cmd.Args)
	case "DBSIZE":
		return h.handleDBSize(cmd.Args)
	case "MGET":
		return h.handleMGet(cmd.Args)
	case "MSET":
		return h.handleMSet(cmd.Args)
	case "INCR":
		return h.handleIncr(cmd.Args)
	case "DECR":
		return h.handleDecr(cmd.Args)
	case "INCRBY":
		return h.handleIncrBy(cmd.Args)
	case "DECRBY":
		return h.handleDecrBy(cmd.Args)
	case "QUIT":
		return proto.NewSimpleString("OK")
	default:
		return proto.NewError("ERR unknown command '" + cmd.Name + "'")
	}
}

// handlePing handles the PING command
func (h *Handler) handlePing(args []string) *proto.Response {
	if len(args) == 0 {
		return proto.NewSimpleString("PONG")
	}
	if len(args) == 1 {
		return proto.NewBulkString(args[0])
	}
	return proto.NewError("ERR wrong number of arguments for 'ping' command")
}

// handleEcho handles the ECHO command
func (h *Handler) handleEcho(args []string) *proto.Response {
	if len(args) != 1 {
		return proto.NewError("ERR wrong number of arguments for 'echo' command")
	}
	return proto.NewBulkString(args[0])
}

// handleInfo handles the INFO command
func (h *Handler) handleInfo(args []string) *proto.Response {
	info := []string{
		"# Server",
		"kv_stash_version:dev",
		"go_version:go1.25",
		"uptime_in_seconds:0",
		"",
		"# Clients",
		"connected_clients:1",
		"",
		"# Memory",
		"used_memory:0",
		"",
		"# Keyspace",
		"db0:keys=" + strconv.FormatInt(h.store.DBSize(), 10) + ",expires=0,avg_ttl=0",
	}
	return proto.NewBulkString(strings.Join(info, "\r\n"))
}

// handleGet handles the GET command
func (h *Handler) handleGet(args []string) *proto.Response {
	if len(args) != 1 {
		return proto.NewError("ERR wrong number of arguments for 'get' command")
	}

	value, exists := h.store.Get(args[0])
	if !exists {
		return proto.NewNullBulkString()
	}

	return proto.NewBulkString(value)
}

// handleSet handles the SET command
func (h *Handler) handleSet(args []string) *proto.Response {
	if len(args) < 2 {
		return proto.NewError("ERR wrong number of arguments for 'set' command")
	}

	key := args[0]
	value := args[1]
	var expiration *time.Duration

	// Parse options
	for i := 2; i < len(args); i++ {
		option := strings.ToUpper(args[i])
		switch option {
		case "EX":
			if i+1 >= len(args) {
				return proto.NewError("ERR syntax error")
			}
			seconds, err := strconv.Atoi(args[i+1])
			if err != nil {
				return proto.NewError("ERR value is not an integer or out of range")
			}
			duration := time.Duration(seconds) * time.Second
			expiration = &duration
			i++ // skip next argument
		case "PX":
			if i+1 >= len(args) {
				return proto.NewError("ERR syntax error")
			}
			milliseconds, err := strconv.Atoi(args[i+1])
			if err != nil {
				return proto.NewError("ERR value is not an integer or out of range")
			}
			duration := time.Duration(milliseconds) * time.Millisecond
			expiration = &duration
			i++ // skip next argument
		default:
			return proto.NewError("ERR syntax error")
		}
	}

	h.store.Set(key, value, expiration)
	return proto.NewSimpleString("OK")
}

// handleDel handles the DEL command
func (h *Handler) handleDel(args []string) *proto.Response {
	if len(args) == 0 {
		return proto.NewError("ERR wrong number of arguments for 'del' command")
	}

	var deleted int64
	for _, key := range args {
		if h.store.Delete(key) {
			deleted++
		}
	}

	return proto.NewInteger(deleted)
}

// handleExists handles the EXISTS command
func (h *Handler) handleExists(args []string) *proto.Response {
	if len(args) == 0 {
		return proto.NewError("ERR wrong number of arguments for 'exists' command")
	}

	var count int64
	for _, key := range args {
		if h.store.Exists(key) {
			count++
		}
	}

	return proto.NewInteger(count)
}

// handleExpire handles the EXPIRE command
func (h *Handler) handleExpire(args []string) *proto.Response {
	if len(args) != 2 {
		return proto.NewError("ERR wrong number of arguments for 'expire' command")
	}

	key := args[0]
	seconds, err := strconv.Atoi(args[1])
	if err != nil {
		return proto.NewError("ERR value is not an integer or out of range")
	}

	duration := time.Duration(seconds) * time.Second
	if h.store.Expire(key, duration) {
		return proto.NewInteger(1)
	}

	return proto.NewInteger(0)
}

// handleTTL handles the TTL command
func (h *Handler) handleTTL(args []string) *proto.Response {
	if len(args) != 1 {
		return proto.NewError("ERR wrong number of arguments for 'ttl' command")
	}

	ttl := h.store.TTL(args[0])
	return proto.NewInteger(ttl)
}

// handleDBSize handles the DBSIZE command
func (h *Handler) handleDBSize(args []string) *proto.Response {
	if len(args) != 0 {
		return proto.NewError("ERR wrong number of arguments for 'dbsize' command")
	}

	size := h.store.DBSize()
	return proto.NewInteger(size)
}

// handleMGet handles the MGET command
func (h *Handler) handleMGet(args []string) *proto.Response {
	if len(args) == 0 {
		return proto.NewError("ERR wrong number of arguments for 'mget' command")
	}

	values := make([]any, len(args))
	for i, key := range args {
		if value, exists := h.store.Get(key); exists {
			values[i] = value
		} else {
			values[i] = nil
		}
	}

	return proto.NewArray(values)
}

// handleMSet handles the MSET command
func (h *Handler) handleMSet(args []string) *proto.Response {
	if len(args) == 0 || len(args)%2 != 0 {
		return proto.NewError("ERR wrong number of arguments for 'mset' command")
	}

	for i := 0; i < len(args); i += 2 {
		key := args[i]
		value := args[i+1]
		h.store.Set(key, value, nil)
	}

	return proto.NewSimpleString("OK")
}

// handleIncr handles the INCR command
func (h *Handler) handleIncr(args []string) *proto.Response {
	if len(args) != 1 {
		return proto.NewError("ERR wrong number of arguments for 'incr' command")
	}

	return h.incrementBy(args[0], 1)
}

// handleDecr handles the DECR command
func (h *Handler) handleDecr(args []string) *proto.Response {
	if len(args) != 1 {
		return proto.NewError("ERR wrong number of arguments for 'decr' command")
	}

	return h.incrementBy(args[0], -1)
}

// handleIncrBy handles the INCRBY command
func (h *Handler) handleIncrBy(args []string) *proto.Response {
	if len(args) != 2 {
		return proto.NewError("ERR wrong number of arguments for 'incrby' command")
	}

	increment, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return proto.NewError("ERR value is not an integer or out of range")
	}

	return h.incrementBy(args[0], increment)
}

// handleDecrBy handles the DECRBY command
func (h *Handler) handleDecrBy(args []string) *proto.Response {
	if len(args) != 2 {
		return proto.NewError("ERR wrong number of arguments for 'decrby' command")
	}

	decrement, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return proto.NewError("ERR value is not an integer or out of range")
	}

	return h.incrementBy(args[0], -decrement)
}

// incrementBy increments a key by the given amount
func (h *Handler) incrementBy(key string, increment int64) *proto.Response {
	value, exists := h.store.Get(key)
	var current int64

	if exists {
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return proto.NewError("ERR value is not an integer or out of range")
		}
		current = parsed
	}

	newValue := current + increment
	h.store.Set(key, strconv.FormatInt(newValue, 10), nil)
	return proto.NewInteger(newValue)
}
