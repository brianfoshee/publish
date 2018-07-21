package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"

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
	flag.Parse()

	fmt.Println("publishing")

	postsCh := make(chan cli.Post)

	go func() {
		for p := range postsCh {
			if !p.Draft {
				log.Printf("%s", p.Title)
			}
		}
	}()

	if err := filepath.Walk(*path, cli.PostWalker(postsCh)); err != nil {
		log.Println("error walking path: ", err)
		return
	}

	// this works as long as file processing isn't happening in goroutines
	close(postsCh)
}
