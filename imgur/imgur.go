package imgur

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"time"
)

/* ***Galleries and Photos***
--pics-path is similar to --blog-path, it's the dir where photo gallery md and photo md and jpg files are

New flag, --preparePics=path, which renames each picture with a new shortID and creates a md file with that name too.
The md file will have some basic info like the gallery it belongs to (can get this from the base dir) and the ID, with
empty fields that need to be filled in.

On build, go through pics path folders to find a base gallery folder containing gallery md files and image/md files.
Parse gallery md and generate gallery pages. For every image in the gallery folder, add it to the gallery data structure.
For every image, run retrobatch via command line to convert original image into new files in the build directory folder.
  * get from http://flyingmeat.com/download/latest/#retrobatch
Validate: a md file with the name of the folder exists (08/iceland-year-off/ must contain iceland-year-off.md)
*/

func Build(path string, drafts bool) {
	galleryCh := make(chan Gallery)

	go func() {
		if err := filepath.Walk(path, imgurWalker(galleryCh)); err != nil {
			log.Println("error walking imgur path: ", err)
		}
		// this works as long as file processing isn't happening in goroutines
		close(galleryCh)
	}()

	var galleries dataGalleries
	for g := range galleryCh {
		// add to feed if published at is in the past
		if time.Now().After(g.PublishedAt) || drafts {
			d := dataGallery{
				Type:       "galleries",
				ID:         g.Slug,
				Attributes: g,
			}
			galleries = append(galleries, d)

			photosIncluded := make(dataPhotos, len(g.Photos))

			// create individual photo files
			for i, p := range g.Photos {
				pd := dataPhoto{
					Type:       "photos",
					ID:         p.Slug,
					Attributes: p,
				}
				photosIncluded[i] = pd

				pd.Relationships = &galleryRelationships{
					Gallery: base{
						Data: relationshipData{
							ID:   g.Slug,
							Type: "galleries",
						},
					},
				}

				base := base{
					Data:     pd,
					Included: []interface{}{d},
				}

				f, err := os.Create("dist/photos/" + p.Slug + ".json")
				if err != nil {
					log.Printf("could not create file for photo %q: %s", p.Slug, err)
					continue
				}
				if err := json.NewEncoder(f).Encode(base); err != nil {
					// don't return here, keep going with the logged error.
					log.Printf("error encoding photo to json %s: %s", p.Slug, err)
				}
				f.Close()
			}

			// sort photos within a gallery
			sort.Slice(photosIncluded, func(i, j int) bool {
				return photosIncluded[i].Attributes.CreatedAt.Before(photosIncluded[j].Attributes.CreatedAt)
			})

			photos := make([]relationshipData, len(g.Photos))
			pi := make([]interface{}, len(photosIncluded))
			for i, p := range photosIncluded {
				pi[i] = p

				photos[i] = relationshipData{
					ID:   p.Attributes.Slug,
					Type: "photos",
				}
			}

			d.Relationships = &photoRelationships{
				Photos: base{
					Data: photos,
				},
			}

			galleryBase := base{
				Data:     d,
				Included: pi,
			}

			// create individual gallery file
			f, err := os.Create("dist/galleries/" + g.Slug + ".json")
			if err != nil {
				log.Printf("could not create file for gallery %q: %s", g.Slug, err)
				continue
			}
			if err := json.NewEncoder(f).Encode(galleryBase); err != nil {
				// don't return here, keep going with the logged error.
				log.Printf("error encoding gallery to json %s: %s", g.Slug, err)
			}
			f.Close()

			// run procesing on photos
			cmd := exec.Command(
				"/Applications/Retrobatch.app/Contents/MacOS/Retrobatch",
				"--workflow",
				"/Users/brian/Code/brianfoshee-content/imgur/imgur.retrobatch",
				"--output",
				"dist/photos",
				g.path)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				log.Printf("Command finished with error: %v", err)
			}
		}
	}

	// Sort galleries in reverse-chron
	sort.Sort(sort.Reverse(galleries))

	// write out main feed, and pages if more than 10 galleries
	// main feed is index 0-9. next page should be 10-19
	const galleriesPerPage = 20
	total := len(galleries)
	pages := math.Ceil(float64(total) / galleriesPerPage)
	for i := 1; i <= int(pages); i += 1 {
		fname := fmt.Sprintf("dist/galleries/page/%d.json", i)

		f, err := os.Create(fname)
		if err != nil {
			log.Printf("could not open %s: %s", fname, err)
			continue
		}

		low := galleriesPerPage*i - galleriesPerPage
		high := galleriesPerPage * i
		if high > total {
			high = total
		}

		// 'next' is the next page of galleries. So page 1's next is page 2.
		// 'prev' is the previous page of galleries. So page 2's prev is page 1.
		next, prev := "", ""
		if int(pages) > i {
			prev = fmt.Sprintf("https://www.brianfoshee.com/px/page/%d", i+1)
		}
		if i > 1 {
			next = fmt.Sprintf("https://www.brianfoshee.com/px/page/%d", i-1)
		}

		b := base{
			Data: galleries[low:high],
			Links: &links{
				First: "https://www.brianfoshee.com/px",
				Last:  fmt.Sprintf("https://www.brianfoshee.com/px/page/%d", int(pages)),
				Next:  next,
				Prev:  prev,
			},
		}
		if err := json.NewEncoder(f).Encode(b); err != nil {
			log.Printf("error encoding posts page %d: %s", i, err)
		}
		f.Close()
	}
}

