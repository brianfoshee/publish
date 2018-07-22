package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"

	"github.com/brianfoshee/cli"
)

// output directory format
// dist -|
//		posts.json
//		posts -|
//			   im-taking-a-year-off.json
//			   archives.json
//			   archives -|
//						2018.json
//					    2018 -|
//							 february.json
//							 march.json
//			   page -|
//						2.json

// check for dist folder, create it if it doesn't exist
// read all md files in current folder (recursive)
//		separate frontmatter (yaml) & markdown
//		validate frontmatter contains all required fields
//		convert markdown into html
// if drafts flag is true, put draft posts in the feeds
// generate main feed of latest 10 posts
// generate individual post files
// generate page 2 etc feed from other groups of 10
// generate archive feeds eg 2018/feb 2018/march etc
// push to B2
// bust cloudflare cache

func main() {
	path := flag.String("path", "./", "Path with blog post markdown files")
	drafts := flag.Bool("drafts", false, "Include drafts in generated feeds")
	clean := flag.Bool("clean", false, "Remove generated files")
	flag.Parse()

	if *clean {
		if err := os.RemoveAll("./dist"); err != nil {
			log.Println("error deleting dist directory", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if _, err := os.Stat("dist"); os.IsNotExist(err) {
		os.Mkdir("dist", os.ModeDir|os.ModePerm)
		os.Mkdir("dist/posts", os.ModeDir|os.ModePerm)
		os.Mkdir("dist/posts/page", os.ModeDir|os.ModePerm)
		os.Mkdir("dist/posts/archives", os.ModeDir|os.ModePerm)
	}

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
		// add to feed if post has a Slug. If drafts flag is true, include
		// drafts.
		// TODO when parsing yaml does validation, slug check won't be necessary
		if p.Slug != "" && (*drafts || !p.Draft) {
			d := dataPost{
				Type:       "posts",
				ID:         p.Slug,
				Attributes: p,
			}
			posts = append(posts, d)

			// create individual post file
			f, err := os.Create("dist/posts/" + p.Slug + ".json")
			if err != nil {
				log.Printf("could not create file for post %q: %s", p.Slug, err)
			}
			if err := json.NewEncoder(f).Encode(base{d}); err != nil {
				// don't return here, keep going with the logged error.
				log.Printf("error encoding post to json %s: %s", p.Slug, err)
			}
			f.Close()
		}
	}

	// Sort posts in reverse-chron
	sort.Sort(sort.Reverse(posts))

	// main feed is index 0-9. next page should be 10-19
	total := len(posts)
	pages := math.Ceil(float64(total) / 10.0)
	log.Printf("total: %d pages: %f", total, pages)
	for i := 1; i <= int(pages); i += 1 {
		fname := fmt.Sprintf("dist/posts/page/%d.json", i)
		if i == 1 {
			fname = "dist/posts.json"
		}

		f, err := os.Create(fname)
		if err != nil {
			log.Printf("could not open %s", fname, err)
			return
		}

		low := 10*i - 10
		high := 10 * i
		if high > total {
			high = total
		}

		if err := json.NewEncoder(f).Encode(base{posts[low:high]}); err != nil {
			log.Printf("error encoding posts page %d: %s", i, err)
		}
		f.Close()
	}
}

// to satisfy JSONAPI
type dataPost struct {
	Type       string   `json:"type"`
	ID         string   `json:"id"`
	Attributes cli.Post `json:"attributes"`
}

// base is the base JSONAPI for either arrays or individual structs
type base struct {
	Data interface{} `json:"data"`
}

// dataPosts is a type to use with sort.Sort etc
type dataPosts []dataPost

func (d dataPosts) Len() int { return len(d) }

func (d dataPosts) Less(i, j int) bool {
	return d[i].Attributes.PublishedAt.Before(d[j].Attributes.PublishedAt)
}

func (d dataPosts) Swap(i, j int) { d[i], d[j] = d[j], d[i] }
