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

		// nothing needs to happen with directories or hidden things
		if info.IsDir() ||
			strings.Contains(path, ".git") ||
			info.Name() == ".DS_Store" ||
			info.Name() == "README.md" {
			return nil
		}

		if strings.HasSuffix(info.Name(), ".md") {
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
