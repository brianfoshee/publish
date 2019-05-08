package blog

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/gorilla/feeds"

	blackfriday "gopkg.in/russross/blackfriday.v2"
	yaml "gopkg.in/yaml.v2"
)

// Post is a blog post
type Post struct {
	Title       string    `json:"title"`
	Slug        string    `json:"id"`
	Description string    `json:"description"`
	Draft       bool      `json:"draft"`
	PublishedAt time.Time `json:"published-at" yaml:"published-at"`
	Body        string    `json:"body", yaml:"-"`
}

// Item satisfies the feed.Feeder interface
func (p Post) Item() feeds.Item {
	fullSlug := "/blog/" + p.Slug
	domain := "https://www.brianfoshee.com"
	link := domain + fullSlug
	content := strings.ReplaceAll(
		p.Body,
		`<a href="/`,
		`<a href="https://www.brianfoshee.com/`)

	return feeds.Item{
		Id:          link,
		Title:       p.Title,
		Link:        &feeds.Link{Href: link},
		Description: p.Description,
		Created:     p.PublishedAt,
		Content:     content,
	}
}

// TODO validate
func (p *Post) parseYAML(b []byte) error {
	return yaml.Unmarshal(b, p)
}

func (p *Post) parseMarkdown(b []byte) error {
	output := blackfriday.Run(b)
	p.Body = string(output)
	return nil
}

func (p *Post) processFile(name string) error {
	f, err := os.Open(name)
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
		return fmt.Errorf("splitting file %s did not give three parts", name)
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
