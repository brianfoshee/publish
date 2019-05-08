package feed

import (
	"io"
	"os"
	"strings"

	"github.com/gorilla/feeds"
)

// Feeder is an interface that content types implement so they can be in the
// RSS feed.
type Feeder interface {
	// Item returns the appropriate feeds.Item for the content type
	// implementing this method.
	Item() feeds.Item
}

func Build(content chan Feeder) error {
	feed := &feeds.Feed{
		Title: "Brian Foshee",
		Link: &feeds.Link{
			Href: "https://www.brianfoshee.com/",
			Rel:  "self",
		},
		Description: "Hi. I'm Brian Foshee. I write software.",
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
