package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/brianfoshee/cli"
	"github.com/kurin/blazer/b2"
)

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
				continue
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

	// write out main feed and pages if more than 10 posts
	// main feed is index 0-9. next page should be 10-19
	total := len(posts)
	pages := math.Ceil(float64(total) / 10.0)
	for i := 1; i <= int(pages); i += 1 {
		fname := fmt.Sprintf("dist/posts/page/%d.json", i)
		if i == 1 {
			fname = "dist/posts.json"
		}

		f, err := os.Create(fname)
		if err != nil {
			log.Printf("could not open %s: %s", fname, err)
			continue
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

	// create archives
	// TODO make archives of each year
	archives := map[string]dataPosts{}
	for _, p := range posts {
		bucket := p.Attributes.PublishedAt.Format("2006-01")
		if a, ok := archives[bucket]; ok {
			archives[bucket] = append(a, p)
		} else {
			archives[bucket] = dataPosts{p}
		}
	}

	for _, v := range archives {
		// no bounds check required, if there's a value for this map it means
		// there's at least one element in it.
		year := v[0].Attributes.PublishedAt.Year()
		month := v[0].Attributes.PublishedAt.Month().String()
		dir := fmt.Sprintf("dist/posts/archives/%d", year)
		fname := fmt.Sprintf("%s/%s.json", dir, month)

		if _, err := os.Stat(dir); os.IsNotExist(err) {
			os.Mkdir(dir, os.ModeDir|os.ModePerm)
		}

		f, err := os.Create(fname)
		if err != nil {
			log.Printf("could not open %s: %s", fname, err)
			continue
		}
		if err := json.NewEncoder(f).Encode(base{v}); err != nil {
			log.Printf("error encoding archives %s: %s", fname, err)
		}
		f.Close()
	}

	account := os.Getenv("B2_KEY_ID")
	key := os.Getenv("B2_APPLICATION_KEY")

	ctx := context.TODO()
	c, err := b2.NewClient(ctx, account, key, b2.UserAgent("brianfoshee"))
	if err != nil {
		log.Println("error creating new b2 client", err)
		os.Exit(1)
	}

	bucket, err := c.Bucket(ctx, "brianfoshee-cdn")
	if err != nil {
		log.Println("error getting brianfoshee-cdn bucket from b2 client", err)
		os.Exit(1)
	}

	if err := filepath.Walk("dist/", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.HasSuffix(info.Name(), ".json") {
			// get rid of everything before dist/ in path
			parts := strings.Split(path, "dist/")
			// destination should not have .json extension
			cleanPath := strings.TrimSuffix(parts[1], ".json")
			dst := fmt.Sprintf("www/v1/%s", cleanPath)

			log.Printf("copying %q to b2 %q", path, dst)

			if err := copyFile(ctx, bucket, path, dst); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		log.Println("error walking dist path:", err)
		os.Exit(1)
	}
}

func copyFile(ctx context.Context, bucket *b2.Bucket, src, dst string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	obj := bucket.Object(dst)
	w := obj.NewWriter(ctx)
	ct := &b2.Attrs{
		ContentType: "application/vnd.api+json",
	}
	b2.WithAttrsOption(ct)(w)
	if _, err := io.Copy(w, f); err != nil {
		w.Close()
		return err
	}
	return w.Close()
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
