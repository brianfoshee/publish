package imgur

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	blackfriday "gopkg.in/russross/blackfriday.v2"
	yaml "gopkg.in/yaml.v2"
)

type Gallery struct {
	Title       string    `json:"title"`
	Slug        string    `json:"id"`
	Description string    `json:"description" yaml:"-"` // from md
	PublishedAt time.Time `json:"published-at" yaml:"published-at"`
	Photos      []Photo   `json:"-" yaml:"-"`

	path string // used when running retrobatch on image dir
}

func (g *Gallery) open(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	// separate frontmatter (yaml) & markdown
	arr := strings.Split(string(b), "---")
	if len(arr) != 3 {
		return fmt.Errorf("splitting file %s did not give three parts", path)
	}
	fm := arr[1]
	md := arr[2]

	// validate frontmatter contains all required fields
	if err := g.parseYAML([]byte(fm)); err != nil {
		return err
	}
	// convert markdown into html
	if err := g.parseMarkdown([]byte(md)); err != nil {
		return err
	}

	dir, _ := filepath.Split(path)
	g.path = dir

	// process image files too
	return g.processPhotos(dir)
}

func (g *Gallery) parseYAML(b []byte) error {
	return yaml.Unmarshal(b, g)
}

func (g *Gallery) parseMarkdown(b []byte) error {
	output := blackfriday.Run(b)
	g.Description = string(output)
	return nil
}

func (g *Gallery) processPhotos(photosPath string) error {
	galleryMdname := g.Slug + ".md"

	err := filepath.Walk(photosPath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// All images should be flattened at the gallery path
			if info.IsDir() && path != photosPath {
				log.Printf("gallery should not have dirs %s", path)
				return filepath.SkipDir
			}

			// skip the gallery md file
			if info.Name() == galleryMdname {
				return nil
			}

			// only work on md files
			ext := filepath.Ext(path)
			if ext != ".md" {
				return nil
			}

			// open the md file related to the jpg. if one doesn't exist, log.
			// store that data in g.Photos
			var p Photo
			if err := p.open(path); err != nil {
				return err
			}

			g.Photos = append(g.Photos, p)

			return nil
		})

	return err
}
