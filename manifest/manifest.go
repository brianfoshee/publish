package manifest

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Manifest map[string]string

type manifestFile struct {
	Manifest  Manifest  `json:"manifest"`
	UpdatedAt time.Time `json:"updated_at"`
}

func Generate() error {
	prefix := os.Getenv("KV_PREFIX")
	if prefix == "" {
		prefix = "www/v1"
	}

	manifest := Manifest{}

	if err := filepath.Walk("dist/", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || info.Size() == 0 {
			return nil
		}

		// get rid of everything before dist/ in path
		parts := strings.Split(path, "dist/")

		var dst string
		if strings.HasSuffix(info.Name(), ".json") {
			// destination should not have .json extension
			cleanPath := strings.TrimSuffix(parts[1], ".json")
			dst = fmt.Sprintf("%s/%s", prefix, cleanPath)
		} else if !strings.HasSuffix(info.Name(), ".jpg") {
			// handle everything other than images. feeds, js, html etc
			dst = fmt.Sprintf("%s/%s", prefix, parts[1])
		}

		if dst != "" {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()

			// create sha1 hash of file's contents
			h := sha1.New()
			if _, err := io.Copy(h, f); err != nil {
				return fmt.Errorf("could not copy from file to hash: %v", err)
			}
			sum := h.Sum(nil)
			sha := fmt.Sprintf("%x", sum) // convert to string

			// add file to manifest
			manifest[dst] = sha
		}

		return nil
	}); err != nil {
		return fmt.Errorf("manifest: error walking dist path: %v", err)
	}

	// write manifest file
	mf := manifestFile{Manifest: manifest, UpdatedAt: time.Now()}
	f, err := os.Create("dist/manifest.json")
	if err != nil {
		return fmt.Errorf("error writing manifest file: %v", err)
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(mf); err != nil {
		return fmt.Errorf("error encoding json to manifest file: %v", err)
	}

	return nil
}
