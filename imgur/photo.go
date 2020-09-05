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

type Gallery struct {
	Title string
	Date  string
}

func (g *Gallery) parseYAML(b []byte) error {
	return yaml.Unmarshal(b, g)
}

type Photo struct {
	Title       string
	Album       string
	URL         string
	Description string    `yaml:"-"`
	CreatedAt   time.Time `yaml:"date"`
	// TODO add camera/lens/settings details
}

func (p *Photo) parseYAML(b []byte) error {
	return yaml.Unmarshal(b, p)
}

func (p *Photo) parseMarkdown(b []byte) error {
	output := blackfriday.Run(b)
	p.Description = string(output)
	return nil
}

// gallery file goes in hugoPath/content/px/GALLERY.md
// img md files go in hugoPath/content/img/GALLERY/IMG.md
// img files go in hugoPath/imgur/GALLERY/IMG.jpg
func Prepare(galleryPath string, hugoPath string) error {
	// gallery name is taken from last element of path when separated by "/"
	gallery := strings.Replace(galleryPath, filepath.Dir(galleryPath)+"/", "", 1)

	galleryMD := hugoPath + "/content/px/" + gallery + ".md"
	// create gallery file if it doesn't exist
	if !fileExists(galleryMD) {
		f, err := os.Create(galleryMD)
		if err != nil {
			return fmt.Errorf("error creating galleryMD file %s: %v", galleryMD, err)
		}
		defer f.Close()

		g := Gallery{
			Title: gallery,
			Date:  time.Now().Format("2006-01-02"),
		}

		f.Write([]byte("---"))
		f.Write([]byte("\n"))

		if err := yaml.NewEncoder(f).Encode(g); err != nil {
			return err
		}

		f.Write([]byte("---"))
		f.Write([]byte("\n"))
	}

	// check to see if hugoPath/content/img/GALLERY exists. if not, create it
	createDir(hugoPath + "/content/img/" + gallery)

	// img md files go in hugoPath/content/img/GALLERY/IMG.md
	// img files go in hugoPath/imgur/GALLERY/IMG.jpg
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
			if strings.ToLower(ext) != ".jpeg" {
				return nil
			}

			// some base variables.
			_, fileName := filepath.Split(path)
			imgName := strings.Split(fileName, ".")[0]
			mdOutDir := hugoPath + "/content/img/" + gallery + "/"

			// no need to process if a md file exists for this jpg file
			mdName := mdOutDir + imgName + ".md"
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
			// output should be jpg
			nfName = strings.Replace(nfName, ext, ".jpg", 1)
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
				CreatedAt: tm.UTC(),
				URL:       "/img/" + sid,
				Album:     gallery,
			}

			fname := mdOutDir + sid + ".md"
			f, err := os.Create(fname)
			if err != nil {
				return err
			}
			defer f.Close()

			f.Write([]byte("---"))
			f.Write([]byte("\n"))

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

func createDir(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.Mkdir(dir, os.ModeDir|os.ModePerm)
	}
}
