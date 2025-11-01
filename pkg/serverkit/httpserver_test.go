package serverkit_test

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/lucho2027/lcc/pkg/serverkit"
)

func TestHTTPServerLifecycle(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, "pong")
	})

	// :0 asks the OS to pick a free port, avoiding conflicts in CI.
	s, err := serverkit.NewHTTP("127.0.0.1:0", mux)
	if err != nil {
		t.Fatal(err)
	}

	if err := s.Start(context.Background()); err != nil {
		t.Fatal(err)
	}

	// Give the listener a tick to start
	time.Sleep(50 * time.Millisecond)

	resp, err := http.Get("http://" + s.Addr() + "/ping")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := s.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	<-s.Done()

	if s.Err() != nil {
		t.Fatalf("server had error: %v", s.Err())
	}
}
