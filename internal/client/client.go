package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"
)

const DefaultBaseURL = "https://api.firehose.com"

type auth int

const (
	authNone auth = iota
	authMgmt
	authTap
)

type Options struct {
	BaseURL    string
	MgmtKey    string
	TapToken   string
	Version    string
	HTTPClient *http.Client
}

type Client struct {
	httpClient *http.Client
	baseURL    string
	mgmtKey    string
	tapToken   string
	userAgent  string
}

func New(opts Options) *Client {
	base := opts.BaseURL
	if base == "" {
		base = DefaultBaseURL
	}
	base = strings.TrimRight(base, "/")
	hc := opts.HTTPClient
	if hc == nil {
		hc = &http.Client{
			Timeout: 30 * time.Second,
		}
	}
	version := opts.Version
	if version == "" {
		version = "dev"
	}
	return &Client{
		httpClient: hc,
		baseURL:    base,
		mgmtKey:    opts.MgmtKey,
		tapToken:   opts.TapToken,
		userAgent:  fmt.Sprintf("firehose-cli/%s (go/%s)", version, runtime.Version()),
	}
}

func (c *Client) BaseURL() string  { return c.baseURL }
func (c *Client) UserAgent() string { return c.userAgent }
func (c *Client) HasMgmtKey() bool { return c.mgmtKey != "" }
func (c *Client) HasTapToken() bool { return c.tapToken != "" }

type ErrMissingAuth struct {
	Need string
	Env  string
	Flag string
}

func (e *ErrMissingAuth) Error() string {
	return fmt.Sprintf("%s required: set %s or pass %s", e.Need, e.Env, e.Flag)
}

func (c *Client) tokenFor(a auth) (string, error) {
	switch a {
	case authMgmt:
		if c.mgmtKey == "" {
			return "", &ErrMissingAuth{Need: "management key", Env: "FIREHOSE_MANAGEMENT_KEY", Flag: "--management-key"}
		}
		return c.mgmtKey, nil
	case authTap:
		if c.tapToken == "" {
			return "", &ErrMissingAuth{Need: "tap token", Env: "FIREHOSE_TAP_TOKEN", Flag: "--tap-token"}
		}
		return c.tapToken, nil
	}
	return "", nil
}

func (c *Client) do(ctx context.Context, method, path string, body any, a auth, out any) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		reqBody = bytes.NewReader(buf)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	if a != authNone {
		tok, err := c.tokenFor(a)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return respBody, parseAPIError(resp.StatusCode, respBody)
	}
	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return respBody, fmt.Errorf("decode response: %w", err)
		}
	}
	return respBody, nil
}
