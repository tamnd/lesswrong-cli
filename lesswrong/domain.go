package lesswrong

import (
	"context"
	"errors"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

// Host is the LessWrong hostname claimed by the kit domain.
const Host = "www.lesswrong.com"

// domain.go exposes lesswrong as a kit Domain. A blank import from a multi-domain
// host such as ant is enough to enable the driver:
//
//	import _ "github.com/tamnd/lesswrong-cli/lesswrong"
//
// The same Domain also drives the standalone lw binary via cli.Root(), so the
// binary and any host share one source of truth.
func init() { kit.Register(Domain{}) }

// Domain is the LessWrong driver. It carries no state; the per-run client is
// built by the factory Register hands to kit.
type Domain struct{}

// Info returns the scheme, claimed hostnames, and binary identity.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme: "lesswrong",
		Hosts:  []string{Host},
		Identity: kit.Identity{
			Binary: "lesswrong",
			Short:  "A command line for LessWrong.",
			Long: `A command line for LessWrong. Browse frontpage, curated, and top posts, or filter by tag. No API key required.

lw is an independent tool and is not affiliated with LessWrong.`,
			Site: "https://www.lesswrong.com",
			Repo: "https://github.com/tamnd/lesswrong-cli",
		},
	}
}

// Register installs the client factory and four operations onto app.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	kit.Handle(app, kit.OpMeta{
		Name:    "frontpage",
		Group:   "posts",
		List:    true,
		URIType: "post",
		Summary: "LessWrong frontpage posts",
	}, frontpagePosts)

	kit.Handle(app, kit.OpMeta{
		Name:    "curated",
		Group:   "posts",
		List:    true,
		URIType: "post",
		Summary: "LessWrong curated posts",
	}, curatedPosts)

	kit.Handle(app, kit.OpMeta{
		Name:    "top",
		Group:   "posts",
		List:    true,
		URIType: "post",
		Summary: "LessWrong top posts by karma",
	}, topPosts)

	kit.Handle(app, kit.OpMeta{
		Name:    "tag",
		Group:   "posts",
		List:    true,
		URIType: "post",
		Summary: "LessWrong posts by tag",
		Args:    []kit.Arg{{Name: "tagID", Help: "tag ID or slug"}},
	}, tagPosts)
}

// newClient builds the LessWrong client from the kit Config.
func newClient(_ context.Context, cfg kit.Config) (any, error) {
	c := DefaultConfig()
	if cfg.UserAgent != "" {
		c.UserAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		c.Rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		c.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		c.Timeout = cfg.Timeout
	}
	return NewClient(c), nil
}

// --- input structs ---

type noArgIn struct {
	Limit  int     `kit:"flag,inherit" help:"max results"`
	Client *Client `kit:"inject"`
}

type tagIn struct {
	TagID  string  `kit:"arg" help:"tag ID or slug"`
	Limit  int     `kit:"flag,inherit" help:"max results"`
	Client *Client `kit:"inject"`
}

// --- handlers ---

func frontpagePosts(ctx context.Context, in noArgIn, emit func(*Post) error) error {
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	posts, err := in.Client.Posts(ctx, "frontpage", limit, "")
	if err != nil {
		return mapErr(err)
	}
	for i := range posts {
		if err := emit(&posts[i]); err != nil {
			return err
		}
	}
	return nil
}

func curatedPosts(ctx context.Context, in noArgIn, emit func(*Post) error) error {
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	posts, err := in.Client.Posts(ctx, "curated", limit, "")
	if err != nil {
		return mapErr(err)
	}
	for i := range posts {
		if err := emit(&posts[i]); err != nil {
			return err
		}
	}
	return nil
}

func topPosts(ctx context.Context, in noArgIn, emit func(*Post) error) error {
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	posts, err := in.Client.Posts(ctx, "top", limit, "")
	if err != nil {
		return mapErr(err)
	}
	for i := range posts {
		if err := emit(&posts[i]); err != nil {
			return err
		}
	}
	return nil
}

func tagPosts(ctx context.Context, in tagIn, emit func(*Post) error) error {
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	posts, err := in.Client.TagPosts(ctx, in.TagID, limit)
	if err != nil {
		return mapErr(err)
	}
	for i := range posts {
		if err := emit(&posts[i]); err != nil {
			return err
		}
	}
	return nil
}

// mapErr converts a library error into the appropriate kit error kind.
func mapErr(err error) error {
	if errors.Is(err, ErrNotFound) {
		return errs.NotFound("%s", err.Error())
	}
	return err
}
