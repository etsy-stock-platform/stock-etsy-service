package authclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	ErrUnauthorized              = errors.New("unauthorized")
	ErrAuthServiceUnexpected     = errors.New("auth service unexpected response")
	ErrAuthServiceUnavailable    = errors.New("auth service unavailable")
	ErrInvalidAuthServiceBaseURL = errors.New("invalid auth service base url")
)

type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
}

type CurrentUser struct {
	ID              string     `json:"id"`
	Email           string     `json:"email"`
	Name            string     `json:"name"`
	EmailVerifiedAt *time.Time `json:"email_verified_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type meResponse struct {
	User CurrentUser `json:"user"`
}

func New(baseURL string, timeout time.Duration) (*Client, error) {
	parsed, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, ErrInvalidAuthServiceBaseURL
	}

	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	return &Client{
		baseURL: parsed,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

func (c *Client) CurrentUser(ctx context.Context, cookies []*http.Cookie) (*CurrentUser, error) {
	endpoint := c.baseURL.JoinPath("/auth/me")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("%w: build request", ErrAuthServiceUnavailable)
	}

	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAuthServiceUnavailable, err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		return nil, ErrUnauthorized
	case resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices:
		return nil, fmt.Errorf("%w: status %d", ErrAuthServiceUnexpected, resp.StatusCode)
	}

	var decoded meResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("%w: decode response", ErrAuthServiceUnexpected)
	}

	if decoded.User.ID == "" {
		return nil, fmt.Errorf("%w: missing user id", ErrAuthServiceUnexpected)
	}

	return &decoded.User, nil
}
