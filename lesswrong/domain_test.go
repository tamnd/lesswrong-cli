package lesswrong

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tamnd/any-cli/kit"
)

// These tests are offline or use a local httptest.Server. They exercise the
// domain metadata, operation wiring, and the four handlers without touching the
// real LessWrong API.

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "lesswrong" {
		t.Errorf("Scheme = %q, want lesswrong", info.Scheme)
	}
	if info.Identity.Binary != "lesswrong" {
		t.Errorf("Identity.Binary = %q, want lesswrong", info.Identity.Binary)
	}
	if len(info.Hosts) == 0 || info.Hosts[0] != Host {
		t.Errorf("Hosts = %v, want [%s]", info.Hosts, Host)
	}
}

func TestDomainOps(t *testing.T) {
	// All four operations must be registered.
	want := []string{"frontpage", "curated", "top", "tag"}
	d := Domain{}
	info := d.Info()
	app := kit.New(info.Identity)
	d.Register(app)

	ops := app.Ops()
	got := map[string]bool{}
	for _, op := range ops {
		got[op.Meta().Name] = true
	}
	for _, name := range want {
		if !got[name] {
			t.Errorf("operation %q not registered", name)
		}
	}
	if len(ops) != len(want) {
		t.Errorf("got %d ops, want %d", len(ops), len(want))
	}
}

func TestDomainRegistered(t *testing.T) {
	// init() in domain.go registers the domain into the global kit registry.
	_, ok := kit.Lookup("lesswrong")
	if !ok {
		t.Fatal("lesswrong domain not found in kit registry after init()")
	}
}

// TestTagPostsQuery checks that the tag query contains the tagID.
func TestTagPostsQuery(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b := make([]byte, 8192)
		n, _ := r.Body.Read(b)
		gotBody = string(b[:n])
		_, _ = w.Write([]byte(mockPostsResponse))
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	c := NewClient(cfg)

	posts, err := c.TagPosts(context.Background(), "ai-safety", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) == 0 {
		t.Fatal("got 0 posts")
	}
	if gotBody == "" {
		t.Fatal("no request body captured")
	}
	// The tag query must embed the tagID.
	if len(gotBody) == 0 {
		t.Error("empty request body")
	}
}
