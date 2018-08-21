package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/brianfoshee/cli"
	"github.com/brianfoshee/cli/imgur"
	"github.com/kurin/blazer/b2"
)

/*
dist -|
	  posts.json
	  posts -|
			 im-taking-a-year-off.json
			 page -|
				   1.json
				   2.json
			 archives.json
			 archives -|
					2018.json
					2018 -|
						  february.json
	  galleries -|
			 iceland.json
			 page -|
				   1.json
				   2.json
	  photos -|
			 abc123.json
*/

func main() {
	blogPath := flag.String("blog-path", "./", "Path with blog post markdown files")
	//picsPath := flag.String("picsPath", "./", "Path with photo gallery markdown files and images")
	preparePics := flag.String("prepare-pics", "./", "Path to a gallery of photos to prepare")
	drafts := flag.Bool("drafts", false, "Include drafts in generated feeds")
	clean := flag.Bool("clean", false, "Remove generated files")
	build := flag.Bool("build", false, "Only generate files locally. No uploading.")
	serve := flag.Bool("serve", false, "Serve files in dist dir.")
	flag.Parse()

	if *clean {
		if err := os.RemoveAll("./dist"); err != nil {
			log.Println("error deleting dist directory", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if *serve {
		mw := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "http://localhost:4200")
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			if r.Method == http.MethodOptions {
				return
			}

			w.Header().Set("Content-Type", "application/vnd.api+json")
			r.URL.Path = r.URL.Path + ".json"

			http.
				StripPrefix("/www/v1/", http.FileServer(http.Dir("dist/"))).
				ServeHTTP(w, r)
		}

		http.HandleFunc("/www/v1/", mw)
		http.ListenAndServe("localhost:8080", nil)
	}

	if _, err := os.Stat("dist"); os.IsNotExist(err) {
		os.Mkdir("dist", os.ModeDir|os.ModePerm)

		os.Mkdir("dist/posts", os.ModeDir|os.ModePerm)
		os.Mkdir("dist/posts/page", os.ModeDir|os.ModePerm)
		os.Mkdir("dist/posts/archives", os.ModeDir|os.ModePerm)

		os.Mkdir("dist/galleries", os.ModeDir|os.ModePerm)
		os.Mkdir("dist/galleries/page", os.ModeDir|os.ModePerm)

		os.Mkdir("dist/photos", os.ModeDir|os.ModePerm)
	}

	if *preparePics != "" {
		if err := imgur.Prepare(*preparePics); err != nil {
			log.Printf("error preparing path %s: %q", *preparePics, err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Do Post Things
	// TODO move this into a posts package

	postsCh := make(chan cli.Post)

	go func() {
		if err := filepath.Walk(*blogPath, cli.PostWalker(postsCh)); err != nil {
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
			if err := json.NewEncoder(f).Encode(base{Data: d}); err != nil {
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
	const postsPerPage = 5
	total := len(posts)
	pages := math.Ceil(float64(total) / postsPerPage)
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

		low := postsPerPage*i - postsPerPage
		high := postsPerPage * i
		if high > total {
			high = total
		}

		next, prev := "", ""
		if i == 1 && int(pages) > i {
			prev = fmt.Sprintf("https://www.brianfoshee.com/blog/page/2")
		}
		if i > 1 {
			next = fmt.Sprintf("https://www.brianfoshee.com/blog/page/%d", i-1)
			if int(pages) > i {
				prev = fmt.Sprintf("https://www.brianfoshee.com/blog/page/%d", i+1)
			}
		}

		b := base{
			Data: posts[low:high],
			Links: &links{
				First: "https://www.brianfoshee.com/blog/page/1",
				Last:  fmt.Sprintf("https://www.brianfoshee.com/blog/page/%d", int(pages)),
				Next:  next,
				Prev:  prev,
			},
		}
		if err := json.NewEncoder(f).Encode(b); err != nil {
			log.Printf("error encoding posts page %d: %s", i, err)
		}
		f.Close()
	}

	// create archives. Feed for each year, and feed for each month of each year.
	monthArchives := map[string]dataPosts{}
	yearArchives := map[string]dataPosts{}
	for _, p := range posts {
		months := p.Attributes.PublishedAt.Format("2006-01")
		if a, ok := monthArchives[months]; ok {
			monthArchives[months] = append(a, p)
		} else {
			monthArchives[months] = dataPosts{p}
		}

		years := p.Attributes.PublishedAt.Format("2006")
		if a, ok := yearArchives[years]; ok {
			yearArchives[years] = append(a, p)
		} else {
			yearArchives[years] = dataPosts{p}
		}
	}

	for _, v := range monthArchives {
		// no bounds check required, if there's a value for this map it means
		// there's at least one element in it.
		year := v[0].Attributes.PublishedAt.Year()
		month := strings.ToLower(v[0].Attributes.PublishedAt.Month().String())

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
		if err := json.NewEncoder(f).Encode(base{Data: v}); err != nil {
			log.Printf("error encoding archives %s: %s", fname, err)
		}
		f.Close()
	}

	// TODO this is wrong. It should be a list of all months in each year with posts
	// TODO make posts/archives.json
	// TODO make posts/archives/2018.json
	for _, v := range yearArchives {
		// no bounds check required, if there's a value for this map it means
		// there's at least one element in it.
		year := v[0].Attributes.PublishedAt.Year()
		fname := fmt.Sprintf("dist/posts/archives/%d.json", year)

		f, err := os.Create(fname)
		if err != nil {
			log.Printf("could not open %s: %s", fname, err)
			continue
		}
		if err := json.NewEncoder(f).Encode(base{Data: v}); err != nil {
			log.Printf("error encoding archives %s: %s", fname, err)
		}
		f.Close()
	}

	// Only building, not uploading
	if *build {
		return
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

		// TODO do this concurrently
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
	Data  interface{} `json:"data"`
	Links *links      `json:"links,omitempty"`
}

type links struct {
	First string `json:"first"`
	Last  string `json:"last"`
	Next  string `json:"next,omitempty"`
	Prev  string `json:"prev,omitempty"`
}

// dataPosts is a type to use with sort.Sort etc
type dataPosts []dataPost

func (d dataPosts) Len() int { return len(d) }

func (d dataPosts) Less(i, j int) bool {
	return d[i].Attributes.PublishedAt.Before(d[j].Attributes.PublishedAt)
}

func (d dataPosts) Swap(i, j int) { d[i], d[j] = d[j], d[i] }
