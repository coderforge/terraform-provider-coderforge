// Copyright (c) 2026 CoderForge.org Ltd.
// Licensed under the MIT License.

// Package client is the transport layer between the provider and terraform-api.
// It owns everything about HTTP - retries, timeouts, error decoding - so the
// resource implementations deal only in Go types and errors.
package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	// DefaultEndpoint points at the terraform-api service. It is a default,
	// not a constant: every installation overrides it.
	DefaultEndpoint = "https://terraform.coderforge.org"

	// apiPrefix is fixed by the contract; only the host is configurable.
	apiPrefix = "/api/1.0/tf"

	// A create call blocks server-side until the resource has settled, which
	// for a machine means a full guest boot. The HTTP client must therefore
	// outlast the longest waiter, not the average request.
	DefaultRequestTimeout = 60 * time.Minute

	DefaultMaxRetries = 3

	// contractVersion is the major version of the API this provider is
	// written against. A mismatch is reported once, as a warning, rather than
	// failing the run: a newer service is expected to stay compatible.
	contractVersion = "1"
)

// Config describes how to reach terraform-api. Everything but Token has a
// working default.
type Config struct {
	Endpoint       string
	Token          string
	Insecure       bool
	MaxRetries     int
	RequestTimeout time.Duration
	UserAgent      string
	HTTPClient     *http.Client
}

// Client is safe for concurrent use, which matters: Terraform walks the
// resource graph in parallel and will call it from many goroutines at once.
type Client struct {
	endpoint   string
	token      string
	userAgent  string
	maxRetries int
	http       *http.Client
}

// New validates the configuration and builds a client from it.
func New(cfg Config) (*Client, error) {
	endpoint := strings.TrimRight(cfg.Endpoint, "/")
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}

	parsed, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("endpoint %q is not a valid URL: %w", endpoint, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("endpoint %q must use http or https, got %q", endpoint, parsed.Scheme)
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("endpoint %q has no host", endpoint)
	}

	if cfg.Token == "" {
		return nil, fmt.Errorf("an API token is required")
	}

	timeout := cfg.RequestTimeout
	if timeout <= 0 {
		timeout = DefaultRequestTimeout
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		if cfg.Insecure {
			transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // #nosec G402 - opt-in, for internal CAs
		}
		httpClient = &http.Client{Transport: transport, Timeout: timeout}
	}

	maxRetries := cfg.MaxRetries
	if maxRetries < 0 {
		maxRetries = 0
	}

	return &Client{
		endpoint:   endpoint,
		token:      cfg.Token,
		userAgent:  cfg.UserAgent,
		maxRetries: maxRetries,
		http:       httpClient,
	}, nil
}

// Endpoint returns the configured base URL, for diagnostics.
func (c *Client) Endpoint() string { return c.endpoint }

// do performs one request, decoding a success body into out and any failure
// into an *APIError. out may be nil when the response body is not needed.
func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("could not encode the request body: %w", err)
		}
	}

	target := c.endpoint + apiPrefix + path
	var lastErr error

	for attempt := 0; ; attempt++ {
		var reader io.Reader
		if payload != nil {
			reader = bytes.NewReader(payload)
		}

		req, err := http.NewRequestWithContext(ctx, method, target, reader)
		if err != nil {
			return fmt.Errorf("could not build the request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("X-Terraform-API-Version", contractVersion)
		if c.userAgent != "" {
			req.Header.Set("User-Agent", c.userAgent)
		}
		if payload != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		tflog.Debug(ctx, "Calling the CoderForge API", map[string]any{
			"method":  method,
			"path":    path,
			"attempt": attempt + 1,
		})

		err = c.roundTrip(req, out)
		if err == nil {
			return nil
		}
		lastErr = err

		// Only transport failures and explicitly retryable statuses are
		// replayed, and only for requests that can safely happen twice.
		if attempt >= c.maxRetries || !isSafeToRetry(method, err) {
			return err
		}

		backoff := time.Duration(1<<attempt) * 500 * time.Millisecond
		tflog.Warn(ctx, "Retrying a failed API call", map[string]any{
			"method":  method,
			"path":    path,
			"attempt": attempt + 1,
			"backoff": backoff.String(),
			"error":   err.Error(),
		})

		select {
		case <-ctx.Done():
			return fmt.Errorf("%w (last error: %v)", ctx.Err(), lastErr)
		case <-time.After(backoff):
		}
	}
}

func (c *Client) roundTrip(req *http.Request, out any) error {
	res, err := c.http.Do(req)
	if err != nil {
		// A transport failure has no status code, so it is wrapped as an
		// unavailable upstream: that is what it means, and it is retryable.
		return &APIError{
			StatusCode: 0,
			Code:       CodeUpstreamUnavailable,
			Message: fmt.Sprintf(
				"could not reach the CoderForge API at %s: %v", req.URL.Host, err),
		}
	}
	defer func() { _ = res.Body.Close() }()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return &APIError{
			StatusCode: res.StatusCode,
			Code:       CodeUpstreamUnavailable,
			Message:    fmt.Sprintf("could not read the API response: %v", err),
		}
	}

	if res.StatusCode >= 400 {
		return decodeError(res, responseBody)
	}

	if out == nil || len(responseBody) == 0 {
		return nil
	}

	if err := json.Unmarshal(responseBody, out); err != nil {
		return fmt.Errorf("could not decode the API response: %w (body: %s)", err, truncate(responseBody, 512))
	}
	return nil
}

func decodeError(res *http.Response, body []byte) error {
	apiErr := &APIError{
		StatusCode: res.StatusCode,
		RequestID:  res.Header.Get("X-Request-Id"),
	}

	var envelope errorEnvelope
	if err := json.Unmarshal(body, &envelope); err == nil && envelope.Error.Code != "" {
		apiErr.Code = envelope.Error.Code
		apiErr.Message = envelope.Error.Message
		apiErr.Resource = envelope.Error.Resource
		apiErr.Details = envelope.Error.Details
		if envelope.Error.RequestID != "" {
			apiErr.RequestID = envelope.Error.RequestID
		}
		return apiErr
	}

	// A body that is not the documented envelope usually means something other
	// than terraform-api answered - a proxy, or the wrong endpoint entirely.
	// Say so, and include what did answer.
	apiErr.Code = fmt.Sprintf("http_%d", res.StatusCode)
	apiErr.Message = fmt.Sprintf(
		"the API returned HTTP %d with an unrecognised body. Check that the endpoint points at "+
			"terraform-api and not at something else. Body: %s",
		res.StatusCode, truncate(body, 512))
	return apiErr
}

// isSafeToRetry keeps replays to requests that cannot create a second
// resource. A POST that timed out may well have succeeded upstream.
func isSafeToRetry(method string, err error) bool {
	if !IsRetryable(err) {
		return false
	}
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodDelete:
		return true
	default:
		return false
	}
}

func truncate(body []byte, max int) string {
	text := strings.TrimSpace(string(body))
	if len(text) <= max {
		return text
	}
	return text[:max] + "..."
}

// --- Verb helpers, used by the resource-specific files ---

func (c *Client) get(ctx context.Context, path string, out any) error {
	return c.do(ctx, http.MethodGet, path, nil, out)
}

func (c *Client) post(ctx context.Context, path string, body, out any) error {
	return c.do(ctx, http.MethodPost, path, body, out)
}

func (c *Client) patch(ctx context.Context, path string, body, out any) error {
	return c.do(ctx, http.MethodPatch, path, body, out)
}

func (c *Client) delete(ctx context.Context, path string, out any) error {
	return c.do(ctx, http.MethodDelete, path, nil, out)
}
