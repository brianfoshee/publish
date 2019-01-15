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

	"github.com/rwcarlsen/goexif/exif"
	"github.com/teris-io/shortid"
)

type Photo struct {
	Slug        string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description" yaml:"-"`
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

func Prepare(galleryPath string) error {
	// gallery name is taken from last element of path when separated by "/"
	gallery := strings.Replace(galleryPath, filepath.Dir(galleryPath)+"/", "", 1)

	galleryMD := filepath.Dir(galleryPath) + "/" + gallery + ".md"
	if !fileExists(galleryMD) {
		// TODO: check if gallery.md file exists. If not, create it.
	}

	err := filepath.Walk(galleryPath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// All images should be flattened at the gallery path
			if info.IsDir() && path != galleryPath {
				return filepath.SkipDir
			}

			// only work on jpg files
			ext := filepath.Ext(path)
			if strings.ToLower(ext) != ".jpg" {
				return nil
			}

			// some base variables.
			dir, fileName := filepath.Split(path)
			imgName := strings.Split(fileName, ".")[0]

			// no need to process if a md file exists for this jpg file
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
			nfName = strings.Replace(nfName, ext, strings.ToLower(ext), 1)
			if err := os.Rename(path, nfName); err != nil {
				return err
			}

			// get the exif DateTime for when the image was created
			imf, err := os.Open(nfName)
			if err != nil {
				return nil
			}
			defer imf.Close()

			x, err := exif.Decode(imf)
			if err != nil {
				return err
			}

			tm, err := x.DateTime()
			if err != nil {
				return err
			}
			// lat, long, _ := x.LatLong()

			// Create sid.md file with some default contents
			p := &Photo{
				CreatedAt: tm,
				Slug:      sid,
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

// open takes a markdown file represnting an image
func (p *Photo) open(path string) error {
	dir, mdfile := filepath.Split(path)
	imgName := strings.Replace(mdfile, ".md", ".jpg", 1)
	imgPath := dir + imgName
	if !fileExists(imgPath) {
		return fmt.Errorf("img for md file doesn't exist: %s", imgPath)
	}

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
	if err := p.parseYAML([]byte(fm)); err != nil {
		return err
	}
	// convert markdown into html
	if err := p.parseMarkdown([]byte(md)); err != nil {
		return err
	}

	return nil
}

func fileExists(name string) bool {
	if _, err := os.Stat(name); err == nil {
		return true
	}
	return false
}
