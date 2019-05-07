package feed

import (
	"fmt"
	"log"
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

func Build() {
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
