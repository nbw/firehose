package client

import (
	"context"
	"net/http"
)

type Tap struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Token       string  `json:"token,omitempty"`
	TokenPrefix string  `json:"token_prefix,omitempty"`
	RulesCount  int     `json:"rules_count,omitempty"`
	LastUsedAt  *string `json:"last_used_at,omitempty"`
	CreatedAt   string  `json:"created_at,omitempty"`
}

type tapEnvelope struct {
	Data Tap `json:"data"`
}

type tapsEnvelope struct {
	Data []Tap `json:"data"`
}

type createTapResponse struct {
	Data  Tap    `json:"data"`
	Token string `json:"token"`
}

type CreateTapResult struct {
	Tap   Tap
	Token string
}

type TapCreate struct {
	Name string `json:"name"`
}

type TapUpdate struct {
	Name *string `json:"name,omitempty"`
}

func (c *Client) ListTaps(ctx context.Context) ([]Tap, []byte, error) {
	var env tapsEnvelope
	raw, err := c.do(ctx, http.MethodGet, "/v1/taps", nil, authMgmt, &env)
	if err != nil {
		return nil, raw, err
	}
	return env.Data, raw, nil
}

func (c *Client) CreateTap(ctx context.Context, name string) (*CreateTapResult, []byte, error) {
	var env createTapResponse
	raw, err := c.do(ctx, http.MethodPost, "/v1/taps", TapCreate{Name: name}, authMgmt, &env)
	if err != nil {
		return nil, raw, err
	}
	if env.Data.Token == "" && env.Token != "" {
		env.Data.Token = env.Token
	}
	return &CreateTapResult{Tap: env.Data, Token: env.Token}, raw, nil
}

func (c *Client) GetTap(ctx context.Context, id string) (*Tap, []byte, error) {
	var env tapEnvelope
	raw, err := c.do(ctx, http.MethodGet, "/v1/taps/"+id, nil, authMgmt, &env)
	if err != nil {
		return nil, raw, err
	}
	return &env.Data, raw, nil
}

func (c *Client) UpdateTap(ctx context.Context, id string, upd TapUpdate) (*Tap, []byte, error) {
	var env tapEnvelope
	raw, err := c.do(ctx, http.MethodPut, "/v1/taps/"+id, upd, authMgmt, &env)
	if err != nil {
		return nil, raw, err
	}
	return &env.Data, raw, nil
}

func (c *Client) DeleteTap(ctx context.Context, id string) ([]byte, error) {
	return c.do(ctx, http.MethodDelete, "/v1/taps/"+id, nil, authMgmt, nil)
}
