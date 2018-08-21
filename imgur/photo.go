package imgur

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v2"

	"github.com/teris-io/shortid"
)

type Photo struct {
	Slug        string    `json:"id"`
	Description string    `json:"description"`
	Gallery     string    `json:"gallery"`
	CreatedAt   time.Time `json:"created-at" yaml:"created-at"`
}

func Prepare(path string) error {
	// gallery name is taken from last element of path when separated by "/"
	gallery := strings.Split(path, "/")[0]

	err := filepath.Walk(path,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// only work on jpg files
			if !strings.HasSuffix(path, "JPG") {
				return nil
			}

			// All images should be flattened at the gallery path
			if info.IsDir() {
				return filepath.SkipDir
			}

			// Note: if this is ever run concurrently, use sid workers?
			sid, err := shortid.Generate()
			if err != nil {
				return err
			}

			// rename path. new filename will have sid instead of original name
			// TODO figure out how to move over file stats
			imgName := strings.Split(info.Name(), ".")[0]
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