func imgurWalker(ch chan Gallery) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("error accessing path %s: %v\n", path, err)
			return err
		}

		// skip git dir
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}

		// find a base gallery directory containing md files and jpg files
		if info.IsDir() {
			_, mdname := filepath.Split(path)
			mdname = path + "/" + mdname + ".md"
			if _, err := os.Stat(mdname); err == nil || os.IsExist(err) {
				var g Gallery
				if err := g.open(mdname); err != nil {
					return err
				}
				ch <- g
			}

		}

		return nil
	}
}

// to satisfy JSONAPI
type dataGallery struct {
	Type          string              `json:"type"`
	ID            string              `json:"id"`
	Attributes    Gallery             `json:"attributes"`
	Relationships *photoRelationships `json:"relationships,omitempty"`
}

type photoRelationships struct {
	Photos base `json:"photos"`
}

type dataPhoto struct {
	Type          string                `json:"type"`
	ID            string                `json:"id"`
	Attributes    Photo                 `json:"attributes"`
	Relationships *galleryRelationships `json:"relationships,omitempty"`
}

type galleryRelationships struct {
	Gallery base `json:"gallery"`
}

type relationshipData struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// base is the base JSONAPI for either arrays or individual structs
// TODO this is duped in blog pkg
type base struct {
	Data     interface{}   `json:"data"`
	Links    *links        `json:"links,omitempty"`
	Included []interface{} `json:"included,omitempty"`
}

// TODO this is duped in blog pkg
type links struct {
	First string `json:"first"`
	Last  string `json:"last"`
	Next  string `json:"next,omitempty"`
	Prev  string `json:"prev,omitempty"`
}

// dataGalleries is a type to use with sort.Sort
type dataGalleries []dataGallery

func (d dataGalleries) Len() int { return len(d) }

func (d dataGalleries) Less(i, j int) bool {
	return d[i].Attributes.PublishedAt.Before(d[j].Attributes.PublishedAt)
}

func (d dataGalleries) Swap(i, j int) { d[i], d[j] = d[j], d[i] }

// dataPhotos is a type to use with sort.Sort
type dataPhotos []dataPhoto

func (d dataPhotos) Len() int { return len(d) }

func (d dataPhotos) Less(i, j int) bool {
	return d[i].Attributes.CreatedAt.Before(d[j].Attributes.CreatedAt)
}

func (d dataPhotos) Swap(i, j int) { d[i], d[j] = d[j], d[i] }
