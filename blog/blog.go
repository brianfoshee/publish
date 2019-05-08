package blog

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/brianfoshee/publish/archive"
	"github.com/brianfoshee/publish/feed"
)

// TODO this should eventually take one channel and the main method will fan out
// from there to feeder, archiver, etc.
func Build(path string, drafts bool, feeder chan feed.Feeder) {
	postsCh := make(chan Post)

	go func() {
		if err := filepath.Walk(path, postWalker(postsCh)); err != nil {
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
		if p.Slug != "" && (drafts || !p.Draft) {
			d := dataPost{
				Type:       "posts",
				ID:         p.Slug,
				Attributes: p,
			}
			posts = append(posts, d)

			// Add post to feed channel for processing
			// TODO this will eventually be one channel of Posts to be returned
			// back to the main function.
			feeder <- p

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

	close(feeder)

	// Sort posts in reverse-chron
	sort.Sort(sort.Reverse(posts))

	// write out main feed and pages if more than 10 posts
	// main feed is index 0-9. next page should be 10-19
	const postsPerPage = 5
	total := len(posts)
	pages := math.Ceil(float64(total) / postsPerPage)
	for i := 1; i <= int(pages); i += 1 {
		fname := fmt.Sprintf("dist/posts/page/%d.json", i)

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
		if int(pages) > i {
			prev = fmt.Sprintf("https://www.brianfoshee.com/blog/page/%d", i+1)
		}
		if i > 1 {
			next = fmt.Sprintf("https://www.brianfoshee.com/blog/page/%d", i-1)
		}

		b := base{
			Data: posts[low:high],
			Links: &links{
				First: "https://www.brianfoshee.com/blog",
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

	// create archives. Feed for each month of each year.
	monthArchives := map[string]dataPosts{}
	archives := []archive.Archive{}
	for _, p := range posts {
		months := p.Attributes.PublishedAt.Format("2006-01")
		if a, ok := monthArchives[months]; ok {
			monthArchives[months] = append(a, p)
		} else {
			monthArchives[months] = dataPosts{p}
			a := archive.Archive{
				Kind:  "posts",
				Year:  p.Attributes.PublishedAt.Year(),
				Month: strings.ToLower(p.Attributes.PublishedAt.Month().String()),
			}
			archives = append(archives, a)
		}
	}

	if err := archive.WriteArchives("dist/archives/posts.json", archives); err != nil {
		log.Printf("error writing archives %s", err)
	}

	// make archives/posts/2018/february.json
	for _, v := range monthArchives {
		sort.Sort(sort.Reverse(v))

		// no bounds check required, if there's a value for this map it means
		// there's at least one element in it.
		year := v[0].Attributes.PublishedAt.Year()
		month := strings.ToLower(v[0].Attributes.PublishedAt.Month().String())

		dir := fmt.Sprintf("dist/archives/posts/%d", year)
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
}

// to satisfy JSONAPI
type dataPost struct {
	Type       string `json:"type"`
	ID         string `json:"id"`
	Attributes Post   `json:"attributes"`
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
