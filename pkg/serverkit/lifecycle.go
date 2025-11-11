package serverkit

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Lifecycle manages application startup and graceful shutdown
type Lifecycle struct {
	httpServer      *HTTPServer
	cleanups        []func() error
	shutdownTimeout time.Duration
	started         bool
}

// NewLifecycle creates a new lifecycle manager
func NewLifecycle(httpServer *HTTPServer, shutdownTimeout time.Duration) *Lifecycle {
	if shutdownTimeout == 0 {
		shutdownTimeout = 30 * time.Second
	}
	return &Lifecycle{
		httpServer:      httpServer,
		cleanups:        make([]func() error, 0),
		shutdownTimeout: shutdownTimeout,
	}
}

// RegisterCleanup adds a cleanup function to be called on shutdown
// Cleanup functions are called in reverse order (LIFO)
func (l *Lifecycle) RegisterCleanup(fn func() error) {
	l.cleanups = append(l.cleanups, fn)
}

// Start starts the HTTP server (non-blocking)
func (l *Lifecycle) Start(ctx context.Context) error {
	if l.started {
		return errors.New("lifecycle already started")
	}
	l.started = true

	if err := l.httpServer.Start(ctx); err != nil {
		return err
	}

	log.Printf("Server listening on %s", l.httpServer.Addr())
	return nil
}

// Wait blocks until shutdown signal is received or server error occurs
func (l *Lifecycle) Wait() error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		log.Printf("Received signal: %v. Starting graceful shutdown...", sig)
		return nil
	case <-l.httpServer.Done():
		if err := l.httpServer.Err(); err != nil {
			return err
		}
		return nil
	}
}

// Shutdown gracefully shuts down the server and runs cleanup functions
func (l *Lifecycle) Shutdown(ctx context.Context) error {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), l.shutdownTimeout)
		defer cancel()
	}

	// Shutdown HTTP server
	log.Println("Shutting down HTTP server...")
	if err := l.httpServer.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	// Run cleanup functions in reverse order (LIFO)
	log.Println("Running cleanup functions...")
	var cleanupErr error
	for i := len(l.cleanups) - 1; i >= 0; i-- {
		if err := l.cleanups[i](); err != nil {
			log.Printf("Cleanup error: %v", err)
			if cleanupErr == nil {
				cleanupErr = err
			}
		}
	}

	log.Println("Shutdown complete")
	return cleanupErr
}

// Run is a convenience method that calls Start(), Wait(), and Shutdown()
func (l *Lifecycle) Run(ctx context.Context) error {
	// Start server
	if err := l.Start(ctx); err != nil {
		return err
	}

	// Wait for shutdown signal
	if err := l.Wait(); err != nil {
		return err
	}

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), l.shutdownTimeout)
	defer cancel()

	return l.Shutdown(shutdownCtx)
}
