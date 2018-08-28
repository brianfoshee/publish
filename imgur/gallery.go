package imgur

import (
	"fmt"
	"io/ioutil"
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
}

func (g *Gallery) open(path string) error {
	dir, _ := filepath.Split(path)

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
	if len(arr) != 2 {
		return fmt.Errorf("splitting file did not give two parts")
	}
	fm := arr[0]
	md := arr[1]

	// validate frontmatter contains all required fields
	if err := g.parseYAML([]byte(fm)); err != nil {
		return err
	}
	// convert markdown into html
	if err := g.parseMarkdown([]byte(md)); err != nil {
		return err
	}

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

func (g *Gallery) processPhotos(path string) error {
	return nil
	/*

		err := filepath.Walk(path,
			func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				// only work on jpg files
				ext := filepath.Ext(path)
				if ext != "JPG" {
					return nil
				}

				// All images should be flattened at the gallery path
				if info.IsDir() {
					return filepath.SkipDir
				}

				return nil
			})

		return err
	*/
}
