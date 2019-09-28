package feed

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/gorilla/feeds"
)

// Feeder is an interface that content types implement so they can be in the
// RSS feed.
type Feeder interface {
	// Item returns the appropriate feeds.Item for the content type
	// implementing this method.
	Item() feeds.Item
}

func Build(content []Feeder) error {
	// If there's nothing in the feed, don't generate it.
	if len(content) == 0 {
		return nil
	}

	year := time.Now().Year()

	feed := &feeds.Feed{
		Title: "Brian Foshee",
		Link: &feeds.Link{
			Href: "https://www.brianfoshee.com/",
			Rel:  "self",
		},
		Author:      &feeds.Author{Name: "Brian Foshee", Email: "brian@brianfoshee.com"},
		Description: "Hi. I'm Brian Foshee. I write software.",
		Copyright:   fmt.Sprintf("Copyright 2017 - %d Brian Foshee", year),
	}

	for _, f := range content {
		item := f.Item()
		feed.Items = append(feed.Items, &item)
	}

	feed.Sort(func(a, b *feeds.Item) bool {
		return a.Created.After(b.Created)
	})

	// set the published/updated field on the feed itself
	feed.Created = feed.Items[0].Created

	atomFeed := (&feeds.Atom{Feed: feed}).AtomFeed()
	atomFeed.Link.Href = "https://www.brianfoshee.com/feeds/atom"

	atom, err := feeds.ToXML(atomFeed)
	if err != nil {
		return err
	}

	rssFeed := (&feeds.Rss{Feed: feed}).RssFeed()
	rssFeed.Link.Href = "https://www.brianfoshee.com/feeds/rss"
	rss, err := feed.ToRss()
	if err != nil {
		return err
	}

	jsonFeed := (&feeds.JSON{Feed: feed}).JSONFeed()
	jsonFeed.FeedUrl = "https://www.brianfoshee.com/feeds/json"
	js, err := jsonFeed.ToJSON()
	if err != nil {
		return err
	}

	rssf, err := os.Create("dist/feeds/rss")
	if err != nil {
		return err
	}
	io.Copy(rssf, strings.NewReader(rss))

	atomf, err := os.Create("dist/feeds/atom")
	if err != nil {
		return err
	}
	io.Copy(atomf, strings.NewReader(atom))

	jsonf, err := os.Create("dist/feeds/json")
	if err != nil {
		return err
	}
	io.Copy(jsonf, strings.NewReader(js))

	return nil
}
