package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/brianfoshee/cli"
)

// output directory format
// dist -|
//		drafts.json
//		drafts -|
//			   one-week-off-grid.json
//		posts.json
//		posts -|
//			   im-taking-a-year-off.json
//			   archives.json
//			   archives -|
//						2018.json
//					    2018 -|
//							 february.json
//							 march.json

// check for dist folder, create it if it doesn't exist
// read all md files in current folder (recursive)
//		separate frontmatter (yaml) & markdown
//		validate frontmatter contains all required fields
//		convert markdown into html
// if drafts flag is true, put draft posts in the feeds
// generate main feed of latest 10 posts
// generate page 2 etc feed from other groups of 10
// generate archive feeds eg 2018/feb 2018/march etc
// push to B2
// bust cloudflare cache

func main() {
	path := flag.String("path", "./", "Path with blog post markdown files")
	// drafts := flag.Bool("drafts", false, "Include drafts in generated feeds")
	flag.Parse()

	postsCh := make(chan cli.Post)

	go func() {
		if err := filepath.Walk(*path, cli.PostWalker(postsCh)); err != nil {
			log.Println("error walking path: ", err)
			return
		}
		// this works as long as file processing isn't happening in goroutines
		close(postsCh)
	}()

	var posts dataPosts
	for p := range postsCh {
		if !p.Draft {
			d := dataPost{
				Type:       "posts",
				ID:         p.Slug,
				Attributes: p,
			}
			posts = append(posts, d)
			f, err := os.Create("dist/posts/" + p.Slug + ".json")
			if err != nil {
				log.Printf("could not open file for post %q: %s", p.Slug, err)
			}
			defer f.Close()
			if err := json.NewEncoder(f).Encode(base{d}); err != nil {
				log.Printf("error encoding post to json %s: %s", p.Slug, err)
			}
		}
	}

	sort.Sort(sort.Reverse(posts))

	// write out posts.json feed
	// TODO only 10 latest posts
	f, err := os.Create("dist/posts.json")
	if err != nil {
		log.Println("could not open posts.json", err)
		return
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(base{posts}); err != nil {
		log.Println("error encoding posts", err)
		return
	}
}

type dataPost struct {
	Type       string   `json:"type"`
	ID         string   `json:"id"`
	Attributes cli.Post `json:"attributes"`
}

type base struct {
	Data interface{} `json:"data"`
}

type dataPosts []dataPost

func (d dataPosts) Len() int {
	return len(d)
}

func (d dataPosts) Less(i, j int) bool {
	return d[i].Attributes.PublishedAt.Before(d[j].Attributes.PublishedAt)
}

func (d dataPosts) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}
