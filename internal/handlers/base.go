package handlers

import (
	"context"

	"github.com/user/gendocs/internal/config"
	"github.com/user/gendocs/internal/logging"
)

// Handler is the interface that all handlers must implement
type Handler interface {
	// Handle executes the handler logic
	Handle(ctx context.Context) error
}

// BaseHandler provides common functionality for all handlers
type BaseHandler struct {
	Config config.BaseConfig
	Logger *logging.Logger
}

// NewBaseHandler creates a new base handler
func NewBaseHandler(cfg config.BaseConfig, logger *logging.Logger) *BaseHandler {
	return &BaseHandler{
		Config: cfg,
		Logger: logger,
	}
}
