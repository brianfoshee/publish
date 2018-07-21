package cli

import (
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func TestPostParseFile(t *testing.T) {
	var cases = []struct {
		File        string
		Title       string
		Slug        string
		Description string
		PublishedAt time.Time
		Draft       bool
		Body        string
	}{
		{
			File:        "fixtures/im-taking-a-year-off",
			Title:       "I'm Taking a Year Off",
			Slug:        "im-taking-a-year-off",
			Description: "I left my job writing software at The New York Times to travel for a year.",
			PublishedAt: time.Date(2018, 5, 4, 0, 0, 0, 0, time.UTC),
			Draft:       false,
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.File, func(t *testing.T) {
			var p Post
			if err := p.processFile(c.File + ".md"); err != nil {
				t.Fatal("error processing md file", err)
			}

			if p.Title != c.Title {
				t.Fatalf("title expected %q, got %q", c.Title, p.Title)
			}
			if p.Slug != c.Slug {
				t.Errorf("slug expected %q, got %q", c.Slug, p.Slug)
			}
			if p.Description != c.Description {
				t.Errorf("description expected %q, got %q", c.Description, p.Description)
			}
			if !p.PublishedAt.Equal(c.PublishedAt) {
				t.Errorf("publishedAt expected %q got %q", c.PublishedAt.Format("2006-01-02"), p.PublishedAt.Format("2006-01-02"))
			}
			if p.Draft != c.Draft {
				t.Errorf("draft expected %v got %v", c.Draft, p.Draft)
			}

			f, err := os.Open(c.File + ".html")
			if err != nil {
				t.Fatal("error opening html file", err)
			}
			defer f.Close()

			b, err := ioutil.ReadAll(f)
			if err != nil {
				t.Fatal("error reading all of html file", err)
			}

			if p.Body != string(b) {
				t.Fatalf("body expected %q\ngot %q", string(b), p.Body)
			}
		})
	}

}
