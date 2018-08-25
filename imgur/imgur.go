package imgur

import (
	"log"
	"os"
	"path/filepath"
	"strings"
)

/* ***Galleries and Photos***
--pics-path is similar to --blog-path, it's the dir where photo gallery md and photo md and jpg files are

New flag, --preparePics=path, which renames each picture with a new shortID and creates a md file with that name too.
The md file will have some basic info like the gallery it belongs to (can get this from the base dir) and the ID, with
empty fields that need to be filled in.

On build, go through pics path folders to find a base gallery folder containing gallery md files and image/md files.
Parse gallery md and generate gallery pages. For every image in the gallery folder, add it to the gallery data structure.
For every image, run retrobatch via command line to convert original image into new files in the build directory folder.
  * get from http://flyingmeat.com/download/latest/#retrobatch
Validate: a md file with the name of the folder exists (08/iceland-year-off/ must contain iceland-year-off.md)
*/

func Build(path string, drafts bool) {
	galleryCh := make(chan Gallery)

	go func() {
		if err := filepath.Walk(path, imgurWalker(galleryCh)); err != nil {
			log.Println("error walking imgur path: ", err)
			return
		}
		// this works as long as file processing isn't happening in goroutines
		close(galleryCh)
	}()
}

func imgurWalker(ch chan Gallery) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("error accessing path %s: %v\n", path, err)
			return err
		}

		// skip git dir
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}

		// find a base gallery directory containing md files and jpg files
		if strings.HasSuffix(info.Name(), ".md") && info.Name() != "README.md" {
			var g Gallery
			/*
				err := p.processFile(path)
				if err != nil {
					log.Printf("error processing file %s: %v", path, err)
					return nil
				}
			*/
			ch <- g
		}

		return nil
	}
}
