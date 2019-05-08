package feed

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gorilla/feeds"
)

// Feeder is an interface that content types implement so they can be in the
// RSS feed.
type Feeder interface {
	// Item returns the appropriate feeds.Item for the content type
	// implementing this method. Item.Author doesn't need to be populated,
	// that will be done internally in this package for each Item.
	Item() feeds.Item
}

var author = &feeds.Author{Name: "Brian Foshee", Email: "brian@brianfoshee.com"}

func Build(content chan Feeder) error {
	feed := &feeds.Feed{
		Title: "Brian Foshee",
		Link: &feeds.Link{
			Href: "https://www.brianfoshee.com/",
			Rel:  "self",
		},
		Description: "Hi. I'm Brian Foshee. I write software.",
		Author:      author,
		Copyright:   "Copyright 2019 Brian Foshee",
	}

	for f := range content {
		item := f.Item()
		feed.Items = append(feed.Items, &item)
	}

	feed.Sort(func(a, b *feeds.Item) bool {
		return a.Created.After(b.Created)
	})

	if len(feed.Items) > 0 {
		feed.Created = feed.Items[0].Created
	}

	atomFeed := (&feeds.Atom{Feed: feed}).AtomFeed()
	atomFeed.Link.Href = "https://www.brianfoshee.com/feeds/atom"

	atom, err := feeds.ToXML(atomFeed)
	if err != nil {
		return err
	}
	rss, err := feed.ToRss()
	if err != nil {
		return err
	}

	rssf, err := os.Create("feed/feed.rss")
	if err != nil {
		return err
	}
	io.Copy(rssf, strings.NewReader(rss))

	atomf, err := os.Create("feed/feed.atom")
	if err != nil {
		return err
	}
	io.Copy(atomf, strings.NewReader(atom))

	return nil
}

func build() {
	now := time.Now()
	feed := &feeds.Feed{
		Title:       "Brian Foshee",
		Link:        &feeds.Link{Href: "https://www.brianfoshee.com"},
		Description: "Hi. I'm Brian Foshee. I write software.",
		Author:      author,
		Created:     now,
		Copyright:   "Copyright Brian Foshee",
	}

	feed.Items = []*feeds.Item{
		&feeds.Item{
			Id:          "/blog/how-to-setup-github-actions",
			Title:       "GitHub Actions",
			Link:        &feeds.Link{Href: "https://www.brianfoshee.com/blog/how-to-setup-github-actions"},
			Description: "How to setup GitHub Actions",
			Author:      author,
			Created:     now,
			Content:     "<h1>Github Actions</h1><p>This is how you setup github actions</p>",
		},
	}

	atom, err := feed.ToAtom()
	if err != nil {
		log.Fatal(err)
	}

	rss, err := feed.ToRss()
	if err != nil {
		log.Fatal(err)
	}

	json, err := feed.ToJSON()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(atom, "\n", rss, "\n", json)
}
