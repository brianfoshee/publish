package main

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/brianfoshee/publish/blog"
	"github.com/brianfoshee/publish/feed"
	"github.com/brianfoshee/publish/imgur"
	"github.com/brianfoshee/publish/manifest"
	"github.com/kurin/blazer/b2"
)

func main() {
	blogPath := flag.String("blog-path", "", "Path with blog post markdown files")
	imgurPath := flag.String("imgur-path", "", "Path with photo gallery markdown files and images")
	preparePics := flag.String("prepare-pics", "", "Path to a gallery of photos to prepare")
	drafts := flag.Bool("drafts", false, "Include drafts in generated feeds")
	clean := flag.Bool("clean", false, "Remove generated files")
	serve := flag.Bool("serve", false, "Serve files in dist dir.")
	uploadB2 := flag.Bool("uploadb2", false, "Upload images in dist dir to B2.")
	uploadCF := flag.Bool("uploadcf", false, "Upload json files in dist dir to Cloudflare.")
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

	if *blogPath != "" || *imgurPath != "" {
		// make sure directories are created before building
		createDir("dist")
		createDir("dist/archives")
		createDir("dist/feeds")
	}

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

		log.Println("Generating Manifest File")
		if err := manifest.Generate(); err != nil {
			log.Println(err)
		}
	}

	if *uploadB2 {
		log.Println("Uploading images to B2...")
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

		fileChan := make(chan string)
		wg := sync.WaitGroup{}
		workers, err := strconv.ParseInt(os.Getenv("UPLOAD_WORKERS"), 10, 64)
		if err != nil || workers == 0 {
			workers = int64(runtime.NumCPU() * 2)
		}
		// run uploads concurrently
		for i := 0; i < int(workers); i++ {
			log.Printf("Working %d starting", i)
			wg.Add(1)
			go func() {
				for path := range fileChan {

					// get rid of everything before dist/ in path
					parts := strings.Split(path, "dist/")

					dst := fmt.Sprintf("www/v1/%s", parts[1])

					// only images make it into this channel to go to B2. See
					// the Walk func.
					if err := copyFile(ctx, bucket, path, dst, "image/jpeg"); err != nil {
						log.Printf("error copying %s to %s: %s", path, dst, err)
					}
				}
				wg.Done()
			}()
		}

		if err := filepath.Walk("dist/", func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			// Only upload images to B2
			if !strings.HasSuffix(info.Name(), ".jpg") {
				return nil
			}

			fileChan <- path

			return nil
		}); err != nil {
			log.Println("error walking dist path:", err)
			os.Exit(1)
		}

		close(fileChan)
		wg.Wait()

	}

	if *uploadCF {
		log.Println("Uploading to Cloudflare...")
		if err := publishToCloudflare(); err != nil {
			log.Printf("error publishing to cloudflare %v", err)
		}
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

type cfkv struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Base64 bool   `json:"base64"`
}

func publishToCloudflare() error {
	account := os.Getenv("CF_ACCOUNT_ID")
	nsid := os.Getenv("CF_NAMESPACE_ID")
	cfemail := os.Getenv("CF_AUTH_EMAIL")
	cfkey := os.Getenv("CF_AUTH_KEY")
	kvPrefix := os.Getenv("KV_PREFIX")
	if kvPrefix == "" {
		kvPrefix = "www/v1"
	}

	var kvs []cfkv

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
		if info.Name() == "manifest.json" {
			dst = "manifest.json"
		} else if strings.HasSuffix(info.Name(), ".json") {
			// destination should not have .json extension for jsonapi files
			cleanPath := strings.TrimSuffix(parts[1], ".json")
			dst = fmt.Sprintf("%s/%s", kvPrefix, cleanPath)
		} else if !strings.HasSuffix(info.Name(), ".jpg") {
			// handle everything other than images. feeds, js, html etc
			dst = fmt.Sprintf("%s/%s", kvPrefix, parts[1])
		}

		if dst != "" {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()

			var b []byte
			var b64 bool
			// base64 encode image files
			if strings.HasSuffix(info.Name(), ".png") || strings.HasSuffix(info.Name(), ".ico") {
				var buf bytes.Buffer
				encoder := base64.NewEncoder(base64.StdEncoding, &buf)
				if _, err := io.Copy(encoder, f); err != nil {
					return err
				}
				encoder.Close()

				b = buf.Bytes()
				b64 = true
			} else {
				by, err := ioutil.ReadAll(f)
				if err != nil {
					return err
				}

				b = by
			}

			kv := cfkv{
				Key:    dst,
				Value:  fmt.Sprintf("%s", b),
				Base64: b64,
			}
			kvs = append(kvs, kv)
		}

		return nil
	}); err != nil {
		log.Println("error walking dist path:", err)
		os.Exit(1)
	}

	if len(kvs) > 10000 {
		return fmt.Errorf("cannot have more than 10,000 kvs in bulk request")
	}

	if len(kvs) == 0 {
		return nil
	}

	// encode kvs as json
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(kvs); err != nil {
		return err
	}

	// make request to cf
	hc := &http.Client{
		Timeout: 10 * time.Second,
	}

	cfURL := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/storage/kv/namespaces/%s/bulk", account, nsid)
	req, err := http.NewRequest(http.MethodPut, cfURL, &buf)
	if err != nil {
		return err
	}
	req.Header.Add("X-Auth-Key", cfkey)
	req.Header.Add("X-Auth-Email", cfemail)
	req.Header.Add("Content-Type", "application/json")

	res, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	var cfres cloudflareResponse
	if err := json.NewDecoder(res.Body).Decode(&cfres); err != nil {
		return fmt.Errorf("could not decode cloudflare response")
	}

	if !cfres.Success || res.StatusCode != http.StatusOK {
		return fmt.Errorf("cloudflare response not ok %d, %v", res.StatusCode, cfres.Errors)
	}

	return nil
}

type cloudflareError struct {
	Code    int    `json:code"`
	Message string `json:"message"`
}

type cloudflareResponse struct {
	Result   interface{}       `json:"result"`
	Success  bool              `json:"success"`
	Errors   []cloudflareError `json:"errors"`
	Messages []string          `json:"messages"`
}
