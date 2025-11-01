package serverkit

import "context"

type lifecycle interface {
	Start(ctx context.Context) error
	Shutdown(ctx context.Context) error
	Done() <-chan struct{}
	Err() error
}
