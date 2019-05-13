package imgur

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/brianfoshee/publish/archive"
	"github.com/brianfoshee/publish/jsonapi"
	"github.com/brianfoshee/publish/utils"
)

/* ***Galleries and Photos***
--imgur-path is similar to --blog-path, it's the dir where photo gallery md and photo md and jpg files are

New flag, --preparePics=path, which renames each picture with a new shortID and creates a md file with that name too.
The md file will have some basic info like the gallery it belongs to (can get this from the base dir) and the ID, with
empty fields that need to be filled in.

On build, go through pics path folders to find a base gallery folder containing gallery md files and image/md files.
Parse gallery md and generate gallery pages. For every image in the gallery folder, add it to the gallery data structure.
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
					Gallery: jsonapi.Base{
						Data: relationshipData{
							ID:   g.Slug,
							Type: "galleries",
						},
					},
				}

				base := jsonapi.Base{
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
				Photos: jsonapi.Base{
					Data: photos,
				},
			}

			galleryBase := jsonapi.Base{
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
			// not necessary anymore - images are processed by imagemagick in a
			// github action.
		}
	}

	// Sort galleries in reverse-chron
	sort.Sort(sort.Reverse(galleries))

	// write out paginated index pages
	if err := utils.Paginate(galleries); err != nil {
		log.Println(err)
	}

	// create archives. Feed for each month of each year.
	monthArchives := map[string]dataGalleries{}
	archives := []archive.Archive{}
	for _, g := range galleries {
		months := g.Attributes.PublishedAt.Format("2006-01")
		if a, ok := monthArchives[months]; ok {
			monthArchives[months] = append(a, g)
		} else {
			monthArchives[months] = dataGalleries{g}
			a := archive.Archive{
				Kind:  "galleries",
				Year:  g.Attributes.PublishedAt.Year(),
				Month: strings.ToLower(g.Attributes.PublishedAt.Month().String()),
			}
			archives = append(archives, a)
		}
	}

	if err := archive.WriteArchives("dist/archives/galleries.json", archives); err != nil {
		log.Printf("error writing archives %s", err)
	}

	// make archives/galleries/2018/february.json
	for _, v := range monthArchives {
		sort.Sort(sort.Reverse(v))

		// no bounds check required, if there's a value for this map it means
		// there's at least one element in it.
		year := v[0].Attributes.PublishedAt.Year()
		month := strings.ToLower(v[0].Attributes.PublishedAt.Month().String())

		dir := fmt.Sprintf("dist/archives/galleries/%d", year)
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
	Photos jsonapi.Base `json:"photos"`
}

type dataPhoto struct {
	Type          string                `json:"type"`
	ID            string                `json:"id"`
	Attributes    Photo                 `json:"attributes"`
	Relationships *galleryRelationships `json:"relationships,omitempty"`
}

type galleryRelationships struct {
	Gallery jsonapi.Base `json:"gallery"`
}

type relationshipData struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// dataGalleries is a type to use with sort.Sort
type dataGalleries []dataGallery

func (d dataGalleries) Len() int { return len(d) }

func (d dataGalleries) Less(i, j int) bool {
	return d[i].Attributes.PublishedAt.Before(d[j].Attributes.PublishedAt)
}

func (d dataGalleries) Swap(i, j int) { d[i], d[j] = d[j], d[i] }

func (d dataGalleries) PerPage() int {
	return 10
}

func (d dataGalleries) Kind() string {
	return "galleries"
}

func (d dataGalleries) Data() []interface{} {
	arr := make([]interface{}, len(d))
	for i, g := range d {
		arr[i] = g
	}
	return arr
}

// dataPhotos is a type to use with sort.Sort
type dataPhotos []dataPhoto

func (d dataPhotos) Len() int { return len(d) }

func (d dataPhotos) Less(i, j int) bool {
	return d[i].Attributes.CreatedAt.Before(d[j].Attributes.CreatedAt)
}

func (d dataPhotos) Swap(i, j int) { d[i], d[j] = d[j], d[i] }
