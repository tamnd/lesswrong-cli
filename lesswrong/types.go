package lesswrong

import (
	"strings"
	"time"
)

// Post is the record emitted for post list commands (top, new, best, search)
// and the single-post command.
type Post struct {
	Rank     int    `json:"rank"`
	Title    string `json:"title"`
	Author   string `json:"author"`
	Score    int    `json:"score"`
	Comments int    `json:"comments"`
	Words    int    `json:"words"`
	Tags     string `json:"tags"`
	Posted   string `json:"posted"`
	URL      string `json:"url"      kit:"id" table:"url,url"`
}

// ─── GraphQL wire types ───────────────────────────────────────────────────────

type lwPost struct {
	ID           string  `json:"_id"`
	Title        string  `json:"title"`
	URL          string  `json:"url"`
	PageURL      string  `json:"pageUrl"`
	PostedAt     string  `json:"postedAt"`
	Score        float64 `json:"score"`
	BaseScore    float64 `json:"baseScore"`
	CommentCount int     `json:"commentCount"`
	WordCount    int     `json:"wordCount"`
	VoteCount    int     `json:"voteCount"`
	User         lwUser  `json:"user"`
	Tags         []lwTag `json:"tags"`
}

type lwUser struct {
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
}

type lwTag struct {
	Name string `json:"name"`
}

type postsResponse struct {
	Data struct {
		Posts struct {
			Results []lwPost `json:"results"`
		} `json:"posts"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

type postResponse struct {
	Data struct {
		Post struct {
			Result *lwPost `json:"result"`
		} `json:"post"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func wireToPost(p lwPost, rank int) Post {
	author := p.User.DisplayName
	if author == "" {
		author = p.User.Username
	}

	var tagNames []string
	for i, t := range p.Tags {
		if i >= 3 {
			break
		}
		tagNames = append(tagNames, t.Name)
	}
	tags := strings.Join(tagNames, ", ")

	posted := p.PostedAt
	if t, err := time.Parse(time.RFC3339, p.PostedAt); err == nil {
		posted = t.Format("2006-01-02")
	} else if t2, err2 := time.Parse("2006-01-02T15:04:05.000Z", p.PostedAt); err2 == nil {
		posted = t2.Format("2006-01-02")
	}

	return Post{
		Rank:     rank,
		Title:    p.Title,
		Author:   author,
		Score:    int(p.Score),
		Comments: p.CommentCount,
		Words:    p.WordCount,
		Tags:     tags,
		Posted:   posted,
		URL:      p.PageURL,
	}
}
