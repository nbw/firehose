package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newSSEServer(body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}))
}

func collect(srv *httptest.Server, opts StreamOptions) ([]StreamEvent, error) {
	c := New(Options{BaseURL: srv.URL, TapToken: "fh_test"})
	var events []StreamEvent
	err := c.Stream(context.Background(), opts, func(ev StreamEvent) error {
		events = append(events, ev)
		return nil
	})
	return events, err
}

func TestStream_BasicEvents(t *testing.T) {
	body := "event: connected\ndata: []\n\n" +
		"id: 0-1\nevent: update\ndata: {\"query_id\":\"1\"}\n\n" +
		"event: end\ndata: []\n\n"
	srv := newSSEServer(body)
	defer srv.Close()

	events, err := collect(srv, StreamOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("got %d events, want 3: %+v", len(events), events)
	}
	if events[0].Event != "connected" {
		t.Errorf("events[0] event = %q, want connected", events[0].Event)
	}
	if events[1].Event != "update" || events[1].ID != "0-1" {
		t.Errorf("events[1] = %+v", events[1])
	}
	if string(events[1].Data) != `{"query_id":"1"}` {
		t.Errorf("events[1] data = %q", events[1].Data)
	}
	if events[2].Event != "end" {
		t.Errorf("events[2] event = %q", events[2].Event)
	}
}

func TestStream_CRLFLineEndings(t *testing.T) {
	body := "event: update\r\ndata: {\"a\":1}\r\n\r\nevent: end\r\ndata: []\r\n\r\n"
	srv := newSSEServer(body)
	defer srv.Close()
	events, err := collect(srv, StreamOptions{})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(events) != 2 || string(events[0].Data) != `{"a":1}` {
		t.Fatalf("got %+v", events)
	}
}

func TestStream_CommentLinesIgnored(t *testing.T) {
	body := ": keepalive\n: another comment\nevent: update\ndata: {}\n\nevent: end\ndata: []\n\n"
	srv := newSSEServer(body)
	defer srv.Close()
	events, err := collect(srv, StreamOptions{})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}
	if events[0].Event != "update" {
		t.Errorf("events[0] = %+v", events[0])
	}
}

func TestStream_MultilineData(t *testing.T) {
	body := "event: update\ndata: line1\ndata: line2\ndata: line3\n\nevent: end\ndata: []\n\n"
	srv := newSSEServer(body)
	defer srv.Close()
	events, err := collect(srv, StreamOptions{})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if string(events[0].Data) != "line1\nline2\nline3" {
		t.Errorf("data = %q", events[0].Data)
	}
}

func TestStream_LeadingSpaceStrippedOnce(t *testing.T) {
	// Per spec: exactly one leading space stripped. So "data:  hi" -> " hi".
	body := "event: update\ndata:  hi\n\nevent: end\ndata: []\n\n"
	srv := newSSEServer(body)
	defer srv.Close()
	events, err := collect(srv, StreamOptions{})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if string(events[0].Data) != " hi" {
		t.Errorf("data = %q, want %q", events[0].Data, " hi")
	}
}

func TestStream_LargeFrame(t *testing.T) {
	huge := strings.Repeat("x", 200_000)
	body := fmt.Sprintf("event: update\ndata: %s\n\nevent: end\ndata: []\n\n", huge)
	srv := newSSEServer(body)
	defer srv.Close()
	events, err := collect(srv, StreamOptions{})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(events[0].Data) != 200_000 {
		t.Fatalf("got data length %d, want 200000", len(events[0].Data))
	}
}

func TestStream_GracefulEnd(t *testing.T) {
	body := "event: update\ndata: {}\n\nevent: end\ndata: []\n\n"
	srv := newSSEServer(body)
	defer srv.Close()
	c := New(Options{BaseURL: srv.URL, TapToken: "fh_test"})
	err := c.Stream(context.Background(), StreamOptions{}, func(ev StreamEvent) error { return nil })
	if err != nil {
		t.Fatalf("graceful end should be nil error, got %v", err)
	}
}

func TestStream_MidFrameEOFIsError(t *testing.T) {
	// No trailing blank line — the parser sees io.EOF mid-frame.
	body := "event: update\ndata: {\"q\":1}\n"
	srv := newSSEServer(body)
	defer srv.Close()
	c := New(Options{BaseURL: srv.URL, TapToken: "fh_test"})
	var got int
	err := c.Stream(context.Background(), StreamOptions{}, func(ev StreamEvent) error {
		got++
		return nil
	})
	if err == nil {
		t.Fatal("expected error on mid-frame EOF, got nil")
	}
	if got != 0 {
		t.Errorf("expected 0 dispatched events, got %d", got)
	}
}

func TestStream_AuthHeaderAndQueryParams(t *testing.T) {
	var (
		gotAuth string
		gotURL  string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotURL = r.URL.String()
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("event: end\ndata: []\n\n"))
	}))
	defer srv.Close()

	c := New(Options{BaseURL: srv.URL, TapToken: "fh_secret"})
	opts := StreamOptions{
		Timeout: 60, TimeoutSet: true,
		Since: "5m",
		Limit: 100, LimitSet: true,
	}
	err := c.Stream(context.Background(), opts, func(StreamEvent) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer fh_secret" {
		t.Errorf("auth header = %q", gotAuth)
	}
	if !strings.Contains(gotURL, "timeout=60") || !strings.Contains(gotURL, "since=5m") || !strings.Contains(gotURL, "limit=100") {
		t.Errorf("url = %q, expected timeout/since/limit query params", gotURL)
	}
}

func TestStream_MissingAuth(t *testing.T) {
	c := New(Options{BaseURL: "http://example", TapToken: ""})
	err := c.Stream(context.Background(), StreamOptions{}, func(StreamEvent) error { return nil })
	var missing *ErrMissingAuth
	if !errors.As(err, &missing) {
		t.Fatalf("expected ErrMissingAuth, got %v", err)
	}
}

func TestStream_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte(`{"error":{"message":"bad token"}}`))
	}))
	defer srv.Close()
	c := New(Options{BaseURL: srv.URL, TapToken: "fh_x"})
	err := c.Stream(context.Background(), StreamOptions{}, func(StreamEvent) error { return nil })
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Status != 401 {
		t.Fatalf("expected 401 APIError, got %v", err)
	}
	if apiErr.Msg != "bad token" {
		t.Errorf("msg = %q", apiErr.Msg)
	}
}

func TestStream_ContextCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		f, _ := w.(http.Flusher)
		w.Write([]byte("event: connected\ndata: []\n\n"))
		f.Flush()
		<-r.Context().Done()
	}))
	defer srv.Close()
	c := New(Options{BaseURL: srv.URL, TapToken: "fh_x"})
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	err := c.Stream(ctx, StreamOptions{}, func(StreamEvent) error { return nil })
	if err == nil {
		t.Fatal("expected error on cancel")
	}
}
