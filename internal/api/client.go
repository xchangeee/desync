// Package api provides a rate-limit-aware HTTP client for the deSEC REST API.
// It is the single entry point for all network I/O: every operation in this
// package routes through do(), which transparently retries on HTTP 429
// responses and honors the Retry-After header.
package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

const baseURL = "https://desec.io/api/v1"

// ErrNotFound is returned when the API responds with 404.
var ErrNotFound = errors.New("not found")

// reLinkNext extracts the URL from a Link header rel="next" entry.
var reLinkNext = regexp.MustCompile(`<([^>]+)>;\s*rel="next"`)

// reLinkFirst extracts the URL from a Link header rel="first" entry,
// used when the server signals that pagination is required via a 400 response.
var reLinkFirst = regexp.MustCompile(`<([^>]+)>;\s*rel="first"`)

// Client is a thin wrapper around net/http.Client that adds
// authentication headers and automatic rate-limit backoff.
type Client struct {
	token string
	http  *http.Client
}

// NewClient creates a Client authenticated with the given deSEC API token.
func NewClient(token string) *Client {
	return &Client{
		token: token,
		http:  &http.Client{Timeout: 30 * time.Second},
	}
}

// do executes an HTTP request against a full URL. On HTTP 429 it sleeps for
// the duration indicated by the Retry-After header (defaulting to 5 s) and
// retries indefinitely. body must already be JSON-encoded.
func (c *Client) do(method, url string, body []byte) (*http.Response, error) {
	for {
		var r io.Reader
		if body != nil {
			r = bytes.NewReader(body)
		}
		req, err := http.NewRequest(method, url, r)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Token "+c.token)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := c.http.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusTooManyRequests {
			return resp, nil
		}

		resp.Body.Close()
		wait := 5 * time.Second
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if secs, err := strconv.Atoi(ra); err == nil {
				// Add one extra second to avoid hitting the boundary.
				wait = time.Duration(secs+1) * time.Second
			}
		}
		fmt.Printf("  rate limited; retrying in %s\n", wait)
		time.Sleep(wait)
	}
}

// request marshals reqBody (if non-nil) as JSON, calls method on the given
// API path (relative to baseURL), and decodes the response into respBody
// (if non-nil). It returns the HTTP status code alongside any error.
func (c *Client) request(method, path string, reqBody, respBody any) (int, error) {
	var data []byte
	if reqBody != nil {
		var err error
		data, err = json.Marshal(reqBody)
		if err != nil {
			return 0, err
		}
	}

	resp, err := c.do(method, baseURL+path, data)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if respBody != nil && resp.StatusCode/100 == 2 {
		if err := json.NewDecoder(resp.Body).Decode(respBody); err != nil {
			return resp.StatusCode, fmt.Errorf("decoding response from %s %s: %w", method, path, err)
		}
	} else if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, fmt.Errorf("%s %s: %s: %s", method, path, resp.Status, b)
	}

	return resp.StatusCode, nil
}

// listAll fetches all items from a paginated list endpoint. It starts without a
// cursor; if the server requires pagination (HTTP 400 + Link: first), it
// follows cursor pages until there is no "next" link.
func (c *Client) listAll(path string, out any) error {
	var all []json.RawMessage
	url := baseURL + path
	firstAttempt := true

	for url != "" {
		resp, err := c.do("GET", url, nil)
		if err != nil {
			return err
		}

		// The server signals "pagination required" via 400 + Link: first.
		if firstAttempt && resp.StatusCode == http.StatusBadRequest {
			link := resp.Header.Get("Link")
			resp.Body.Close()
			m := reLinkFirst.FindStringSubmatch(link)
			if m == nil {
				return fmt.Errorf("GET %s: 400 Bad Request (pagination required but no Link header)", path)
			}
			url = m[1]
			firstAttempt = false
			continue
		}

		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return fmt.Errorf("GET %s: %s: %s", path, resp.Status, b)
		}

		var page []json.RawMessage
		err = json.NewDecoder(resp.Body).Decode(&page)
		link := resp.Header.Get("Link")
		resp.Body.Close()
		if err != nil {
			return fmt.Errorf("decoding list from GET %s: %w", path, err)
		}
		all = append(all, page...)
		firstAttempt = false

		url = ""
		if m := reLinkNext.FindStringSubmatch(link); m != nil {
			url = m[1]
		}
	}

	data, err := json.Marshal(all)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}
