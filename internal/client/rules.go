package client

import (
	"context"
	"net/http"
)

type Rule struct {
	ID      string `json:"id"`
	Value   string `json:"value"`
	Tag     string `json:"tag,omitempty"`
	NSFW    *bool  `json:"nsfw,omitempty"`
	Quality *bool  `json:"quality,omitempty"`
}

type ruleEnvelope struct {
	Data Rule `json:"data"`
}

type rulesEnvelope struct {
	Data []Rule `json:"data"`
	Meta struct {
		Count int `json:"count"`
	} `json:"meta"`
}

type RuleCreate struct {
	Value   string `json:"value"`
	Tag     string `json:"tag,omitempty"`
	NSFW    *bool  `json:"nsfw,omitempty"`
	Quality *bool  `json:"quality,omitempty"`
}

type RuleUpdate struct {
	Value   *string `json:"value,omitempty"`
	Tag     *string `json:"tag,omitempty"`
	NSFW    *bool   `json:"nsfw,omitempty"`
	Quality *bool   `json:"quality,omitempty"`
}

func (c *Client) ListRules(ctx context.Context) ([]Rule, []byte, error) {
	var env rulesEnvelope
	raw, err := c.do(ctx, http.MethodGet, "/v1/rules", nil, authTap, &env)
	if err != nil {
		return nil, raw, err
	}
	return env.Data, raw, nil
}

func (c *Client) CreateRule(ctx context.Context, r RuleCreate) (*Rule, []byte, error) {
	var env ruleEnvelope
	raw, err := c.do(ctx, http.MethodPost, "/v1/rules", r, authTap, &env)
	if err != nil {
		return nil, raw, err
	}
	return &env.Data, raw, nil
}

func (c *Client) GetRule(ctx context.Context, id string) (*Rule, []byte, error) {
	var env ruleEnvelope
	raw, err := c.do(ctx, http.MethodGet, "/v1/rules/"+id, nil, authTap, &env)
	if err != nil {
		return nil, raw, err
	}
	return &env.Data, raw, nil
}

func (c *Client) UpdateRule(ctx context.Context, id string, upd RuleUpdate) (*Rule, []byte, error) {
	var env ruleEnvelope
	raw, err := c.do(ctx, http.MethodPut, "/v1/rules/"+id, upd, authTap, &env)
	if err != nil {
		return nil, raw, err
	}
	return &env.Data, raw, nil
}

func (c *Client) DeleteRule(ctx context.Context, id string) ([]byte, error) {
	return c.do(ctx, http.MethodDelete, "/v1/rules/"+id, nil, authTap, nil)
}
