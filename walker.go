package cli

import (
	"log"
	"os"
	"path/filepath"
	"strings"
)

func PostWalker(ch chan Post) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("error accessing path %s: %v\n", path, err)
			return err
		}

		// skip git dir
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}

		if strings.HasSuffix(info.Name(), ".md") && info.Name() != "README.md" {
			var p Post
			err := p.processFile(path)
			if err != nil {
				log.Printf("error processing file %s: %v", path, err)
				return nil
			}
			ch <- p
		}

		return nil
	}
}
