package imgur

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	blackfriday "gopkg.in/russross/blackfriday.v2"
	yaml "gopkg.in/yaml.v2"

	"github.com/teris-io/shortid"
)

type Photo struct {
	Slug        string    `json:"id"`
	Description string    `json:"description" yaml:"-"`
	Gallery     string    `json:"gallery"`
	CreatedAt   time.Time `json:"created-at" yaml:"created-at"`
}

func (p *Photo) parseYAML(b []byte) error {
	return yaml.Unmarshal(b, p)
}

func (p *Photo) parseMarkdown(b []byte) error {
	output := blackfriday.Run(b)
	p.Description = string(output)
	return nil
}

func Prepare(path string) error {
	// gallery name is taken from last element of path when separated by "/"
	gallery := strings.Replace(path, filepath.Dir(path)+"/", "", 1)

	galleryMD := filepath.Dir(path) + "/" + gallery + ".md"
	if !fileExists(galleryMD) {
		// TODO: check if gallery.md file exists. If not, create it.
	}

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

			// some base variables.
			dir, fileName := filepath.Split(path)
			imgName := strings.Split(fileName, ".")[0]

			// no need to process if a md file exists for this md file
			mdName := dir + imgName + ".md"
			if fileExists(mdName) {
				return nil
			}

			// Note: if this is ever run concurrently, use sid workers?
			sid, err := shortid.Generate()
			if err != nil {
				return err
			}

			// rename path. new filename will have sid instead of original name
			nfName := strings.Replace(path, imgName, sid, 1)
			nf, err := os.Create(nfName)
			if err != nil {
				return err
			}
			defer nf.Close()

			img, err := os.Open(path)
			if err != nil {
				return err
			}
			defer img.Close()
			if _, err := io.Copy(nf, img); err != nil {
				return err
			}

			// Remove the original file
			if err := os.Remove(path); err != nil {
				return err
			}

			// Create sid.md file with some default contents
			p := &Photo{
				CreatedAt: info.ModTime(),
				Slug:      sid,
				Gallery:   gallery,
			}

			fname := filepath.Dir(path) + "/" + p.Slug + ".md"
			f, err := os.Create(fname)
			if err != nil {
				return err
			}
			defer f.Close()

			if err := yaml.NewEncoder(f).Encode(p); err != nil {
				return err
			}

			f.Write([]byte("---"))
			f.Write([]byte("\n"))

			return nil
		})

	return err
}

func fileExists(name string) bool {
	if _, err := os.Stat(name); err == nil {
		return true
	}
	return false
}
