package serverkit

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
	"time"
)

type Hook func(ctx context.Context)

type HTTPServer struct {
	srv     *http.Server
	ln      net.Listener
	done    chan struct{}
	once    sync.Once
	mu      sync.Mutex
	err     error
	onStart []Hook
	onStop  []Hook
}

type Option func(*HTTPServer)

func WithOnStart(h Hook) Option { return func(s *HTTPServer) { s.onStart = append(s.onStart, h) } }

func WithOnStop(h Hook) Option { return func(s *HTTPServer) { s.onStop = append(s.onStop, h) } }

func NewHTTP(addr string, handler http.Handler, opts ...Option) (*HTTPServer, error) {
	s := &HTTPServer{
		srv:  &http.Server{Addr: addr, Handler: handler},
		done: make(chan struct{}),
	}

	for _, o := range opts {
		o(s)
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	s.ln = ln
	return s, nil
}

func (s *HTTPServer) Start(ctx context.Context) error {
	s.once.Do(func() {
		for _, h := range s.onStart {
			h(ctx)
		}
		go func() {
			defer close(s.done)
			if err := s.srv.Serve(s.ln); !errors.Is(err, http.ErrServerClosed) && err != nil {
				s.mu.Lock()
				s.err = err
				s.mu.Unlock()
			}
		}()
	})
	return nil
}

func (s *HTTPServer) Shutdown(ctx context.Context) error {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
	}
	return s.srv.Shutdown(ctx)
}

func (s *HTTPServer) Done() <-chan struct{} { return s.done }
func (s *HTTPServer) Err() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.err
}
func (s *HTTPServer) Addr() string { return s.srv.Addr }
