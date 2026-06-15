// Package lesswrong is the library behind the lw command: the HTTP client,
// request shaping, and the typed data models for LessWrong.
//
// The client posts queries to the public LessWrong GraphQL endpoint at
// https://www.lesswrong.com/graphql. No authentication is required. It sets a
// real User-Agent, paces requests, and retries transient 429/5xx errors with
// exponential backoff.
package lesswrong

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// DefaultUserAgent identifies the client to LessWrong.
const DefaultUserAgent = "lw/dev (+https://github.com/tamnd/lesswrong-cli)"

// ErrNotFound is returned when the API returns a null result for a single post.
var ErrNotFound = errors.New("not found")

// Config holds constructor parameters.
type Config struct {
	BaseURL   string
	UserAgent string
	Rate      time.Duration
	Retries   int
	Timeout   time.Duration
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		BaseURL:   "https://www.lesswrong.com",
		UserAgent: DefaultUserAgent,
		Rate:      500 * time.Millisecond,
		Retries:   3,
		Timeout:   30 * time.Second,
	}
}

// Client talks to the LessWrong GraphQL API.
type Client struct {
	cfg        Config
	httpClient *http.Client
	mu         sync.Mutex
	last       time.Time
}

// NewClient returns a Client with the given config.
func NewClient(cfg Config) *Client {
	return &Client{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: cfg.Timeout},
	}
}

// graphql POSTs a GraphQL query and returns the raw response body.
func (c *Client) graphql(ctx context.Context, query string) ([]byte, error) {
	body, err := json.Marshal(map[string]string{"query": query})
	if err != nil {
		return nil, err
	}
	url := c.cfg.BaseURL + "/graphql"

	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		b, retry, err := c.post(ctx, url, body)
		if err == nil {
			return b, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("graphql: %w", lastErr)
}

func (c *Client) post(ctx context.Context, url string, body []byte) ([]byte, bool, error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}

	b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

func (c *Client) pace() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cfg.Rate <= 0 {
		return
	}
	if wait := c.cfg.Rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}

// ─── query builders ───────────────────────────────────────────────────────────

func postsQuery(view string, limit int, after string) string {
	terms := fmt.Sprintf("limit: %d, view: %q", limit, view)
	if after != "" {
		terms += fmt.Sprintf(", after: %q", after)
	}
	return fmt.Sprintf(`{
  posts(input: {terms: {%s}}) {
    results {
      _id title url pageUrl postedAt score baseScore commentCount wordCount
      voteCount
      user { username displayName }
      tags { name }
    }
  }
}`, terms)
}

func searchQuery(q string, limit int) string {
	return fmt.Sprintf(`{
  posts(input: {terms: {limit: %d, view: "magic", search: %q}}) {
    results {
      _id title pageUrl postedAt score commentCount
      user { username displayName }
      tags { name }
    }
  }
}`, limit, q)
}

func tagPostsQuery(tagID string, limit int) string {
	return fmt.Sprintf(`{
  posts(input: {terms: {limit: %d, filterSettings: {tags: [{tagId: %q, filterMode: "Required"}]}}}) {
    results {
      _id title pageUrl postedAt score baseScore commentCount wordCount
      voteCount
      user { username displayName }
      tags { name }
    }
  }
}`, limit, tagID)
}

func singlePostQuery(id string) string {
	return fmt.Sprintf(`{
  post(input: {selector: {_id: %q}}) {
    result {
      _id title pageUrl postedAt score baseScore commentCount wordCount
      voteCount
      user { username displayName }
      tags { name }
    }
  }
}`, id)
}

// ─── public API ───────────────────────────────────────────────────────────────

// Posts fetches a post list for the given view.
// afterDate may be empty (no date filter). Format: "2006-01-02".
func (c *Client) Posts(ctx context.Context, view string, limit int, afterDate string) ([]Post, error) {
	q := postsQuery(view, limit, afterDate)
	raw, err := c.graphql(ctx, q)
	if err != nil {
		return nil, err
	}
	var resp postsResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("decode posts: %w", err)
	}
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("graphql error: %s", resp.Errors[0].Message)
	}
	out := make([]Post, 0, len(resp.Data.Posts.Results))
	for i, p := range resp.Data.Posts.Results {
		out = append(out, wireToPost(p, i+1))
	}
	return out, nil
}

// Search searches posts by query string.
func (c *Client) Search(ctx context.Context, query string, limit int) ([]Post, error) {
	q := searchQuery(query, limit)
	raw, err := c.graphql(ctx, q)
	if err != nil {
		return nil, err
	}
	var resp postsResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("decode search: %w", err)
	}
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("graphql error: %s", resp.Errors[0].Message)
	}
	out := make([]Post, 0, len(resp.Data.Posts.Results))
	for i, p := range resp.Data.Posts.Results {
		out = append(out, wireToPost(p, i+1))
	}
	return out, nil
}

// TagPosts fetches posts filtered by a tag ID or slug.
func (c *Client) TagPosts(ctx context.Context, tagID string, limit int) ([]Post, error) {
	q := tagPostsQuery(tagID, limit)
	raw, err := c.graphql(ctx, q)
	if err != nil {
		return nil, err
	}
	var resp postsResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("decode tag posts: %w", err)
	}
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("graphql error: %s", resp.Errors[0].Message)
	}
	out := make([]Post, 0, len(resp.Data.Posts.Results))
	for i, p := range resp.Data.Posts.Results {
		out = append(out, wireToPost(p, i+1))
	}
	return out, nil
}

// Post fetches a single post by ID. Returns ErrNotFound if the post does not exist.
func (c *Client) Post(ctx context.Context, id string) (Post, error) {
	q := singlePostQuery(id)
	raw, err := c.graphql(ctx, q)
	if err != nil {
		return Post{}, err
	}
	var resp postResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return Post{}, fmt.Errorf("decode post: %w", err)
	}
	if len(resp.Errors) > 0 {
		return Post{}, fmt.Errorf("graphql error: %s", resp.Errors[0].Message)
	}
	if resp.Data.Post.Result == nil {
		return Post{}, ErrNotFound
	}
	return wireToPost(*resp.Data.Post.Result, 1), nil
}
