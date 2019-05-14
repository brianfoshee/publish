package blog

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/brianfoshee/publish/archive"
	"github.com/brianfoshee/publish/jsonapi"
	"github.com/brianfoshee/publish/utils"
)

func Build(path string, drafts bool) ([]Post, error) {
	postsCh := make(chan Post)

	go func() {
		if err := filepath.Walk(path, postWalker(postsCh)); err != nil {
			log.Println("error walking blog path: ", err)
		}
		// this works as long as file processing isn't happening in goroutines
		close(postsCh)
	}()

	var posts dataPosts
	var returnPosts []Post
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
			returnPosts = append(returnPosts, p)

			// create individual post file
			f, err := os.Create("dist/posts/" + p.Slug + ".json")
			if err != nil {
				log.Printf("could not create file for post %q: %s", p.Slug, err)
				continue
			}
			if err := json.NewEncoder(f).Encode(jsonapi.Base{Data: d}); err != nil {
				// don't return here, keep going with the logged error.
				log.Printf("error encoding post to json %s: %s", p.Slug, err)
			}
			f.Close()
		}
	}

	// Sort posts in reverse-chron
	sort.Sort(sort.Reverse(posts))

	// write out paginated index pages
	if err := utils.Paginate(posts); err != nil {
		log.Println("Paginate: ", err)
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
		if err := json.NewEncoder(f).Encode(jsonapi.Base{Data: v}); err != nil {
			log.Printf("error encoding archives %s: %s", fname, err)
		}
		f.Close()
	}

	return returnPosts, nil
}

// to satisfy JSONAPI
type dataPost struct {
	Type       string `json:"type"`
	ID         string `json:"id"`
	Attributes Post   `json:"attributes"`
}

// dataPosts is a type to use with sort.Sort etc
type dataPosts []dataPost

func (d dataPosts) Len() int { return len(d) }

func (d dataPosts) Less(i, j int) bool {
	return d[i].Attributes.PublishedAt.Before(d[j].Attributes.PublishedAt)
}

func (d dataPosts) Swap(i, j int) { d[i], d[j] = d[j], d[i] }

func (d dataPosts) PerPage() int {
	return 5
}

func (d dataPosts) Kind() string {
	return "posts"
}

func (d dataPosts) Data() []interface{} {
	arr := make([]interface{}, len(d))
	for i, g := range d {
		arr[i] = g
	}
	return arr
}
