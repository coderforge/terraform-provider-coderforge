// Copyright (c) 2026 CoderForge.org Ltd.
// Licensed under the MIT License.

package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func newTestClient(t *testing.T, handler http.Handler) *Client {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	c, err := New(Config{
		Endpoint:       server.URL,
		Token:          "test-token",
		MaxRetries:     0,
		RequestTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("New() returned an unexpected error: %v", err)
	}
	return c
}

func TestNewRejectsAnUnusableConfiguration(t *testing.T) {
	cases := map[string]Config{
		"no token":       {Endpoint: "http://example.test"},
		"no scheme":      {Endpoint: "example.test:8080", Token: "t"},
		"unknown scheme": {Endpoint: "ftp://example.test", Token: "t"},
		"no host at all": {Endpoint: "http://", Token: "t"},
	}

	for name, cfg := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := New(cfg); err == nil {
				t.Fatal("expected an error, got none")
			}
		})
	}
}

func TestEveryRequestCarriesTheToken(t *testing.T) {
	var seen string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Get("Authorization")
		_ = json.NewEncoder(w).Encode(Machine{Name: "web01"})
	}))

	if _, err := c.GetMachine(context.Background(), "web01"); err != nil {
		t.Fatalf("GetMachine() returned an unexpected error: %v", err)
	}
	if seen != "Bearer test-token" {
		t.Fatalf("Authorization header = %q, want %q", seen, "Bearer test-token")
	}
}

func TestNotFoundIsRecognisedThroughErrorsIs(t *testing.T) {
	// The resource Read methods branch on this to decide whether to drop a
	// resource from state, so it has to work through errors.Is and not only
	// on a status-code comparison.
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"not_found","message":"No machine named 'ghost'."}}`))
	}))

	_, err := c.GetMachine(context.Background(), "ghost")
	if err == nil {
		t.Fatal("expected an error, got none")
	}
	if !IsNotFound(err) {
		t.Fatalf("IsNotFound() = false for %v, want true", err)
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("errors.Is(err, ErrNotFound) = false for %v, want true", err)
	}
}

func TestTheApiMessageReachesTheCaller(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-Id", "abc123")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error":{"code":"conflict","message":"A machine named 'web01' already exists."}}`))
	}))

	_, err := c.CreateMachine(context.Background(), MachineCreateRequest{Name: "web01"})

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected an *APIError, got %T", err)
	}
	if apiErr.Code != CodeConflict {
		t.Errorf("Code = %q, want %q", apiErr.Code, CodeConflict)
	}
	if apiErr.RequestID != "abc123" {
		t.Errorf("RequestID = %q, want %q", apiErr.RequestID, "abc123")
	}
	// The message a practitioner sees has to be the one the service wrote,
	// with the request id attached so a log line can be found from it.
	if got := apiErr.Error(); got != "A machine named 'web01' already exists. (request id: abc123)" {
		t.Errorf("Error() = %q", got)
	}
}

func TestAnUnrecognisedBodySaysWhatProbablyWentWrong(t *testing.T) {
	// Pointing the provider at a web server rather than at terraform-api is a
	// common mistake, and "unexpected end of JSON input" would not help.
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("<html><body>502 Bad Gateway</body></html>"))
	}))

	_, err := c.GetMachine(context.Background(), "web01")

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected an *APIError, got %T", err)
	}
	if want := "terraform-api"; !strings.Contains(apiErr.Message, want) {
		t.Errorf("Error() = %q, want it to mention %q", apiErr.Message, want)
	}
}

func TestReadsAreRetriedAndWritesAreNot(t *testing.T) {
	var calls int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":{"code":"upstream_unavailable","message":"down"}}`))
			return
		}
		_ = json.NewEncoder(w).Encode(Machine{Name: "web01"})
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	c, err := New(Config{
		Endpoint: server.URL, Token: "t", MaxRetries: 3, RequestTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("New() returned an unexpected error: %v", err)
	}

	if _, err := c.GetMachine(context.Background(), "web01"); err != nil {
		t.Fatalf("a GET should have been retried past the outage, got: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Fatalf("GET made %d calls, want 3", got)
	}

	// A POST must not be replayed: the first attempt may have created the
	// machine even though the response never arrived.
	atomic.StoreInt32(&calls, 0)
	if _, err := c.CreateMachine(context.Background(), MachineCreateRequest{Name: "web01"}); err == nil {
		t.Fatal("expected the POST to fail rather than retry")
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("POST made %d calls, want 1", got)
	}
}

func TestAnUnreachableEndpointIsReportedAsRetryable(t *testing.T) {
	c, err := New(Config{
		// Reserved for documentation (RFC 5737); nothing listens there.
		Endpoint: "http://192.0.2.1:9", Token: "t", MaxRetries: 0,
		RequestTimeout: 500 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("New() returned an unexpected error: %v", err)
	}

	_, err = c.GetMachine(context.Background(), "web01")
	if err == nil {
		t.Fatal("expected an error, got none")
	}
	if !IsRetryable(err) {
		t.Fatalf("IsRetryable() = false for %v, want true", err)
	}
}

func TestNamesWithAwkwardCharactersAreEscaped(t *testing.T) {
	var seenPath string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path
		_ = json.NewEncoder(w).Encode(Storage{ID: "store:0001"})
	}))

	// A resource id contains a colon, which must survive the round trip.
	if _, err := c.GetStorage(context.Background(), "store:0001"); err != nil {
		t.Fatalf("GetStorage() returned an unexpected error: %v", err)
	}
	if want := "/api/v1/tf/storages/store:0001"; seenPath != want {
		t.Fatalf("path = %q, want %q", seenPath, want)
	}
}
