package client

import (
	"encoding/json"
	"errors"
	"fmt"
)

type APIError struct {
	Status int
	Code   string
	Msg    string
	Body   []byte
}

func (e *APIError) Error() string {
	if e.Msg != "" {
		return fmt.Sprintf("api error: %d %s", e.Status, e.Msg)
	}
	return fmt.Sprintf("api error: %d", e.Status)
}

func parseAPIError(status int, body []byte) *APIError {
	e := &APIError{Status: status, Body: body}
	if len(body) == 0 {
		return e
	}
	var envelope struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		Message string `json:"message"`
		Errors  []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil {
		switch {
		case envelope.Error.Message != "":
			e.Code = envelope.Error.Code
			e.Msg = envelope.Error.Message
		case envelope.Message != "":
			e.Msg = envelope.Message
		case len(envelope.Errors) > 0:
			e.Msg = envelope.Errors[0].Message
		}
	}
	return e
}

func IsAuth(err error) bool {
	var missing *ErrMissingAuth
	if errors.As(err, &missing) {
		return true
	}
	return statusIn(err, 401, 403)
}
func IsNotFound(err error) bool   { return statusIn(err, 404) }
func IsValidation(err error) bool { return statusIn(err, 422) }
func IsRateLimit(err error) bool  { return statusIn(err, 429) }
func IsServer(err error) bool {
	var e *APIError
	if errors.As(err, &e) {
		return e.Status >= 500 && e.Status < 600
	}
	return false
}

func statusIn(err error, codes ...int) bool {
	var e *APIError
	if !errors.As(err, &e) {
		return false
	}
	for _, c := range codes {
		if e.Status == c {
			return true
		}
	}
	return false
}

func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	switch {
	case IsAuth(err):
		return 2
	case IsNotFound(err):
		return 3
	case IsValidation(err):
		return 4
	case IsRateLimit(err):
		return 5
	case IsServer(err):
		return 6
	}
	var e *APIError
	if errors.As(err, &e) {
		return 6
	}
	return 1
}
