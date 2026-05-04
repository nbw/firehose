//go:build ignore

// Standalone Firehose SSE consumer.
//
// Run it directly: `go run examples/stream.go`. Educational only — the CLI's
// `firehose stream` does the same thing with more features (reconnect,
// JSONL output, signal handling, exit codes). Copy and adapt to your sink.
//
// Stdlib only.

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

type document struct {
	URL          string   `json:"url"`
	Title        string   `json:"title,omitempty"`
	PublishTime  string   `json:"publish_time,omitempty"`
	Language     string   `json:"language,omitempty"`
	PageCategory []string `json:"page_category,omitempty"`
	PageTypes    []string `json:"page_types,omitempty"`
}

type update struct {
	QueryID   string   `json:"query_id"`
	MatchedAt string   `json:"matched_at"`
	TapID     string   `json:"tap_id"`
	Document  document `json:"document"`
}

func main() {
	token := os.Getenv("FIREHOSE_TAP_TOKEN")
	if token == "" {
		fmt.Fprintln(os.Stderr, "FIREHOSE_TAP_TOKEN is required")
		os.Exit(2)
	}
	base := os.Getenv("FIREHOSE_BASE_URL")
	if base == "" {
		base = "https://api.firehose.com"
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, base, token); err != nil {
		if errors.Is(err, context.Canceled) {
			os.Exit(130)
		}
		fmt.Fprintf(os.Stderr, "stream: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, base, token string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/v1/stream?timeout=300", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	hc := &http.Client{
		Timeout: 0,
		Transport: &http.Transport{
			ResponseHeaderTimeout: 30 * time.Second,
		},
	}
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	r := bufio.NewReader(resp.Body)
	var (
		eventName string
		dataBuf   strings.Builder
	)
	for {
		line, err := r.ReadString('\n')
		eof := errors.Is(err, io.EOF)
		if err != nil && !eof {
			return err
		}
		line = strings.TrimRight(line, "\r\n")

		if line == "" {
			if eventName != "" || dataBuf.Len() > 0 {
				dispatch(eventName, dataBuf.String())
				eventName = ""
				dataBuf.Reset()
			}
			if eof {
				return nil
			}
			continue
		}
		if strings.HasPrefix(line, ":") {
			if eof {
				return nil
			}
			continue
		}
		idx := strings.IndexByte(line, ':')
		var field, value string
		if idx < 0 {
			field = line
		} else {
			field = line[:idx]
			value = strings.TrimPrefix(line[idx+1:], " ")
		}
		switch field {
		case "event":
			eventName = value
		case "data":
			if dataBuf.Len() > 0 {
				dataBuf.WriteByte('\n')
			}
			dataBuf.WriteString(value)
		}
		if eof {
			return nil
		}
	}
}

func dispatch(event, data string) {
	switch event {
	case "update":
		var u update
		if err := json.Unmarshal([]byte(data), &u); err != nil {
			fmt.Fprintf(os.Stderr, "decode update: %v\n", err)
			return
		}
		fmt.Printf("[%s] rule=%s %s\n", u.MatchedAt, u.QueryID, u.Document.URL)
	case "error":
		fmt.Fprintf(os.Stderr, "stream error: %s\n", data)
	case "end":
		fmt.Fprintln(os.Stderr, "stream ended")
	case "connected":
		fmt.Fprintln(os.Stderr, "connected")
	}
}
