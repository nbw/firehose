package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListTaps(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v1/taps" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer fhm_test" {
			t.Errorf("auth = %q", r.Header.Get("Authorization"))
		}
		w.Write([]byte(`{"data":[{"id":"abc","name":"one","token_prefix":"fh_a","rules_count":2,"created_at":"2026-01-01T00:00:00Z"}]}`))
	}))
	defer srv.Close()
	c := New(Options{BaseURL: srv.URL, MgmtKey: "fhm_test"})
	taps, _, err := c.ListTaps(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(taps) != 1 || taps[0].ID != "abc" || taps[0].RulesCount != 2 {
		t.Fatalf("got %+v", taps)
	}
}

func TestCreateTap_ReturnsToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("method = %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		var req TapCreate
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatal(err)
		}
		if req.Name != "Smoke" {
			t.Errorf("name = %q", req.Name)
		}
		w.WriteHeader(201)
		w.Write([]byte(`{"data":{"id":"t1","name":"Smoke","token_prefix":"fh_x","created_at":"2026-01-01"},"token":"fh_full_secret"}`))
	}))
	defer srv.Close()
	c := New(Options{BaseURL: srv.URL, MgmtKey: "fhm_x"})
	res, _, err := c.CreateTap(context.Background(), "Smoke")
	if err != nil {
		t.Fatal(err)
	}
	if res.Token != "fh_full_secret" {
		t.Errorf("Token = %q", res.Token)
	}
	if res.Tap.ID != "t1" {
		t.Errorf("Tap.ID = %q", res.Tap.ID)
	}
}

func TestUpdateTap_OmitsUnsetFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Fatalf("method = %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		// no name field set => empty {} body
		if strings.TrimSpace(string(body)) != "{}" {
			t.Errorf("body = %q, want empty object", string(body))
		}
		w.Write([]byte(`{"data":{"id":"t1","name":"unchanged"}}`))
	}))
	defer srv.Close()
	c := New(Options{BaseURL: srv.URL, MgmtKey: "fhm_x"})
	_, _, err := c.UpdateTap(context.Background(), "t1", TapUpdate{})
	if err != nil {
		t.Fatal(err)
	}
}

func TestDeleteTap_204(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Fatal(r.Method)
		}
		w.WriteHeader(204)
	}))
	defer srv.Close()
	c := New(Options{BaseURL: srv.URL, MgmtKey: "fhm_x"})
	if _, err := c.DeleteTap(context.Background(), "t1"); err != nil {
		t.Fatal(err)
	}
}

func TestListTaps_MissingAuth(t *testing.T) {
	c := New(Options{BaseURL: "http://x", MgmtKey: ""})
	_, _, err := c.ListTaps(context.Background())
	if !IsAuth(err) {
		t.Fatalf("expected auth error, got %v", err)
	}
}
