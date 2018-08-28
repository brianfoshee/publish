package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/brianfoshee/cli/blog"
	"github.com/brianfoshee/cli/imgur"
	"github.com/kurin/blazer/b2"
)

/*
dist -|
	  posts.json
	  posts -|
			 im-taking-a-year-off.json
			 page -|
				   1.json
				   2.json
			 archives.json
			 archives -|
					2018.json
					2018 -|
						  february.json
	  galleries -|
			 iceland.json
			 page -|
				   1.json
				   2.json
	  photos -|
			 abc123.json
*/

func main() {
	blogPath := flag.String("blog-path", "", "Path with blog post markdown files")
	imgurPath := flag.String("imgur-path", "", "Path with photo gallery markdown files and images")
	preparePics := flag.String("prepare-pics", "", "Path to a gallery of photos to prepare")
	drafts := flag.Bool("drafts", false, "Include drafts in generated feeds")
	clean := flag.Bool("clean", false, "Remove generated files")
	build := flag.Bool("build", false, "Only generate files locally. No uploading.")
	serve := flag.Bool("serve", false, "Serve files in dist dir.")
	flag.Parse()

	if *clean {
		if err := os.RemoveAll("./dist"); err != nil {
			log.Println("error deleting dist directory", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if *serve {
		mw := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "http://localhost:4200")
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			if r.Method == http.MethodOptions {
				return
			}

			w.Header().Set("Content-Type", "application/vnd.api+json")
			r.URL.Path = r.URL.Path + ".json"

			http.
				StripPrefix("/www/v1/", http.FileServer(http.Dir("dist/"))).
				ServeHTTP(w, r)
		}

		http.HandleFunc("/www/v1/", mw)
		http.ListenAndServe("localhost:8080", nil)
	}

	if *preparePics != "" {
		log.Println("Preparing imgur")
		if err := imgur.Prepare(*preparePics); err != nil {
			log.Printf("error preparing path %s: %q", *preparePics, err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// make sure directories are created before building
	createDir("dist")

	// Do blog post building
	if *blogPath != "" {
		log.Println("Building blog")

		createDir("dist/posts")
		createDir("dist/posts/page")
		createDir("dist/posts/archives")

		blog.Build(*blogPath, *drafts)
	}

	// Do imgur building
	if *imgurPath != "" {
		log.Println("Building imgur")

		createDir("dist/galleries")
		createDir("dist/galleries/page")
		createDir("dist/photos")

		imgur.Build(*imgurPath, *drafts)
	}

	// Only building, not uploading
	if *build {
		return
	}

	account := os.Getenv("B2_KEY_ID")
	key := os.Getenv("B2_APPLICATION_KEY")

	ctx := context.TODO()
	c, err := b2.NewClient(ctx, account, key, b2.UserAgent("brianfoshee"))
	if err != nil {
		log.Println("error creating new b2 client", err)
		os.Exit(1)
	}

	bucket, err := c.Bucket(ctx, "brianfoshee-cdn")
	if err != nil {
		log.Println("error getting brianfoshee-cdn bucket from b2 client", err)
		os.Exit(1)
	}

	if err := filepath.Walk("dist/", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// TODO do this concurrently
		if strings.HasSuffix(info.Name(), ".json") {
			// get rid of everything before dist/ in path
			parts := strings.Split(path, "dist/")
			// destination should not have .json extension
			cleanPath := strings.TrimSuffix(parts[1], ".json")
			dst := fmt.Sprintf("www/v1/%s", cleanPath)

			log.Printf("copying %q to b2 %q", path, dst)

			if err := copyFile(ctx, bucket, path, dst); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		log.Println("error walking dist path:", err)
		os.Exit(1)
	}
}

func copyFile(ctx context.Context, bucket *b2.Bucket, src, dst string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	obj := bucket.Object(dst)
	w := obj.NewWriter(ctx)
	ct := &b2.Attrs{
		ContentType: "application/vnd.api+json",
	}
	b2.WithAttrsOption(ct)(w)
	if _, err := io.Copy(w, f); err != nil {
		w.Close()
		return err
	}
	return w.Close()
}

func createDir(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.Mkdir(dir, os.ModeDir|os.ModePerm)
	}
}
