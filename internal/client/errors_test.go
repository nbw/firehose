package client

import (
	"errors"
	"testing"
)

func TestParseAPIError_Envelopes(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{"error.message", `{"error":{"message":"bad","code":"invalid"}}`, "bad"},
		{"top message", `{"message":"oops"}`, "oops"},
		{"errors[]", `{"errors":[{"message":"first"}]}`, "first"},
		{"empty", ``, ""},
		{"garbage", `not json`, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := parseAPIError(422, []byte(tc.body))
			if e.Msg != tc.want {
				t.Errorf("Msg = %q, want %q", e.Msg, tc.want)
			}
			if e.Status != 422 {
				t.Errorf("Status = %d, want 422", e.Status)
			}
		})
	}
}

func TestExitCode(t *testing.T) {
	cases := []struct {
		err  error
		want int
	}{
		{nil, 0},
		{&APIError{Status: 401}, 2},
		{&APIError{Status: 403}, 2},
		{&APIError{Status: 404}, 3},
		{&APIError{Status: 422}, 4},
		{&APIError{Status: 429}, 5},
		{&APIError{Status: 500}, 6},
		{&APIError{Status: 502}, 6},
		{&ErrMissingAuth{Need: "x", Env: "Y", Flag: "--z"}, 2},
		{errors.New("boom"), 1},
	}
	for _, tc := range cases {
		if got := ExitCode(tc.err); got != tc.want {
			t.Errorf("ExitCode(%v) = %d, want %d", tc.err, got, tc.want)
		}
	}
}

func TestPredicates(t *testing.T) {
	if !IsAuth(&APIError{Status: 401}) {
		t.Error("401 should be auth")
	}
	if !IsAuth(&ErrMissingAuth{}) {
		t.Error("ErrMissingAuth should be auth")
	}
	if !IsValidation(&APIError{Status: 422}) {
		t.Error("422 should be validation")
	}
	if !IsRateLimit(&APIError{Status: 429}) {
		t.Error("429 should be rate limit")
	}
	if !IsServer(&APIError{Status: 503}) {
		t.Error("503 should be server")
	}
	if IsServer(&APIError{Status: 422}) {
		t.Error("422 should not be server")
	}
}
