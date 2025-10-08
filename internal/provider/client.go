package provider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const HostURL string = "https://api.coderforge.org"

var ErrNotFound = errors.New("not_found")

type Client struct {
	StackId    string
	HostURL    string
	HTTPClient *http.Client
	Token      string
	CloudSpace string
	Locations  []string
}

func NewClient(token *string, cloudSpace *string, locations *[]string, stackId *string) (*Client, error) {
	c := Client{
		StackId:    *stackId,
		HostURL:    HostURL,
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
		Token:      *token,
		CloudSpace: *cloudSpace,
		Locations:  *locations,
	}
	return &c, nil
}

func (c *Client) doRequest(ctx context.Context, req *http.Request) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	req = req.WithContext(ctx)
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	req.Header.Set("X-CoderForge.org-Context", "{\"userId\": \"u00001\"}")
	req.Header.Set("Content-Type", "application/json")
	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
    defer func(Body io.ReadCloser) {
        _ = Body.Close()
    }(res.Body)

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

    if res.StatusCode < 200 || res.StatusCode >= 300 {
        if res.StatusCode == http.StatusNotFound {
            return nil, ErrNotFound
        }
        return nil, fmt.Errorf("status: %d, body: %s", res.StatusCode, body)
    }

	return body, err
}
