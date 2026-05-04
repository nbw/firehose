package client

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type StreamOptions struct {
	Timeout      int
	Since        string
	Offset       int64
	Limit        int
	ResumeFrom   string
	Reconnect    bool
	OffsetSet    bool
	LimitSet     bool
	TimeoutSet   bool
}

type StreamEvent struct {
	Event string
	Data  []byte
	ID    string
}

var ErrStreamEnded = errors.New("stream ended")

func (c *Client) Stream(ctx context.Context, opts StreamOptions, handler func(StreamEvent) error) error {
	if c.tapToken == "" {
		return &ErrMissingAuth{Need: "tap token", Env: "FIREHOSE_TAP_TOKEN", Flag: "--tap-token"}
	}
	resumeID := opts.ResumeFrom
	backoff := time.Second
	const maxBackoff = 30 * time.Second

	for {
		err := c.streamOnce(ctx, opts, resumeID, func(ev StreamEvent) error {
			if ev.ID != "" {
				resumeID = ev.ID
			}
			return handler(ev)
		})
		if err == nil || errors.Is(err, ErrStreamEnded) {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if !opts.Reconnect {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

func (c *Client) streamOnce(ctx context.Context, opts StreamOptions, resumeID string, handler func(StreamEvent) error) error {
	q := url.Values{}
	if opts.TimeoutSet {
		q.Set("timeout", strconv.Itoa(opts.Timeout))
	}
	if opts.Since != "" {
		q.Set("since", opts.Since)
	}
	if opts.OffsetSet {
		q.Set("offset", strconv.FormatInt(opts.Offset, 10))
	}
	if opts.LimitSet {
		q.Set("limit", strconv.Itoa(opts.Limit))
	}
	endpoint := c.baseURL + "/v1/stream"
	if encoded := q.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.tapToken)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("User-Agent", c.userAgent)
	if resumeID != "" {
		req.Header.Set("Last-Event-ID", resumeID)
	}

	hc := &http.Client{
		Timeout: 0,
		Transport: &http.Transport{
			ResponseHeaderTimeout: 30 * time.Second,
			DisableCompression:    true,
		},
	}
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return parseAPIError(resp.StatusCode, body)
	}

	r := bufio.NewReader(resp.Body)
	var (
		eventName string
		dataBuf   strings.Builder
		lastID    string
	)
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				if line == "" {
					return io.EOF
				}
			} else {
				return err
			}
		}
		line = strings.TrimRight(line, "\r\n")

		if line == "" {
			if eventName == "" && dataBuf.Len() == 0 {
				continue
			}
			ev := StreamEvent{
				Event: eventName,
				Data:  []byte(dataBuf.String()),
				ID:    lastID,
			}
			if err := handler(ev); err != nil {
				return err
			}
			if ev.Event == "end" {
				return ErrStreamEnded
			}
			eventName = ""
			dataBuf.Reset()
			continue
		}

		if strings.HasPrefix(line, ":") {
			continue
		}

		field, value, _ := splitSSEField(line)
		switch field {
		case "event":
			eventName = value
		case "data":
			if dataBuf.Len() > 0 {
				dataBuf.WriteByte('\n')
			}
			dataBuf.WriteString(value)
		case "id":
			lastID = value
		case "retry":
			// not honored
		}

		if errors.Is(err, io.EOF) {
			return io.EOF
		}
	}
}

func splitSSEField(line string) (field, value string, ok bool) {
	idx := strings.IndexByte(line, ':')
	if idx < 0 {
		return line, "", true
	}
	field = line[:idx]
	value = line[idx+1:]
	value = strings.TrimPrefix(value, " ")
	return field, value, true
}

