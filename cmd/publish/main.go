package main

import (
	"context"
	"crypto/sha1"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/brianfoshee/publish/blog"
	"github.com/brianfoshee/publish/feed"
	"github.com/brianfoshee/publish/imgur"
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
	serve := flag.Bool("serve", false, "Serve files in dist dir.")
	upload := flag.Bool("upload", false, "Upload files in dist dir to B2.")
	flag.Parse()

	if *clean {
		if err := os.RemoveAll("./dist"); err != nil {
			log.Println("error deleting dist directory", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if *serve {
		vnd := "application/vnd.api+json"
		mw := func(w http.ResponseWriter, r *http.Request) {
			// TODO test which of these are required to reflect in CDN
			w.Header().Set("Access-Control-Allow-Origin", "http://localhost:4200")
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			if r.Method == http.MethodOptions {
				return
			}

			log.Println(r.URL.Path)

			if r.Header.Get("Accept") == vnd {
				w.Header().Set("Content-Type", vnd)
				r.URL.Path = r.URL.Path + ".json"
			}

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
	createDir("dist/archives")
	createDir("dist/feeds")

	var feeders []feed.Feeder

	// Do blog post building
	if *blogPath != "" {
		log.Println("Building blog")

		createDir("dist/posts")
		createDir("dist/posts/page")
		createDir("dist/archives/posts")

		blogs, err := blog.Build(*blogPath, *drafts)
		if err != nil {
			fmt.Println(err)
		}
		for _, b := range blogs[:10] {
			feeders = append(feeders, b)
		}
	}

	// Do imgur building
	if *imgurPath != "" {
		log.Println("Building imgur")

		createDir("dist/galleries")
		createDir("dist/galleries/page")
		createDir("dist/photos")
		createDir("dist/archives/galleries")

		albums, err := imgur.Build(*imgurPath, *drafts)
		if err != nil {
			fmt.Println(err)
		}
		for _, a := range albums[:10] {
			feeders = append(feeders, a)
		}
	}

	if len(feeders) > 0 && (*imgurPath != "" && *blogPath != "") {
		log.Println("Building feeds")
		if err := feed.Build(feeders); err != nil {
			log.Println(err)
		}
	}

	if *upload {
		log.Println("Uploading...")
		// TODO extract uploading into a package

		account := os.Getenv("B2_ACCOUNT_ID")
		key := os.Getenv("B2_APPLICATION_KEY")
		bucketName := os.Getenv("B2_BUCKET")

		ctx := context.TODO()
		c, err := b2.NewClient(ctx, account, key, b2.UserAgent("brianfoshee"))
		if err != nil {
			log.Println("error creating new b2 client", err)
			os.Exit(1)
		}

		bucket, err := c.Bucket(ctx, bucketName)
		if err != nil {
			log.Printf("error getting %s bucket from b2 client: %v", bucketName, err)
			os.Exit(1)
		}

		// run uploads concurrently
		// cpus is how many workers will be spun up
		type files struct {
			path string
			info os.FileInfo
		}
		fileChan := make(chan files)
		wg := sync.WaitGroup{}
		workers, err := strconv.ParseInt(os.Getenv("UPLOAD_WORKERS"), 10, 64)
		if err != nil || workers == 0 {
			workers = int64(runtime.NumCPU() * 2)
		}
		for i := 0; i < int(workers); i++ {
			log.Printf("Working %d starting", i)
			wg.Add(1)
			go func() {
				for f := range fileChan {
					path := f.path
					info := f.info

					// get rid of everything before dist/ in path
					parts := strings.Split(path, "dist/")

					if strings.HasSuffix(info.Name(), ".json") {
						// destination should not have .json extension
						cleanPath := strings.TrimSuffix(parts[1], ".json")
						dst := fmt.Sprintf("www/v1/%s", cleanPath)

						if err := copyFile(ctx, bucket, path, dst, "application/vnd.api+json"); err != nil {
							log.Printf("error copying %s to %s: %s", path, dst, err)
						}
					} else if strings.HasSuffix(info.Name(), ".jpg") {
						dst := fmt.Sprintf("www/v1/%s", parts[1])

						if err := copyFile(ctx, bucket, path, dst, "image/jpeg"); err != nil {
							log.Printf("error copying %s to %s: %s", path, dst, err)
						}
					}
				}
				wg.Done()
			}()
		}

		if err := filepath.Walk("dist/", func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			fileChan <- files{path, info}

			return nil
		}); err != nil {
			log.Println("error walking dist path:", err)
			os.Exit(1)
		}

		close(fileChan)
		wg.Wait()

	}

	log.Println("Bye.")
}

func copyFile(ctx context.Context, bucket *b2.Bucket, src, dst, cont string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	obj := bucket.Object(dst)

	// check if this object already exists. If so don't upload.
	if attrs, err := obj.Attrs(ctx); err == nil {
		// calculate the sha1 sum of the file
		h := sha1.New()
		if _, err := io.Copy(h, f); err != nil {
			log.Fatalf("could not copy from file to hash: %s", err)
		}
		sum := h.Sum(nil)
		sha := fmt.Sprintf("%x", sum) // convert to string

		if attrs.SHA1 == sha {
			// log.Printf("object exists %q on b2", src)
			return nil
		}
	}

	log.Printf("copying %q to b2 %q", src, dst)

	w := obj.NewWriter(ctx)
	if cont != "" {
		ct := &b2.Attrs{
			ContentType: cont,
		}
		b2.WithAttrsOption(ct)(w)
	}
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
