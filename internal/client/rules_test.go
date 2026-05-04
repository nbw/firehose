package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateRule_BoolPointersOmitWhenUnset(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatal(err)
		}
		w.Write([]byte(`{"data":{"id":"r1","value":"x","tag":"t"}}`))
	}))
	defer srv.Close()
	c := New(Options{BaseURL: srv.URL, TapToken: "fh_x"})
	_, _, err := c.CreateRule(context.Background(), RuleCreate{Value: "x", Tag: "t"})
	if err != nil {
		t.Fatal(err)
	}
	if _, present := got["nsfw"]; present {
		t.Error("nsfw should not be present when unset")
	}
	if _, present := got["quality"]; present {
		t.Error("quality should not be present when unset")
	}
	if got["value"] != "x" {
		t.Errorf("value = %v", got["value"])
	}
}

func TestCreateRule_BoolPointersSentWhenSet(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &got)
		w.Write([]byte(`{"data":{"id":"r1","value":"x"}}`))
	}))
	defer srv.Close()
	c := New(Options{BaseURL: srv.URL, TapToken: "fh_x"})
	tru := true
	fal := false
	_, _, err := c.CreateRule(context.Background(), RuleCreate{Value: "x", NSFW: &tru, Quality: &fal})
	if err != nil {
		t.Fatal(err)
	}
	if got["nsfw"] != true {
		t.Errorf("nsfw = %v", got["nsfw"])
	}
	if got["quality"] != false {
		t.Errorf("quality = %v", got["quality"])
	}
}

func TestUpdateRule_PartialBody(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Fatal(r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &got)
		w.Write([]byte(`{"data":{"id":"r1","value":"x","tag":"new"}}`))
	}))
	defer srv.Close()
	c := New(Options{BaseURL: srv.URL, TapToken: "fh_x"})
	tag := "new"
	_, _, err := c.UpdateRule(context.Background(), "r1", RuleUpdate{Tag: &tag})
	if err != nil {
		t.Fatal(err)
	}
	if got["tag"] != "new" {
		t.Errorf("tag = %v", got["tag"])
	}
	if _, present := got["value"]; present {
		t.Error("value should be omitted when unset")
	}
}

func TestListRules(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data":[{"id":"1","value":"a","tag":"x"},{"id":"2","value":"b"}],"meta":{"count":2}}`))
	}))
	defer srv.Close()
	c := New(Options{BaseURL: srv.URL, TapToken: "fh_x"})
	rules, _, err := c.ListRules(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 2 || rules[0].ID != "1" {
		t.Fatalf("got %+v", rules)
	}
}

func TestRules_Validation422(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(422)
		w.Write([]byte(`{"error":{"message":"rule limit reached","code":"limit"}}`))
	}))
	defer srv.Close()
	c := New(Options{BaseURL: srv.URL, TapToken: "fh_x"})
	_, _, err := c.CreateRule(context.Background(), RuleCreate{Value: "x"})
	if !IsValidation(err) {
		t.Fatalf("expected validation error, got %v", err)
	}
}
