package scp

import (
	"context"

	"github.com/codeninja55/go-radx/dimse/dimse"
)

// DefaultEchoHandler provides a simple C-ECHO handler that always returns success
type DefaultEchoHandler struct{}

// HandleEcho implements EchoHandler
func (h *DefaultEchoHandler) HandleEcho(ctx context.Context, req *EchoRequest) *EchoResponse {
	return &EchoResponse{
		Status: dimse.StatusSuccess,
	}
}

// NewDefaultEchoHandler creates a new default echo handler
func NewDefaultEchoHandler() *DefaultEchoHandler {
	return &DefaultEchoHandler{}
}

// EchoHandlerFunc is a function adapter for EchoHandler
type EchoHandlerFunc func(ctx context.Context, req *EchoRequest) *EchoResponse

// HandleEcho implements EchoHandler
func (f EchoHandlerFunc) HandleEcho(ctx context.Context, req *EchoRequest) *EchoResponse {
	return f(ctx, req)
}

// StoreHandlerFunc is a function adapter for StoreHandler
type StoreHandlerFunc func(ctx context.Context, req *StoreRequest) *StoreResponse

// HandleStore implements StoreHandler
func (f StoreHandlerFunc) HandleStore(ctx context.Context, req *StoreRequest) *StoreResponse {
	return f(ctx, req)
}

// FindHandlerFunc is a function adapter for FindHandler
type FindHandlerFunc func(ctx context.Context, req *FindRequest) *FindResponse

// HandleFind implements FindHandler
func (f FindHandlerFunc) HandleFind(ctx context.Context, req *FindRequest) *FindResponse {
	return f(ctx, req)
}

// GetHandlerFunc is a function adapter for GetHandler
type GetHandlerFunc func(ctx context.Context, req *GetRequest) *GetResponse

// HandleGet implements GetHandler
func (f GetHandlerFunc) HandleGet(ctx context.Context, req *GetRequest) *GetResponse {
	return f(ctx, req)
}

// MoveHandlerFunc is a function adapter for MoveHandler
type MoveHandlerFunc func(ctx context.Context, req *MoveRequest) *MoveResponse

// HandleMove implements MoveHandler
func (f MoveHandlerFunc) HandleMove(ctx context.Context, req *MoveRequest) *MoveResponse {
	return f(ctx, req)
}
