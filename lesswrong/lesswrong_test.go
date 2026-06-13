package lesswrong

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const mockPostsResponse = `{
  "data": {
    "posts": {
      "results": [
        {
          "_id": "abc123",
          "title": "The Alignment Problem",
          "pageUrl": "https://www.lesswrong.com/posts/abc123/the-alignment-problem",
          "postedAt": "2024-03-15T10:00:00.000Z",
          "score": 150,
          "baseScore": 200,
          "commentCount": 42,
          "wordCount": 3500,
          "voteCount": 88,
          "user": {"username": "eliezer", "displayName": "Eliezer Yudkowsky"},
          "tags": [{"name": "AI Safety"}, {"name": "Alignment"}]
        },
        {
          "_id": "def456",
          "title": "Bayesian Epistemology",
          "pageUrl": "https://www.lesswrong.com/posts/def456/bayesian-epistemology",
          "postedAt": "2024-03-10T08:30:00.000Z",
          "score": 95,
          "baseScore": 120,
          "commentCount": 18,
          "wordCount": 2100,
          "voteCount": 55,
          "user": {"username": "rationalist42", "displayName": ""},
          "tags": [{"name": "Epistemology"}, {"name": "Rationality"}, {"name": "Bayesianism"}]
        }
      ]
    }
  }
}`

const mockPostResponse = `{
  "data": {
    "post": {
      "result": {
        "_id": "abc123",
        "title": "The Alignment Problem",
        "pageUrl": "https://www.lesswrong.com/posts/abc123/the-alignment-problem",
        "postedAt": "2024-03-15T10:00:00.000Z",
        "score": 150,
        "baseScore": 200,
        "commentCount": 42,
        "wordCount": 3500,
        "voteCount": 88,
        "user": {"username": "eliezer", "displayName": "Eliezer Yudkowsky"},
        "tags": [{"name": "AI Safety"}, {"name": "Alignment"}]
      }
    }
  }
}`

const mockPostNotFoundResponse = `{
  "data": {
    "post": {
      "result": null
    }
  }
}`

const mockGraphQLErrorResponse = `{
  "errors": [{"message": "not authorised"}]
}`

func newTestClient(ts *httptest.Server) *Client {
	cfg := DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	return NewClient(cfg)
}

func TestPostsSendsUserAgent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("request carried no User-Agent")
		}
		_, _ = w.Write([]byte(mockPostsResponse))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	_, err := c.Posts(context.Background(), "top", 2, "")
	if err != nil {
		t.Fatal(err)
	}
}

func TestPostsTopParsesResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(mockPostsResponse))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	posts, err := c.Posts(context.Background(), "top", 2, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) != 2 {
		t.Fatalf("got %d posts, want 2", len(posts))
	}

	p := posts[0]
	if p.Rank != 1 {
		t.Errorf("rank = %d, want 1", p.Rank)
	}
	if p.Title != "The Alignment Problem" {
		t.Errorf("title = %q", p.Title)
	}
	if p.Author != "Eliezer Yudkowsky" {
		t.Errorf("author = %q, want displayName", p.Author)
	}
	if p.Score != 150 {
		t.Errorf("score = %d, want 150", p.Score)
	}
	if p.Comments != 42 {
		t.Errorf("comments = %d, want 42", p.Comments)
	}
	if p.Tags != "AI Safety, Alignment" {
		t.Errorf("tags = %q", p.Tags)
	}
	if p.Posted != "2024-03-15" {
		t.Errorf("posted = %q, want 2024-03-15", p.Posted)
	}

	// Second post: empty displayName falls back to username, 3 tags capped.
	p2 := posts[1]
	if p2.Author != "rationalist42" {
		t.Errorf("author = %q, want username fallback", p2.Author)
	}
	if p2.Tags != "Epistemology, Rationality, Bayesianism" {
		t.Errorf("tags = %q, want 3 capped", p2.Tags)
	}
}

func TestPostsTopWithAfterDate(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b := make([]byte, 4096)
		n, _ := r.Body.Read(b)
		gotBody = string(b[:n])
		_, _ = w.Write([]byte(mockPostsResponse))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	_, err := c.Posts(context.Background(), "top", 5, "2024-01-01")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotBody, "2024-01-01") {
		t.Errorf("request body %q does not contain after date", gotBody)
	}
}

func TestSearchParsesResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(mockPostsResponse))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	posts, err := c.Search(context.Background(), "AI alignment", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) == 0 {
		t.Fatal("got 0 posts from search")
	}
}

func TestPostNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(mockPostNotFoundResponse))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	_, err := c.Post(context.Background(), "noexist")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}

func TestPostFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(mockPostResponse))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	p, err := c.Post(context.Background(), "abc123")
	if err != nil {
		t.Fatal(err)
	}
	if p.Title != "The Alignment Problem" {
		t.Errorf("title = %q", p.Title)
	}
	if p.Rank != 1 {
		t.Errorf("rank = %d, want 1", p.Rank)
	}
}

func TestGetRetriesOn503(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(mockPostsResponse))
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	cfg.Retries = 5
	c := NewClient(cfg)

	start := time.Now()
	_, err := c.Posts(context.Background(), "top", 2, "")
	if err != nil {
		t.Fatal(err)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
	if time.Since(start) < 500*time.Millisecond {
		t.Error("retries did not back off")
	}
}

func TestGraphQLErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(mockGraphQLErrorResponse))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	_, err := c.Posts(context.Background(), "top", 5, "")
	if err == nil {
		t.Fatal("expected error from graphql errors array, got nil")
	}
	if !strings.Contains(err.Error(), "not authorised") {
		t.Errorf("error = %v, want 'not authorised'", err)
	}
}
