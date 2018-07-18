package cli

import (
	"log"
	"os"
	"path/filepath"
	"strings"
)

func postWalker(ch chan Post) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("error accessing path %s: %v\n", info.Name(), err)
			return err
		}

		// nothing needs to happen with directories
		if info.IsDir() {
			return nil
		}

		if strings.HasSuffix(".md", info.Name()) {
			log.Printf("opening file %s\n", info.Name())
			var p Post
			err := p.processFile(path)
			if err != nil {
				log.Printf("error processing file %s: %v", path, err)
				return err
			}
			ch <- p
		}

		return nil
	}
}
