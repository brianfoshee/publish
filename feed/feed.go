package feed

import (
	"fmt"
	"log"
	"time"

	"github.com/gorilla/feeds"
)

var author = &feeds.Author{Name: "Brian Foshee", Email: "brian@brianfoshee.com"}

func Build() {
	now := time.Now()
	feed := &feeds.Feed{
		Title:       "Brian Foshee",
		Link:        &feeds.Link{Href: "https://www.brianfoshee.com"},
		Description: "Hi. I'm Brian Foshee. I write software.",
		Author:      author,
		Created:     now,
	}

	feed.Items = []*feeds.Item{
		&feeds.Item{
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
