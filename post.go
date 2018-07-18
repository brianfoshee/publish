package cli

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	blackfriday "gopkg.in/russross/blackfriday.v2"
	yaml "gopkg.in/yaml.v2"
)

type Post struct {
	Title       string
	Slug        string
	Description string
	Draft       bool
	PublishedAt time.Time `yaml:"published-at"`
	Body        string    `yaml:"-"` // should this be byte?
}

func (p *Post) ParseYAML(b []byte) error {
	return yaml.Unmarshal(b, p)
}

func (p *Post) ParseMarkdown(b []byte) error {
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
	if len(arr) != 2 {
		return fmt.Errorf("splitting file did not give two parts")
	}
	fm := arr[0]
	md := arr[1]

	// validate frontmatter contains all required fields
	if err := p.ParseYAML([]byte(fm)); err != nil {
		return err
	}
	// convert markdown into html
	if err := p.ParseMarkdown([]byte(md)); err != nil {
		return err
	}

	return nil
}
